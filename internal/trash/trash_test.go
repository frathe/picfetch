package trash

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestMoveLinux_PrefersGio(t *testing.T) {
	origGio, origTrashPut, origRun := lookupGio, lookupTrashPut, runTrashCommand
	t.Cleanup(func() { lookupGio, lookupTrashPut, runTrashCommand = origGio, origTrashPut, origRun })

	lookupGio = func() (string, error) { return "/usr/bin/gio", nil }
	lookupTrashPut = func() (string, error) {
		t.Fatal("trash-put should not be consulted when gio is present")
		return "", nil
	}

	var gotArgs []string
	runTrashCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = cmd.Args
		return nil, nil
	}

	if err := moveLinux("/tmp/photo.jpg"); err != nil {
		t.Fatalf("moveLinux() error = %v", err)
	}

	if !strings.Contains(strings.Join(gotArgs, " "), "trash /tmp/photo.jpg") {
		t.Errorf("gio args = %v, want them to run \"trash /tmp/photo.jpg\"", gotArgs)
	}
}

func TestMoveLinux_FallsBackToTrashPut(t *testing.T) {
	origGio, origTrashPut, origRun := lookupGio, lookupTrashPut, runTrashCommand
	t.Cleanup(func() { lookupGio, lookupTrashPut, runTrashCommand = origGio, origTrashPut, origRun })

	lookupGio = func() (string, error) { return "", errors.New("not found") }
	lookupTrashPut = func() (string, error) { return "/usr/bin/trash-put", nil }

	var gotPath string
	var gotArgs []string
	runTrashCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath = cmd.Path
		gotArgs = cmd.Args
		return nil, nil
	}

	if err := moveLinux("/tmp/photo.jpg"); err != nil {
		t.Fatalf("moveLinux() error = %v", err)
	}
	if !strings.Contains(gotPath, "trash-put") {
		t.Errorf("cmd.Path = %q, want it to run trash-put", gotPath)
	}
	if !strings.Contains(strings.Join(gotArgs, " "), "/tmp/photo.jpg") {
		t.Errorf("trash-put args = %v, want the path passed through", gotArgs)
	}
}

func TestMoveLinux_PreservesOrdinaryXDGAndNormalizesSnap(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultDir := filepath.Join(home, ".local", "share")
	for _, backend := range []string{"gio", "trash-put"} {
		t.Run(backend, func(t *testing.T) {
			for _, tc := range []struct {
				name, value, want string
				absent            bool
			}{
				{name: "absent", absent: true},
				{name: "empty"},
				{name: "default", value: defaultDir, want: defaultDir},
				{name: "custom", value: "/mnt/data/desktop", want: "/mnt/data/desktop"},
				{name: "custom snap component", value: "/mnt/snap/personal/data", want: "/mnt/snap/personal/data"},
				{name: "another home", value: "/home/someone-else/snap/code/257/.local/share", want: "/home/someone-else/snap/code/257/.local/share"},
				{name: "snap revision", value: filepath.Join(home, "snap/code/257/.local/share"), want: defaultDir},
				{name: "snap current", value: filepath.Join(home, "snap/code/current/.local/share"), want: defaultDir},
				{name: "snap common", value: filepath.Join(home, "snap/code/common/.local/share"), want: defaultDir},
				{name: "non revision", value: filepath.Join(home, "snap/photos/originals/.local/share"), want: filepath.Join(home, "snap/photos/originals/.local/share")},
				{name: "outside data home", value: filepath.Join(home, "snap/code/257/photos"), want: filepath.Join(home, "snap/code/257/photos")},
				{name: "traversal", value: home + "/snap/code/257/../.local/share", want: home + "/snap/code/257/../.local/share"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					origGio, origTrashPut, origRun := lookupGio, lookupTrashPut, runTrashCommand
					t.Cleanup(func() { lookupGio, lookupTrashPut, runTrashCommand = origGio, origTrashPut, origRun })
					t.Setenv("XDG_DATA_HOME", tc.value)
					if tc.absent {
						if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
							t.Fatal(err)
						}
					}
					t.Setenv("PICFETCH_TRASH_ENV_FIXTURE", "untouched")
					lookupGio = func() (string, error) {
						if backend == "gio" {
							return "/usr/bin/gio", nil
						}
						return "", errors.New("missing")
					}
					lookupTrashPut = func() (string, error) { return "/usr/bin/trash-put", nil }
					var got []string
					runTrashCommand = func(cmd *exec.Cmd) ([]byte, error) { got = cmd.Env; return nil, nil }
					if err := moveLinux("/tmp/photo.jpg"); err != nil {
						t.Fatal(err)
					}
					matches := 0
					for _, kv := range got {
						if key, value, _ := strings.Cut(kv, "="); key == "XDG_DATA_HOME" {
							matches++
							if value != tc.want {
								t.Errorf("XDG_DATA_HOME = %q, want %q", value, tc.want)
							}
						}
					}
					wantMatches := 1
					if tc.absent {
						wantMatches = 0
					}
					if matches != wantMatches {
						t.Errorf("XDG_DATA_HOME entries = %d, want %d", matches, wantMatches)
					}
					if !slices.Contains(got, "PICFETCH_TRASH_ENV_FIXTURE=untouched") {
						t.Error("unrelated environment was lost")
					}
					if value, present := os.LookupEnv("XDG_DATA_HOME"); value != tc.value || present == tc.absent {
						t.Error("parent environment changed")
					}
				})
			}
		})
	}
}

