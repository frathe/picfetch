package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestNativeSuitesSelectPlatformAndDistributionGuards(t *testing.T) {
	for _, tc := range []struct{ name, os, pkg, test, tags string }{
		{"windows", "windows", "github.com/frathe/picfetch/internal/wallpaper", "TestSetWindows_TargetValidationFailsBeforeMutation", ""},
		{"windows", "windows", "github.com/frathe/picfetch/internal/clipboard", "TestCopyFilesWindows_DecodesUTF8WithNonUTF8Default", ""},
		{"windows", "windows", "github.com/frathe/picfetch/internal/filepicker", "TestWindowsPickerTransport_EmitsUTF8PathArrays", ""},
		{"windows", "windows", "github.com/frathe/picfetch/internal/update", "TestApplyWindows_MissingStagedBinaryRestoresDest", ""},
		{"windows", "windows", "github.com/frathe/picfetch/internal/update", "TestDownloadedCompanionsRemainVerifiedAfterPersistence", ""},
		{"windows", "windows", "github.com/frathe/picfetch/internal/update", "TestApplyInstallsCompanionsAndRollsBackOnBinaryFailure", ""},
		{"windows", "windows", "github.com/frathe/picfetch/internal/distribution", "TestStoreManaged_DefaultBuildIsFalse", ""},
		{"macos", "darwin", "github.com/frathe/picfetch", "TestInstall_GraftsOntoGLFWsDelegate", ""},
		{"heic-macos", "darwin", "github.com/frathe/picfetch/internal/heicdecode/client", "TestNativeMacSandboxHelper", "heicnative"},
		{"heic-macos", "darwin", "github.com/frathe/picfetch/internal/update", "TestNativeMacUpdatePreservesSignedBundle", "heicnative"},
		{"heic-macos", "darwin", "github.com/frathe/picfetch/internal/update", "TestNativeMacLegacyUpdateRequiresCompleteReinstall", "heicnative"},
		{"heic-linux", "linux", "github.com/frathe/picfetch/internal/heicdecode/client", "TestNativeSandboxHelper", "heicnative"},
		{"heic-macos", "darwin", "github.com/frathe/picfetch/internal/ui", "TestNativePackagedHEICActivation", "heicnative"},
		{"heic-macos", "darwin", "github.com/frathe/picfetch/internal/similarity", "TestNativeHEICAnalysisPixels", "heicnative"},
		{"heic-linux", "linux", "github.com/frathe/picfetch/internal/similarity", "TestNativeHEICAnalysisPixels", "heicnative"},
		{"heic-windows", "windows", "github.com/frathe/picfetch/internal/similarity", "TestNativeHEICAnalysisPixels", "heicnative"},
		{"heic-linux", "linux", "github.com/frathe/picfetch/internal/ui", "TestNativePackagedHEICActivation", "heicnative"},
		{"heic-windows", "windows", "github.com/frathe/picfetch/internal/ui", "TestNativePackagedHEICActivation", "heicnative"},
		{"heic-linux", "linux", "github.com/frathe/picfetch/internal/heicdecode/worker", "TestNativeLinuxRuntimePolicy", "heicnative"},
		{"heic-windows", "windows", "github.com/frathe/picfetch/internal/heicdecode/client", "TestInheritedRemoteCancellationJoins", "heicnative"},
		{"heic-windows", "windows", "github.com/frathe/picfetch/internal/similarity", "TestAnalysisWorkersUseHEICOwner", "heicnative"},
		{"heic-windows", "windows", "github.com/frathe/picfetch/internal/heicdecode/winisolation", "TestNativeWindowsJobAndToken", "heicnative"},
		{"heic-windows", "windows", "github.com/frathe/picfetch/internal/heicdecode/client", "TestNativeSandboxHelper", "heicnative"},
		{"store", "darwin", "github.com/frathe/picfetch/internal/distribution", "TestStoreManaged_MicrosoftStoreBuildIsTrue", "microsoftstore"},
	} {
		t.Run(tc.name+"/"+tc.test, func(t *testing.T) {
			s, err := suiteFor(tc.name, tc.os)
			if err != nil || s.tags != tc.tags || !slices.Contains(s.guards, guard{tc.pkg, tc.test}) {
				t.Fatalf("suite=%+v, error=%v; missing expected guard/tags", s, err)
			}
		})
	}
	for _, tc := range [][2]string{{"windows", "darwin"}, {"macos", "windows"}, {"unknown", "linux"}} {
		if _, err := suiteFor(tc[0], tc[1]); err == nil {
			t.Errorf("accepted %v", tc)
		}
	}
}

