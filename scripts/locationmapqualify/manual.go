package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"

	"github.com/frathe/picfetch/internal/locationtrial"
)

type manualRunOptions struct {
	binary, evidence string
	timeout          time.Duration
}

func parseManualRun(args []string, output io.Writer) (manualRunOptions, error) {
	var options manualRunOptions
	flags := flag.NewFlagSet("manual", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&options.binary, "binary", "", "native application binary")
	flags.StringVar(&options.evidence, "evidence", "", "new private evidence directory")
	flags.DurationVar(&options.timeout, "timeout", 30*time.Minute, "finite whole-run deadline")
	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() != 0 || options.binary == "" || options.evidence == "" || options.timeout <= 0 {
		return options, errors.New("binary, new evidence directory and positive timeout are required")
	}
	return options, nil
}

// manualReport describes observations only. It carries no approval or latency verdict.
type manualReport struct {
	Schema            int                  `json:"schema"`
	Observation       string               `json:"observation"`
	BuildID           string               `json:"build_id,omitempty"`
	PID               int                  `json:"pid,omitempty"`
	StartedNS         int64                `json:"started_ns"`
	EndedNS           int64                `json:"ended_ns"`
	Exit              string               `json:"exit"`
	Failure           string               `json:"failure,omitempty"`
	FinalState        *locationtrial.State `json:"final_state,omitempty"`
	StateObservations int                  `json:"state_observations"`
	Memory            []MemorySample       `json:"memory"`
	Errors            []string             `json:"errors,omitempty"`
}

type manualObservation struct {
	Kind     string               `json:"kind"`
	AtNS     int64                `json:"at_ns"`
	State    *locationtrial.State `json:"state,omitempty"`
	RSSBytes int64                `json:"rss_bytes,omitempty"`
}

func manualState(path string) (*locationtrial.State, error) {
	data, err := boundedFile(path, maxReportBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state locationtrial.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func runManual(ctx context.Context, options manualRunOptions, output io.Writer, start func(*exec.Cmd) (*nativeProcess, error), sampleMemory func(context.Context, int) *memoryObservation) (resultErr error) {
	binary, err := filepath.Abs(options.binary)
	if err != nil {
		return err
	}
	evidence, err := filepath.Abs(options.evidence)
	if err != nil {
		return err
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		return fmt.Errorf("evidence directory must be new: %w", err)
	}
	report := manualReport{Schema: 1, Observation: "manual-state-and-rss", StartedNS: time.Now().UnixNano()}
	defer func() {
		report.EndedNS = time.Now().UnixNano()
		if resultErr != nil {
			report.Failure = resultErr.Error()
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(evidence, "manual-report.json"), append(data, '\n'), 0o600)
		}
		resultErr = errors.Join(resultErr, err)
	}()
	report.BuildID, err = binaryIdentity(binary)
	if err != nil {
		report.Exit = "preflight-failed"
		return fmt.Errorf("build identity: %w", err)
	}
	console, err := os.OpenFile(filepath.Join(evidence, "native-console.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		report.Exit = "preflight-failed"
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, console.Close()) }()
	observations, err := os.OpenFile(filepath.Join(evidence, "observations.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		report.Exit = "preflight-failed"
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, observations.Close()) }()
	trialDir := filepath.Join(evidence, "native")
	command := exec.Command(binary, "--location-map-trial", trialDir, "--max-files=100000")
	command.Stdout, command.Stderr = console, console
	app, err := start(command)
	if err != nil {
		report.Exit = "launch-failed"
		return fmt.Errorf("launch native application: %w", err)
	}
	report.PID = app.command.Process.Pid
	if _, err := fmt.Fprintf(output, "Manual Location Map trial running (PID %d). Load images and use the app; observations are in %s. Exit the app to finish.\n", report.PID, evidence); err != nil {
		report.Exit = "observer-failed"
		return errors.Join(err, app.stop())
	}
	memory := sampleMemory(context.Background(), report.PID)
	encoder := json.NewEncoder(observations)
	statePath := filepath.Join(trialDir, "state.json")
	memoryIndex := 0
	collect := func() error {
		state, err := manualState(statePath)
		if err != nil {
			return fmt.Errorf("trial state: %w", err)
		}
		if state != nil && !reflect.DeepEqual(report.FinalState, state) {
			if err := encoder.Encode(manualObservation{Kind: "state", AtNS: time.Now().UnixNano(), State: state}); err != nil {
				return err
			}
			report.FinalState = state
			report.StateObservations++
		}
		memory.mu.Lock()
		samples := append([]MemorySample(nil), memory.samples[memoryIndex:]...)
		memory.mu.Unlock()
		for _, sample := range samples {
			if err := encoder.Encode(manualObservation{Kind: "rss", AtNS: sample.AtNS, RSSBytes: sample.RSSBytes}); err != nil {
				return err
			}
			memoryIndex++
		}
		return nil
	}
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := collect(); err != nil {
			report.Exit = "observer-failed"
			resultErr = errors.Join(err, app.stop())
			break
		}
		select {
		case <-app.done:
			report.Exit = "exited"
			resultErr = app.err
			goto finished
		case <-ctx.Done():
			report.Exit = "interrupted"
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				report.Exit = "deadline"
			}
			resultErr = errors.Join(ctx.Err(), app.stop())
			goto finished
		case <-tick.C:
		}
	}
finished:
	report.Memory, err = memory.finish()
	if err != nil {
		report.Errors = append(report.Errors, "RSS sampling: "+err.Error())
	}
	resultErr = errors.Join(resultErr, collect())
	if resultErr == nil {
		_, resultErr = fmt.Fprintf(output, "Manual trial ended; private observations saved in %s. No visual or latency verdict was generated.\n", evidence)
	}
	return resultErr
}
