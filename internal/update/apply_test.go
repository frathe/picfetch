//go:build !windows

package update

import (
	"os"
	"path/filepath"
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

	if err := applyUnix(Stage{BinaryPath: staged}, dest); err != nil {
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

	err := applyUnix(Stage{BinaryPath: staged, PlistPath: plist}, dest)
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

	err := applyUnix(Stage{BinaryPath: staged}, dest)
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

	err := applyUnix(Stage{BinaryPath: filepath.Join(dir, "missing")}, dest)
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
	real := filepath.Join(dir, "real")
	if err := os.WriteFile(real, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := applyUnix(Stage{BinaryPath: staged}, link); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("resolved dest = %q, want new", got)
	}
}
