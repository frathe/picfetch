// Package filepicker opens the current OS's own file/folder browser and
// returns the paths the user picked. Fyne's built-in dialog is never used
// here: it can select neither a folder nor more than one file, so it can't
// stand in for any of the three OS-specific choosers this package
// dispatches to (Linux and Windows below; darwin is cgo/AppKit and lives in
// darwin.go).
package filepicker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
)

// Choose returns the exact selected file/folder identities in selection order.
// A nil list with no error means cancellation. Native transport is decoded here,
// so consumers never split filenames or interpret command output themselves.
var Choose = func() ([]fyne.URI, error) {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "darwin":
		out, err = chooseFilesDarwin()
	case "windows":
		out, err = chooseFilesWindows()
	default:
		out, err = chooseFilesLinux()
	}
	return decodePickedPaths(out, err)
}

// ChooseSave returns exactly the destination confirmed by the native panel.
// Nil with no error means cancellation; more than one result is a protocol error.
// suggestedPath supplies both the initial directory and the proposed name.
var ChooseSave = func(suggestedPath string) (fyne.URI, error) {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "darwin":
		out, err = chooseSaveDarwin(suggestedPath)
	case "windows":
		out, err = chooseSaveWindows(suggestedPath)
	default:
		out, err = chooseSaveLinux(suggestedPath)
	}
	return decodePickedDestination(out, err)
}

func decodePickedDestination(out []byte, err error) (fyne.URI, error) {
	picked, err := decodePickedPaths(out, err)
	if err != nil || picked == nil {
		return nil, err
	}
	if len(picked) != 1 {
		return nil, errors.New("save panel returned more than one destination")
	}
	return picked[0], nil
}

// Native adapters emit a JSON array of paths, or null for cancellation.
// An empty successful selection is distinct from cancellation and cannot be used.
func decodePickedPaths(out []byte, err error) ([]fyne.URI, error) {
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(out) {
		return nil, errors.New("file chooser returned invalid UTF-8")
	}
	var paths []string
	if err := json.Unmarshal(out, &paths); err != nil {
		return nil, fmt.Errorf("invalid file chooser result: %w", err)
	}
	if paths == nil {
		return nil, nil
	}
	if len(paths) == 0 {
		return nil, errors.New("file chooser returned an empty selection")
	}
	uris := make([]fyne.URI, len(paths))
	for i, picked := range paths {
		if strings.ContainsRune(picked, 0) || !(path.IsAbs(picked) || filepath.IsAbs(picked)) {
			return nil, errors.New("file chooser returned an invalid absolute path")
		}
		uris[i] = storage.NewFileURI(picked)
	}
	return uris, nil
}

// lookupZenity finds the zenity binary; a var so tests can force either
// outcome of chooseFilesLinux deterministically instead of depending on
// whether the machine running the test happens to have zenity installed.
var lookupZenity = exec.LookPath

// runZenityCommand runs the already-built zenity command and returns its
// stdout; a var so tests can stub the process out entirely.
var runZenityCommand = func(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }

// chooseFilesLinux runs Zenity and validates its native path framing. Missing
// tools and execution/transport failures remain distinct from cancellation.
func chooseFilesLinux() ([]byte, error) {
	path, err := lookupZenity("zenity")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(path, "--file-selection", "--multiple",
		"--separator="+zenityPathSeparator, "--title="+lang.L("Open images"))
	out, err := runZenityCommand(cmd)
	return zenityResult(out, err, true)
}

// chooseSaveLinux is chooseFilesLinux's save-panel twin: the same zenity
// file selector in --save mode. The native save panel owns overwrite
// confirmation (older Zenity needs --confirm-overwrite). Multi-select is
// meaningless here, so --multiple is deliberately absent.
func chooseSaveLinux(suggestedPath string) ([]byte, error) {
	path, err := lookupZenity("zenity")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(path, "--file-selection", "--save", "--confirm-overwrite",
		"--filename="+suggestedPath, "--title="+lang.L("Export image"))
	out, err := runZenityCommand(cmd)
	return zenityResult(out, err, false)
}

