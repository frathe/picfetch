// Package wallpaper sets the current OS's desktop wallpaper to an image
// file, for the Actions > "Set as Wallpaper" action. Each platform needs its
// own approach: NSWorkspace on macOS (darwin.go, cgo/AppKit - not an
// AppleScript "tell application \"System Events\"" shell-out, which would
// trigger the one-time Automation permission prompt a direct framework call
// avoids entirely, the same reasoning internal/trash's darwin.go is built
// on); SystemParametersInfo via PowerShell on Windows; gsettings, or
// plasma-apply-wallpaperimage on KDE, on Linux.
//
// The path handed to Set must stay readable for as long as it is the
// wallpaper: every platform here stores a reference to the file rather than
// a copy of its pixels. internal/ui/wallpaper.go is what guarantees that, by
// writing its own PNG into the user's cache directory rather than pointing
// any of this at a file the user can move or delete from the app itself.
package wallpaper

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/frathe/picfetch/internal/displays"
)

// Request names the persistent PicFetch-owned image and, optionally, the
// opaque display that should receive it. A zero Target preserves the legacy
// all-desktop action.
type Request struct {
	Path   string
	Target displays.ID

	// Solo tells a platform that cannot truthfully address one display among
	// several that Target is nonetheless safe to honor as a global change:
	// the caller has confirmed it is the only display currently attached, so
	// there is no other desktop a global write could wrongly affect. Ignored
	// by platforms (Windows, macOS) that already target displays for real.
	Solo bool
}

// ErrBusy reports that another PicFetch wallpaper copy/set/cleanup operation
// owns the shared lifecycle. Callers can retain their current preview and try
// again after that operation completes.
var ErrBusy = errors.New("another wallpaper change is already in progress")

// TargetUnsupportedError reports a platform that cannot truthfully apply a
// selected-display request without changing other displays too.
type TargetUnsupportedError struct {
	Platform string
	Reason   string
}

func (e *TargetUnsupportedError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("%s cannot set wallpaper for one selected display", e.Platform)
	}

	return fmt.Sprintf("%s cannot set wallpaper for one selected display: %s", e.Platform, e.Reason)
}

// Set dispatches to the current OS's own way of making request.Path the desktop
// wallpaper. A var so callers' tests can stub the whole platform dispatch.
var Set = func(request Request) error {
	switch runtime.GOOS {
	case "darwin":
		return setDarwinRequest(request)
	case "windows":
		return setWindowsRequest(request)
	default:
		return setLinuxRequest(request)
	}
}

// lookupGsettings/lookupPlasmaApply find their respective binaries; vars so
// tests can force either outcome deterministically instead of depending on
// what's installed on the machine running the test.
var lookupGsettings = func() (string, error) { return exec.LookPath("gsettings") }
var lookupPlasmaApply = func() (string, error) { return exec.LookPath("plasma-apply-wallpaperimage") }

// runWallpaperCommand runs the already-built command; a var so tests can
// stub the process out entirely.
var runWallpaperCommand = func(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }

// gnomeBackgroundSchema is the GSettings schema GNOME and the desktops built
// on its session (Cinnamon, Budgie, Unity) read the wallpaper from.
// gnomePictureURI is the picture on a light desktop; gnomePictureURIDark is
// its GNOME 42-and-later dark-mode twin, and the only one of the two on
// screen for a user in dark mode.
const (
	gnomeBackgroundSchema = "org.gnome.desktop.background"
	gnomePictureURI       = "picture-uri"
	gnomePictureURIDark   = "picture-uri-dark"
)

// desktopCommand builds a host desktop command whose GSettings schema lookup
// is the host's own - see hostSchemaEnv.
func desktopCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env = hostSchemaEnv(os.Environ())

	return cmd
}

