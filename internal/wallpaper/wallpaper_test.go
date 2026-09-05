package wallpaper

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/displays"
)

func TestRequestAndTargetUnsupportedAreTyped(t *testing.T) {
	request := Request{Path: "/tmp/mosaic.png", Target: displays.ID("opaque-monitor")}
	if request.Path != "/tmp/mosaic.png" || request.Target != "opaque-monitor" {
		t.Fatalf("request = %+v", request)
	}
	err := error(&TargetUnsupportedError{Platform: "linux"})
	var unsupported *TargetUnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Platform != "linux" {
		t.Fatalf("errors.As(%v) failed", err)
	}
}

func TestLinuxTarget_ReturnsUnsupportedBeforeAnyLookupOrCommand(t *testing.T) {
	lookups := 0
	originalG, originalP := lookupGsettings, lookupPlasmaApply
	t.Cleanup(func() { lookupGsettings, lookupPlasmaApply = originalG, originalP })
	lookupGsettings = func() (string, error) { lookups++; return "gsettings", nil }
	lookupPlasmaApply = func() (string, error) { lookups++; return "plasma", nil }
	commands := captureCommands(t, nil)

	err := setLinuxRequest(Request{Path: "/tmp/mosaic.png", Target: "display-1"})
	var unsupported *TargetUnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("setLinuxRequest() = %v, want TargetUnsupportedError", err)
	}
	if lookups != 0 || len(*commands) != 0 {
		t.Fatalf("targeted Linux request performed %d lookups and %d commands", lookups, len(*commands))
	}
}

// TestLinuxTarget_SoleDisplayAppliesGlobally covers the common single-monitor
// desktop: mosaicwin always sends a real Target, but when that target is the
// only attached display there is no other desktop a global change could
// wrongly affect, so the request carries Solo and must be honored exactly
// like a no-target request rather than rejected outright.
func TestLinuxTarget_SoleDisplayAppliesGlobally(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	got := captureCommands(t, nil)

	err := setLinuxRequest(Request{Path: "/home/me/mosaic.png", Target: "display-1", Solo: true})
	if err != nil {
		t.Fatalf("setLinuxRequest() = %v, want nil for the sole attached display", err)
	}
	if len(*got) == 0 {
		t.Fatal("a solo targeted request ran no wallpaper command")
	}
}

// TestHostSchemaEnv_DropsSandboxInjectedSchemaSources pins the reason the
// gsettings child gets a scrubbed environment. A snap- or flatpak-wrapped
// launcher (the VS Code snap is the one this was found on) redirects all
// three GSettings schema sources at its own bundled copies of the *host's*
// desktop schemas, which are frequently years out of date - the VS Code snap
// still ships a pre-GNOME-42 org.gnome.desktop.background with no
// picture-uri-dark. picfetch is writing a host desktop setting, so it has to
// resolve the host's schemas no matter who launched it.
func TestHostSchemaEnv_DropsSandboxInjectedSchemaSources(t *testing.T) {
	got := hostSchemaEnv([]string{
		"GSETTINGS_SCHEMA_DIR=/home/me/snap/code/259/.local/share/glib-2.0/schemas",
		"XDG_DATA_HOME=/home/me/snap/code/259/.local/share",
		"XDG_DATA_DIRS=/home/me/snap/code/259/.local/share:/snap/code/259/usr/share:/var/lib/snapd/desktop:/usr/local/share:/usr/share",
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
	})

	for _, unwanted := range []string{"GSETTINGS_SCHEMA_DIR", "XDG_DATA_HOME"} {
		if slices.ContainsFunc(got, func(entry string) bool {
			return strings.HasPrefix(entry, unwanted+"=")
		}) {
			t.Errorf("env still carries the sandbox's %s: %v", unwanted, got)
		}
	}
	if !slices.Contains(got, "XDG_DATA_DIRS=/usr/local/share:/usr/share") {
		t.Errorf("env = %v, want XDG_DATA_DIRS reduced to the host entries", got)
	}
	// Scrubbing must stay surgical: gsettings reaches dconf over the session
	// bus, so losing this would break every write instead of fixing one.
	if !slices.Contains(got, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus") {
		t.Errorf("env = %v, want the session bus address preserved", got)
	}
}

