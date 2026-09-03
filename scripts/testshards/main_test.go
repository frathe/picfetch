package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSummarize_ReportsInterleavedPackagesAndTopLevelTests(t *testing.T) {
	path := writeEventStream(t, `
{"Action":"start","Package":"example/b"}
{"Action":"start","Package":"example/a"}
{"Action":"run","Package":"example/a","Test":"TestParent"}
{"Action":"run","Package":"example/a","Test":"TestParent/child"}
{"Action":"pass","Package":"example/a","Test":"TestParent/child","Elapsed":0.5}
{"Action":"pass","Package":"example/a","Test":"TestParent","Elapsed":1.5}
{"Action":"skip","Package":"example/b","Elapsed":0}
{"Action":"pass","Package":"example/a","Elapsed":2}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"summarize", "-json", path}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	want := strings.TrimSpace(`
summary	runs=1
package	example/a	run-1=pass:2.000	median=2.000
package	example/b	run-1=skip:0.000	median=0.000
test	example/a	TestParent	run-1=pass:1.500	median=1.500
`) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestSummarize_RejectsDuplicateTerminalOutcome(t *testing.T) {
	path := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestRepeated"}
{"Action":"pass","Package":"example/ui","Test":"TestRepeated","Elapsed":1}
{"Action":"fail","Package":"example/ui","Test":"TestRepeated","Elapsed":2}
{"Action":"fail","Package":"example/ui","Elapsed":3}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"summarize", "-json", path}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "duplicate terminal outcome for example/ui/TestRepeated") {
		t.Fatalf("got %v, want duplicate TestRepeated diagnostic", err)
	}
}

func TestSummarize_RejectsMissingTerminalOutcome(t *testing.T) {
	tests := []struct {
		name     string
		events   string
		wantText string
	}{
		{
			name: "test",
			events: `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestNeverFinished"}
{"Action":"fail","Package":"example/ui","Elapsed":1}
`,
			wantText: "missing terminal outcome for example/ui/TestNeverFinished",
		},
		{
			name: "package",
			events: `
{"Action":"start","Package":"example/unfinished"}
`,
			wantText: "missing terminal outcome for package example/unfinished",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeEventStream(t, test.events)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run([]string{"summarize", "-json", path}, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("got %v, want %q", err, test.wantText)
			}
		})
	}
}

func TestSummarize_RejectsTopLevelTestAbsentFromRequiredRun(t *testing.T) {
	first := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestPresent"}
{"Action":"pass","Package":"example/ui","Test":"TestPresent","Elapsed":1}
{"Action":"run","Package":"example/ui","Test":"TestMissing"}
{"Action":"pass","Package":"example/ui","Test":"TestMissing","Elapsed":2}
{"Action":"pass","Package":"example/ui","Elapsed":3}
`)
	second := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestPresent"}
{"Action":"pass","Package":"example/ui","Test":"TestPresent","Elapsed":1.5}
{"Action":"pass","Package":"example/ui","Elapsed":2}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"summarize", "-json", first, "-json", second}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "run 2") || !strings.Contains(err.Error(), "missing top-level test example/ui/TestMissing") {
		t.Fatalf("got %v, want run 2 missing TestMissing diagnostic", err)
	}
}

func TestSummarize_RejectsMalformedTruncatedAndEmptyInput(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		wantText string
	}{
		{name: "malformed", contents: `{not-json}`, wantText: "parse"},
		{name: "truncated", contents: `{"Action":"start"`, wantText: "unexpected EOF"},
		{name: "empty", contents: "", wantText: "contains no Go test events"},
		{name: "not test events", contents: `{"Time":"2026-09-03T00:00:00Z"}`, wantText: "contains no package start events"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeEventStream(t, test.contents)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run([]string{"summarize", "-json", path}, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("got %v, want diagnostic containing %q", err, test.wantText)
			}
		})
	}
}

func TestSummarize_RejectsTerminalWithoutBeginning(t *testing.T) {
	tests := []struct {
		name     string
		events   string
		wantText string
	}{
		{
			name:     "package",
			events:   `{"Action":"pass","Package":"example/orphan","Elapsed":1}`,
			wantText: "terminal outcome for package example/orphan has no start event",
		},
		{
			name: "test",
			events: `
{"Action":"start","Package":"example/ui"}
{"Action":"pass","Package":"example/ui","Test":"TestOrphan","Elapsed":1}
{"Action":"pass","Package":"example/ui","Elapsed":2}
`,
			wantText: "terminal outcome for example/ui/TestOrphan has no run event",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeEventStream(t, test.events)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run([]string{"summarize", "-json", path}, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("got %v, want %q", err, test.wantText)
			}
		})
	}
}

