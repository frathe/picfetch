//go:build windows

package filepicker

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWindowsPickerTransport_EmitsUTF8PathArrays(t *testing.T) {
	paths := []string{filepath.Join(t.TempDir(), "café 東京 😀.png"), filepath.Join(t.TempDir(), " spaced name.png")}
	input, err := json.Marshal(paths)
	if err != nil {
		t.Fatal(err)
	}
	// Hosted runners can spend over 20 seconds starting a cold PowerShell
	// process while the other native test packages are running.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	// Run the actual picker serializer, without opening WinForms or modifying
	// desktop state. Environment transport keeps the fixture out of script text.
	script := powerShellPickerScript(`Write-PickedPaths (ConvertFrom-Json $env:PICFETCH_PICKER_TEST_PATHS)`)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "PICFETCH_PICKER_TEST_PATHS="+string(input))
	out, err := cmd.Output()
	selected, err := decodePickedPaths(out, err)
	if err != nil {
		t.Fatalf("picker transport: %v (context: %v)", err, ctx.Err())
	}
	if len(selected) != len(paths) {
		t.Fatalf("selected %d paths, want %d", len(selected), len(paths))
	}
	for i, uri := range selected {
		if filepath.FromSlash(uri.Path()) != paths[i] {
			t.Errorf("path %d = %q, want %q", i, uri.Path(), paths[i])
		}
	}
}

// Exercise the actual WinForms properties and PowerShell parser without showing
// a native dialog. Only the modal call is replaced with its synthetic result.
func TestWindowsSaveTransport_PathIsData(t *testing.T) {
	for _, name := range []string{"ordinary.jpg", "“;Write-Output INJECTED;#” $dollar`tick ‘quote’ 東京.jpg"} {
		for _, cancelDialog := range []bool{false, true} {
			path := filepath.Join(t.TempDir(), name)
			cmd := buildPowerShellSaveCmd(path)
			script := cmd.Args[len(cmd.Args)-1]
			result := "[System.Windows.Forms.DialogResult]::OK"
			if cancelDialog {
				result = "[System.Windows.Forms.DialogResult]::Cancel"
			}
			if strings.Count(script, "$dlg.ShowDialog()") != 1 {
				t.Fatal("missing modal seam")
			}
			script = strings.Replace(script, "$dlg.ShowDialog()", result, 1)
			// Check the real save-panel fields independently of its serialized result.
			script = strings.Replace(script, "if ("+result, `if ($dlg.FileName -cne $env:PICFETCH_SAVE_PATH -or $dlg.InitialDirectory -cne $env:PICFETCH_SAVE_TEST_DIR -or -not $dlg.OverwritePrompt) { throw 'save properties changed' }
if (`+result, 1)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			run := exec.CommandContext(ctx, cmd.Path, append(cmd.Args[1:len(cmd.Args)-1], script)...)
			run.Env = append(cmd.Environ(), "PICFETCH_SAVE_TEST_DIR="+filepath.Dir(path))
			run.SysProcAttr = cmd.SysProcAttr
			out, err := run.Output()
			cancel()
			selected, err := decodePickedPaths(out, err)
			if err != nil {
				t.Fatalf("save transport: %v (output %q)", err, out)
			}
			if cancelDialog {
				if selected != nil {
					t.Fatalf("cancel selected paths: %v", selected)
				}
			} else if len(selected) != 1 || filepath.FromSlash(selected[0].Path()) != path {
				t.Fatalf("selected %v, want exact path %q", selected, path)
			}
		}
	}
}
