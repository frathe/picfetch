//go:build windows

package clipboard

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCopyFilesWindows_DecodesUTF8WithNonUTF8Default(t *testing.T) {
	paths := []string{}
	for _, name := range []string{"café.jpg", "東京.png", "😀 $one `two.png"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	for _, tc := range []struct {
		name        string
		paths       []string
		missingList bool
	}{
		{"multiple", paths, false},
		{"single", paths[1:2], false},
		{"read failure", paths, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := runClipboardCommand
			t.Cleanup(func() { runClipboardCommand = original })
			var listPath string
			var decoded []string
			runClipboardCommand = func(command *exec.Cmd) ([]byte, error) {
				for _, item := range command.Env {
					if key, value, _ := strings.Cut(item, "="); key == "PICFETCH_CLIPBOARD_LIST" {
						listPath = value
					}
				}
				if tc.missingList {
					if err := os.Remove(listPath); err != nil {
						t.Fatal(err)
					}
				}
				// A command-specific non-UTF8 default makes the old implicit
				// decoder fail even on a UTF8-configured Windows host. Only the
				// clipboard mutation is replaced; execute the production script.
				const setup = `$PSDefaultParameterValues = @{'Get-Content:Encoding' = 'ASCII'}
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
function Set-Clipboard {
    param([string[]]$LiteralPath)
    [Console]::Write((ConvertTo-Json -InputObject @($LiteralPath) -Compress))
}
`
				args := slices.Clone(command.Args[1:])
				args[len(args)-1] = setup + args[len(args)-1]
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				child := exec.CommandContext(ctx, command.Path, args...)
				child.Env = command.Env
				hideConsoleWindow(child)
				out, err := child.Output()
				if err == nil {
					if err := json.Unmarshal(out, &decoded); err != nil {
						t.Fatalf("decode %q: %v", out, err)
					}
				}
				return out, err
			}
			err := copyFilesWindows(tc.paths)
			if tc.missingList {
				if err == nil {
					t.Fatal("missing list reported success")
				}
			} else if err != nil || !slices.Equal(decoded, tc.paths) {
				t.Fatalf("decoded %q, error %v; want %q", decoded, err, tc.paths)
			}
			if listPath == "" {
				t.Fatal("production command did not provide its list path")
			}
			if _, err := os.Stat(listPath); !os.IsNotExist(err) {
				t.Fatalf("list cleanup: %v", err)
			}
		})
	}
}