// TestHostSchemaEnv_DropsAnEntirelySandboxedDataDirList pins the difference
// between an empty XDG_DATA_DIRS and an absent one: glib falls back to
// /usr/local/share:/usr/share only when the variable is gone, so setting it
// to "" would strand the lookup with nowhere to search.
func TestHostSchemaEnv_DropsAnEntirelySandboxedDataDirList(t *testing.T) {
	got := hostSchemaEnv([]string{"XDG_DATA_DIRS=/snap/code/259/usr/share:/var/lib/snapd/desktop"})

	if len(got) != 0 {
		t.Errorf("hostSchemaEnv() = %v, want XDG_DATA_DIRS dropped rather than emptied", got)
	}
}

// TestHostSchemaEnv_LeavesAnOrdinaryEnvironmentAlone guards against the
// scrub firing on a normal desktop session, where every one of these
// variables is the host's own and must survive untouched.
func TestHostSchemaEnv_LeavesAnOrdinaryEnvironmentAlone(t *testing.T) {
	env := []string{
		"XDG_DATA_HOME=/home/me/.local/share",
		"XDG_DATA_DIRS=/usr/local/share:/usr/share",
		"PATH=/usr/bin",
	}

	if got := hostSchemaEnv(env); !slices.Equal(got, env) {
		t.Errorf("hostSchemaEnv(%v) = %v, want it unchanged", env, got)
	}
}

// TestSetLinux_ReportsAFailedDarkKeyWrite is the regression test for the bug
// this whole path was reported for: on GNOME 42+ in dark mode picture-uri-dark
// is the *only* key on screen, so swallowing its failure told the user the
// wallpaper had changed while their desktop kept the old picture.
func TestSetLinux_ReportsAFailedDarkKeyWrite(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	stubDarkKey(t, true, nil)
	captureCommands(t, func(args []string) error {
		if slices.Contains(args, "picture-uri-dark") {
			return errors.New("No such key “picture-uri-dark”")
		}
		return nil
	})

	err := setLinux("/home/me/photo.png")
	if err == nil {
		t.Fatal("setLinux() = nil, want the failed dark-key write reported")
	}
	if !strings.Contains(err.Error(), "picture-uri-dark") {
		t.Errorf("setLinux() error = %v, want it to name the key that failed", err)
	}
}

// TestSetLinux_SkipsTheDarkKeyBeforeGnome42 replaces the old
// "ignore a missing dark key" behavior: rather than running a command that
// is known to fail and discarding the error, the schema is asked first and
// the key is simply not written when it does not exist.
func TestSetLinux_SkipsTheDarkKeyBeforeGnome42(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	stubDarkKey(t, false, nil)
	got := captureCommands(t, nil)

	if err := setLinux("/home/me/photo.png"); err != nil {
		t.Errorf("setLinux() error = %v, want nil on a pre-42 GNOME", err)
	}
	if len(*got) != 1 {
		t.Errorf("ran %d commands, want only the picture-uri write: %v", len(*got), *got)
	}
}

// TestSetLinux_StaysForgivingWhenTheSchemaCannotBeRead keeps an unreadable
// schema listing from turning a wallpaper change that may well have worked
// into a reported failure.
func TestSetLinux_StaysForgivingWhenTheSchemaCannotBeRead(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	stubDarkKey(t, false, errors.New("list-keys failed"))
	captureCommands(t, func(args []string) error {
		if slices.Contains(args, "picture-uri-dark") {
			return errors.New("No such key")
		}
		return nil
	})

	if err := setLinux("/home/me/photo.png"); err != nil {
		t.Errorf("setLinux() error = %v, want nil when the schema could not be read", err)
	}
}

