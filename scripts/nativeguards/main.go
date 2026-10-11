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
	"regexp"
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
	runTests         string
	skipTests        string
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
	case "launch-policy", "launch-policy-store":
		if hostOS != "linux" && hostOS != "windows" && hostOS != "darwin" {
			return suite{}, fmt.Errorf("%s requires Linux, Windows or macOS, running on %s", name, hostOS)
		}
		if name == "launch-policy-store" && hostOS != "windows" {
			return suite{}, fmt.Errorf("launch-policy-store requires native Windows, running on %s", hostOS)
		}
		s.goos = hostOS
		if name == "launch-policy-store" {
			s.tags = "microsoftstore"
		}
		s.requireLaunchPolicy()
		return s, nil
	case "favorite-ownership":
		if hostOS != "linux" && hostOS != "windows" && hostOS != "darwin" {
			return suite{}, fmt.Errorf("favorite-ownership requires Linux, Windows or macOS, running on %s", hostOS)
		}
		s.goos = hostOS
		s.requireFavoriteOwnership()
		return s, nil
	case "command-admission":
		if hostOS != "linux" && hostOS != "windows" && hostOS != "darwin" {
			return suite{}, fmt.Errorf("command-admission requires Linux, Windows or macOS, running on %s", hostOS)
		}
		s.goos = hostOS
		tests := []string{"TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset"}
		if hostOS == "darwin" {
			tests = append(tests, "TestSetMenuItemModifierMask_ClearsDefaultCommand")
		}
		s.require("internal/ui", tests...)
		// Native UI guards must not pull in Linux-only golden rendering or
		// codec qualification. Existing platform suites remain full-package.
		s.runTests = "^(" + strings.Join(tests, "|") + ")$"
		return s, nil
	case "linux":
		s.goos = "linux"
	case "windows":
		s.goos = "windows"
		s.require("internal/wallpaper", "TestSetWindows_TargetPreservesOpaqueIDAndUnicodePath", "TestSetWindows_TargetValidationFailsBeforeMutation")
		s.require("internal/clipboard", "TestCopyFilesWindows_DecodesUTF8WithNonUTF8Default")
		s.require("internal/filepicker", "TestWindowsPickerTransport_EmitsUTF8PathArrays", "TestWindowsSaveTransport_PathIsData")
		s.require("internal/trash", "TestWindowsTrashTransport_PathIsData")
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
		s.goos = "windows"
		s.tags = "microsoftstore"
		s.require("internal/distribution", "TestStoreManaged_MicrosoftStoreBuildIsTrue")
		s.require("internal/ui/autoupdate", "TestUpdater_AutomaticAndManualShareCompleteTransaction")
	default:
		return suite{}, fmt.Errorf("unknown native suite %q", name)
	}
	if s.goos != "" && s.goos != hostOS {
		return suite{}, fmt.Errorf("suite %s requires native %s, running on %s", name, s.goos, hostOS)
	}
	s.requireHEIC()
	return s, nil
}

