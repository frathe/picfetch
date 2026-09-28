package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
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

func TestLaunchPolicyNativeSuite(t *testing.T) {
	for _, tc := range []struct {
		name, host, tags string
		count            int
	}{
		{"launch-policy", "linux", "", 165}, {"launch-policy", "windows", "", 166},
		{"launch-policy", "darwin", "", 169}, {"launch-policy-store", "windows", "microsoftstore", 166},
	} {
		t.Run(tc.name+"/"+tc.host, func(t *testing.T) {
			s, err := suiteFor(tc.name, tc.host)
			if err != nil {
				t.Fatalf("focused native launch suite unavailable: %v", err)
			}
			if s.tags != tc.tags || s.skipTests != "" || len(s.guards) != tc.count {
				t.Fatalf("wrong build selection or inventory: tags=%q skip=%q guards=%d, want %d", s.tags, s.skipTests, len(s.guards), tc.count)
			}
			seenPackages := map[string]bool{}
			for _, pkg := range s.packages {
				if seenPackages[pkg] {
					t.Fatalf("duplicate package execution: %s", pkg)
				}
				seenPackages[pkg] = true
			}
			for _, pkg := range []string{".", "./internal/launch", "./internal/ui", "./internal/ui/autoupdate", "./internal/ui/settingswin", "./internal/distribution"} {
				if !seenPackages[pkg] {
					t.Errorf("omitted required native package %s", pkg)
				}
			}
			wantTags := "-tags=no_emoji,nodynamic"
			if tc.tags != "" {
				wantTags += "," + tc.tags
			}
			if slices.Contains(s.testArgs(), "-skip") || !slices.Contains(s.testArgs(), wantTags) {
				t.Fatalf("wrong native test arguments: %v", s.testArgs())
			}
			for pkg, name := range map[string]string{
				"":                        "TestLaunchStartupContract/ordering/compiled_distribution_and_identity",
				"internal/launch":         "TestLaunchPreparationContract/evidence/default_Explorer_probe_reports_actual_platform_result",
				"internal/ui":             "TestLaunchPolicyIntegration/settings/store_explorer",
				"internal/ui/autoupdate":  "TestUpdaterLaunchPolicy/admission/missing_policy",
				"internal/ui/settingswin": "TestUpdatesTabLaunchPolicy/missing_policy",
			} {
				full := "github.com/frathe/picfetch"
				if pkg != "" {
					full += "/" + pkg
				}
				if !slices.Contains(s.guards, guard{full, name}) {
					t.Errorf("missing launch guard %s %s", full, name)
				}
			}
			if tc.host == "darwin" && !slices.Contains(s.guards, guard{"github.com/frathe/picfetch/internal/openwith", "TestInvokeOpenURLs_DeliversDecodedPaths"}) {
				t.Error("macOS native Open With guard absent")
			}
			if tc.host == "windows" && !slices.Contains(s.guards, guard{"github.com/frathe/picfetch/internal/update", "TestWindowsRelaunchCommand_PassesThePIDInTheInheritedEnvironment"}) {
				t.Error("Windows predecessor guard absent")
			}
			compiled := "TestStoreManaged_DefaultBuildIsFalse"
			if tc.tags != "" {
				compiled = "TestStoreManaged_MicrosoftStoreBuildIsTrue"
			}
			if !slices.Contains(s.guards, guard{"github.com/frathe/picfetch/internal/distribution", compiled}) {
				t.Error("compiled distribution guard absent")
			}
			for _, name := range []string{"store", "explorer", "location_map", "store_explorer", "store_location_map", "ordinary"} {
				if !slices.Contains(s.guards, guard{"github.com/frathe/picfetch/internal/ui", "TestLaunchPolicyIntegration/settings/" + name}) {
					t.Errorf("missing integrated Settings guard %s", name)
				}
			}
			filter := regexp.MustCompile(s.runTests)
			for _, g := range s.guards {
				top, _, _ := strings.Cut(g.Test, "/")
				if !filter.MatchString(top) || filter.MatchString(top+"Extra") {
					t.Fatalf("guard outside exact focus: %v", g)
				}
			}
			for _, unrelated := range []string{"TestE2E_Golden", "TestHEICNativeQualification", "TestLaunchStartupContractExtra"} {
				if filter.MatchString(unrelated) {
					t.Fatalf("selected unrelated test %s", unrelated)
				}
			}
		})
	}
	for _, mismatch := range [][2]string{{"launch-policy", "plan9"}, {"launch-policy-store", "linux"}, {"launch-policy-store", "darwin"}} {
		if _, err := suiteFor(mismatch[0], mismatch[1]); err == nil {
			t.Errorf("accepted wrong native selection %v", mismatch)
		}
	}
	t.Run("evidence_metadata", func(t *testing.T) {
		parent := guard{"github.com/frathe/picfetch", "TestLaunchStartupContract"}
		child := guard{parent.Package, parent.Test + "/ordering/compiled_distribution_and_identity"}
		s := suite{name: "launch-policy", goos: runtime.GOOS, packages: []string{"."}, guards: []guard{parent, child}}
		for _, tc := range []struct {
			name, stream, tags, goos string
			runErr                   error
			complete                 bool
		}{
			{"passed", eventsFor(parent, "run") + eventsFor(child, "run", "pass") + eventsFor(parent, "pass"), "", "", nil, true},
			{"missing_child", eventsFor(parent, "run", "pass"), "", "", errors.New("required child absent"), false},
			{"skipped_child", eventsFor(parent, "run") + eventsFor(child, "run", "skip") + eventsFor(parent, "pass"), "", "", errors.New("required child skipped"), false},
			{"process_failure", eventsFor(parent, "run") + eventsFor(child, "run", "pass") + eventsFor(parent, "pass"), "", "", errors.New("go test failed"), false},
			{"wrong_tags", eventsFor(parent, "run") + eventsFor(child, "run", "pass") + eventsFor(parent, "pass"), "microsoftstore", "", nil, false},
			{"wrong_host", eventsFor(parent, "run") + eventsFor(child, "run", "pass") + eventsFor(parent, "pass"), "", "plan9", nil, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				path := t.TempDir() + "/capture.json"
				if err := os.WriteFile(path, []byte(tc.stream), 0o600); err != nil {
					t.Fatal(err)
				}
				selected := s
				selected.tags = tc.tags
				if tc.goos != "" {
					selected.goos = tc.goos
				}
				metadataErr := writeLaunchMetadata(selected, path, tc.runErr)
				if (metadataErr == nil) != (tc.name == "passed" || tc.name == "process_failure") {
					t.Fatalf("metadata selection/evidence error = %v", metadataErr)
				}
				data, err := os.ReadFile(path + ".metadata.json")
				if err != nil {
					t.Fatalf("raw capture lacks provenance and outcomes: %v", err)
				}
				var metadata struct {
					Suite, HostOS, HostArch, GoVersion, Revision, Tags string
					Complete                                           bool
					Required                                           []struct {
						Package, Test string
						Runs, Passes  int
						Rejected      bool
					}
				}
				if err := json.Unmarshal(data, &metadata); err != nil {
					t.Fatal(err)
				}
				wantTags := "no_emoji,nodynamic"
				if tc.tags != "" {
					wantTags += "," + tc.tags
				}
				if metadata.Suite != s.name || metadata.HostOS != runtime.GOOS || metadata.HostArch != runtime.GOARCH || metadata.GoVersion != runtime.Version() || len(metadata.Revision) != 40 || metadata.Tags != wantTags || metadata.Complete != tc.complete || len(metadata.Required) != 2 {
					t.Fatalf("invalid provenance or completion: %+v", metadata)
				}
				if metadata.Required[0].Package != parent.Package || metadata.Required[0].Test != parent.Test || metadata.Required[1].Test != child.Test {
					t.Fatalf("required outcomes do not identify exact guards: %+v", metadata.Required)
				}
				if tc.name == "missing_child" && (metadata.Required[1].Runs != 0 || metadata.Required[1].Passes != 0) {
					t.Fatalf("missing child marked as seen: %+v", metadata.Required[1])
				}
				if tc.name == "skipped_child" && !metadata.Required[1].Rejected {
					t.Fatal("skipped child not marked rejected")
				}
			})
		}
	})
	t.Run("runner_refusal", func(t *testing.T) {
		s, err := suiteFor("launch-policy", "linux")
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name, target, problem string
		}{
			{"valid", "", ""},
			{"missing_parent", "TestLaunchStartupContract", "inventory"},
			{"missing_child", "TestLaunchPreparationContract/evidence/default_Explorer_probe_reports_actual_platform_result", "missing"},
			{"skipped_required", "TestUpdaterLaunchPolicy/admission/missing_policy", "skip"},
			{"skipped_descendant", "TestLaunchPolicyIntegration/update_entrypoints", "descendant"},
			{"failed_process", "", "process"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var capture bytes.Buffer
				var executed bool
				execute := func(_ context.Context, args []string, out io.Writer) error {
					if !slices.Contains(args, "-tags=no_emoji,nodynamic") || slices.Contains(args, "-skip") {
						t.Fatalf("incorrect build selection: %v", args)
					}
					if slices.Contains(args, "-list") {
						pkg := strings.TrimPrefix(args[len(args)-1], "./")
						full := "github.com/frathe/picfetch"
						if pkg != "." {
							full += "/" + pkg
						}
						seen := map[string]bool{}
						for _, g := range s.guards {
							parent, _, _ := strings.Cut(g.Test, "/")
							if g.Package == full && !seen[parent] && !(tc.problem == "inventory" && parent == tc.target) {
								_, _ = fmt.Fprintln(out, parent)
								seen[parent] = true
							}
						}
						return nil
					}
					executed = true
					for _, g := range s.guards {
						if tc.problem == "missing" && g.Test == tc.target {
							continue
						}
						actions := []string{"run", "pass"}
						if tc.problem == "skip" && g.Test == tc.target {
							actions[1] = "skip"
						}
						_, _ = io.WriteString(out, eventsFor(g, actions...))
						if tc.problem == "descendant" && g.Test == tc.target {
							_, _ = io.WriteString(out, eventsFor(guard{g.Package, g.Test + "/new_case"}, "run", "skip"))
						}
					}
					if tc.problem == "process" {
						return errors.New("go test failed")
					}
					return nil
				}
				err := runSuite(context.Background(), s, execute, io.Discard, &capture)
				if (err == nil) != (tc.problem == "") {
					t.Fatalf("wrong native capture result: %v", err)
				}
				if tc.problem == "inventory" && executed {
					t.Fatal("executed after absent build-selected parent")
				}
				if tc.problem != "inventory" && !executed {
					t.Fatal("never exercised event validation")
				}
			})
		}
	})
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

