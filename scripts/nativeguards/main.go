// Command nativeguards executes native platform suites and requires named,
// build-selected guards to run and pass without skipping any of their cases.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"
)

type guard struct{ Package, Test string }
type suite struct {
	name, goos, tags string
	packages         []string
	guards           []guard
}
type goRunner func(context.Context, []string, io.Writer) error

func (s *suite) require(pkg string, tests ...string) {
	path := "./" + pkg
	if pkg == "" {
		path = "."
	}
	s.packages = append(s.packages, path)
	full := "github.com/frathe/picfetch"
	if pkg != "" {
		full += "/" + pkg
	}
	for _, test := range tests {
		s.guards = append(s.guards, guard{full, test})
	}
}

func suiteFor(name, hostOS string) (suite, error) {
	s := suite{name: name}
	switch name {
	case "windows":
		s.goos = "windows"
		s.require("internal/wallpaper", "TestSetWindows_TargetPreservesOpaqueIDAndUnicodePath", "TestSetWindows_TargetValidationFailsBeforeMutation")
		s.require("internal/clipboard", "TestCopyFilesWindows_DecodesUTF8WithNonUTF8Default")
		s.require("internal/filepicker", "TestWindowsPickerTransport_EmitsUTF8PathArrays")
		s.require("internal/update", "TestApplyWindows_ReplacesDestAndKeepsOld", "TestApplyWindows_MissingStagedBinaryRestoresDest", "TestWindowsRelaunchCommand_PassesThePIDInTheInheritedEnvironment", "TestClassifyApplyError_WindowsErrno", "TestWaitMilliseconds_NeverConvertsToAnUnboundedWait")
		s.require("internal/ui/autoupdate", "TestUpdater_AutomaticAndManualShareCompleteTransaction", "TestApplyStagedUpdate_SuccessRemovesTheStageOnEveryPlatform")
		s.require("internal/distribution", "TestStoreManaged_DefaultBuildIsFalse")
	case "macos":
		s.goos = "darwin"
		s.require("", "TestInstall_GraftsOntoGLFWsDelegate")
		s.require("internal/openwith", "TestInvokeOpenURLs_DeliversDecodedPaths", "TestInvokeOpenURLs_DecodesPercentEncodedUnicode", "TestInvokeOpenFiles_DeliversEquivalentURIsFromPlainPaths")
		s.require("internal/displays", "TestDisplaySnapshot_PreservesNativeBoundsAndIDs")
		s.require("internal/winpos", "TestPoller_StopDiscardsQueuedReadWithoutDrainingUI", "TestPoller_StopDuringNativeReadWaitsForReadWithoutPublishing")
		s.require("internal/filepicker", "TestDarwinPathTransport_RoundTripsNativeURLPaths")
	case "store":
		s.tags = "microsoftstore"
		s.require("internal/distribution", "TestStoreManaged_MicrosoftStoreBuildIsTrue")
		s.require("internal/ui/autoupdate", "TestUpdater_AutomaticAndManualShareCompleteTransaction")
	default:
		return suite{}, fmt.Errorf("unknown native suite %q", name)
	}
	if s.goos != "" && s.goos != hostOS {
		return suite{}, fmt.Errorf("suite %s requires native %s, running on %s", name, s.goos, hostOS)
	}
	return s, nil
}

func (s *suite) testArgs(flags ...string) []string {
	args := append([]string{"test", "-count=1"}, flags...)
	if s.tags != "" {
		args = append(args, "-tags="+s.tags)
	}
	return args
}

func runSuite(ctx context.Context, s suite, execute goRunner, log, capture io.Writer) error {
	for _, pkg := range s.packages {
		var inventory bytes.Buffer
		if err := execute(ctx, append(s.testArgs("-list", "."), pkg), &inventory); err != nil {
			return fmt.Errorf("inventory %s: %w\n%s", pkg, err, inventory.String())
		}
		_, _ = fmt.Fprintf(log, "Selected %s (%s):\n%s", pkg, s.tags, inventory.String())
		full := "github.com/frathe/picfetch"
		if pkg != "." {
			full += "/" + strings.TrimPrefix(pkg, "./")
		}
		names := strings.Fields(inventory.String())
		for _, required := range s.guards {
			if required.Package == full && !slices.Contains(names, required.Test) {
				return fmt.Errorf("required guard not build-selected: %s %s", full, required.Test)
			}
		}
	}
	var raw bytes.Buffer
	args := append(s.testArgs("-json", "-v", "-timeout=30m"), s.packages...)
	executionErr := execute(ctx, args, io.MultiWriter(capture, &raw))
	evidenceErr := validateEvents(&raw, s.guards, log)
	return errors.Join(executionErr, evidenceErr)
}

type event struct{ Action, Package, Test string }
type guardResult struct {
	runs, passes int
	rejected     bool
}

func validateEvents(input io.Reader, required []guard, log io.Writer) error {
	states := make(map[guard]guardResult, len(required))
	for _, g := range required {
		states[g] = guardResult{}
	}
	decoder := json.NewDecoder(input)
	var failures []error
	for {
		var e event
		if err := decoder.Decode(&e); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("invalid go test event stream: %w", err)
		}
		if e.Action == "fail" {
			failures = append(failures, fmt.Errorf("failed: %s %s", e.Package, e.Test))
		}
		for g, state := range states {
			if e.Package != g.Package || (e.Test != g.Test && !strings.HasPrefix(e.Test, g.Test+"/")) {
				continue
			}
			if e.Action == "skip" || e.Action == "fail" {
				state.rejected = true
			}
			if e.Test == g.Test {
				if e.Action == "run" {
					state.runs++
				}
				if e.Action == "pass" {
					state.passes++
				}
			}
			states[g] = state
		}
	}
	for _, g := range required {
		state := states[g]
		if state.runs != 1 || state.passes != 1 || state.rejected {
			failures = append(failures, fmt.Errorf("required guard did not run and pass without skips: %s %s (run=%d pass=%d rejected=%v)", g.Package, g.Test, state.runs, state.passes, state.rejected))
		} else {
			_, _ = fmt.Fprintf(log, "PASS required: %s %s\n", g.Package, g.Test)
		}
	}
	return errors.Join(failures...)
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("nativeguards", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("suite", "", "windows, macos, or store")
	capturePath := flags.String("capture", "", "raw go test JSON output path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *capturePath == "" {
		return errors.New("usage: nativeguards -suite windows|macos|store -capture <json-file>")
	}
	s, err := suiteFor(*name, runtime.GOOS)
	if err != nil {
		return err
	}
	capture, err := os.Create(*capturePath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	execute := func(ctx context.Context, args []string, out io.Writer) error {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Stdout = out
		cmd.Stderr = stderr
		return cmd.Run()
	}
	_, _ = fmt.Fprintf(stdout, "Native guards: suite=%s runtime=%s/%s Go=%s tags=%q\n", s.name, runtime.GOOS, runtime.GOARCH, runtime.Version(), s.tags)
	err = runSuite(ctx, s, execute, stdout, capture)
	return errors.Join(err, capture.Close())
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "nativeguards:", err)
		os.Exit(1)
	}
}
