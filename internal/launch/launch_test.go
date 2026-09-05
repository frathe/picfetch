package launch

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/preferences"
)

// TestParse_FlagsAnywhereAmongPaths is the reason this package hand-rolls a
// parser instead of using flag.FlagSet: flag stops at the first non-flag
// argument, which would make "picfetch ~/photos --slideshow" open a file
// named "--slideshow" rather than start picture-frame mode.
func TestParse_FlagsAnywhereAmongPaths(t *testing.T) {
	paths, opts, err := Parse([]string{"/a.jpg", "--slideshow", "/b", "--interval=8s"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if want := []string{"/a.jpg", "/b"}; !equalStrings(paths, want) {
		t.Errorf("paths = %q, want %q", paths, want)
	}
	if !opts.PictureFrame {
		t.Error("PictureFrame = false, want true")
	}
	if opts.Interval == nil {
		t.Fatal("Interval is nil, want 8s")
	}
	if *opts.Interval != 8*time.Second {
		t.Errorf("Interval = %v, want 8s", *opts.Interval)
	}
}

// TestParse_ValueFlagsAcceptBothForms pins "--x=v" and "--x v", and the
// single-dash spelling of each, since a shell user reaches for whichever one
// their muscle memory holds.
func TestParse_ValueFlagsAcceptBothForms(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"double dash, equals", []string{"--interval=8s", "--max-files=500", "--sort=date"}},
		{"double dash, separated", []string{"--interval", "8s", "--max-files", "500", "--sort", "date"}},
		{"single dash, equals", []string{"-interval=8s", "-max-files=500", "-sort=date"}},
		{"single dash, separated", []string{"-interval", "8s", "-max-files", "500", "-sort", "date"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			paths, opts, err := Parse(tc.args)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(paths) != 0 {
				t.Errorf("paths = %q, want none", paths)
			}
			if opts.Interval == nil || *opts.Interval != 8*time.Second {
				t.Errorf("Interval = %v, want 8s", opts.Interval)
			}
			if opts.MaxFiles == nil || *opts.MaxFiles != 500 {
				t.Errorf("MaxFiles = %v, want 500", opts.MaxFiles)
			}
			if opts.Sort == nil || *opts.Sort != preferences.SortByCaptureDate {
				t.Errorf("Sort = %v, want %q", opts.Sort, preferences.SortByCaptureDate)
			}
		})
	}
}