func fixtureSuite() suite {
	return suite{name: "fixture", tags: "microsoftstore", packages: []string{"./internal/distribution"}, guards: []guard{{"github.com/frathe/picfetch/internal/distribution", "TestRequired"}}}
}

func eventsFor(g guard, actions ...string) string {
	var out bytes.Buffer
	for _, action := range actions {
		_ = json.NewEncoder(&out).Encode(map[string]string{"Package": g.Package, "Test": g.Test, "Action": action})
	}
	return out.String()
}

func TestNativeRunnerListsMatchingTagsBeforeExecutingFullSuite(t *testing.T) {
	s := fixtureSuite()
	var calls [][]string
	var capture, log bytes.Buffer
	execute := func(_ context.Context, args []string, out io.Writer) error {
		calls = append(calls, slices.Clone(args))
		if slices.Contains(args, "-list") {
			_, _ = fmt.Fprintln(out, "TestRequired")
			return nil
		}
		_, _ = io.WriteString(out, eventsFor(s.guards[0], "run", "pass"))
		return nil
	}
	if err := runSuite(context.Background(), s, execute, &log, &capture); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("commands=%v, want inventory then execution", calls)
	}
	for _, args := range calls {
		if !slices.Contains(args, "-tags=no_emoji,nodynamic,microsoftstore") || !slices.Contains(args, "./internal/distribution") {
			t.Fatalf("selection mismatch: %v", args)
		}
	}
	if !slices.Contains(calls[1], "-json") || !slices.Contains(calls[1], "-count=1") || !slices.Contains(calls[1], "-v") || slices.Contains(calls[1], "-run") {
		t.Fatalf("execution=%v, want full uncached verbose JSON suite", calls[1])
	}
	if !strings.Contains(log.String(), "TestRequired") || !strings.Contains(capture.String(), `"Action":"pass"`) {
		t.Fatal("named result or raw evidence missing")
	}
}

func TestNativeRunnerRejectsMissingInventoryBeforeExecution(t *testing.T) {
	calls := 0
	execute := func(_ context.Context, _ []string, out io.Writer) error {
		calls++
		_, _ = fmt.Fprintln(out, "TestSomethingElse")
		return nil
	}
	if err := runSuite(context.Background(), fixtureSuite(), execute, io.Discard, io.Discard); err == nil {
		t.Fatal("missing build-selected guard accepted")
	}
	if calls != 1 {
		t.Fatalf("commands=%d, wanted only inventory", calls)
	}
}