// TestDarkBackgroundKeyExists_ReadsTheHostSchemaListing covers the probe
// itself, including that it runs against the scrubbed host environment.
func TestDarkBackgroundKeyExists_ReadsTheHostSchemaListing(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		listing string
		want    bool
	}{
		{name: "gnome 46", listing: "picture-options\npicture-uri\npicture-uri-dark\nprimary-color\n", want: true},
		{name: "pre-42", listing: "picture-options\npicture-uri\nshow-desktop-icons\n", want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// A real sandbox override has to be present in this process's own
			// environment, or "the child does not carry it" would hold even
			// for a command that simply inherits everything.
			t.Setenv("GSETTINGS_SCHEMA_DIR", "/home/me/snap/code/259/.local/share/glib-2.0/schemas")

			var ran *exec.Cmd
			orig := runWallpaperCommand
			t.Cleanup(func() { runWallpaperCommand = orig })
			runWallpaperCommand = func(cmd *exec.Cmd) ([]byte, error) {
				ran = cmd
				return []byte(testCase.listing), nil
			}

			got, err := darkBackgroundKeyExists("/usr/bin/gsettings")
			if err != nil || got != testCase.want {
				t.Fatalf("darkBackgroundKeyExists() = %v, %v, want %v, nil", got, err, testCase.want)
			}
			if !slices.Contains(ran.Args, "list-keys") || !slices.Contains(ran.Args, gnomeBackgroundSchema) {
				t.Errorf("probe command = %v, want it to list the background schema's keys", ran.Args)
			}
			// A nil Env is exec's "inherit the parent's", which would hand the
			// child the sandbox override this whole path exists to escape.
			if ran.Env == nil {
				t.Fatal("probe inherited the ambient environment, want an explicit scrubbed one")
			}
			if slices.ContainsFunc(ran.Env, func(entry string) bool {
				return strings.HasPrefix(entry, "GSETTINGS_SCHEMA_DIR=")
			}) {
				t.Errorf("probe env = %v, want the sandbox schema override dropped", ran.Env)
			}
		})
	}
}

// stubDarkKey pins what the background schema is found to contain, so a test
// can choose its GNOME generation without stubbing the listing command.
func stubDarkKey(t *testing.T, exists bool, err error) {
	t.Helper()

	orig := darkBackgroundKeyExists
	t.Cleanup(func() { darkBackgroundKeyExists = orig })
	darkBackgroundKeyExists = func(string) (bool, error) { return exists, err }
}

// captureCommands swaps runWallpaperCommand for a recorder, returning a
// pointer to the argument lists of every command the code under test ran, in
// order. fail, when non-nil, decides which of them report an error - the
// Linux path deliberately treats two of its three possible commands
// differently on failure, and that distinction is most of what there is to
// test here.
func captureCommands(t *testing.T, fail func(args []string) error) *[][]string {
	t.Helper()

	orig := runWallpaperCommand
	t.Cleanup(func() { runWallpaperCommand = orig })

	var got [][]string
	runWallpaperCommand = func(cmd *exec.Cmd) ([]byte, error) {
		got = append(got, cmd.Args)
		if fail == nil {
			return nil, nil
		}
		return nil, fail(cmd.Args)
	}

	return &got
}

// stubLookups forces both binary lookups to a fixed outcome, so a test never
// depends on what happens to be installed on the machine running it.
func stubLookups(t *testing.T, gsettings, plasma string) {
	t.Helper()

	origG, origP := lookupGsettings, lookupPlasmaApply
	t.Cleanup(func() { lookupGsettings, lookupPlasmaApply = origG, origP })

	lookupGsettings = func() (string, error) {
		if gsettings == "" {
			return "", errors.New("not found")
		}
		return gsettings, nil
	}
	lookupPlasmaApply = func() (string, error) {
		if plasma == "" {
			return "", errors.New("not found")
		}
		return plasma, nil
	}
}