func TestSummarize_RejectsTerminalWithoutElapsed(t *testing.T) {
	path := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestNoDuration"}
{"Action":"pass","Package":"example/ui","Test":"TestNoDuration"}
{"Action":"fail","Package":"example/ui","Elapsed":1}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"summarize", "-json", path}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "terminal pass for example/ui/TestNoDuration is missing Elapsed") {
		t.Fatalf("got %v, want missing TestNoDuration elapsed diagnostic", err)
	}
}

func TestSummarize_UsesMedianAndPreservesTerminalOutcomes(t *testing.T) {
	paths := []string{
		writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestOutcome"}
{"Action":"pass","Package":"example/ui","Test":"TestOutcome","Elapsed":9}
{"Action":"pass","Package":"example/ui","Elapsed":10}
`),
		writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestOutcome"}
{"Action":"skip","Package":"example/ui","Test":"TestOutcome","Elapsed":1}
{"Action":"skip","Package":"example/ui","Elapsed":2}
`),
		writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestOutcome"}
{"Action":"fail","Package":"example/ui","Test":"TestOutcome","Elapsed":5}
{"Action":"fail","Package":"example/ui","Elapsed":6}
`),
	}
	args := []string{"summarize"}
	for _, path := range paths {
		args = append(args, "-json", path)
	}

	var first bytes.Buffer
	var stderr bytes.Buffer
	if err := run(args, &first, &stderr); err != nil {
		t.Fatalf("first run: %v\nstderr: %s", err, stderr.String())
	}
	var second bytes.Buffer
	stderr.Reset()
	if err := run(args, &second, &stderr); err != nil {
		t.Fatalf("second run: %v\nstderr: %s", err, stderr.String())
	}
	if first.String() != second.String() {
		t.Fatalf("identical inputs produced different output:\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}

	want := strings.TrimSpace(`
summary	runs=3
package	example/ui	run-1=pass:10.000	run-2=skip:2.000	run-3=fail:6.000	median=6.000
test	example/ui	TestOutcome	run-1=pass:9.000	run-2=skip:1.000	run-3=fail:5.000	median=5.000
`) + "\n"
	if first.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", first.String(), want)
	}
}

func TestPlan_UsesMedianLPTAndDeterministicTieBreaks(t *testing.T) {
	paths := []string{
		writePlanningStream(t, map[string]float64{
			"ExampleA": 30, "FuzzB": 3, "TestC": 3,
			"ExampleD": 10, "FuzzE": 1, "TestF": 1,
		}),
		writePlanningStream(t, map[string]float64{
			"ExampleA": 3, "FuzzB": 30, "TestC": 3,
			"ExampleD": 1, "FuzzE": 10, "TestF": 1,
		}),
		writePlanningStream(t, map[string]float64{
			"ExampleA": 3, "FuzzB": 3, "TestC": 30,
			"ExampleD": 1, "FuzzE": 1, "TestF": 10,
		}),
	}
	args := []string{
		"plan",
		"-package", "example/ui",
		"-shards", "3",
		"-baseline-sha", "abc123",
		"-baseline-run", "42",
	}
	for _, path := range paths {
		args = append(args, "-json", path)
	}

	var first bytes.Buffer
	var stderr bytes.Buffer
	if err := run(args, &first, &stderr); err != nil {
		t.Fatalf("first run: %v\nstderr: %s", err, stderr.String())
	}
	var second bytes.Buffer
	stderr.Reset()
	if err := run(args, &second, &stderr); err != nil {
		t.Fatalf("second run: %v\nstderr: %s", err, stderr.String())
	}
	if first.String() != second.String() {
		t.Fatalf("identical inputs produced different plans:\nfirst:\n%s\nsecond:\n%s", first.String(), second.String())
	}

	want := strings.TrimSpace(`
# PicFetch test shard manifest v1
# package: example/ui
# shards: 3
# baseline-sha: abc123
# baseline-run: 42
# baseline-attempts: 1,2,3
# ui-1: 2 entries, 4.000s median-weight sum
# ui-2: 2 entries, 4.000s median-weight sum
# ui-3: 2 entries, 4.000s median-weight sum
# name	shard
ExampleA	ui-1
ExampleD	ui-1
FuzzB	ui-2
FuzzE	ui-2
TestC	ui-3
TestF	ui-3
`) + "\n"
	if first.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", first.String(), want)
	}
}

func TestPlan_ResolvesRelativePackageFromEventInventory(t *testing.T) {
	path := writeEventStream(t, `
{"Action":"start","Package":"example.com/project/internal/ui"}
{"Action":"run","Package":"example.com/project/internal/ui","Test":"TestOnly"}
{"Action":"pass","Package":"example.com/project/internal/ui","Test":"TestOnly","Elapsed":1}
{"Action":"pass","Package":"example.com/project/internal/ui","Elapsed":2}
`)
	args := []string{
		"plan",
		"-package", "./internal/ui",
		"-shards", "1",
		"-baseline-sha", "abc123",
		"-baseline-run", "42",
		"-json", path,
		"-json", path,
		"-json", path,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "# package: example.com/project/internal/ui\n") {
		t.Fatalf("manifest did not contain the resolved import path:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "TestOnly\tui-1\n") {
		t.Fatalf("manifest did not assign TestOnly:\n%s", stdout.String())
	}
}

func TestPlan_RejectsNonPassingBaselineOutcome(t *testing.T) {
	passing := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestNeedsPass"}
{"Action":"pass","Package":"example/ui","Test":"TestNeedsPass","Elapsed":1}
{"Action":"pass","Package":"example/ui","Elapsed":2}
`)
	skipped := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestNeedsPass"}
{"Action":"skip","Package":"example/ui","Test":"TestNeedsPass","Elapsed":0}
{"Action":"pass","Package":"example/ui","Elapsed":1}
`)
	args := []string{
		"plan",
		"-package", "example/ui",
		"-shards", "1",
		"-baseline-sha", "abc123",
		"-baseline-run", "42",
		"-json", passing,
		"-json", skipped,
		"-json", passing,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(args, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "run 2") || !strings.Contains(err.Error(), "TestNeedsPass ended with skip") {
		t.Fatalf("got %v, want run 2 skipped TestNeedsPass diagnostic", err)
	}
}

func TestPlan_EqualDecimalLoadsPreferLowerShard(t *testing.T) {
	path := writeEventStream(t, `
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestA"}
{"Action":"pass","Package":"example/ui","Test":"TestA","Elapsed":0.8}
{"Action":"run","Package":"example/ui","Test":"TestB"}
{"Action":"pass","Package":"example/ui","Test":"TestB","Elapsed":0.7}
{"Action":"run","Package":"example/ui","Test":"TestC"}
{"Action":"pass","Package":"example/ui","Test":"TestC","Elapsed":0.1}
{"Action":"run","Package":"example/ui","Test":"TestD"}
{"Action":"pass","Package":"example/ui","Test":"TestD","Elapsed":0.05}
{"Action":"pass","Package":"example/ui","Elapsed":2}
`)
	args := []string{
		"plan",
		"-package", "example/ui",
		"-shards", "2",
		"-baseline-sha", "abc123",
		"-baseline-run", "42",
		"-json", path,
		"-json", path,
		"-json", path,
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}
	for _, row := range []string{"TestA\tui-1\n", "TestD\tui-1\n", "TestB\tui-2\n", "TestC\tui-2\n"} {
		if !strings.Contains(stdout.String(), row) {
			t.Fatalf("manifest missing equal-load tie assignment %q:\n%s", strings.TrimSpace(row), stdout.String())
		}
	}
}

func TestRegex_ProducesExactGoCompatibleFilter(t *testing.T) {
	manifest := writeManifest(t, `