// requireLaunchPolicy keeps the native qualification focused on the six launch
// contracts and the existing platform-specific startup predecessors.
func (s *suite) requireLaunchPolicy() {
	type family struct {
		pkg, parent string
		children    []string
	}
	families := []family{
		{"", "TestLaunchStartupContract", []string{
			"early_exit", "early_exit/help", "early_exit/malformed_flags", "early_exit/private_heic", "early_exit/private_similarity",
			"ordering", "ordering/native_install_predecessor_ordinary_run", "ordering/capture_failure", "ordering/prepare_failure", "ordering/app_failure", "ordering/run_failure", "ordering/compiled_distribution_and_identity", "ordering/captured_explorer-trial", "ordering/captured_location-map-trial",
			"validation", "validation/dual_trials", "validation/absent_capture",
			"prepared_cleanup", "prepared_cleanup/explorer-trial_app", "prepared_cleanup/explorer-trial_run", "prepared_cleanup/explorer-trial_normal", "prepared_cleanup/location-map-trial_app", "prepared_cleanup/location-map-trial_run", "prepared_cleanup/location-map-trial_normal", "prepared_cleanup/used_path_before_app",
		}},
		{"internal/launch", "TestLaunchPolicyContract", []string{
			"matrix", "matrix/portable_ordinary", "matrix/store_ordinary", "matrix/portable_explorer", "matrix/store_explorer", "matrix/portable_location", "matrix/store_location",
			"identity", "storage", "storage/explorer", "storage/location", "validity", "validity/empty_identity", "validity/dual_trial",
		}},
		{"internal/launch", "TestLaunchPreparationContract", []string{
			"reservation", "reservation/explorer", "reservation/location",
			"resources", "resources/missing_policy_refuses_external_operations", "resources/ordinary_owns_no_external_resource", "resources/Explorer_prerequisite_refuses_before_reservation", "resources/Location_does_not_probe", "resources/nil_acquisition_cannot_claim_a_reservation", "resources/nil_acquisition_cannot_claim_a_reservation/Explorer", "resources/nil_acquisition_cannot_claim_a_reservation/Location", "resources/cancellation_refuses_and_closes_acquisition", "resources/cancellation_refuses_and_closes_acquisition/before", "resources/cancellation_refuses_and_closes_acquisition/after_Explorer_probe", "resources/cancellation_refuses_and_closes_acquisition/after_Location_acquisition",
			"evidence", "evidence/Explorer_incomplete_evidence_and_repeated_close", "evidence/partial_Explorer_acquisition_keeps_evidence_and_both_errors", "evidence/Location_final_flush_and_repeated_close", "evidence/Location_flush_error_remains_stable_on_repeated_close", "evidence/partial_Location_acquisition_joins_flush_failure", "evidence/default_Explorer_probe_reports_actual_platform_result", "evidence/nil_Prepared_methods",
		}},
		{"internal/ui", "TestLaunchPolicyIntegration", []string{
			"construction", "construction/absent_policy_before_storage", "construction/run_absent_policy_before_trial", "construction/explicit_ordinary", "construction/denied_storage_before_preferences_and_consumers", "construction/ordinary_fallbacks_and_identity_derived_cache",
			"feature_lifetime", "feature_lifetime/immutable_explorer", "feature_lifetime/immutable_location_map", "feature_lifetime/independent_viewers",
			"update_entrypoints", "update_records", "backup_order", "backup_order/restore", "backup_order/copy", "backup_order/unreadable", "backup_order/absent", "shutdown", "shutdown/explorer_held_producer_production_hook_postrun", "shutdown/location_map_held_producer_production_hook_postrun",
		}},
		{"internal/ui/autoupdate", "TestUpdaterLaunchPolicy", []string{
			"admission", "admission/store-managed_ordinary", "admission/Explorer_trial", "admission/Location_Map_trial", "admission/store-managed_Explorer_trial", "admission/store-managed_Location_Map_trial", "admission/missing_policy",
			"preconfigured", "persistence", "records", "records/missing", "records/store", "records/explorer", "records/location_map", "records/store_explorer", "records/store_location_map",
		}},
		{"internal/ui/settingswin", "TestUpdatesTabLaunchPolicy", []string{
			"ordinary_portable", "ordinary_store", "explorer_portable", "explorer_store", "location_map_portable", "location_map_store", "missing_policy",
		}},
		{"", "TestTrialLaunchPreservesPredecessorArtifacts", []string{"ordinary", "location-map-trial", "explorer-trial"}},
	}
	for _, purpose := range []string{"explorer", "location"} {
		for _, step := range []string{"new", "invalid", "used", "competing"} {
			families[2].children = append(families[2].children, "reservation/"+purpose+"/"+step)
		}
	}
	for _, name := range []string{"ordinary_portable", "ordinary_store", "explorer_portable", "explorer_store", "location_map_portable", "location_map_store"} {
		families[3].children = append(families[3].children, "construction/"+name+"_consumer_construction")
	}
	restricted := []string{"store", "explorer", "location_map", "store_explorer", "store_location_map"}
	for _, name := range restricted {
		families[3].children = append(families[3].children,
			"update_entrypoints/"+name+"_checks_and_staging", "update_records/"+name+"_startup_and_direct_reporting")
	}
	for _, name := range append(restricted, "ordinary") {
		families[3].children = append(families[3].children, "update_records/"+name+"_last_check_restore_and_persist")
		families[3].children = append(families[3].children, "settings/"+name)
		for _, suffix := range []string{"normal", "explicit"} {
			families[3].children = append(families[3].children, "shutdown/"+name+"_"+suffix)
		}
	}
	families[3].children = append(families[3].children, "settings")
	if s.goos == "darwin" {
		families = append(families,
			family{"", "TestInstall_GraftsOntoGLFWsDelegate", nil},
			family{"internal/openwith", "TestInvokeOpenURLs_DeliversDecodedPaths", nil},
			family{"internal/openwith", "TestInvokeOpenURLs_DecodesPercentEncodedUnicode", nil},
			family{"internal/openwith", "TestInvokeOpenFiles_DeliversEquivalentURIsFromPlainPaths", nil},
		)
	}
	if s.goos == "windows" {
		families = append(families, family{"internal/update", "TestWindowsRelaunchCommand_PassesThePIDInTheInheritedEnvironment", nil})
	}
	distributionGuard := "TestStoreManaged_DefaultBuildIsFalse"
	if s.name == "launch-policy-store" {
		distributionGuard = "TestStoreManaged_MicrosoftStoreBuildIsTrue"
	}
	families = append(families, family{"internal/distribution", distributionGuard, nil})
	var parents []string
	for _, item := range families {
		tests := []string{item.parent}
		for _, child := range item.children {
			tests = append(tests, item.parent+"/"+child)
		}
		s.require(item.pkg, tests...)
		parents = append(parents, item.parent)
	}
	packages := s.packages
	s.packages = nil
	for _, pkg := range packages {
		if !slices.Contains(s.packages, pkg) {
			s.packages = append(s.packages, pkg)
		}
	}
	s.runTests = "^(" + strings.Join(parents, "|") + ")$"
}