// TestParse_BoolFlagsAcceptExplicitValue covers "--merge=false", which has to
// stay distinguishable from an absent --merge: the first overrides a saved
// merge preference off for the run, the second leaves it alone. The
// separated form is deliberately not supported - "--merge false" would have
// to swallow the next argument, and that argument is a path.
func TestParse_BoolFlagsAcceptExplicitValue(t *testing.T) {
	_, opts, err := Parse([]string{"--merge=false", "--shuffle=true"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if opts.Merge == nil {
		t.Fatal("Merge is nil, want an explicit false")
	}
	if *opts.Merge {
		t.Error("Merge = true, want false")
	}
	if opts.Shuffle == nil || !*opts.Shuffle {
		t.Errorf("Shuffle = %v, want true", opts.Shuffle)
	}
}

func TestParse_BareBoolFlagMeansTrue(t *testing.T) {
	_, opts, err := Parse([]string{"--merge", "--shuffle", "/photos"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if opts.Merge == nil || !*opts.Merge {
		t.Errorf("Merge = %v, want true", opts.Merge)
	}
	if opts.Shuffle == nil || !*opts.Shuffle {
		t.Errorf("Shuffle = %v, want true", opts.Shuffle)
	}
}

// TestParse_DoubleDashEndsFlagParsing is how someone opens an image whose
// name starts with a dash.
func TestParse_DoubleDashEndsFlagParsing(t *testing.T) {
	paths, opts, err := Parse([]string{"--merge", "--", "--slideshow", "-weird.jpg"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if want := []string{"--slideshow", "-weird.jpg"}; !equalStrings(paths, want) {
		t.Errorf("paths = %q, want %q", paths, want)
	}
	if opts.PictureFrame {
		t.Error("PictureFrame = true, want false - --slideshow came after the terminator")
	}
	if opts.Merge == nil || !*opts.Merge {
		t.Errorf("Merge = %v, want true - --merge came before the terminator", opts.Merge)
	}
}

// TestParse_IgnoresProcessSerialNumberArgument guards a macOS launch.
// LaunchServices can hand a GUI app "-psn_0_12345"; before this package
// existed that became a bogus URI and was harmlessly skipped, and a strict
// parser must not turn it into an exit-2 failure to start.
func TestParse_IgnoresProcessSerialNumberArgument(t *testing.T) {
	paths, opts, err := Parse([]string{"-psn_0_12345", "/a.jpg"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if want := []string{"/a.jpg"}; !equalStrings(paths, want) {
		t.Errorf("paths = %q, want %q", paths, want)
	}
	if opts != (Options{}) {
		t.Errorf("opts = %+v, want the zero Options", opts)
	}
}

func TestParse_HelpReturnsErrHelp(t *testing.T) {
	for _, arg := range []string{"--help", "-h", "-help", "--h"} {
		if _, _, err := Parse([]string{arg}); !errors.Is(err, ErrHelp) {
			t.Errorf("Parse(%q) error = %v, want ErrHelp", arg, err)
		}
	}
}

// TestParse_NoArgumentsLeavesEveryOptionUnset pins the plain launch: nothing
// set means nothing overridden, which is what keeps a double-click on the
// app icon from touching any saved preference.
func TestParse_NoArgumentsLeavesEveryOptionUnset(t *testing.T) {
	paths, opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("paths = %q, want none", paths)
	}
	if opts != (Options{}) {
		t.Errorf("opts = %+v, want the zero Options", opts)
	}
}

// TestParse_Rejects covers every way a launch should fail loudly rather than
// start with the flag quietly ignored - a typo in a Pi autostart unit is the
// case this exists for. The --sort case matters most: filesort.FromPref maps
// anything unrecognised to ByName, so without validation here "--sort=dtae"
// would silently sort by name.
func TestParse_Rejects(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown flag", []string{"--slidshow"}, "--slidshow"},
		{"unparseable duration", []string{"--interval=soon"}, "soon"},
		{"unparseable count", []string{"--max-files=lots"}, "lots"},
		{"unknown sort mode", []string{"--sort=dtae"}, "dtae"},
		{"missing value at end", []string{"--interval"}, "--interval"},
		{"missing value before flag", []string{"--sort", "--merge"}, "--sort"},
		{"value given to a bool flag", []string{"--merge=perhaps"}, "perhaps"},
		{"bare dash", []string{"-"}, "-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Parse(tc.args)
			if err == nil {
				t.Fatalf("Parse(%q) error = nil, want a failure", tc.args)
			}
			if errors.Is(err, ErrHelp) {
				t.Fatalf("Parse(%q) returned ErrHelp, want a real failure", tc.args)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Parse(%q) error = %q, want it to name %q", tc.args, err, tc.want)
			}
		})
	}
}

// TestParse_RejectsBeforeReturningPaths keeps a bad flag from being half
// honoured: main exits on the error, so the paths it would otherwise have
// opened must not matter, but a caller that ignored the error should not
// find a usable file set either.
func TestParse_RejectsBeforeReturningPaths(t *testing.T) {
	paths, opts, err := Parse([]string{"/a.jpg", "--nope"})
	if err == nil {
		t.Fatal("Parse error = nil, want a failure")
	}
	if len(paths) != 0 {
		t.Errorf("paths = %q, want none on a failed parse", paths)
	}
	if opts != (Options{}) {
		t.Errorf("opts = %+v, want the zero Options on a failed parse", opts)
	}
}

// TestUsage_DocumentsEveryFlag is the guard that stops a flag from shipping
// undocumented: it walks the same table Parse dispatches on, so adding a
// flag without a usage line fails here rather than in a bug report.
func TestUsage_DocumentsEveryFlag(t *testing.T) {
	usage := Usage()

	for _, spec := range flagSpecs {
		if !strings.Contains(usage, "--"+spec.name) {
			t.Errorf("Usage() does not mention --%s", spec.name)
		}
		if spec.help == "" {
			t.Errorf("--%s has no help text", spec.name)
		}
	}
	if !strings.Contains(usage, "--help") {
		t.Error("Usage() does not mention --help")
	}
}

// TestUsage_ListsEverySortMode keeps the --sort help in step with what Parse
// actually accepts.
func TestUsage_ListsEverySortMode(t *testing.T) {
	usage := Usage()

	for _, mode := range sortModes {
		if !strings.Contains(usage, mode) {
			t.Errorf("Usage() does not list the sort mode %q", mode)
		}
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