TestA+B	ui-1
TestDollar$	ui-1
FuzzDot.	ui-2
ExampleBrackets[1]	ui-3
`)
	tests := []struct {
		shard      string
		want       string
		matches    []string
		nonMatches []string
	}{
		{
			shard:      "ui-1",
			want:       `^(TestA\+B|TestDollar\$)$`,
			matches:    []string{"TestA+B", "TestDollar$"},
			nonMatches: []string{"FuzzDot.", "ExampleBrackets[1]", "TestA+B/child", "prefixTestA+B"},
		},
		{
			shard:      "ui-2",
			want:       `^(FuzzDot\.)$`,
			matches:    []string{"FuzzDot."},
			nonMatches: []string{"TestA+B", "FuzzDotX"},
		},
		{
			shard:      "ui-3",
			want:       `^(ExampleBrackets\[1\])$`,
			matches:    []string{"ExampleBrackets[1]"},
			nonMatches: []string{"TestDollar$", "ExampleBrackets1"},
		},
	}

	for _, test := range tests {
		t.Run(test.shard, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run([]string{"regex", "-manifest", manifest, "-shard", test.shard}, &stdout, &stderr)
			if err != nil {
				t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
			}

			got := strings.TrimSpace(stdout.String())
			if got != test.want {
				t.Fatalf("filter = %q, want %q", got, test.want)
			}
			compiled, err := regexp.Compile(got)
			if err != nil {
				t.Fatalf("compile filter %q: %v", got, err)
			}
			for _, name := range test.matches {
				if !compiled.MatchString(name) {
					t.Errorf("filter %q does not match assigned name %q", got, name)
				}
			}
			for _, name := range test.nonMatches {
				if compiled.MatchString(name) {
					t.Errorf("filter %q unexpectedly matches %q", got, name)
				}
			}
		})
	}
}

func TestRegex_ValidatesWholeManifestBeforeSelectingShard(t *testing.T) {
	manifest := writeManifest(t, `