// TestSetLinux_SetsBothGnomeBackgroundKeys pins the reason this writes two
// keys rather than one: GNOME 42 split the background into a light and a
// dark picture-uri, and a user in dark mode sees nothing at all change when
// only the light one is written.
func TestSetLinux_SetsBothGnomeBackgroundKeys(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	stubDarkKey(t, true, nil)
	got := captureCommands(t, nil)

	if err := setLinux("/home/me/photo.png"); err != nil {
		t.Fatalf("setLinux() error = %v", err)
	}

	if len(*got) != 2 {
		t.Fatalf("ran %d commands, want 2 (picture-uri and picture-uri-dark): %v", len(*got), *got)
	}
	for i, key := range []string{"picture-uri", "picture-uri-dark"} {
		args := (*got)[i]
		if !slices.Contains(args, key) {
			t.Errorf("command %d = %v, want it to set %q", i, args, key)
		}
		if !slices.Contains(args, "org.gnome.desktop.background") {
			t.Errorf("command %d = %v, want it to address the background schema", i, args)
		}
		if !slices.Contains(args, "file:///home/me/photo.png") {
			t.Errorf("command %d = %v, want it to carry the file:// URI", i, args)
		}
	}
}

func TestSetLinux_ReturnsErrorWhenTheLightKeyIsRejected(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	captureCommands(t, func(args []string) error {
		if slices.Contains(args, "picture-uri") {
			return errors.New("no schema")
		}
		return nil
	})

	if err := setLinux("/home/me/photo.png"); err == nil {
		t.Error("expected an error when gsettings rejects picture-uri")
	}
}

// TestSetLinux_PercentEncodesThePath guards the difference between a path
// and a URI: a '#' in a file name would otherwise start a fragment and
// truncate the URI, leaving gsettings pointed at a file that isn't there.
func TestSetLinux_PercentEncodesThePath(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "/usr/bin/gsettings", "")
	got := captureCommands(t, nil)

	if err := setLinux("/home/me/my #1 photo.png"); err != nil {
		t.Fatalf("setLinux() error = %v", err)
	}

	want := "file:///home/me/my%20%231%20photo.png"
	if !slices.Contains((*got)[0], want) {
		t.Errorf("command = %v, want the URI %q", (*got)[0], want)
	}
}

// TestSetLinux_PrefersPlasmaOnKDE covers the one case where gsettings would
// succeed and change nothing: KDE, where glib is usually installed (so the
// lookup succeeds) but the desktop reads its wallpaper from somewhere else
// entirely, so the user would be told it worked and see no change.
func TestSetLinux_PrefersPlasmaOnKDE(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	stubLookups(t, "/usr/bin/gsettings", "/usr/bin/plasma-apply-wallpaperimage")
	got := captureCommands(t, nil)

	if err := setLinux("/home/me/photo.png"); err != nil {
		t.Fatalf("setLinux() error = %v", err)
	}

	if len(*got) != 1 {
		t.Fatalf("ran %d commands, want 1: %v", len(*got), *got)
	}
	args := (*got)[0]
	if !strings.Contains(args[0], "plasma-apply-wallpaperimage") {
		t.Errorf("ran %v, want plasma-apply-wallpaperimage", args)
	}
	if !slices.Contains(args, "/home/me/photo.png") {
		t.Errorf("args = %v, want the plain path passed through", args)
	}
}

// TestSetLinux_FallsBackToGsettingsOnKDEWithoutPlasmaApply covers Plasma
// before 5.24, which ships no plasma-apply-wallpaperimage: gsettings is a
// long shot there, but it is strictly better than refusing outright.
func TestSetLinux_FallsBackToGsettingsOnKDEWithoutPlasmaApply(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	stubLookups(t, "/usr/bin/gsettings", "")
	got := captureCommands(t, nil)

	if err := setLinux("/home/me/photo.png"); err != nil {
		t.Fatalf("setLinux() error = %v", err)
	}

	if len(*got) == 0 || !strings.Contains((*got)[0][0], "gsettings") {
		t.Errorf("ran %v, want a gsettings fallback", *got)
	}
}