// hostSchemaEnv strips the three environment variables through which a
// sandboxed launcher redirects GSettings schema lookup into its own bundle.
// A snap or flatpak ships copies of the host's desktop schemas frozen at
// whatever release it was built against, and exports GSETTINGS_SCHEMA_DIR,
// XDG_DATA_HOME, and XDG_DATA_DIRS so its own libraries find them. Every
// child it spawns inherits that view - so picfetch launched from such a
// session would read, say, the VS Code snap's pre-GNOME-42
// org.gnome.desktop.background, which has no picture-uri-dark at all, and
// conclude the running GNOME 46 desktop cannot do dark wallpapers.
//
// The wallpaper is a *host* setting, so it has to be written against the
// host's schemas whoever launched picfetch. Only these schema-lookup
// variables are dropped: gsettings still needs the rest of the environment,
// the session bus address above all, to reach dconf.
func hostSchemaEnv(env []string) []string {
	scrubbed := make([]string, 0, len(env))
	for _, entry := range env {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			scrubbed = append(scrubbed, entry)
			continue
		}

		switch name {
		case "GSETTINGS_SCHEMA_DIR":
			// An explicit override always points at the bundle that set it.
			continue
		case "XDG_DATA_HOME":
			// Highest precedence of the three, and the last one to be found
			// during this bug: the snap redirects the user's data home into
			// its own tree.
			if isSandboxPath(value) {
				continue
			}
		case "XDG_DATA_DIRS":
			host := hostDataDirs(value)
			if host == "" {
				// Dropping it entirely is what restores glib's own default
				// of /usr/local/share:/usr/share; an empty value would not.
				continue
			}
			scrubbed = append(scrubbed, name+"="+host)
			continue
		}
		scrubbed = append(scrubbed, entry)
	}

	return scrubbed
}

// hostDataDirs keeps only the entries of an XDG_DATA_DIRS list that belong to
// the host rather than to a snap mounted into it.
func hostDataDirs(list string) string {
	entries := strings.Split(list, string(os.PathListSeparator))
	host := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry == "" || isSandboxPath(entry) {
			continue
		}
		host = append(host, entry)
	}

	return strings.Join(host, string(os.PathListSeparator))
}

// isSandboxPath reports whether path lives inside a snap tree, covering both
// the read-only mount (/snap/code/259/usr/share), the per-snap home
// (/home/me/snap/code/259/.local/share), and snapd's own exports
// (/var/lib/snapd/desktop).
func isSandboxPath(path string) bool {
	return strings.Contains(path, "/snap/") || strings.Contains(path, "/snapd/")
}

// darkBackgroundKeyExists reports whether the host's background schema
// defines picture-uri-dark, which is what separates a GNOME that simply
// predates the light/dark pair from one where a failed write is a real
// failure. A var so tests can pin either GNOME generation.
var darkBackgroundKeyExists = func(bin string) (bool, error) {
	out, err := runWallpaperCommand(desktopCommand(bin, "list-keys", gnomeBackgroundSchema))
	if err != nil {
		return false, err
	}

	return slices.Contains(strings.Fields(string(out)), gnomePictureURIDark), nil
}

// setLinux prefers plasma-apply-wallpaperimage on KDE and gsettings
// everywhere else. The desktop check comes first because it is the one case
// where the fallback would silently do nothing: glib - and so gsettings - is
// installed on most KDE systems, so a lookup-order fallback would find it,
// write the GNOME schema successfully, and leave the user's actual desktop
// unchanged while the app reported success.
func setLinux(path string) error {
	if isKDE() {
		if bin, err := lookupPlasmaApply(); err == nil {
			_, err := runWallpaperCommand(desktopCommand(bin, path))
			return err
		}
		// Plasma before 5.24 ships no plasma-apply-wallpaperimage. gsettings
		// is a long shot there, but it beats refusing outright.
	}

	bin, err := lookupGsettings()
	if err != nil {
		return errors.New("neither gsettings nor plasma-apply-wallpaperimage is installed")
	}

	uri := fileURI(path)
	if _, err := runWallpaperCommand(desktopCommand(bin, "set", gnomeBackgroundSchema, gnomePictureURI, uri)); err != nil {
		return err
	}

	return setGnomeDarkPicture(bin, uri)
}