func TestWindowsCISkipsOnlyInstalledHEICCodecTests(t *testing.T) {
	codecTests := []string{
		"TestHEICNativeQualification", "TestHEICNativePrimarySelection", "TestHEICNativeCorpus",
		"TestHEICWindowsWICProbe", "TestHEICWindowsPrimaryVariants", "TestHEICAnalysisWorker",
	}
	for _, name := range []string{"windows", "store"} {
		t.Run(name, func(t *testing.T) {
			s, err := suiteFor(name, "windows")
			if err != nil {
				t.Fatal(err)
			}
			originalGuards, originalPackages := slices.Clone(s.guards), slices.Clone(s.packages)
			if slices.Contains(s.testArgs(), "-skip") {
				t.Fatal("default native qualification skips tests")
			}
			if err := s.skipHEICCodecs(true); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(s.packages, originalPackages) {
				t.Fatal("codec exception removed package coverage")
			}
			args := s.testArgs()
			i := slices.Index(args, "-skip")
			if i < 0 || i+1 >= len(args) {
				t.Fatalf("codec exception has no execution filter: %v", args)
			}
			filter := regexp.MustCompile(args[i+1])
			for _, name := range codecTests {
				if !filter.MatchString(name) || filter.MatchString(name+"Extra") {
					t.Errorf("codec filter must match exactly %s", name)
				}
			}
			var evidence strings.Builder
			for _, g := range originalGuards {
				top, _, _ := strings.Cut(g.Test, "/")
				excluded := slices.Contains(codecTests, top)
				if filter.MatchString(top) != excluded || slices.Contains(s.guards, g) == excluded {
					t.Errorf("incorrect execution/evidence selection for %v", g)
				}
				if !excluded {
					evidence.WriteString(eventsFor(g, "run", "pass"))
				}
			}
			for _, name := range []string{"TestHEICWindowsWorkerRestrictions", "TestHEICWindowsAlphaMetadata", "TestHEICWorkerLifecycle", "TestHEICWorkerProtocol"} {
				if filter.MatchString(name) {
					t.Errorf("codec exception disabled %s", name)
				}
			}
			if err := validateEvents(strings.NewReader(evidence.String()), s.guards, io.Discard); err != nil {
				t.Fatalf("CI still requires excluded codec evidence: %v", err)
			}
			worker := guard{"github.com/frathe/picfetch/internal/heic", "TestHEICWindowsWorkerRestrictions"}
			missingWorker := strings.ReplaceAll(evidence.String(), eventsFor(worker, "run", "pass"), "")
			if err := validateEvents(strings.NewReader(missingWorker), s.guards, io.Discard); err == nil {
				t.Fatal("CI no longer requires Windows worker restrictions")
			}
		})
	}
}