func TestMoveLinux_ReturnsErrorWhenNeitherToolInstalled(t *testing.T) {
	origGio, origTrashPut := lookupGio, lookupTrashPut
	t.Cleanup(func() { lookupGio, lookupTrashPut = origGio, origTrashPut })

	lookupGio = func() (string, error) { return "", errors.New("not found") }
	lookupTrashPut = func() (string, error) { return "", errors.New("not found") }

	if err := moveLinux("/tmp/photo.jpg"); err == nil {
		t.Error("expected an error when neither gio nor trash-put is installed")
	}
}

func TestMoveWindows_BuildsExpectedScript(t *testing.T) {
	tests := []struct {
		name       string
		makeTarget func(*testing.T) string
		method     string
		notMethod  string
	}{
		{
			name: "file",
			makeTarget: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "photo.jpg")
				if err := os.WriteFile(path, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			method:    "DeleteFile",
			notMethod: "DeleteDirectory",
		},
		{
			name:       "directory",
			makeTarget: func(t *testing.T) string { return t.TempDir() },
			method:     "DeleteDirectory",
			notMethod:  "DeleteFile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origRun := runTrashCommand
			t.Cleanup(func() { runTrashCommand = origRun })

			var gotScript string
			runTrashCommand = func(cmd *exec.Cmd) ([]byte, error) {
				for i, a := range cmd.Args {
					if a == "-Command" && i+1 < len(cmd.Args) {
						gotScript = cmd.Args[i+1]
					}
				}
				return nil, nil
			}

			path := tt.makeTarget(t)
			if err := moveWindows(path); err != nil {
				t.Fatalf("moveWindows() error = %v", err)
			}

			for _, want := range []string{
				"Microsoft.VisualBasic",
				"[Microsoft.VisualBasic.FileIO.FileSystem]::" + tt.method,
				"SendToRecycleBin",
				"catch",
				"exit 1",
				"$env:PICFETCH_TRASH_PATH",
			} {
				if !strings.Contains(gotScript, want) {
					t.Errorf("script does not contain %q:\n%s", want, gotScript)
				}
			}
			if strings.Contains(gotScript, "[Microsoft.VisualBasic.FileIO.FileSystem]::"+tt.notMethod) {
				t.Errorf("script unexpectedly contains %s:\n%s", tt.notMethod, gotScript)
			}
		})
	}
}