// Favorite ownership uses focused parents so native root UI checks do not
// select platform-specific golden rendering or installed-codec qualification.
func (s *suite) requireFavoriteOwnership() {
	families := []struct {
		pkg, parent string
		children    []string
	}{
		{"internal/favstore", "TestFavoriteOwnership", []string{"move", "directory_replacement", "list_replacement", "list_change", "idle_removal", "active_release"}},
		{"internal/favstore", "TestFavoriteCancellation", []string{"cancelled", "changed_while_reading", "growth", "save_before", "save_after", "save_after_replacement"}},
		{"internal/favstore", "TestFavoriteConflicts", []string{
			"publication", "retirement", "occupied", "identical_list", "directory", "base", "missing_list",
			"removal_identical_list", "removal_directory", "removal_move", "removal_missing_list", "removal_malformed", "removal_cancelled", "removal_committed_after_cancel",
		}},
		{"internal/similarity", "TestAnalysisCacheFileURIPathsReopen", []string{"general", "favorite"}},
		{"internal/favthumbs", "TestSyncFavoriteOwnership", []string{"held_read", "publication", "cleanup", "sweep", "fresh_record", "move", "remove", "identical_list", "directory"}},
		{"internal/ui/favorites", "TestFavoriteStorageLifecycle", []string{"close", "source", "root", "modal", "stop", "active_native_call"}},
		{"internal/ui/favorites", "TestFavoriteStorageCommittedEffects", []string{"removal_close", "removal_root", "removal_stop", "removal_failed_after_close", "close", "root", "stop", "replacement", "refresh_failure"}},
		{"internal/ui", "TestFavoriteOwnershipIntegration", []string{
			"preview_queued_open", "preview_queued_save", "native_removal_close", "native_removal_root", "native_removal_shutdown",
			"committed_save_close", "committed_save_opt_out", "committed_save_shutdown", "queued_replay", "modal", "source", "malformed", "identical_replacement", "shutdown",
		}},
		{"internal/ui", "TestFavoritePreviewAfterCommitBeforeNotificationRejectsOldMemoryHit", nil},
		{"internal/ui", "TestShutdownCancelsFavoritePreviews", nil},
	}
	var parents []string
	for _, family := range families {
		tests := []string{family.parent}
		for _, child := range family.children {
			tests = append(tests, family.parent+"/"+child)
		}
		s.require(family.pkg, tests...)
		parents = append(parents, family.parent)
	}
	s.packages = slices.Compact(s.packages)
	s.runTests = "^(" + strings.Join(parents, "|") + ")$"
}