func TestHEICCodecExceptionRequiresWindowsCI(t *testing.T) {
	for _, tc := range []struct {
		name, host string
		ci         bool
	}{
		{"windows", "windows", false}, {"store", "windows", false},
		{"linux", "linux", true}, {"macos", "darwin", true},
		{"command-admission", "windows", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := suiteFor(tc.name, tc.host)
			if err != nil {
				t.Fatal(err)
			}
			originalGuards := slices.Clone(s.guards)
			if err := s.skipHEICCodecs(tc.ci); err == nil {
				t.Fatal("accepted codec exception outside Windows CI")
			}
			if !slices.Equal(s.guards, originalGuards) || slices.Contains(s.testArgs(), "-skip") {
				t.Fatal("rejected codec exception changed qualification")
			}
		})
	}
}

func fixtureSuite() suite {
	return suite{name: "fixture", tags: "microsoftstore", packages: []string{"./internal/distribution"}, guards: []guard{{"github.com/frathe/picfetch/internal/distribution", "TestRequired"}}}
}

func TestFavoriteOwnershipNativeSuiteRequiresCompleteEvidence(t *testing.T) {
	for _, host := range []string{"linux", "windows", "darwin"} {
		t.Run(host, func(t *testing.T) {
			s, err := suiteFor("favorite-ownership", host)
			if err != nil {
				t.Fatal(err)
			}
			for pkg, name := range map[string]string{
				"internal/favstore":     "TestFavoriteOwnership/active_release",
				"internal/similarity":   "TestAnalysisCacheFileURIPathsReopen/favorite",
				"internal/favthumbs":    "TestSyncFavoriteOwnership/publication",
				"internal/ui/favorites": "TestFavoriteStorageLifecycle/active_native_call",
				"internal/ui":           "TestFavoriteOwnershipIntegration/preview_queued_save",
			} {
				if !slices.Contains(s.guards, guard{"github.com/frathe/picfetch/" + pkg, name}) {
					t.Fatalf("missing required Favorite guard: %s %s", pkg, name)
				}
			}
			filter := regexp.MustCompile(s.runTests)
			if filter.MatchString("TestE2E_Golden") || filter.MatchString("TestHEICNativeQualification") || s.skipTests != "" {
				t.Fatal("Favorite suite includes unrelated tests or skip exemptions")
			}
			var valid strings.Builder
			for _, g := range s.guards {
				top, _, _ := strings.Cut(g.Test, "/")
				if !filter.MatchString(top) || filter.MatchString(top+"Extra") {
					t.Fatalf("required guard outside exact execution filter: %v", g)
				}
				valid.WriteString(eventsFor(g, "run", "pass"))
			}
			if err := validateEvents(strings.NewReader(valid.String()), s.guards, io.Discard); err != nil {
				t.Fatal(err)
			}
			for _, g := range s.guards {
				for _, replacement := range []string{"", eventsFor(g, "run", "skip"), eventsFor(g, "run", "fail")} {
					invalid := strings.Replace(valid.String(), eventsFor(g, "run", "pass"), replacement, 1)
					if err := validateEvents(strings.NewReader(invalid), s.guards, io.Discard); err == nil {
						t.Fatalf("accepted incomplete Favorite evidence: %v", g)
					}
				}
			}
			if err := s.skipHEICCodecs(true); err == nil {
				t.Fatal("Favorite qualification accepted a codec exemption")
			}
		})
	}
}

