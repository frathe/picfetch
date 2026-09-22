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
		{"windows", "windows", "github.com/frathe/picfetch/internal/distribution", "TestStoreManaged_DefaultBuildIsFalse", ""},
		{"macos", "darwin", "github.com/frathe/picfetch", "TestInstall_GraftsOntoGLFWsDelegate", ""},
		{"store", "windows", "github.com/frathe/picfetch/internal/distribution", "TestStoreManaged_MicrosoftStoreBuildIsTrue", "microsoftstore"},
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

func TestHEICNativeInventory(t *testing.T) {
	for _, tc := range []struct{ name, host, platformTest string }{
		{"linux", "linux", ""},
		{"macos", "darwin", "TestHEICDarwinNativeQualification"},
		{"windows", "windows", "TestHEICWindowsWICProbe"},
		{"store", "windows", "TestHEICWindowsWICProbe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := suiteFor(tc.name, tc.host)
			if err != nil {
				t.Fatal(err)
			}
			for _, required := range []guard{
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeQualification"},
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeQualification/full_primary_10_bit"},
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativePrimarySelection"},
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeCorpus/icc-srgb8"},
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeCorpus/icc-p3-linear10"},
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeCorpus/grid10"},
				{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeCorpus/alpha-premultiplied10"},
				{"github.com/frathe/picfetch/internal/similarity", "TestHEICAnalysisWorker/native_finite"},
				{"github.com/frathe/picfetch/internal/similarity", "TestHEICAnalysisWorker/native_retained_search"},
				{"github.com/frathe/picfetch/internal/similarity", "TestHEICAnalysisWorker/native_limits"},
				{"github.com/frathe/picfetch/internal/imaging", "TestHEICNativeQualification"},
				{"github.com/frathe/picfetch/internal/imaging", "TestHEICNativeQualification/nonfirst-primary"},
			} {
				if !slices.Contains(s.guards, required) {
					t.Errorf("native inventory omitted %v", required)
				}
			}
			if tc.platformTest != "" && !slices.Contains(s.guards, guard{"github.com/frathe/picfetch/internal/heic", tc.platformTest}) {
				t.Errorf("native inventory omitted %s", tc.platformTest)
			}
		})
	}
	if _, err := suiteFor("store", "linux"); err == nil {
		t.Fatal("Store native qualification accepted a Linux host")
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
		"pass before run": eventsFor(g, "pass", "run"),
		"null event":      valid + "null\n",
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

func TestNativeEventsHandleBuildDiagnostics(t *testing.T) {
	g := fixtureSuite().guards[0]
	valid := eventsFor(g, "run", "pass")
	for _, tc := range []struct {
		name, diagnostic string
		wantError        string
	}{
		{"linker warning", `{"ImportPath":"example.test","Action":"build-output","Output":"ld: warning: duplicate library\\n"}`, ""},
		{"failed build", `{"ImportPath":"example.test","Action":"build-fail"}`, "failed build: example.test"},
		{"missing import path", `{"Action":"build-output","Output":"warning"}`, "invalid go build event"},
		{"diagnostic is not test evidence", `{"ImportPath":"example.test","Action":"build-output","Test":"TestRequired"}`, "invalid go build event"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEvents(strings.NewReader(tc.diagnostic+"\n"+valid), []guard{g}, io.Discard)
			if tc.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("validation = %v, want %q", err, tc.wantError)
			}
		})
	}
	if err := validateEvents(strings.NewReader(`{"ImportPath":"example.test","Action":"build-output"}`), []guard{g}, io.Discard); err == nil {
		t.Fatal("build output substituted for a required test")
	}
}

func TestHEICNativeRunnerRequiresEveryFixtureEvent(t *testing.T) {
	parent := guard{"github.com/frathe/picfetch/internal/heic", "TestHEICNativeQualification"}
	child := guard{parent.Package, parent.Test + "/full_primary_10_bit"}
	s := suite{name: "fixture", packages: []string{"./internal/heic"}, guards: []guard{parent, child}}
	for name, childEvents := range map[string]string{
		"present": eventsFor(child, "run", "pass"),
		"missing": "",
		"skipped": eventsFor(child, "run", "skip"),
		"failed":  eventsFor(child, "run", "fail"),
	} {
		t.Run(name, func(t *testing.T) {
			var capture bytes.Buffer
			execute := func(_ context.Context, args []string, out io.Writer) error {
				if slices.Contains(args, "-list") {
					_, _ = fmt.Fprintln(out, parent.Test)
				} else {
					_, _ = io.WriteString(out, eventsFor(parent, "run")+childEvents+eventsFor(parent, "pass"))
				}
				return nil
			}
			err := runSuite(context.Background(), s, execute, io.Discard, &capture)
			if (err == nil) != (name == "present") {
				t.Fatalf("fixture evidence %q: %v", name, err)
			}
			if capture.Len() == 0 {
				t.Fatal("top-level build inventory prevented fixture event validation")
			}
		})
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
	for _, name := range []string{"linux", "windows", "macos", "store"} {
		if !strings.Contains(text, "./scripts/nativeguards -suite "+name+" -capture") {
			t.Errorf("CI omits %s guard runner", name)
		}
	}
	if !strings.Contains(text, "native-guards-${{ runner.os }}") || !strings.Contains(text, "if: always()") {
		t.Fatal("raw native guard evidence not retained")
	}
}
