//go:build windows

package update

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestApplyWindows_ReplacesDestAndKeepsOld(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch.exe")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged.exe")
	if err := os.WriteFile(staged, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := applyWindows(verifiedSwapStage(t, staged), dest, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "new" {
		t.Fatalf("dest = %q, %v; want %q", got, err, "new")
	}
	if _, err := os.Stat(dest + ".old"); err != nil {
		t.Errorf("dest.old must survive for the next launch to sweep: %v", err)
	}
	if _, err := os.Stat(staged); err != nil {
		t.Errorf("staged binary must remain (copy, not rename): %v", err)
	}
	if _, err := os.Stat(dest + ".apply.cmd"); !os.IsNotExist(err) {
		t.Errorf("apply must not write a cmd script any more")
	}
}

func TestApplyWindows_MissingStagedBinaryRestoresDest(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch.exe")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged := writeSwapFile(t, filepath.Join(dir, "staged.exe"), "new")
	stage := verifiedSwapStage(t, staged)
	if err := os.Remove(staged); err != nil {
		t.Fatal(err)
	}

	err := applyWindows(stage, dest, ApplyOptions{})
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) {
		t.Fatalf("applyWindows = %T (%v), want *ApplyError", err, err)
	}
	if applyErr.Op != "verify" {
		t.Errorf("Op = %q, want %q", applyErr.Op, "verify")
	}
	if applyErr.Path != dest {
		t.Errorf("Path = %q, want %q", applyErr.Path, dest)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error must still unwrap to its cause: %v", err)
	}

	got, readErr := os.ReadFile(dest)
	if readErr != nil || string(got) != "old" {
		t.Fatalf("dest = %q, %v; want the installed executable unchanged at %q", got, readErr, "old")
	}
	if _, err := os.Stat(dest + ".old"); !os.IsNotExist(err) {
		t.Errorf("verification must fail before creating dest.old: %v", err)
	}
}

func TestApplyWindows_RejectsUnverifiedStageBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	dest := writeSwapFile(t, filepath.Join(dir, "picfetch.exe"), "installed")
	staged := writeSwapFile(t, filepath.Join(dir, "staged.exe"), "download")
	if err := applyWindows(Stage{BinaryPath: staged}, dest, ApplyOptions{}); err == nil {
		t.Fatal("applied a stage without verified provenance")
	}
	assertSwapFile(t, dest, "installed")
	if _, err := os.Stat(dest + ".old"); !os.IsNotExist(err) {
		t.Fatalf("unexpected backup after failed verification: %v", err)
	}
}

func TestOpenVerifiedStageBinary_RejectsActiveWindowsWriter(t *testing.T) {
	path := writeSwapFile(t, filepath.Join(t.TempDir(), "staged.exe"), "download")
	stage := verifiedSwapStage(t, path)
	writer, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	handle, err := openVerifiedStageBinary(stage)
	if err == nil {
		_ = handle.Close()
		t.Fatal("admitted a source with an active writer")
	}
}

func TestWindowsRelaunchCommand_PassesThePIDInTheInheritedEnvironment(t *testing.T) {
	t.Setenv("PICFETCH_TEST_INHERITED", "kept")
	dir := filepath.Join(t.TempDir(), "Pic Fetch & More")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "picfetch.exe")
	if err := os.WriteFile(dest, []byte("MZ"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := windowsRelaunchCommand(dest, 4242)

	if cmd.Err != nil {
		t.Fatalf("command not startable: %v", cmd.Err)
	}
	if cmd.Path != dest {
		t.Errorf("command path = %q, want %q", cmd.Path, dest)
	}
	// A spaced install path must stay one argument, and no argument may be
	// added: main.go has no flag parsing, so argsToURIs would take anything
	// here for a file to open.
	if !reflect.DeepEqual(cmd.Args, []string{dest}) {
		t.Errorf("command args = %#v, want %#v", cmd.Args, []string{dest})
	}

	env := os.Environ()
	if len(cmd.Env) != len(env)+1 {
		t.Fatalf("env has %d entries, want the %d inherited ones plus the await pid", len(cmd.Env), len(env))
	}
	if !reflect.DeepEqual(cmd.Env[:len(env)], env) {
		t.Error("the inherited environment was not preserved verbatim")
	}
	if want := AwaitPIDEnv + "=4242"; cmd.Env[len(env)] != want {
		t.Errorf("last env entry = %q, want %q", cmd.Env[len(env)], want)
	}
	if !slices.Contains(cmd.Env, "PICFETCH_TEST_INHERITED=kept") {
		t.Error("an update must not strip the user's own environment variables")
	}
}
