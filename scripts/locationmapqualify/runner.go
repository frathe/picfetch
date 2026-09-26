package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frathe/picfetch/internal/locationtrial"
)

type nativeProcess struct {
	command *exec.Cmd
	done    chan struct{}
	err     error
}

func startNativeProcess(command *exec.Cmd) (*nativeProcess, error) {
	if err := command.Start(); err != nil {
		return nil, err
	}
	process := &nativeProcess{command: command, done: make(chan struct{})}
	go func() { process.err = command.Wait(); close(process.done) }()
	return process, nil
}

func (p *nativeProcess) stop() error {
	select {
	case <-p.done:
		return p.err
	default:
	}
	_ = p.command.Process.Signal(os.Interrupt)
	select {
	case <-p.done:
		return p.err
	case <-time.After(15 * time.Second):
		_ = p.command.Process.Kill()
		<-p.done
		return errors.New("native process required forced termination")
	}
}

type processDriver struct {
	app          *nativeProcess
	statePath    string
	input        *json.Encoder
	output       *json.Decoder
	observations *json.Encoder
}

func (d *processDriver) State(ctx context.Context) (locationtrial.State, error) {
	var state locationtrial.State
	if err := ctx.Err(); err != nil {
		return state, err
	}
	select {
	case <-d.app.done:
		return state, fmt.Errorf("application ended before native protocol completed: %v", d.app.err)
	default:
	}
	data, err := boundedFile(d.statePath, maxReportBytes)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	return state, json.Unmarshal(data, &state)
}

func (d *processDriver) Input(ctx context.Context, command nativeCommand) (nativeObservation, error) {
	var observation nativeObservation
	if err := ctx.Err(); err != nil {
		return observation, err
	}
	if err := d.input.Encode(command); err != nil {
		return observation, err
	}
	if err := d.output.Decode(&observation); err != nil {
		return observation, err
	}
	if err := d.observations.Encode(observation); err != nil {
		return observation, err
	}
	return observation, nil
}

type memoryObservation struct {
	mu            sync.Mutex
	samples       []MemorySample
	browsingStart time.Time
	err           error
	stop          context.CancelFunc
	done          chan struct{}
}

func observeMemory(parent context.Context, pid int) *memoryObservation {
	ctx, cancel := context.WithCancel(parent)
	memory := &memoryObservation{stop: cancel, done: make(chan struct{})}
	go func() {
		defer close(memory.done)
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		for {
			at := time.Now().UnixNano()
			data, err := exec.CommandContext(ctx, "/bin/ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
			if ctx.Err() != nil {
				return
			}
			var rss int64
			if err == nil {
				rss, err = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			}
			if rss <= 0 && err == nil {
				err = errors.New("nonpositive process RSS")
			}
			memory.mu.Lock()
			if err == nil {
				memory.samples = append(memory.samples, MemorySample{AtNS: at, RSSBytes: rss * 1024})
			} else if memory.err == nil {
				memory.err = err
			}
			memory.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
	return memory
}

func (m *memoryObservation) needsSustainedBrowsing() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.browsingStart.IsZero() {
		m.browsingStart = time.Now()
	}
	// This starts after the first forty gestures, never during cold startup or
	// scanning. A slow initial scan cannot satisfy sustained browsing evidence.
	return time.Since(m.browsingStart) < time.Minute
}

func (m *memoryObservation) finish() ([]MemorySample, error) {
	m.stop()
	<-m.done
	return m.samples, m.err
}

func nativeDescription(ctx context.Context, command string, args ...string) (string, error) {
	data, err := exec.CommandContext(ctx, command, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func runNative(ctx context.Context, images, evidence, binary, helper string, output io.Writer) (resultErr error) {
	if runtime.GOOS != "darwin" {
		return errors.New("native Location Map collection currently requires macOS")
	}
	var err error
	images, err = filepath.Abs(images)
	if err != nil {
		return err
	}
	evidence, err = filepath.Abs(evidence)
	if err != nil {
		return err
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		return err
	}
	helper, err = filepath.Abs(helper)
	if err != nil {
		return err
	}
	if err := os.Mkdir(evidence, 0o700); err != nil {
		return fmt.Errorf("evidence directory must be new: %w", err)
	}
	report := Report{Schema: 1, Native: true, Observation: "macos-screen-capture"}
	defer func() {
		report.Complete = resultErr == nil
		if resultErr != nil {
			report.Failure = resultErr.Error()
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(evidence, "report.json"), append(data, '\n'), 0o600)
		}
		resultErr = errors.Join(resultErr, err)
	}()
	info, err := os.Stat(images)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("explicit image directory is unavailable: %s", images)
	}
	report.BuildID, err = binaryIdentity(binary)
	if err != nil {
		return err
	}
	helperID, err := binaryIdentity(helper)
	if err != nil {
		return err
	}
	report.Protocol = "screen-v2: foreground 1200x800 window; 120fps capture requested, actual WindowServer frame cadence; pan/zoom requires identified visual transform (current helper refuses unsupported samples); Shift+arrow pan; 250ms stable-frame admission; all submitted samples retained; cancellation requires observed closed viewer; Mach input/display clock; RSS sampled every 250ms (reported peak is sampled); capture/PNG work runs outside PicFetch; helper_sha256=" + helperID
	hardware, err := nativeDescription(ctx, "/usr/sbin/sysctl", "-n", "hw.model", "hw.memsize", "hw.ncpu")
	if err != nil {
		return err
	}
	report.Hardware = runtime.GOOS + "/" + runtime.GOARCH + "; model, RAM bytes, CPUs: " + strings.ReplaceAll(hardware, "\n", ", ")
	report.Storage, err = nativeDescription(ctx, "/bin/df", "-P", images)
	if err != nil {
		return err
	}
	console, err := os.OpenFile(filepath.Join(evidence, "native-console.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, console.Close()) }()
	trialDir := filepath.Join(evidence, "native")
	command := exec.CommandContext(ctx, binary, "--location-map-trial", trialDir, "--max-files=100000", images)
	command.Stdout, command.Stderr = console, console
	app, err := startNativeProcess(command)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, app.stop()) }()
	memory := observeMemory(ctx, app.command.Process.Pid)
	defer func() { var err error; report.Memory, err = memory.finish(); resultErr = errors.Join(resultErr, err) }()
	capture := exec.CommandContext(ctx, helper, strconv.Itoa(app.command.Process.Pid), evidence)
	capture.Stderr = console
	stdin, err := capture.StdinPipe()
	if err != nil {
		return err
	}
	defer func() { _ = stdin.Close() }()
	stdout, err := capture.StdoutPipe()
	if err != nil {
		return err
	}
	observer, err := startNativeProcess(capture)
	if err != nil {
		return err
	}
	defer func() {
		_ = stdin.Close()
		select {
		case <-observer.done:
			resultErr = errors.Join(resultErr, observer.err)
		case <-time.After(5 * time.Second):
			resultErr = errors.Join(resultErr, observer.stop())
		}
	}()
	observations, err := os.OpenFile(filepath.Join(evidence, "observations.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, observations.Close()) }()
	driver := &processDriver{app: app, statePath: filepath.Join(trialDir, "state.json"), input: json.NewEncoder(stdin), output: json.NewDecoder(stdout), observations: json.NewEncoder(observations)}
	if err := collectNative(ctx, driver, &report, memory.needsSustainedBrowsing); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Collected %d admitted images and %d native gestures in %s. Run the evidence checker; 30k additionally needs Ronin's verdict.\n", report.Images, len(report.Gestures), evidence)
	return err
}