TestOwned	ui-1
TestOther	ui-2
this row is malformed
TestThird	ui-3
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"regex", "-manifest", manifest, "-shard", "ui-1"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "row 9") || !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("got %v, want malformed row 9 diagnostic", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("invalid manifest emitted filter %q", stdout.String())
	}
}

func TestCheck_UsesBuildSelectedTopLevelInventory(t *testing.T) {
	prepareCheckFixture(t, false)
	manifest := writeManifest(t, validCheckRows)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"check", "-package", "./ui", "-manifest", manifest}, &stdout, &stderr)
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		if err != nil {
			t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
		}
		want := "checked\tpackage=example.com/project/ui\trunnables=5\tshards=3\n"
		if stdout.String() != want {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
		return
	}

	if err == nil || !strings.Contains(err.Error(), "canonical shard validation requires linux/amd64") {
		t.Fatalf("got %v, want non-linux/amd64 canonical-validation refusal", err)
	}
	if strings.Contains(err.Error(), "TestMain") || strings.Contains(err.Error(), "TestParent/child") || strings.Contains(err.Error(), "TestIgnored") {
		t.Fatalf("unselected harness or subtest escaped into inventory: %v", err)
	}
}

func TestCheckRejectsInvalidAssignments(t *testing.T) {
	tests := []struct {
		name     string
		rows     string
		wantText []string
	}{
		{
			name: "missing",
			rows: `
TestAlpha	ui-1
TestPlatform	ui-2
FuzzBeta	ui-3
ExampleGamma	ui-3
`,
			wantText: []string{"TestParent", "unassigned"},
		},
		{
			name:     "duplicate",
			rows:     validCheckRows + "\nTestAlpha\tui-2",
			wantText: []string{"TestAlpha", "row", "duplicates"},
		},
		{
			name:     "stale",
			rows:     validCheckRows + "\nTestStale\tui-1",
			wantText: []string{"TestStale", "stale"},
		},
		{
			name:     "malformed",
			rows:     validCheckRows + "\nTestBroken ui-1",
			wantText: []string{"row", "malformed"},
		},
		{
			name: "unknown shard",
			rows: `
TestAlpha	ui-4
TestParent	ui-2
TestPlatform	ui-2
FuzzBeta	ui-3
ExampleGamma	ui-3
`,
			wantText: []string{"TestAlpha", "ui-4", "unknown shard"},
		},
		{
			name: "empty shard",
			rows: `
TestAlpha	ui-1
TestParent	ui-2
TestPlatform	ui-2
FuzzBeta	ui-2
ExampleGamma	ui-2
`,
			wantText: []string{"ui-3", "empty"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prepareCheckFixture(t, false)
			manifest := writeManifest(t, test.rows)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run([]string{"check", "-package", "./ui", "-manifest", manifest}, &stdout, &stderr)
			if err == nil {
				t.Fatal("invalid assignment was accepted")
			}
			for _, want := range test.wantText {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("got %v, want diagnostic containing %q", err, want)
				}
			}
			if stdout.Len() != 0 {
				t.Fatalf("invalid assignment emitted success output %q", stdout.String())
			}
		})
	}
}

func TestCheckRejectsDarwinDerivedInventoryAsCanonical(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin-specific canonical-context guard")
	}
	prepareCheckFixture(t, false)
	manifest := writeManifest(t, validCheckRows)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"check", "-package", "./ui", "-manifest", manifest}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "canonical shard validation requires linux/amd64") || !strings.Contains(err.Error(), "darwin/") {
		t.Fatalf("got %v, want explicit Darwin canonical-validation refusal", err)
	}
}

func TestParallelGuardRejectsSelectedParallelCall(t *testing.T) {
	prepareCheckFixture(t, true)
	manifest := writeManifest(t, validCheckRows)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"check", "-package", "./ui", "-manifest", manifest}, &stdout, &stderr)
	if err == nil {
		t.Fatal("selected parallel test call was accepted")
	}
	for _, want := range []string{"TestAlpha", "common_test.go", ".Parallel()", "explicit safety review"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("got %v, want diagnostic containing %q", err, want)
		}
	}
}