func TestNativeEventsRejectMissingSkippedFailedOrMalformedEvidence(t *testing.T) {
	g := fixtureSuite().guards[0]
	valid := eventsFor(g, "run", "pass")
	for name, stream := range map[string]string{
		"absent": "", "not run": eventsFor(g, "pass"), "unfinished": eventsFor(g, "run"),
		"skipped": eventsFor(g, "run", "skip"), "failed": eventsFor(g, "run", "fail"),
		"skipped child": valid + eventsFor(guard{g.Package, g.Test + "/Unicode"}, "run", "skip"),
		"wrong package": eventsFor(guard{"other", g.Test}, "run", "pass"),
		"duplicate":     valid + valid, "malformed": valid + "{",
		"package failure": valid + eventsFor(guard{g.Package, ""}, "fail"),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateEvents(strings.NewReader(stream), []guard{g}, io.Discard); err == nil {
				t.Fatal("invalid execution evidence accepted")
			}
		})
	}
	if err := validateEvents(strings.NewReader(valid), []guard{g}, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestNativeRunnerPreservesRawOutputAndProcessFailure(t *testing.T) {
	sentinel := errors.New("go test failed")
	s := fixtureSuite()
	var capture bytes.Buffer
	execute := func(_ context.Context, args []string, out io.Writer) error {
		if slices.Contains(args, "-list") {
			_, _ = fmt.Fprintln(out, "TestRequired")
			return nil
		}
		_, _ = io.WriteString(out, eventsFor(s.guards[0], "run", "pass"))
		return sentinel
	}
	if err := runSuite(context.Background(), s, execute, io.Discard, &capture); !errors.Is(err, sentinel) {
		t.Fatalf("error=%v, want process failure", err)
	}
	if capture.Len() == 0 {
		t.Fatal("failure discarded raw evidence")
	}
}

func TestNativeCIExecutesAndRetainsEveryDeclaredSuite(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, name := range []string{"windows", "macos", "heic-macos", "heic-linux", "store"} {
		if !strings.Contains(text, "./scripts/nativeguards -suite "+name+" -capture") {
			t.Errorf("CI omits %s guard runner", name)
		}
	}
	child, err := os.ReadFile("../../packaging/heic/qualify-windows-child.ps1")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"./packaging/heic/qualify-windows.ps1 -Scenario standalone", "./packaging/heic/qualify-windows.ps1 -Scenario msix", "native-guards-Windows-MSIX-${{ matrix.arch }}", "heic-standard-user/", "heic-installed-msix/"} {
		if !strings.Contains(text, required) {
			t.Errorf("CI omits Windows qualification contract %q", required)
		}
	}
	if !strings.Contains(string(child), "./scripts/nativeguards -suite heic-windows -capture") || !strings.Contains(string(child), "HEICInstalledActivation") {
		t.Fatal("standard-user runner lost native or installed activation")
	}
	if !strings.Contains(text, "native-guards-${{ runner.os }}") || !strings.Contains(text, "if: always()") {
		t.Fatal("raw native guard evidence not retained")
	}
}

func TestNativeApplicationGuardRunsWithHelperSuite(t *testing.T) {
	s, err := suiteFor("heic-linux", "linux")
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	execute := func(_ context.Context, args []string, out io.Writer) error {
		calls = append(calls, slices.Clone(args))
		for _, g := range s.guards {
			pkg := "./" + strings.TrimPrefix(g.Package, "github.com/frathe/picfetch/")
			if !slices.Contains(args, pkg) {
				continue
			}
			if slices.Contains(args, "-list") {
				_, _ = fmt.Fprintln(out, g.Test)
			} else {
				_, _ = io.WriteString(out, eventsFor(g, "run", "pass"))
			}
		}
		return nil
	}
	if err = runSuite(context.Background(), s, execute, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	var helper, application, analysis bool
	for _, args := range calls {
		if slices.Contains(args, "-list") {
			continue
		}
		if slices.Contains(args, "./internal/ui") {
			application = slices.Contains(args, "-run") && slices.Contains(args, "^TestNativePackagedHEICActivation$")
		} else if slices.Contains(args, "./internal/similarity") {
			analysis = slices.Contains(args, "-run") && slices.Contains(args, ".") && !slices.Contains(args, "./internal/heicdecode/client")
		} else {
			helper = !slices.Contains(args, "-run")
		}
	}
	if !helper || !application || !analysis {
		t.Fatalf("native runner lost helper, application or analysis coverage: %v", calls)
	}
}

func TestLinuxApplicationUsesCGOWithoutChangingHelperBuild(t *testing.T) {
	t.Setenv("CGO_ENABLED", "unexpected")
	s, err := suiteFor("heic-linux", "linux")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"./internal/ui", "./internal/heicdecode/client", "./internal/heicdecode/worker", "./internal/similarity"} {
		for _, flags := range [][]string{{"-list", "."}, {"-json", "-run", "TestNative"}} {
			command := s.command(context.Background(), append(s.testArgs(flags...), pkg))
			want := "CGO_ENABLED=0"
			if pkg == "./internal/ui" || pkg == "./internal/similarity" {
				want = "CGO_ENABLED=1"
			}
			for _, value := range command.Environ() {
				if strings.HasPrefix(value, "CGO_ENABLED=") && value != want {
					t.Errorf("%s %v: %s, want %s", pkg, flags, value, want)
				}
			}
		}
	}
}