// requireHEIC records the cases that currently exist, including named fixture
// checks. Passing this inventory does not establish the wider platform/package
// qualification evidence required by the HEIC specification.
func (s *suite) requireHEIC() {
	tests := []string{"TestHEICNativeQualification", "TestHEICNativePrimarySelection", "TestHEICNativeCorpus"}
	for _, name := range []string{
		"probe_8_and_10_bit_pixels", "full_primary_8_bit", "full_primary_10_bit",
		"container-rotate", "exif-rotate", "container-and-exif", "sequence",
		"AVIF_keeps_existing_decoder", "HDR", "wide_gamut", "oversized_dimensions", "malformed_box",
	} {
		tests = append(tests, "TestHEICNativeQualification/"+name)
	}
	for _, name := range []string{
		"icc-srgb8", "icc-srgb10", "icc-p3-linear8", "icc-p3-linear10", "color8", "color10",
		"grid8", "grid10", "mirror-horizontal8", "mirror-horizontal10", "mirror-rotate8",
		"alpha-straight8", "alpha-straight10", "alpha-premultiplied8", "alpha-premultiplied10",
	} {
		tests = append(tests, "TestHEICNativeCorpus/"+name)
	}
	switch s.goos {
	case "linux":
		tests = append(tests, "TestHEICLinuxHeaderProbe", "TestHEICLinuxWorkerRestrictions", "TestHEICWorkerDiesWithProducer")
	case "darwin":
		tests = append(tests, "TestHEICDarwinNativeQualification", "TestHEICDarwinInheritedSandbox")
		for _, name := range []string{
			"representative_pixels", "probe8.heic", "probe10.heic", "container-rotate.heic",
			"exif-rotate.heic", "container-and-exif.heic", "HDR_native_rendering", "pixel_budget_refusal",
			"primary_metadata",
		} {
			tests = append(tests, "TestHEICDarwinNativeQualification/"+name)
		}
	case "windows":
		tests = append(tests, "TestHEICWindowsWICProbe", "TestHEICWindowsPrimaryVariants", "TestHEICWindowsWorkerRestrictions", "TestHEICWindowsAlphaMetadata")
		for _, name := range []string{"primary_1_renumber_false", "primary_2_renumber_false", "primary_1_renumber_true", "primary_2_renumber_true"} {
			tests = append(tests, "TestHEICWindowsPrimaryVariants/"+name)
		}
	}
	s.require("internal/heic", tests...)
	imagingTests := []string{"TestHEICNativeQualification"}
	for _, name := range []string{"probe8", "probe10", "container-rotate", "exif-rotate", "container-and-exif", "nonfirst-primary"} {
		imagingTests = append(imagingTests, "TestHEICNativeQualification/"+name)
	}
	s.require("internal/imaging", imagingTests...)
	s.require("internal/similarity", "TestHEICAnalysisWorker",
		"TestHEICAnalysisWorker/native_finite", "TestHEICAnalysisWorker/native_retained_search", "TestHEICAnalysisWorker/native_limits")
}

func (s *suite) testArgs(flags ...string) []string {
	args := append([]string{"test", "-count=1"}, flags...)
	if s.runTests != "" {
		args = append(args, "-run", s.runTests)
	}
	if s.skipTests != "" {
		args = append(args, "-skip", s.skipTests)
	}
	return append(args, "-tags="+s.buildTags())
}

func (s *suite) buildTags() string {
	tags := "no_emoji,nodynamic"
	if s.tags != "" {
		tags += "," + s.tags
	}
	return tags
}