func TestCapture_PreservesRawFailedStreamAndReportsPartitionPackageAndTest(t *testing.T) {
	stream := strings.TrimLeft(`
{"Action":"start","Package":"example/ui"}
{"Action":"run","Package":"example/ui","Test":"TestBroken"}
{"Action":"output","Package":"example/ui","Test":"TestBroken","Output":"main_test.go:12: deliberate failure\n"}
{"Action":"fail","Package":"example/ui","Test":"TestBroken","Elapsed":0.25}
{"Action":"fail","Package":"example/ui","Elapsed":0.5}
`, "\n")
	rawPath := filepath.Join(t.TempDir(), "raw.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithInput(
		[]string{"capture", "-out", rawPath, "-partition", "ui-2"},
		strings.NewReader(stream),
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != stream {
		t.Fatalf("raw stream:\n%s\nwant:\n%s", raw, stream)
	}
	want := strings.TrimSpace(`
failure	partition=ui-2	package=example/ui	test=TestBroken	main_test.go:12: deliberate failure
test	partition=ui-2	package=example/ui	name=TestBroken	action=fail	elapsed=0.250
package	partition=ui-2	package=example/ui	action=fail	elapsed=0.500
`) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestCapture_RejectsCaptureFailure(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithInput(
		[]string{"capture", "-out", t.TempDir(), "-partition", "non-ui"},
		strings.NewReader("{\"Action\":\"start\",\"Package\":\"example/pkg\"}\n"),
		&stdout,
		&stderr,
	)
	if err == nil || !strings.Contains(err.Error(), "create capture") {
		t.Fatalf("got %v, want capture creation failure", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("failed capture emitted output %q", stdout.String())
	}
}

func TestCIFailures_AggregatesEveryFailedTestAcrossJobs(t *testing.T) {
	input := strings.NewReader(strings.TrimSpace(`
Linux race (non-ui, attempt 1)	go test	2026-09-03T16:00:00Z failure	partition=non-ui	package=example/msixstage	test=TestStoreListingAssets	    msixstage_test.go:230: viewer.png is too small
Linux race (non-ui, attempt 1)	Run race tests	2026-09-03T16:00:01Z test	partition=non-ui	package=example/msixstage	name=TestStoreListingAssets	action=fail	elapsed=0.010
Linux race (non-ui, attempt 1)	Run race tests	2026-09-03T16:00:01Z package	partition=non-ui	package=example/msixstage	action=fail	elapsed=0.020
Linux race (ui-2, attempt 1)	Run race tests	2026-09-03T16:00:02Z failure	partition=ui-2	package=example/ui	test=TestViewer	    viewer_test.go:42: images differ
Linux race (ui-2, attempt 1)	Run race tests	2026-09-03T16:00:03Z failure	partition=ui-2	package=example/ui	test=TestViewer	    expected golden master
Linux race (ui-2, attempt 1)	Run race tests	2026-09-03T16:00:04Z test	partition=ui-2	package=example/ui	name=TestViewer	action=fail	elapsed=1.250
Linux race (ui-3, attempt 1)	Run race tests	2026-09-03T16:00:05Z test	partition=ui-3	package=example/ui	name=TestPassing	action=pass	elapsed=0.100
`) + "\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithInput([]string{"ci-failures"}, input, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	want := strings.TrimSpace(`
CI failure details (2 failed tests)

[non-ui] example/msixstage — TestStoreListingAssets
  msixstage_test.go:230: viewer.png is too small

[ui-2] example/ui — TestViewer
  viewer_test.go:42: images differ
  expected golden master
`) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestCIFailures_ReportsPackageAndSetupFailuresWithoutTestEvents(t *testing.T) {
	input := strings.NewReader(strings.TrimSpace(`
Linux race (non-ui, attempt 1)	Run race tests	2026-09-03T16:00:00Z failure	partition=non-ui	package=example/broken	./broken.go:12: undefined: missing
Linux race (non-ui, attempt 1)	Run race tests	2026-09-03T16:00:01Z package	partition=non-ui	package=example/broken	action=fail	elapsed=0.000
Windows build	Install tools	2026-09-03T16:00:02Z ##[error]Process completed with exit code 1.
`) + "\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithInput([]string{"ci-failures"}, input, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}

	want := strings.TrimSpace(`
CI failure details (1 failed package, 1 job error)

[non-ui] example/broken
  ./broken.go:12: undefined: missing

[Windows build] job error
  Process completed with exit code 1.
`) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestMakeCIFailuresFindsLatestCompletedRunOrAcceptsRunID(t *testing.T) {
	output := makeDryRun(t, "ci-failures")
	for _, want := range []string{
		`gh run list --workflow "CI"`,
		`--branch "$branch" --status completed --limit 1`,
		`gh run view "$run" --log-failed`,
		`go run ./scripts/testshards ci-failures`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("make ci-failures output is missing %q:\n%s", want, output)
		}
	}

	explicit := makeDryRun(t, "ci-failures", "CI_RUN=33800732837")
	if !strings.Contains(explicit, `run="33800732837"`) {
		t.Fatalf("make ci-failures does not accept an explicit run ID:\n%s", explicit)
	}
	command := exec.Command("make", "--no-print-directory", "help")
	command.Dir = filepath.Join("..", "..")
	help, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make help: %v\n%s", err, help)
	}
	if !strings.Contains(string(help), "ci-failures") {
		t.Fatalf("make help does not list ci-failures:\n%s", help)
	}
}

func TestPackagePartition_ExcludesOnlyExactMainUIPackage(t *testing.T) {
	root := preparePartitionFixture(t, true)
	t.Chdir(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithInput([]string{"partition", "-package", "./internal/ui"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}
	want := strings.TrimSpace(`
example.com/project
example.com/project/internal/ui/grid
example.com/project/scripts/tool
`) + "\n"
	if stdout.String() != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestPackagePartition_RejectsEmptyOrInconsistentPartition(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		root := preparePartitionFixture(t, false)
		t.Chdir(root)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		err := runWithInput([]string{"partition", "-package", "./internal/ui"}, strings.NewReader(""), &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "non-UI package partition is empty") {
			t.Fatalf("got %v, want empty partition diagnostic", err)
		}
	})

	t.Run("selected package absent from module inventory", func(t *testing.T) {
		root := preparePartitionFixture(t, true)
		t.Chdir(root)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		err := runWithInput([]string{"partition", "-package", "fmt"}, strings.NewReader(""), &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "selected package fmt is absent from the module inventory") {
			t.Fatalf("got %v, want inconsistent partition diagnostic", err)
		}
	})
}

func TestMakeTestRemainsCompleteAndUnsharded(t *testing.T) {
	output := makeDryRun(t, "test")
	for _, want := range []string{"docker run --rm --platform linux/amd64", "go test -timeout 30m", "./..."} {
		if !strings.Contains(output, want) {
			t.Fatalf("make test output is missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"testshards capture", "testshards partition", "-count=1", "-json"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("make test unexpectedly contains %q:\n%s", forbidden, output)
		}
	}
}

func TestMakeCoverageRunsCompleteUnshardedSuiteAndBuildsHTML(t *testing.T) {
	output := makeDryRun(t, "coverage")
	for _, want := range []string{
		"docker run --rm --platform linux/amd64",
		"apt-get install -y -qq make",
		"locale-gen en_US.UTF-8",
		"go test -timeout 30m -coverprofile=\"coverage/coverage.out\" ./...",
		"go tool cover -html=\"coverage/coverage.out\" -o \"coverage/coverage.html\"",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("make coverage output is missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"testshards", "-race", "-run"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("make coverage unexpectedly contains %q:\n%s", forbidden, output)
		}
	}

	command := exec.Command("make", "--no-print-directory", "help")
	command.Dir = filepath.Join("..", "..")
	help, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make help: %v\n%s", err, help)
	}
	if !strings.Contains(string(help), "coverage") || !strings.Contains(string(help), "full unsharded Docker suite") {
		t.Fatalf("make help does not document the coverage contract:\n%s", help)
	}

	makefile, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^\.PHONY:.*\bcoverage\b`).Match(makefile) {
		t.Fatal("coverage target is not phony, so an existing coverage directory would suppress it")
	}

	ignored, err := os.ReadFile(filepath.Join("..", "..", ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ignored), "coverage/\n") {
		t.Fatalf("coverage artifacts are not ignored:\n%s", ignored)
	}
}

func TestMakeRaceRunsCanonicalConcurrentContractInOneContainer(t *testing.T) {
	public := makeDryRun(t, "test-race")
	if count := strings.Count(public, "docker run --rm --platform linux/amd64"); count != 1 {
		t.Fatalf("make test-race starts %d Linux/amd64 containers, want 1:\n%s", count, public)
	}
	for _, want := range []string{"locale-gen en_US.UTF-8", "make --no-print-directory test-race-direct"} {
		if !strings.Contains(public, want) {
			t.Fatalf("make test-race output is missing %q:\n%s", want, public)
		}
	}

	direct := makeDryRun(t, "test-race-direct")
	guard := strings.Index(direct, "testshards check")
	firstPartition := strings.Index(direct, "testshards partition")
	if guard < 0 || firstPartition < 0 || guard >= firstPartition {
		t.Fatalf("race manifest guard must precede partition launch:\n%s", direct)
	}
	for _, want := range []string{
		"test-race-non-ui-direct &",
		"test-race-ui-direct TEST_SHARD=ui-1 &",
		"test-race-ui-direct TEST_SHARD=ui-2 &",
		"test-race-ui-direct TEST_SHARD=ui-3 &",
		"wait \"$pid\"",
	} {
		if !strings.Contains(direct, want) {
			t.Fatalf("concurrent race contract is missing %q:\n%s", want, direct)
		}
	}
}

func TestMakeRaceConcurrentContractWaitsForEveryPartitionAndPropagatesFailure(t *testing.T) {
	for _, failPartition := range []string{"", "ui-2"} {
		name := "success"
		if failPartition != "" {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			result := runRaceContractFixture(t, failPartition)
			if failPartition == "" && result.err != nil {
				t.Fatalf("race contract failed: %v\n%s", result.err, result.output)
			}
			if failPartition != "" && result.err == nil {
				t.Fatalf("race contract ignored %s failure:\n%s", failPartition, result.output)
			}
			for _, partition := range []string{"non-ui", "ui-1", "ui-2", "ui-3"} {
				if !strings.Contains(result.events, "done:"+partition+"\n") {
					t.Fatalf("race contract returned before %s finished:\n%s", partition, result.events)
				}
			}
		})
	}
}

func TestMakeDirectRacePartitionsShareFlagsLocaleAndCapture(t *testing.T) {
	t.Setenv("TEST_CAPTURE", "/tmp/external-capture.json")

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "non-ui",
			args: []string{"test-race-non-ui-direct"},
			want: []string{
				"testshards partition -package \"./internal/ui\"",
				"capture -out \"/tmp/picfetch-test-non-ui.json\"",
				"-partition \"non-ui\"",
			},
		},
		{
			name: "selected UI shard",
			args: []string{"test-race-ui-direct", "TEST_SHARD=ui-2"},
			want: []string{
				"testshards regex -manifest \".github/testshards/internal-ui.tsv\" -shard \"ui-2\"",
				"-run \"$filter\" ./internal/ui",
				"capture -out \"/tmp/picfetch-test-ui-2.json\"",
				"-partition \"ui-2\"",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := makeDryRun(t, test.args...)
			shared := []string{
				"set -eu -o pipefail",
				"LANG=\"en_US.UTF-8\" go test -race -count=1 -timeout 30m -json",
				"testshards capture -out",
			}
			for _, want := range append(shared, test.want...) {
				if !strings.Contains(output, want) {
					t.Fatalf("direct %s output is missing %q:\n%s", test.name, want, output)
				}
			}
		})
	}
}