// chooseFilesWindows shells out to a WinForms OpenFileDialog via
// PowerShell - the closest thing to a native Fyne-dialog replacement on
// Windows. Files only: Windows' own shell dialogs have no mode that
// combines folder selection with multi-file selection (IFileOpenDialog's
// FOS_PICKFOLDERS flag switches the whole dialog to folders-only, mutually
// exclusive with picking files; highlighted folders are silently dropped
// from an OpenFileDialog result). This used to fake folder support with a
// sentinel-filename trick - pick a localized "Select this folder." name to
// mean "the folder I'm currently in" - but that was confusing enough in
// practice to be worse than not having it, so it was removed; Ctrl+O now
// only ever returns files here. Folders are still fully supported, just via
// drag-and-drop instead (handleDrop's recursive expansion, same as every
// other platform).
func chooseFilesWindows() ([]byte, error) {
	return buildPowerShellCmd().Output()
}

func buildPowerShellCmd() *exec.Cmd {
	script := `Add-Type -AssemblyName System.Windows.Forms
$dlg = New-Object System.Windows.Forms.OpenFileDialog
$dlg.Multiselect = $true
$dlg.Title = "` + powerShellEscape(lang.L("Open images")) + `"
if ($dlg.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
	Write-PickedPaths $dlg.FileNames
} else { [Console]::Write("null") }`
	script = powerShellPickerScript(script)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	hideConsoleWindow(cmd)
	return cmd
}

// chooseSaveWindows is chooseFilesWindows' save-panel twin: a WinForms
// SaveFileDialog via PowerShell. None of the open dialog's folder-selection
// trouble applies here - a save panel names exactly one file by design.
func chooseSaveWindows(suggestedPath string) ([]byte, error) {
	return buildPowerShellSaveCmd(suggestedPath).Output()
}

// buildPowerShellSaveCmd builds that dialog's script. The suggested path is
// embedded whole and split by PowerShell's own [System.IO.Path], not by Go's
// filepath: this function's test runs on whatever machine builds the module,
// where a `C:\...` path is one long base name to filepath.Dir. Letting the
// platform that will actually run the script do its own splitting keeps the
// two from disagreeing.
func buildPowerShellSaveCmd(suggestedPath string) *exec.Cmd {
	escaped := powerShellEscape(suggestedPath)
	script := `Add-Type -AssemblyName System.Windows.Forms
$dlg = New-Object System.Windows.Forms.SaveFileDialog
$dlg.FileName = "` + escaped + `"
$dlg.InitialDirectory = [System.IO.Path]::GetDirectoryName("` + escaped + `")
$dlg.OverwritePrompt = $true
$dlg.Title = "` + powerShellEscape(lang.L("Export image")) + `"
if ($dlg.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
	Write-PickedPaths @($dlg.FileName)
} else { [Console]::Write("null") }`
	script = powerShellPickerScript(script)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	hideConsoleWindow(cmd)
	return cmd
}

// powerShellEscape escapes s for embedding inside a double-quoted
// PowerShell string literal.
func powerShellEscape(s string) string {
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, "$", "`$")
	return strings.ReplaceAll(s, `"`, "`\"")
}

// Zenity prints canonical GFile paths. An internal double slash cannot occur
// in one such path, so this separator also preserves every filename character.
// A single save needs no separator: remove only Zenity's final output newline.
const zenityPathSeparator = "//picfetch-file//"

func zenityResult(out []byte, err error, multiple bool) ([]byte, error) {
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 1 {
			return []byte("null"), nil
		}
		return nil, err
	}
	if !utf8.Valid(out) || !strings.HasSuffix(string(out), "\n") || strings.ContainsRune(string(out), 0) {
		return nil, errors.New("unsupported zenity path transport")
	}
	payload := strings.TrimSuffix(string(out), "\n")
	paths := []string{payload}
	if multiple {
		paths = strings.Split(payload, zenityPathSeparator)
	}
	for _, picked := range paths {
		if !path.IsAbs(picked) || path.Clean(picked) != picked {
			return nil, errors.New("zenity returned an empty or noncanonical path")
		}
	}
	return json.Marshal(paths)
}

// The same serializer is exercised without opening a dialog by Windows native
// protocol tests. Console encoding is explicit even on legacy Windows PowerShell.
func powerShellPickerScript(body string) string {
	return `$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
function Write-PickedPaths([string[]]$paths) {
	[Console]::Write((ConvertTo-Json -InputObject $paths -Compress))
}
try {
` + body + `
} catch {
	[Console]::Error.WriteLine($_.Exception.Message)
	exit 1
}`
}
