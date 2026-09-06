// Package launch parses PicFetch's command-line flags into the option set
// internal/ui applies at startup.
//
// It sits here rather than in package main because ui.Run takes Options as a
// parameter and cannot import main, and because keeping the parser out of
// main.go leaves that file to app setup and path conversion (see AGENTS.md).
// Nothing here draws or touches Fyne, so the whole surface is unit-testable
// without an app.
//
// Every option is a pointer, apart from the one-shot PictureFrame: the
// viewer restores a flag-overridden setting to its pre-flag value when it
// saves preferences at shutdown, so "unset" has to stay distinguishable from
// "set to the zero value". A scripted launch must not quietly rewrite the
// user's saved defaults.
package launch

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/frathe/picfetch/internal/preferences"
)

// ErrHelp is returned by Parse when the arguments asked for the usage text
// rather than a launch. It is not a failure: main prints Usage to stdout and
// exits 0.
var ErrHelp = errors.New("launch: help requested")

// Options is one launch's overrides. A nil field was not given on the
// command line and leaves the saved preference alone.
type Options struct {
	// PictureFrame requests picture-frame mode for this launch. Not a
	// pointer, because it is a one-shot action rather than a standing
	// setting: there is no saved value to override and nothing to restore.
	PictureFrame bool

	// Sort is one of the preferences.SortBy* constants, validated by Parse.
	Sort *string

	Merge    *bool
	Shuffle  *bool
	Interval *time.Duration

	// MaxFiles caps how many images one scan gathers - the setting behind
	// viewer.SetMaxScan. Named for what it bounds rather than for recursion,
	// because it also bounds the non-recursive sibling expansion of a single
	// opened image (filescan.Siblings).
	MaxFiles *int
}

// sortModes is every value --sort accepts, in the order the usage text lists
// them. Parse validates against this rather than handing the string straight
// to filesort.FromPref, which maps anything unrecognized to ByName - so
// without this check "--sort=dtae" would silently sort by name.
var sortModes = []string{
	preferences.SortByName,
	preferences.SortByCaptureDate,
	preferences.SortByModTime,
	preferences.SortBySize,
	preferences.SortByDropOrder,
}

// spec is one recognized flag. arg is the value placeholder for the usage
// text, empty for a boolean flag; set applies an already-extracted raw
// value, which is "true" for a boolean flag given without one.
type spec struct {
	name string
	arg  string
	help string
	set  func(o *Options, raw string) error
}

var flagSpecs = []spec{
	{
		name: "slideshow",
		help: "start in picture-frame mode once the files have loaded",
		set: func(o *Options, raw string) error {
			return setBool(&o.PictureFrame, raw)
		},
	},
	{
		name: "shuffle",
		help: "advance to a random file in picture-frame mode",
		set: func(o *Options, raw string) error {
			return setBoolPtr(&o.Shuffle, raw)
		},
	},
	{
		name: "interval",
		arg:  "DURATION",
		help: "picture-frame auto-advance interval, for example 8s or 1m30s",
		set: func(o *Options, raw string) error {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return err
			}
			o.Interval = &d

			return nil
		},
	},
	{
		name: "sort",
		arg:  "MODE",
		help: "order the file set: " + strings.Join(sortModes, "|"),
		set: func(o *Options, raw string) error {
			if !slices.Contains(sortModes, raw) {
				return fmt.Errorf("unknown sort mode %q, want one of: %s", raw, strings.Join(sortModes, ", "))
			}
			o.Sort = &raw

			return nil
		},
	},
	{
		name: "merge",
		help: "merge opened files into the current set instead of replacing it",
		set: func(o *Options, raw string) error {
			return setBoolPtr(&o.Merge, raw)
		},
	},
	{
		name: "max-files",
		arg:  "N",
		help: "stop a scan after N images",
		set: func(o *Options, raw string) error {
			n, err := strconv.Atoi(raw)
			if err != nil {
				return err
			}
			o.MaxFiles = &n

			return nil
		},
	},
}

func setBool(dst *bool, raw string) error {
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return err
	}
	*dst = b

	return nil
}

func setBoolPtr(dst **bool, raw string) error {
	var b bool
	if err := setBool(&b, raw); err != nil {
		return err
	}
	*dst = &b

	return nil
}

// Parse splits args - os.Args[1:] - into the paths to open and the options to
// apply. Flags may appear anywhere among the paths, which is why this does
// not use flag.FlagSet: that stops at the first non-flag argument, so
// "picfetch ~/photos --slideshow" would open a file named "--slideshow".
//
// A single dash reads the same as a double one, values take either
// "--flag=value" or "--flag value", and "--" ends flag parsing so a path may
// start with a dash. Boolean flags take only "--flag" or "--flag=false": the
// separated form would have to swallow the next argument, and that argument
// is a path.
//
// Any unrecognized flag is an error, so a typo in an autostart unit fails
// loudly instead of launching with the flag ignored. The one exception is
// macOS's "-psn_0_12345", which LaunchServices can pass to a GUI app and
// which must never stop it from starting.
func Parse(args []string) (paths []string, opts Options, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			paths = append(paths, args[i+1:]...)

			return paths, opts, nil
		}
		if !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)

			continue
		}

		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name == "help" || name == "h" {
			return nil, Options{}, ErrHelp
		}
		if strings.HasPrefix(name, "psn_") {
			continue
		}

		found := specFor(name)
		if found == nil {
			return nil, Options{}, fmt.Errorf("unknown flag %q", arg)
		}

		if found.arg != "" && !hasValue {
			// A following flag is not this one's value: "--sort --merge" is
			// a missing value, not a sort mode named "--merge".
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return nil, Options{}, fmt.Errorf("flag %s needs a value, for example %s=%s", arg, arg, found.arg)
			}
			i++
			value, hasValue = args[i], true
		}
		if !hasValue {
			value = "true"
		}

		if err := found.set(&opts, value); err != nil {
			return nil, Options{}, fmt.Errorf("flag %s: %w", arg, err)
		}
	}

	return paths, opts, nil
}

func specFor(name string) *spec {
	for i := range flagSpecs {
		if flagSpecs[i].name == name {
			return &flagSpecs[i]
		}
	}

	return nil
}

// Usage is the help text, in English rather than through lang.L: it goes to
// a terminal or a systemd journal rather than to anything the app draws, and
// a flag error that changes wording with the locale is hostile to scripting.
func Usage() string {
	var b strings.Builder

	b.WriteString("PicFetch - a desktop image viewer.\n\n")
	b.WriteString("Usage:\n  picfetch [flags] [file or folder ...]\n\nFlags:\n")

	width := 0
	for _, s := range flagSpecs {
		if n := len(flagLabel(s)); n > width {
			width = n
		}
	}

	for _, s := range flagSpecs {
		b.WriteString(fmt.Sprintf("  %-*s  %s\n", width, flagLabel(s), s.help))
	}
	b.WriteString(fmt.Sprintf("  %-*s  %s\n", width, "--help", "print this help and exit"))

	b.WriteString("\nFlags may appear anywhere among the paths, and -flag reads the same as\n")
	b.WriteString("--flag. Use -- to end flag parsing, for a path that starts with a dash.\n")
	b.WriteString("\nEvery flag applies to this launch alone and leaves saved settings unchanged.\n")

	return b.String()
}

func flagLabel(s spec) string {
	if s.arg == "" {
		return "--" + s.name
	}

	return "--" + s.name + "=" + s.arg
}