func TestCommandAdmissionNativeSuiteRunsFocusedGuards(t *testing.T) {
	for _, tc := range []struct {
		host  string
		tests []string
	}{
		{"linux", []string{"TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset"}},
		{"windows", []string{"TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset"}},
		{"darwin", []string{"TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset", "TestSetMenuItemModifierMask_ClearsDefaultCommand"}},
	} {
		t.Run(tc.host, func(t *testing.T) {
			s, err := suiteFor("command-admission", tc.host)
			if err != nil {
				t.Fatal(err)
			}
			var calls [][]string
			var capture bytes.Buffer
			execute := func(_ context.Context, args []string, out io.Writer) error {
				calls = append(calls, slices.Clone(args))
				if args[len(args)-1] != "./internal/ui" || !slices.Contains(args, "-tags=no_emoji,nodynamic") {
					t.Fatalf("wrong native UI package/tags: %v", args)
				}
				for _, arg := range args {
					if strings.HasPrefix(arg, "./") && arg != "./internal/ui" {
						t.Fatalf("focused suite also selected %s", arg)
					}
				}
				if slices.Contains(args, "-list") {
					_, _ = fmt.Fprintln(out, strings.Join(tc.tests, "\n"))
					return nil
				}
				index := slices.Index(args, "-run")
				if index < 0 || index+1 >= len(args) {
					t.Fatalf("focused suite has no test filter: %v", args)
				}
				filter, err := regexp.Compile(args[index+1])
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range tc.tests {
					if !filter.MatchString(name) || filter.MatchString(name+"Extra") || filter.MatchString("Prefix"+name) {
						t.Errorf("filter must select exactly %s", name)
					}
					_, _ = io.WriteString(out, eventsFor(guard{"github.com/frathe/picfetch/internal/ui", name}, "run", "pass"))
				}
				if filter.MatchString("TestE2E_Golden") || filter.MatchString("TestHEICNativeQualification") || slices.Contains(args, "-skip") {
					t.Fatalf("focused suite includes unrelated tests or skip exemptions: %v", args)
				}
				return nil
			}
			if err := runSuite(context.Background(), s, execute, io.Discard, &capture); err != nil {
				t.Fatal(err)
			}
			if len(calls) != 2 || capture.Len() == 0 {
				t.Fatalf("want one inventory and execution with retained events: calls=%v capture=%q", calls, capture.String())
			}
		})
	}
}