func TestMakeDirectUIRaceRejectsInvalidShardAndPrintsTheContract(t *testing.T) {
	command := exec.Command(
		"make",
		"--no-print-directory",
		"test-race-ui-direct",
		"TEST_SHARD=ui-4",
	)
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("invalid UI shard was accepted")
	}
	for _, want := range []string{
		"TEST_SHARD must be one of ui-1, ui-2, or ui-3",
		"LANG=\"en_US.UTF-8\" go test -race -count=1 -timeout 30m -json",
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("invalid-shard output is missing %q:\n%s", want, output)
		}
	}
}

type raceContractResult struct {
	events string
	output string
	err    error
}

func runRaceContractFixture(t *testing.T, failPartition string) raceContractResult {
	t.Helper()
	dir := t.TempDir()
	runner := filepath.Join(dir, "make-runner")
	writeTestFile(t, runner, `#!/bin/sh
set -eu
partition=
for arg in "$@"; do
	case "$arg" in
		check-test-shards-direct) partition=check ;;
		test-race-non-ui-direct) partition=non-ui ;;
		TEST_SHARD=*) partition=${arg#TEST_SHARD=} ;;
	esac
done
if [ "$partition" = check ]; then
	printf 'check\n' >> "$RACE_TEST_DIR/events"
	exit 0
fi
printf 'start:%s\n' "$partition" >> "$RACE_TEST_DIR/events"
: > "$RACE_TEST_DIR/$partition.started"
while [ ! -f "$RACE_TEST_DIR/release" ]; do sleep 0.01; done
printf 'done:%s\n' "$partition" >> "$RACE_TEST_DIR/events"
if [ "$partition" = "$RACE_FAIL_PARTITION" ]; then exit 7; fi
`)
	if err := os.Chmod(runner, 0o755); err != nil {
		t.Fatal(err)
	}

	command := exec.Command(
		"make",
		"--no-print-directory",
		"test-race-direct",
		"MAKE="+runner,
	)
	command.Dir = filepath.Join("..", "..")
	command.Env = append(os.Environ(), "RACE_TEST_DIR="+dir, "RACE_FAIL_PARTITION="+failPartition)
	var commandOutput bytes.Buffer
	command.Stdout = &commandOutput
	command.Stderr = &commandOutput
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = command.Wait()
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	allStarted := false
	for time.Now().Before(deadline) {
		allStarted = true
		for _, partition := range []string{"non-ui", "ui-1", "ui-2", "ui-3"} {
			if _, err := os.Stat(filepath.Join(dir, partition+".started")); err != nil {
				allStarted = false
				break
			}
		}
		if allStarted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !allStarted {
		writeTestFile(t, filepath.Join(dir, "release"), "")
		<-done
		events, _ := os.ReadFile(filepath.Join(dir, "events"))
		t.Fatalf("partitions were not launched concurrently:\n%s", events)
	}
	select {
	case <-done:
		t.Fatal("race contract returned without waiting for blocked partitions")
	default:
	}

	writeTestFile(t, filepath.Join(dir, "release"), "")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = command.Process.Kill()
		<-done
		t.Fatal("race contract did not return after every partition finished")
	}
	events, err := os.ReadFile(filepath.Join(dir, "events"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(events), "check\n") {
		t.Fatalf("partition launched before the manifest guard:\n%s", events)
	}
	return raceContractResult{events: string(events), output: commandOutput.String(), err: runErr}
}

func writeEventStream(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "events.json")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeManifest(t *testing.T, rows string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.tsv")
	contents := `# PicFetch test shard manifest v1
# package: example.com/project/ui
# shards: 3
# baseline-sha: abc123
# baseline-run: 42
# name	shard
` + strings.TrimSpace(rows) + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validCheckRows = `
TestAlpha	ui-1
TestParent	ui-2
TestPlatform	ui-2
FuzzBeta	ui-3
ExampleGamma	ui-3
`

func prepareCheckFixture(t *testing.T, parallel bool) {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.27.1\n")
	writeTestFile(t, filepath.Join(root, "ui", "ui.go"), "package ui\n\nfunc Gamma() {}\n")

	parallelCall := ""
	if parallel {
		parallelCall = "\tt.Parallel()\n"
	}
	commonTests := fmt.Sprintf(`package ui

import (
	"fmt"
	"os"
	"testing"
)

func TestAlpha(t *testing.T) {
%s}

func TestParent(t *testing.T) {
	t.Run("child", func(t *testing.T) {})
}

func FuzzBeta(f *testing.F) {
	f.Add("seed")
	f.Fuzz(func(t *testing.T, input string) {})
}

func ExampleGamma() {
	fmt.Println("gamma")
	// Output: gamma
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
`, parallelCall)
	writeTestFile(t, filepath.Join(root, "ui", "common_test.go"), commonTests)
	platformTests := fmt.Sprintf("//go:build %s\n\npackage ui\n\nimport \"testing\"\n\nfunc TestPlatform(t *testing.T) {}\n", runtime.GOOS)
	writeTestFile(t, filepath.Join(root, "ui", "platform_"+runtime.GOOS+"_test.go"), platformTests)
	writeTestFile(t, filepath.Join(root, "ui", "ignored_test.go"), `//go:build picfetch_unselected_fixture

package ui

import "testing"

func TestIgnored(t *testing.T) {
	t.Parallel()
}
`)
	t.Setenv("GOOS", runtime.GOOS)
	t.Setenv("GOARCH", runtime.GOARCH)
	t.Setenv("GOWORK", "off")
	t.Chdir(root)
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func preparePartitionFixture(t *testing.T, includeNonUI bool) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/project\n\ngo 1.27.1\n")
	writeTestFile(t, filepath.Join(root, "internal", "ui", "ui.go"), "package ui\n")
	if includeNonUI {
		writeTestFile(t, filepath.Join(root, "root.go"), "package project\n")
		writeTestFile(t, filepath.Join(root, "internal", "ui", "grid", "grid.go"), "package grid\n")
		writeTestFile(t, filepath.Join(root, "scripts", "tool", "main.go"), "package main\n")
	}
	t.Setenv("GOWORK", "off")
	return root
}

func makeDryRun(t *testing.T, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"--no-print-directory", "-n"}, args...)
	command := exec.Command("make", commandArgs...)
	command.Dir = filepath.Join("..", "..")
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TEST_CAPTURE=") {
			command.Env = append(command.Env, entry)
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func writePlanningStream(t *testing.T, durations map[string]float64) string {
	t.Helper()
	var stream strings.Builder
	stream.WriteString("{\"Action\":\"start\",\"Package\":\"example/ui\"}\n")
	order := []string{"TestF", "ExampleA", "TestC", "FuzzE", "ExampleD", "FuzzB"}
	for _, name := range order {
		_, _ = fmt.Fprintf(&stream, "{\"Action\":\"run\",\"Package\":\"example/ui\",\"Test\":%q}\n", name)
		if name == "TestC" {
			stream.WriteString("{\"Action\":\"run\",\"Package\":\"example/ui\",\"Test\":\"TestC/child\"}\n")
			stream.WriteString("{\"Action\":\"pass\",\"Package\":\"example/ui\",\"Test\":\"TestC/child\",\"Elapsed\":0.25}\n")
		}
		_, _ = fmt.Fprintf(&stream, "{\"Action\":\"pass\",\"Package\":\"example/ui\",\"Test\":%q,\"Elapsed\":%g}\n", name, durations[name])
	}
	stream.WriteString("{\"Action\":\"pass\",\"Package\":\"example/ui\",\"Elapsed\":40}\n")
	return writeEventStream(t, stream.String())
}