// Exercise real file/directory admission, but intercept the OS mutation.
func TestMoveWindows_PathIsData(t *testing.T) {
	origRun := runTrashCommand
	t.Cleanup(func() { runTrashCommand = origRun })
	t.Setenv("PICFETCH_TRASH_PATH", "inherited stale target")
	t.Setenv("PICFETCH_TRASH_ENV_CONTROL", "preserved")
	for _, directory := range []bool{false, true} {
		var baseline []string
		for _, name := range []string{
			"ordinary.jpg",
			"“;Write-Output INJECTED;#”.jpg",
			"”);Write-Output INJECTED;#“.jpg",
			"$dollar`tick; (brackets) 'single' ‘curly’.jpg",
			"café 東京 😀 [1]=100%.jpg",
		} {
			path := filepath.Join(t.TempDir(), name)
			if directory {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			runTrashCommand = func(cmd *exec.Cmd) ([]byte, error) {
				calls++
				if baseline == nil {
					baseline = slices.Clone(cmd.Args)
				}
				if !slices.Equal(cmd.Args, baseline) || strings.Contains(strings.Join(cmd.Args, "\n"), path) {
					t.Error("target path changed executable PowerShell source")
				}
				env := cmd.Environ()
				if !slices.Contains(env, "PICFETCH_TRASH_PATH="+path) || slices.Contains(env, "PICFETCH_TRASH_PATH=inherited stale target") {
					t.Error("child environment lost or replaced the exact target path")
				}
				if !slices.Contains(env, "PICFETCH_TRASH_ENV_CONTROL=preserved") {
					t.Error("unrelated environment was lost")
				}
				return nil, nil
			}
			if err := moveWindows(path); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("command calls = %d, want 1", calls)
			}
			if os.Getenv("PICFETCH_TRASH_PATH") != "inherited stale target" {
				t.Fatal("parent environment changed")
			}
		}
	}
}

func TestMoveWindows_ReturnsStatErrorBeforeRunningCommand(t *testing.T) {
	origRun := runTrashCommand
	t.Cleanup(func() { runTrashCommand = origRun })
	runTrashCommand = func(*exec.Cmd) ([]byte, error) {
		t.Fatal("command should not run for a missing target")
		return nil, nil
	}

	if err := moveWindows(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("moveWindows error = %v, want os.ErrNotExist", err)
	}
}

func TestMoveWindows_ReturnsCommandError(t *testing.T) {
	origRun := runTrashCommand
	t.Cleanup(func() { runTrashCommand = origRun })
	wantErr := errors.New("powershell failed")
	runTrashCommand = func(*exec.Cmd) ([]byte, error) { return nil, wantErr }

	if err := moveWindows(t.TempDir()); !errors.Is(err, wantErr) {
		t.Errorf("moveWindows error = %v, want %v", err, wantErr)
	}
}

// Replace only the native deletion sink with a recorder. PowerShell parses the
// production command and receives the real child environment; no file is moved.
func TestWindowsTrashTransport_PathIsData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("requires Windows PowerShell")
	}
	origRun := runTrashCommand
	t.Cleanup(func() { runTrashCommand = origRun })
	for _, directory := range []bool{false, true} {
		for _, name := range []string{"ordinary.jpg", "”);Write-Output INJECTED;#“ $dollar`tick ‘quote’ 東京.jpg"} {
			path := filepath.Join(t.TempDir(), name)
			if directory {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			runTrashCommand = func(cmd *exec.Cmd) ([]byte, error) {
				script := cmd.Args[len(cmd.Args)-1]
				if strings.Count(script, "[Microsoft.VisualBasic.FileIO.FileSystem]::") != 1 {
					t.Fatal("missing recycle seam")
				}
				script = strings.Replace(script, "[Microsoft.VisualBasic.FileIO.FileSystem]::", "[PicFetchTrashProbe]::", 1)
				prefix := `[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
Add-Type @'
using System;
public class PicFetchTrashProbe {
 public static void DeleteFile(string path, string ui, string recycle) { Record(path, ui, recycle, "file"); }
 public static void DeleteDirectory(string path, string ui, string recycle) { Record(path, ui, recycle, "directory"); }
 private static void Record(string path, string ui, string recycle, string kind) {
  if (ui != "OnlyErrorDialogs" || recycle != "SendToRecycleBin" || kind != Environment.GetEnvironmentVariable("PICFETCH_TRASH_TEST_KIND")) { throw new Exception("recycle contract changed"); }
  Console.Write(path);
 }
}
'@
`
				kind := "file"
				if directory {
					kind = "directory"
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()
				run := exec.CommandContext(ctx, cmd.Path, append(cmd.Args[1:len(cmd.Args)-1], prefix+script)...)
				run.Env = append(cmd.Environ(), "PICFETCH_TRASH_TEST_KIND="+kind)
				run.SysProcAttr = cmd.SysProcAttr
				out, err := run.Output()
				if err == nil && string(out) != path {
					t.Errorf("recycle received %q, want %q", out, path)
				}
				return out, err
			}
			if err := moveWindows(path); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("test touched native trash: %v", err)
			}
		}
	}
}
