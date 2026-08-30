//go:build !windows

package update

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestApplyUnix_ReplacesDest(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := applyUnix(Stage{BinaryPath: staged}, dest, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("dest = %q, want %q", got, "new")
	}
	st, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o755 {
		t.Errorf("dest mode = %o, want 0755", st.Mode().Perm())
	}
	if _, err := os.Stat(dest + ".old"); !os.IsNotExist(err) {
		t.Errorf("dest.old still present: %v", err)
	}
	if _, err := os.Stat(staged); err != nil {
		t.Errorf("staged binary must remain (copy, not rename): %v", err)
	}
}

func TestApplyUnix_CopiesPlist(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "PicFetch.app", "Contents", "MacOS", "picfetch")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := filepath.Join(dir, "staged.plist")
	if err := os.WriteFile(plist, []byte("PLIST"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := applyUnix(Stage{BinaryPath: staged, PlistPath: plist}, dest, ApplyOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "PicFetch.app", "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "PLIST" {
		t.Errorf("Info.plist = %q, want PLIST", got)
	}
	bin, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(bin) != "new" {
		t.Errorf("dest = %q, want new", bin)
	}
}

func TestApplyUnix_PlistCopyFailureLeavesInstalledFilesUntouched(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "PicFetch.app", "Contents", "MacOS", "picfetch")
	plistDest := filepath.Join(dir, "PicFetch.app", "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plistDest, []byte("old plist"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := applyUnix(Stage{BinaryPath: staged, PlistPath: filepath.Join(dir, "missing.plist")}, dest, ApplyOptions{})
	if err == nil {
		t.Fatal("expected missing plist error")
	}
	gotBinary, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(gotBinary) != "old binary" {
		t.Errorf("installed binary = %q, want old binary", gotBinary)
	}
	gotPlist, readErr := os.ReadFile(plistDest)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(gotPlist) != "old plist" {
		t.Errorf("installed plist = %q, want old plist", gotPlist)
	}
	for _, leftover := range []string{dest + ".new", dest + ".old", plistDest + ".new", plistDest + ".old"} {
		if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("temporary replacement %q remains: %v", leftover, err)
		}
	}
}

func TestApplyUnix_UnwritableDestLeavesOld(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory write bits")
	}

	dir := t.TempDir()
	destDir := filepath.Join(dir, "locked")
	if err := os.Mkdir(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(destDir, "picfetch")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(destDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(destDir, 0o755) })

	err := applyUnix(Stage{BinaryPath: staged}, dest, ApplyOptions{})
	if err == nil {
		t.Fatal("expected error for unwritable dest dir")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Errorf("dest = %q, want old", got)
	}
}

func TestApplyUnix_CopyFailureRestoresOld(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := applyUnix(Stage{BinaryPath: filepath.Join(dir, "missing")}, dest, ApplyOptions{})
	if err == nil {
		t.Fatal("expected error when staged binary is missing")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Errorf("dest = %q, want old after rollback", got)
	}
}

func TestApplyUnix_EvalSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := applyUnix(Stage{BinaryPath: staged}, link, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("resolved dest = %q, want new", got)
	}
}

func TestApplyUnix_RelaunchesOnlyAfterExecutableAndPlistAreInstalled(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "PicFetch.app", "Contents", "MacOS", "picfetch")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := filepath.Join(dir, "staged.plist")
	if err := os.WriteFile(plist, []byte("PLIST"), 0o644); err != nil {
		t.Fatal(err)
	}

	launches := 0
	resolvedDest, err := filepath.EvalSymlinks(dest)
	if err != nil {
		t.Fatal(err)
	}
	err = applyUnixWithLauncher(
		Stage{BinaryPath: staged, PlistPath: plist},
		dest,
		ApplyOptions{Relaunch: true},
		func(gotDest string) error {
			launches++
			if gotDest != resolvedDest {
				t.Errorf("launch dest = %q, want %q", gotDest, resolvedDest)
			}
			gotBinary, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotBinary) != "new" {
				t.Errorf("binary at launch = %q, want new", gotBinary)
			}
			gotPlist, err := os.ReadFile(filepath.Join(dir, "PicFetch.app", "Contents", "Info.plist"))
			if err != nil {
				t.Fatal(err)
			}
			if string(gotPlist) != "PLIST" {
				t.Errorf("plist at launch = %q, want PLIST", gotPlist)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if launches != 1 {
		t.Errorf("launches = %d, want 1", launches)
	}
}

func TestApplyUnix_NormalApplyDoesNotRelaunch(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := applyUnixWithLauncher(Stage{BinaryPath: staged}, dest, ApplyOptions{}, func(string) error {
		t.Fatal("normal apply must not launch")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplyUnix_RelaunchFailureReturnedAfterInstall(t *testing.T) {
	// applyUnix resolves symlinks in dest, and macOS's temp dir sits behind
	// the /var -> /private/var symlink, so resolve dir before the Path of
	// the reported ApplyError is compared against it.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("launch failed")

	err = applyUnixWithLauncher(Stage{BinaryPath: staged}, dest, ApplyOptions{Relaunch: true}, func(string) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) {
		t.Fatalf("error = %T (%v), want *ApplyError so a failed relaunch is not recorded as a failed install", err, err)
	}
	if applyErr.Op != "relaunch" {
		t.Errorf("Op = %q, want relaunch", applyErr.Op)
	}
	if applyErr.Path != dest {
		t.Errorf("Path = %q, want %q", applyErr.Path, dest)
	}
	got, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "new" {
		t.Errorf("dest after launch failure = %q, want new", got)
	}
}

func TestUnixRelaunchCommand_WaitsForOldProcessAndPassesPathAsArgument(t *testing.T) {
	dest := `/Applications/Pic Fetch & More.app/Contents/MacOS/picfetch;echo unsafe`
	cmd := unixRelaunchCommand(dest, 12345)

	if cmd.Path != "/bin/sh" {
		t.Errorf("command path = %q, want /bin/sh", cmd.Path)
	}
	wantArgs := []string{
		"/bin/sh",
		"-c",
		`while kill -0 "$1" 2>/dev/null; do sleep 0.1; done; exec "$2"`,
		"picfetch-relaunch",
		"12345",
		dest,
	}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Errorf("command args = %#v, want %#v", cmd.Args, wantArgs)
	}
	if cmd.Stderr != os.Stderr {
		t.Error("post-exit relaunch errors are not connected to PicFetch stderr")
	}
}