func TestCommandAdmissionNativeSuiteRejectsIncompleteEvidence(t *testing.T) {
	names := []string{"TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset", "TestSetMenuItemModifierMask_ClearsDefaultCommand"}
	for _, target := range names {
		for _, problem := range []string{"missing inventory", "missing events", "skipped", "skipped child", "failed"} {
			t.Run(target+"/"+problem, func(t *testing.T) {
				s, err := suiteFor("command-admission", "darwin")
				if err != nil {
					t.Fatal(err)
				}
				executed := false
				execute := func(_ context.Context, args []string, out io.Writer) error {
					if slices.Contains(args, "-list") {
						for _, name := range names {
							if name != target || problem != "missing inventory" {
								_, _ = fmt.Fprintln(out, name)
							}
						}
						return nil
					}
					executed = true
					for _, name := range names {
						g := guard{"github.com/frathe/picfetch/internal/ui", name}
						stream := eventsFor(g, "run", "pass")
						if name == target {
							switch problem {
							case "missing events":
								stream = ""
							case "skipped":
								stream = eventsFor(g, "run", "skip")
							case "skipped child":
								stream += eventsFor(guard{g.Package, g.Test + "/fixture"}, "run", "skip")
							case "failed":
								stream = eventsFor(g, "run", "fail")
							}
						}
						_, _ = io.WriteString(out, stream)
					}
					return nil
				}
				if err := runSuite(context.Background(), s, execute, io.Discard, io.Discard); err == nil {
					t.Fatal("accepted incomplete native UI evidence")
				}
				if problem == "missing inventory" && executed {
					t.Fatal("executed after the required guard was absent from the inventory")
				}
			})
		}
	}
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
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "run: go run ./scripts/nativeguards") {
			continue
		}
		windows := strings.Contains(line, "-suite windows ") || strings.Contains(line, "-suite store ")
		if strings.Contains(line, "-skip-heic-codecs") != windows {
			t.Errorf("only Windows/Store CI must skip installed-codec tests: %s", line)
		}
	}
	if !strings.Contains(text, "runs-on: windows-latest") || strings.Contains(text, "windows-11-arm") {
		t.Fatal("Windows CI must retain its x64 hosted runner")
	}
}

