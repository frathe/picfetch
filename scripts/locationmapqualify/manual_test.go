package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/locationtrial"
)

func TestManualCLIRequiresBinaryNewEvidenceAndPositiveDeadline(t *testing.T) {
	var output bytes.Buffer
	options, err := parseManualRun([]string{"-binary", "app", "-evidence", "new-results", "-timeout", "2m"}, &output)
	if err != nil || options.binary != "app" || options.evidence != "new-results" || options.timeout != 2*time.Minute {
		t.Fatalf("manual options: %+v, %v", options, err)
	}
	for _, args := range [][]string{
		nil,
		{"-binary", "app"},
		{"-binary", "app", "-evidence", "out", "-timeout", "0"},
		{"-binary", "app", "-evidence", "out", "-timeout", "-1s"},
		{"-binary", "app", "-evidence", "out", "extra"},
	} {
		if _, err := parseManualRun(args, &output); err == nil {
			t.Fatalf("invalid manual options accepted: %v", args)
		}
	}
}

func TestManualCLIRecognizesMode(t *testing.T) {
	var output bytes.Buffer
	err := runCLI([]string{"manual"}, &output)
	if err == nil || !strings.Contains(err.Error(), "binary, new evidence directory") {
		t.Fatalf("manual mode was not parsed: %v", err)
	}
}