// setGnomeDarkPicture writes the GNOME 42-and-later dark-mode background.
// The pair is not optional decoration: a user in dark mode sees only
// picture-uri-dark, so leaving it behind means picfetch reports a wallpaper
// change the desktop never shows - which is exactly what happened when this
// write's failure was discarded on the assumption that only a pre-42 GNOME
// could make it fail. The schema is asked instead, so a missing key is
// skipped rather than attempted-and-ignored, and every other failure is
// reported. An unreadable schema listing stays forgiving: the write may
// well have landed, and a false alarm is worse than the old silence.
func setGnomeDarkPicture(bin, uri string) error {
	hasDarkKey, probeErr := darkBackgroundKeyExists(bin)
	if probeErr == nil && !hasDarkKey {
		return nil
	}

	if _, err := runWallpaperCommand(desktopCommand(bin, "set", gnomeBackgroundSchema, gnomePictureURIDark, uri)); err != nil {
		if probeErr != nil {
			return nil
		}
		return fmt.Errorf("set %s: %w", gnomePictureURIDark, err)
	}

	return nil
}

func setLinuxRequest(request Request) error {
	if request.Target != "" && !request.Solo {
		return &TargetUnsupportedError{
			Platform: "Linux",
			Reason:   "the active desktop integration applies wallpaper globally; Save Image remains available",
		}
	}

	return setLinux(request.Path)
}

// isKDE reports whether this is a KDE Plasma session. XDG_CURRENT_DESKTOP is
// colon-separated and may name several desktops ("KDE", but also
// "KDE:Plasma"), so this is a substring test rather than an equality one.
func isKDE() bool {
	return strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "KDE")
}

// fileURI converts a filesystem path to the file:// URI gsettings stores.
// Built through net/url rather than by concatenation so the characters that
// are legal in a file name but structural in a URI - '#', which would
// otherwise start a fragment and truncate the path, '?', and spaces - are
// percent-encoded.
func fileURI(path string) string {
	u := url.URL{Scheme: "file", Path: path}
	return u.String()
}

// setWindows shells out to SystemParametersInfo, the Win32 call the
// Personalization settings page itself ends up making. PowerShell has no
// wallpaper cmdlet, so this goes through Add-Type: a P/Invoke declaration
// compiled on the fly, which is the standard way to reach a user32 entry
// point from a script. spiSetDeskWallpaper (20) is the action;
// SPIF_UPDATEINIFILE|SPIF_SENDWININICHANGE (3) is what makes the change
// persist across a reboot and reach the running shell, rather than living
// only in this session's memory.
func setWindows(path string) error {
	script := `Add-Type @"
using System;
using System.Runtime.InteropServices;
public class PicFetchWallpaper {
	[DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
	public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
}
"@
if ([PicFetchWallpaper]::SystemParametersInfo(20, 0, "` + escapePowerShellPath(path) + `", 3) -eq 0) {
	[Console]::Error.WriteLine("SystemParametersInfo could not set the wallpaper")
	exit 1
}`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	hideConsoleWindow(cmd)
	_, err := runWallpaperCommand(cmd)
	return err
}

func setWindowsRequest(request Request) error {
	return setWindowsRequestWith(request, setWindowsTarget)
}

func setWindowsRequestWith(request Request, targeted func(Request) error) error {
	if request.Target == "" {
		return setWindows(request.Path)
	}

	return targeted(request)
}

// escapePowerShellPath escapes path for embedding inside a double-quoted
// PowerShell string literal, exactly as internal/trash's own copy does:
// Windows paths can't contain a literal " (an illegal filename character
// there), so only PowerShell's own metacharacters need guarding - ` (its
// escape character) and $ (variable interpolation), both legal in a Windows
// filename.
func escapePowerShellPath(path string) string {
	path = strings.ReplaceAll(path, "`", "``")
	return strings.ReplaceAll(path, "$", "`$")
}
