//go:build windows

package filepicker

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestWindowsPickerTransport_EmitsUTF8PathArrays(t *testing.T) {
	paths := []string{filepath.Join(t.TempDir(), "café 東京 😀.png"), filepath.Join(t.TempDir(), " spaced name.png")}
	input, err := json.Marshal(paths)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	// Run the actual picker serializer, without opening WinForms or modifying
	// desktop state. Environment transport keeps the fixture out of script text.
	script := powerShellPickerScript(`Write-PickedPaths (ConvertFrom-Json $env:PICFETCH_PICKER_TEST_PATHS)`)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "PICFETCH_PICKER_TEST_PATHS="+string(input))
	out, err := cmd.Output()
	selected, err := decodePickedPaths(out, err)
	if err != nil {
		t.Fatal(err)
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
