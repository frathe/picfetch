package filepicker

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestDecodePickedPaths_PreservesBoundaries(t *testing.T) {
	paths := []string{"/tmp/first\nsecond.jpg", "/tmp/tail\r", "/tmp/tail\n", "/tmp/ spaces ", "/tmp/café 東京 😀.png"}
	encoded, err := json.Marshal(paths)
	if err != nil {
		t.Fatal(err)
	}
	uris, err := decodePickedPaths(encoded, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(uris) != len(paths) {
		t.Fatalf("paths = %v", uris)
	}
	for i, uri := range uris {
		if uri.Path() != paths[i] {
			t.Errorf("path %d = %q, want %q", i, uri.Path(), paths[i])
		}
	}
}

func TestDecodePickedPaths_DistinguishesCancellationAndErrors(t *testing.T) {
	failure := errors.New("native picker failed")
	if _, err := decodePickedPaths(nil, failure); !errors.Is(err, failure) {
		t.Fatal("lost native error")
	}
	if paths, err := decodePickedPaths([]byte("null"), nil); paths != nil || err != nil {
		t.Fatalf("cancel = %v, %v", paths, err)
	}
	for _, out := range []string{"", "[]", `[""]`, `["relative.jpg"]`, `["/tmp/a\u0000b"]`, `"/tmp/a"`, "[", string([]byte{0xff})} {
		if paths, err := decodePickedPaths([]byte(out), nil); err == nil || paths != nil {
			t.Errorf("accepted malformed/empty result %q: %v", out, paths)
		}
	}
}

func TestZenityResult_PreservesExactPaths(t *testing.T) {
	paths := []string{"/tmp/line\nnext.jpg", "/tmp/tail\r", "/tmp/tail\n", "/tmp/ spaced ", "/tmp/café 東京 😀.png"}
	for _, multiple := range []bool{false, true} {
		groups := [][]string{paths}
		if !multiple {
			groups = nil
			for _, p := range paths {
				groups = append(groups, []string{p})
			}
		}
		for _, group := range groups {
			out, err := zenityResult([]byte(strings.Join(group, zenityPathSeparator)+"\n"), nil, multiple)
			uris, err := decodePickedPaths(out, err)
			if err != nil {
				t.Fatal(err)
			}
			if len(uris) != len(group) {
				t.Fatalf("paths = %v", uris)
			}
			for i, uri := range uris {
				if uri.Path() != group[i] {
					t.Errorf("path = %q, want %q", uri.Path(), group[i])
				}
			}
		}
	}
}

func TestZenityResult_RejectsUnsupportedFraming(t *testing.T) {
	for _, out := range []string{"", "\n", "/tmp/a", "/tmp/a/../b\n", "/tmp/a//b\n", "/tmp/a\x00b\n", "/tmp/a" + zenityPathSeparator + "relative\n"} {
		if _, err := zenityResult([]byte(out), nil, true); err == nil {
			t.Errorf("accepted unsupported transport %q", out)
		}
	}
	failure := errors.New("zenity unavailable")
	if _, err := zenityResult(nil, failure, true); !errors.Is(err, failure) {
		t.Fatal("lost execution error")
	}
}

func TestPowerShellEscape(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Open images", "Open images"},
		{`say "hi"`, "say `\"hi`\""},
		{"back`tick", "back``tick"},
		{"costs $5", "costs `$5"},
		{"$HOME`, really", "`$HOME``, really"},
	}
	for _, tt := range tests {
		if got := powerShellEscape(tt.in); got != tt.want {
			t.Errorf("powerShellEscape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Native protocol tests exercise the real serializers without opening a panel.
func TestBuildPowerShellCmd(t *testing.T) {
	cmd := buildPowerShellCmd()

	if got := cmd.Args[0]; !strings.HasSuffix(got, "powershell") {
		t.Errorf("cmd.Args[0] = %q, want it to name powershell", got)
	}

	var script string
	for i, a := range cmd.Args {
		if a == "-Command" && i+1 < len(cmd.Args) {
			script = cmd.Args[i+1]
		}
	}
	if script == "" {
		t.Fatalf("cmd.Args = %v, expected a -Command argument", cmd.Args)
	}

	for _, want := range []string{
		"OpenFileDialog",
		"$dlg.Multiselect = $true",
		"Open images",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not contain %q:\n%s", want, script)
		}
	}

	for _, notWant := range []string{
		"folderSentinel",
		"Select this folder.",
		"GetDirectoryName",
		"ValidateNames",
	} {
		if strings.Contains(script, notWant) {
			t.Errorf("script contains %q, want the folder-sentinel workaround gone:\n%s", notWant, script)
		}
	}
}

func TestBuildPowerShellSaveCmd(t *testing.T) {
	cmd := buildPowerShellSaveCmd(`C:\photos\holiday.png`)

	if got := cmd.Args[0]; !strings.HasSuffix(got, "powershell") {
		t.Errorf("cmd.Args[0] = %q, want it to name powershell", got)
	}

	var script string
	for i, a := range cmd.Args {
		if a == "-Command" && i+1 < len(cmd.Args) {
			script = cmd.Args[i+1]
		}
	}
	if script == "" {
		t.Fatalf("cmd.Args = %v, expected a -Command argument", cmd.Args)
	}

	for _, want := range []string{
		"SaveFileDialog",
		"holiday.png",
		`C:\photos`,
		"$dlg.OverwritePrompt = $true",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not contain %q:\n%s", want, script)
		}
	}
}

func TestChooseSaveLinux_ReturnsErrorWhenZenityMissing(t *testing.T) {
	origLookup := lookupZenity
	t.Cleanup(func() { lookupZenity = origLookup })
	lookupZenity = func(string) (string, error) { return "", errors.New("zenity not found") }

	if _, err := chooseSaveLinux("/photos/holiday.png"); err == nil {
		t.Error("expected an error when zenity isn't installed")
	}
}

func TestChooseSaveLinux_RunsZenityInSaveModeWithTheSuggestedPath(t *testing.T) {
	origLookup := lookupZenity
	origRun := runZenityCommand
	t.Cleanup(func() {
		lookupZenity = origLookup
		runZenityCommand = origRun
	})
	lookupZenity = func(string) (string, error) { return "/usr/bin/zenity", nil }

	var gotArgs []string
	runZenityCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = cmd.Args
		return []byte("/photos/holiday.png\n"), nil
	}

	out, err := chooseSaveLinux("/photos/holiday.png")
	if err != nil {
		t.Fatalf("chooseSaveLinux() error = %v", err)
	}
	if string(out) != `["/photos/holiday.png"]` {
		t.Errorf("out = %q, want the stubbed zenity output", out)
	}

	// --filename carries the whole suggested path, not just the base name,
	// so the panel opens in the image's own folder rather than the CWD.
	for _, want := range []string{"--save", "--confirm-overwrite", "--filename=/photos/holiday.png"} {
		if !slices.Contains(gotArgs, want) {
			t.Errorf("zenity args = %v, want %q present", gotArgs, want)
		}
	}
	if slices.Contains(gotArgs, "--multiple") {
		t.Errorf("zenity args = %v, want no --multiple on a save panel", gotArgs)
	}
}

func TestChooseFilesLinux_ReturnsErrorWhenZenityMissing(t *testing.T) {
	origLookup := lookupZenity
	t.Cleanup(func() { lookupZenity = origLookup })
	lookupZenity = func(string) (string, error) { return "", errors.New("zenity not found") }

	if _, err := chooseFilesLinux(); err == nil {
		t.Error("expected an error when zenity isn't installed")
	}
}

func TestChooseFilesLinux_RunsZenityWithMultiSelect(t *testing.T) {
	origLookup := lookupZenity
	origRun := runZenityCommand
	t.Cleanup(func() {
		lookupZenity = origLookup
		runZenityCommand = origRun
	})
	lookupZenity = func(string) (string, error) { return "/usr/bin/zenity", nil }

	var gotArgs []string
	runZenityCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = cmd.Args
		return []byte("/tmp/a.jpg\n"), nil
	}

	out, err := chooseFilesLinux()
	if err != nil {
		t.Fatalf("chooseFilesLinux() error = %v", err)
	}
	if string(out) != `["/tmp/a.jpg"]` {
		t.Errorf("out = %q, want the stubbed zenity output", out)
	}

	found := false
	for _, a := range gotArgs {
		if a == "--multiple" {
			found = true
		}
	}
	if !found {
		t.Errorf("zenity args = %v, want --multiple present", gotArgs)
	}
}