func TestSetLinux_ReturnsErrorWhenNeitherToolIsInstalled(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	stubLookups(t, "", "")

	if err := setLinux("/home/me/photo.png"); err == nil {
		t.Error("expected an error when neither gsettings nor plasma-apply-wallpaperimage is installed")
	}
}

// scriptOf pulls the -Command argument out of a recorded PowerShell
// invocation, mirroring internal/trash's own windows-path tests.
func scriptOf(t *testing.T, args []string) string {
	t.Helper()

	for i, a := range args {
		if a == "-Command" && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("no -Command argument in %v", args)
	return ""
}

func TestSetWindows_BuildsExpectedScript(t *testing.T) {
	got := captureCommands(t, nil)

	if err := setWindows(`C:\Users\me\photo.png`); err != nil {
		t.Fatalf("setWindows() error = %v", err)
	}

	script := scriptOf(t, (*got)[0])
	for _, want := range []string{
		"Add-Type",
		"user32.dll",
		"SystemParametersInfo",
		// SPI_SETDESKWALLPAPER with SPIF_UPDATEINIFILE|SPIF_SENDWININICHANGE:
		// the flags are what make the change persist across a reboot and
		// reach the running shell rather than only this session's memory.
		`SystemParametersInfo(20, 0, "C:\Users\me\photo.png", 3)`,
		"exit 1",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script does not contain %q:\n%s", want, script)
		}
	}
}

func TestSetWindows_DispatchesTargetWithoutGlobalFallback(t *testing.T) {
	commands := captureCommands(t, nil)
	request := Request{Path: `C:\Users\me\mosaic.png`, Target: `\\?\DISPLAY#opaque`}
	var got Request
	if err := setWindowsRequestWith(request, func(targeted Request) error {
		got = targeted
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if got != request {
		t.Fatalf("target adapter request = %+v, want %+v", got, request)
	}
	if len(*commands) != 0 {
		t.Fatalf("targeted dispatch ran global PowerShell commands: %v", *commands)
	}
}

func TestSetWindows_LegacyRequestUsesExistingPowerShellPath(t *testing.T) {
	commands := captureCommands(t, nil)
	targetCalls := 0
	if err := setWindowsRequestWith(Request{Path: `C:\wallpaper.png`}, func(Request) error {
		targetCalls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if targetCalls != 0 || len(*commands) != 1 {
		t.Fatalf("legacy request targetCalls=%d commands=%v", targetCalls, *commands)
	}
}

// TestSetWindows_EscapesPathMetacharacters mirrors internal/trash's own: the
// path here is derived from a user's file name, and ` and $ are both legal
// in a Windows filename and both special inside a double-quoted PowerShell
// string.
func TestSetWindows_EscapesPathMetacharacters(t *testing.T) {
	got := captureCommands(t, nil)

	if err := setWindows("C:\\Users\\me\\$weird`file.png"); err != nil {
		t.Fatalf("setWindows() error = %v", err)
	}

	script := scriptOf(t, (*got)[0])
	if !strings.Contains(script, "`$weird") {
		t.Errorf("script does not escape $ in the path:\n%s", script)
	}
	if !strings.Contains(script, "``file") {
		t.Errorf("script does not escape ` in the path:\n%s", script)
	}
}

func TestEscapePowerShellPath(t *testing.T) {
	got := escapePowerShellPath("C:\\a$b`c")
	want := "C:\\a`$b``c"
	if got != want {
		t.Errorf("escapePowerShellPath() = %q, want %q", got, want)
	}
}