// skipHEICCodecs preserves worker, metadata and portable regressions on hosted
// Windows CI, which has no Microsoft HEIF/HEVC extensions. Local runs stay strict.
func (s *suite) skipHEICCodecs(githubActions bool) error {
	if !githubActions || s.goos != "windows" || (s.name != "windows" && s.name != "store") {
		return errors.New("-skip-heic-codecs requires a Windows or Store suite in GitHub Actions")
	}
	s.skipTests = "^(TestHEICNativeQualification|TestHEICNativePrimarySelection|TestHEICNativeCorpus|TestHEICWindowsWICProbe|TestHEICWindowsPrimaryVariants|TestHEICAnalysisWorker)$"
	filter := regexp.MustCompile(s.skipTests)
	s.guards = slices.DeleteFunc(s.guards, func(g guard) bool {
		top, _, _ := strings.Cut(g.Test, "/")
		return filter.MatchString(top)
	})
	return nil
}

func runSuite(ctx context.Context, s suite, execute goRunner, log, capture io.Writer) error {
	inventoryPattern := "."
	if s.runTests != "" {
		inventoryPattern = s.runTests
	}
	for _, pkg := range s.packages {
		var inventory bytes.Buffer
		if err := execute(ctx, append(s.testArgs("-list", inventoryPattern), pkg), &inventory); err != nil {
			return fmt.Errorf("inventory %s: %w\n%s", pkg, err, inventory.String())
		}
		_, _ = fmt.Fprintf(log, "Selected %s (%s):\n%s", pkg, s.tags, inventory.String())
		full := "github.com/frathe/picfetch"
		if pkg != "." {
			full += "/" + strings.TrimPrefix(pkg, "./")
		}
		names := strings.Fields(inventory.String())
		for _, required := range s.guards {
			topLevel, _, _ := strings.Cut(required.Test, "/")
			if required.Package == full && !slices.Contains(names, topLevel) {
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

type event struct{ Action, Package, Test, ImportPath string }
type guardResult struct {
	runs, passes int
	rejected     bool
}

func validateEvents(input io.Reader, required []guard, log io.Writer) error {
	_, err := inspectEvents(input, required, log)
	return err
}

func inspectEvents(input io.Reader, required []guard, log io.Writer) (map[guard]guardResult, error) {
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
			return states, fmt.Errorf("invalid go test event stream: %w", err)
		}
		if e.Action == "build-output" || e.Action == "build-fail" {
			if e.ImportPath == "" || e.Package != "" || e.Test != "" {
				return states, errors.New("invalid go build event")
			}
			if e.Action == "build-fail" {
				failures = append(failures, fmt.Errorf("failed build: %s", e.ImportPath))
			}
			continue
		}
		if e.Action == "" || e.Package == "" {
			return states, errors.New("invalid go test event: missing action or package")
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
					if state.runs != 1 || state.passes != 0 {
						state.rejected = true
					}
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
	return states, errors.Join(failures...)
}

func writeLaunchMetadata(s suite, capturePath string, runErr error) error {
	selected, selectionErr := suiteFor(s.name, runtime.GOOS)
	if selectionErr == nil && (s.goos != selected.goos || s.tags != selected.tags) {
		selectionErr = errors.New("capture build selection differs from native suite")
	}
	raw, err := os.Open(capturePath)
	if err != nil {
		return err
	}
	states, evidenceErr := inspectEvents(raw, s.guards, io.Discard)
	closeErr := raw.Close()
	revision, revisionErr := exec.Command("git", "rev-parse", "HEAD").Output()
	dirty, dirtyErr := exec.Command("git", "status", "--porcelain").Output()
	type outcome struct {
		Package, Test string
		Runs, Passes  int
		Rejected      bool
	}
	metadata := struct {
		Suite, HostOS, HostArch, GoVersion, Revision, Tags string
		Dirty, Complete                                    bool
		Required                                           []outcome
	}{
		Suite: s.name, HostOS: runtime.GOOS, HostArch: runtime.GOARCH,
		GoVersion: runtime.Version(), Revision: strings.TrimSpace(string(revision)), Tags: s.buildTags(),
		Dirty:    len(dirty) != 0,
		Complete: runErr == nil && selectionErr == nil && evidenceErr == nil && closeErr == nil && revisionErr == nil && dirtyErr == nil,
	}
	for _, g := range s.guards {
		state := states[g]
		metadata.Required = append(metadata.Required, outcome{g.Package, g.Test, state.runs, state.passes, state.rejected})
	}
	encoded, encodeErr := json.MarshalIndent(metadata, "", "  ")
	if encodeErr != nil {
		return encodeErr
	}
	encoded = append(encoded, '\n')
	writeErr := os.WriteFile(capturePath+".metadata.json", encoded, 0o600)
	return errors.Join(selectionErr, evidenceErr, closeErr, revisionErr, dirtyErr, writeErr)
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("nativeguards", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("suite", "", "linux, windows, macos, store, launch-policy, launch-policy-store, command-admission, or favorite-ownership")
	capturePath := flags.String("capture", "", "raw go test JSON output path")
	skipCodecs := flags.Bool("skip-heic-codecs", false, "exclude installed HEIC codec tests in Windows GitHub Actions only")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *capturePath == "" {
		return errors.New("usage: nativeguards -suite linux|windows|macos|store|launch-policy|launch-policy-store|command-admission|favorite-ownership -capture <json-file> [-skip-heic-codecs]")
	}
	s, err := suiteFor(*name, runtime.GOOS)
	if err != nil {
		return err
	}
	if *skipCodecs {
		if err := s.skipHEICCodecs(os.Getenv("GITHUB_ACTIONS") == "true"); err != nil {
			return err
		}
	}
	capture, err := os.Create(*capturePath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
	defer cancel()
	execute := func(ctx context.Context, args []string, out io.Writer) error {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Env = os.Environ()
		if s.name != "command-admission" && s.name != "favorite-ownership" && s.name != "launch-policy" && s.name != "launch-policy-store" {
			cmd.Env = append(cmd.Env, "PICFETCH_HEIC_NATIVE_TEST=1")
		}
		cmd.Stdout = out
		cmd.Stderr = stderr
		return cmd.Run()
	}
	_, _ = fmt.Fprintf(stdout, "Native guards: suite=%s runtime=%s/%s Go=%s tags=%q\n", s.name, runtime.GOOS, runtime.GOARCH, runtime.Version(), s.tags)
	if s.skipTests != "" {
		_, _ = fmt.Fprintf(stdout, "HEIC: Windows CI excludes installed-codec tests matching %s; this run does not qualify Windows/Store HEIC decoding.\n", s.skipTests)
	}
	if s.name == "command-admission" {
		_, _ = fmt.Fprintln(stdout, "Command admission: filesystem and isolated native-menu guards; physical-input qualification remains separate.")
	} else if s.name == "favorite-ownership" {
		_, _ = fmt.Fprintln(stdout, "Favorite ownership: captured filesystem identities, URI paths, mutation and UI/preview lifecycle guards; native Trash uses isolated fixtures.")
	} else if s.name == "launch-policy" || s.name == "launch-policy-store" {
		_, _ = fmt.Fprintln(stdout, "Launch policy: focused startup, preparation, composition, updater and Settings guards with native predecessor coverage.")
	} else {
		_, _ = fmt.Fprintln(stdout, "HEIC: this inventory checks implemented cases; full fixture, packaged-open, codec-absence/recheck and target-matrix evidence remains a separate qualification requirement.")
	}
	err = runSuite(ctx, s, execute, stdout, capture)
	closeErr := capture.Close()
	if s.name == "launch-policy" || s.name == "launch-policy-store" {
		metadataErr := writeLaunchMetadata(s, *capturePath, errors.Join(err, closeErr))
		return errors.Join(err, closeErr, metadataErr)
	}
	return errors.Join(err, closeErr)
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "nativeguards:", err)
		os.Exit(1)
	}
}