func TestDecodePickedDestination_RequiresExactlyOnePath(t *testing.T) {
	for _, out := range []string{`["/tmp/a.png", "/tmp/b.png"]`, `[]`, ``, `[""]`} {
		if destination, err := decodePickedDestination([]byte(out), nil); destination != nil || err == nil {
			t.Errorf("accepted save result %q: %v (%v)", out, destination, err)
		}
	}
	if destination, err := decodePickedDestination([]byte("null"), nil); destination != nil || err != nil {
		t.Fatalf("cancel = %v (%v)", destination, err)
	}
	destination, err := decodePickedDestination([]byte(`["/tmp/line\nnext\r"]`), nil)
	if err != nil || destination.Path() != "/tmp/line\nnext\r" {
		t.Fatalf("exact destination = %v (%v)", destination, err)
	}
}

func TestZenityResult_DistinguishesCancelAndFailure(t *testing.T) {
	switch os.Getenv("PICFETCH_ZENITY_EXIT_FIXTURE") {
	case "cancel":
		os.Exit(1)
	case "failure":
		os.Exit(2)
	}
	for _, fixture := range []string{"cancel", "failure"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestZenityResult_DistinguishesCancelAndFailure$")
		cmd.Env = append(os.Environ(), "PICFETCH_ZENITY_EXIT_FIXTURE="+fixture)
		out, processErr := cmd.Output()
		if processErr == nil {
			t.Fatal("fixture did not fail")
		}
		out, err := zenityResult(out, processErr, true)
		if fixture == "cancel" {
			if err != nil || string(out) != "null" {
				t.Fatalf("cancel = %q, %v", out, err)
			}
		} else if !errors.Is(err, processErr) {
			t.Fatal("execution failure was lost")
		}
	}
}