func TestManualControlledChild(t *testing.T) {
	if os.Getenv("PICFETCH_MANUAL_CHILD") == "" {
		return
	}
	path := os.Getenv("PICFETCH_MANUAL_STATE")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for sequence := 1; sequence <= 2; sequence++ {
		state := locationtrial.State{Sequence: sequence, Images: 7, Formats: map[string]int{"jpg": 7}, Ready: true, Active: sequence == 2, Visible: sequence == 2}
		data, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		next := path + ".next"
		if err := os.WriteFile(next, data, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(next, path); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("PICFETCH_MANUAL_WAIT") == "1" {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		defer signal.Stop(interrupt)
		<-interrupt
	}
}

func manualFixture(t *testing.T) (manualRunOptions, string) {
	t.Helper()
	root := t.TempDir()
	binary := filepath.Join(root, "app")
	if err := os.WriteFile(binary, []byte("controlled test binary identity"), 0o700); err != nil {
		t.Fatal(err)
	}
	return manualRunOptions{binary: binary, evidence: filepath.Join(root, "evidence"), timeout: time.Minute}, binary
}

func controlledManualProcess(t *testing.T, options manualRunOptions, wait bool) func(*exec.Cmd) (*nativeProcess, error) {
	t.Helper()
	return func(command *exec.Cmd) (*nativeProcess, error) {
		want := []string{options.binary, "--location-map-trial", filepath.Join(options.evidence, "native"), "--max-files=100000"}
		if !reflect.DeepEqual(command.Args, want) {
			t.Errorf("manual launch argv = %q, want %q", command.Args, want)
		}
		child := exec.Command(os.Args[0], "-test.run=^TestManualControlledChild$")
		child.Env = append(os.Environ(), "PICFETCH_MANUAL_CHILD=1", "PICFETCH_MANUAL_STATE="+filepath.Join(options.evidence, "native", "state.json"))
		if wait {
			child.Env = append(child.Env, "PICFETCH_MANUAL_WAIT=1")
		}
		child.Stdout, child.Stderr = command.Stdout, command.Stderr
		return startNativeProcess(child)
	}
}

func controlledManualMemory(_ context.Context, _ int) *memoryObservation {
	done := make(chan struct{})
	close(done)
	return &memoryObservation{samples: []MemorySample{{AtNS: time.Now().UnixNano(), RSSBytes: 12 * 1024 * 1024}}, stop: func() {}, done: done}
}

func readManualReport(t *testing.T, dir string) manualReport {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "manual-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report manualReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func TestManualUserExitRetainsStateRSSAndCannotPassFormalCheck(t *testing.T) {
	options, binary := manualFixture(t)
	var output bytes.Buffer
	if err := runManual(context.Background(), options, &output, controlledManualProcess(t, options, false), controlledManualMemory); err != nil {
		t.Fatal(err)
	}
	report := readManualReport(t, options.evidence)
	build, err := binaryIdentity(binary)
	if err != nil {
		t.Fatal(err)
	}
	if report.BuildID != build || report.PID <= 0 || report.StartedNS <= 0 || report.EndedNS <= report.StartedNS || report.Exit != "exited" {
		t.Fatalf("invalid manual identity/lifetime: %+v", report)
	}
	if report.FinalState == nil || report.FinalState.Sequence != 2 || report.FinalState.Images != 7 || report.StateObservations < 1 || len(report.Memory) == 0 || report.Memory[0].RSSBytes <= 0 {
		t.Fatalf("missing state/RSS: %+v", report)
	}
	observations, err := os.ReadFile(filepath.Join(options.evidence, "observations.jsonl"))
	if err != nil || !bytes.Contains(observations, []byte(`"kind":"state"`)) || !bytes.Contains(observations, []byte(`"kind":"rss"`)) {
		t.Fatalf("missing live observations: %s, %v", observations, err)
	}
	if _, err := os.Stat(filepath.Join(options.evidence, "native-console.log")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(options.evidence, "report.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manual run created formal report: %v", err)
	}
	manualJSON, err := os.ReadFile(filepath.Join(options.evidence, "manual-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, unsupported := range []string{`"verdict"`, `"latency"`, `"gestures"`} {
		if bytes.Contains(manualJSON, []byte(unsupported)) {
			t.Fatalf("manual report makes unsupported claim %s", unsupported)
		}
	}
	if _, err := CheckEvidence(options.evidence, 7, build); err == nil {
		t.Fatal("formal checker accepted manual evidence")
	}
	if !strings.Contains(output.String(), options.evidence) {
		t.Fatalf("manual output omits evidence location: %q", output.String())
	}
}

type cancelOnStart struct {
	bytes.Buffer
	cancel context.CancelFunc
}

func (w *cancelOnStart) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	w.cancel()
	return n, err
}

func TestManualCancellationStopsChildAndPreservesEvidence(t *testing.T) {
	options, _ := manualFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := &cancelOnStart{cancel: cancel}
	err := runManual(ctx, options, output, controlledManualProcess(t, options, true), controlledManualMemory)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("manual cancellation = %v", err)
	}
	report := readManualReport(t, options.evidence)
	if report.Exit != "interrupted" || report.PID <= 0 || report.Failure == "" || report.EndedNS <= report.StartedNS {
		t.Fatalf("interrupted run not retained: %+v", report)
	}
}

func TestManualDeadlineStopsChildAndPreservesEvidence(t *testing.T) {
	options, _ := manualFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := runManual(ctx, options, io.Discard, controlledManualProcess(t, options, true), controlledManualMemory)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("manual deadline = %v", err)
	}
	report := readManualReport(t, options.evidence)
	if report.Exit != "deadline" || report.Failure == "" {
		t.Fatalf("deadline run not retained: %+v", report)
	}
}

func TestManualLaunchFailureAndExistingEvidence(t *testing.T) {
	options, _ := manualFixture(t)
	launchErr := errors.New("controlled launch failure")
	start := func(_ *exec.Cmd) (*nativeProcess, error) { return nil, launchErr }
	if err := runManual(context.Background(), options, io.Discard, start, controlledManualMemory); !errors.Is(err, launchErr) {
		t.Fatalf("launch failure = %v", err)
	}
	report := readManualReport(t, options.evidence)
	if report.Exit != "launch-failed" || !strings.Contains(report.Failure, launchErr.Error()) {
		t.Fatalf("launch failure not retained: %+v", report)
	}
	called := false
	start = func(_ *exec.Cmd) (*nativeProcess, error) {
		called = true
		return nil, nil
	}
	if err := runManual(context.Background(), options, io.Discard, start, controlledManualMemory); err == nil || called {
		t.Fatalf("existing evidence was reused or launch attempted: error %v, called %t", err, called)
	}
	if _, err := os.Stat(filepath.Join(options.evidence, "report.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed manual run created formal report: %v", err)
	}
}