func TestFocusedNativeCIExecutesAndRetainsGuards(t *testing.T) {
	data, err := os.ReadFile("../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Env   map[string]string
			Steps []struct {
				Run  string
				Uses string
				If   string
				With map[string]string
			}
		}
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	t.Run("macos_trial_tools", func(t *testing.T) {
		job := workflow.Jobs["macos-test"]
		if job.Env["PICFETCH_SIMILARITY_ASSETS"] != "${{ github.workspace }}/.scratch/visual-similarity-explorer/assets" {
			t.Error("native trial fixture assets do not match its retained compatibility path")
		}
		found := false
		for _, step := range job.Steps {
			if strings.Contains(step.Run, "^TestNativeLibraryRunner$") {
				found = true
				if step.If != "" || !strings.Contains(step.Run, "-tags no_emoji,nodynamic,explorertrial") || !strings.Contains(step.Run, "native-guards-trial-tools.json") || !strings.Contains(step.Run, "jq -e") {
					t.Error("native trial tool fixture lacks strict, retained native execution")
				}
			}
		}
		if !found {
			t.Error("macOS native trial-tool qualification is absent")
		}
	})
	for _, selected := range []struct{ job, suite string }{
		{"windows-test", "command-admission"}, {"macos-test", "command-admission"},
		{"linux-native", "favorite-ownership"}, {"windows-test", "favorite-ownership"}, {"macos-test", "favorite-ownership"},
		{"linux-native", "launch-policy"}, {"windows-test", "launch-policy"}, {"macos-test", "launch-policy"},
		{"windows-test", "launch-policy-store"},
	} {
		t.Run(selected.job+"/"+selected.suite, func(t *testing.T) {
			jobName := selected.job
			job, ok := workflow.Jobs[jobName]
			if !ok {
				t.Fatal("native job missing")
			}
			var runs, uploads int
			for _, step := range job.Steps {
				if strings.Contains(step.Run, "./scripts/nativeguards -suite "+selected.suite+" ") {
					runs++
					if step.If != "" || strings.Contains(step.Run, "-skip-heic-codecs") || !strings.Contains(step.Run, `-capture "${{ runner.temp }}/native-guards-`+selected.suite+`.json"`) {
						t.Errorf("focused qualification is conditional, exempted or lacks its capture: %+v", step)
					}
				}
				if strings.HasPrefix(step.Uses, "actions/upload-artifact@") && step.With["path"] == "${{ runner.temp }}/native-guards-*.json" {
					uploads++
					if step.If != "always()" || step.With["if-no-files-found"] != "error" {
						t.Error("native guard evidence must be retained even after failure")
					}
				}
			}
			if runs != 1 || uploads != 1 {
				t.Fatalf("focused native commands=%d, evidence uploads=%d; want one of each", runs, uploads)
			}
		})
	}
}
