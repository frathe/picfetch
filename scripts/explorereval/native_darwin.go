//go:build darwin

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/frathe/picfetch/internal/explorertrial"
)

type nativeReport struct {
	Collected, Qualified, Exited bool
	ExecutableSHA256, RSSScope   string
	RSSIntervalSeconds, Seconds  float64
	Samples                      int
	PeakRSSKiB                   int64
	ExitCode                     int
	ForcedStop                   bool
}

type memorySample struct {
	Seconds   float64
	Processes int
	RSSKiB    int64
}

// nativeTrial retains and supervises a real app. It never scans the library;
// PicFetch does that through its normal launch path after verifying OS denial.
func nativeTrial(ctx context.Context, config configuration, executable string, output io.Writer) (resultErr error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(config.Out), 0700); err != nil {
		return err
	}
	if err := os.Mkdir(config.Out, 0700); err != nil {
		return fmt.Errorf("evidence directory must be new: %w", err)
	}
	sessionDir := filepath.Join(config.Out, "session")
	bundle := filepath.Join(config.Out, "PicFetch Explorer Trial.app", "Contents")
	if err := os.MkdirAll(filepath.Join(bundle, "MacOS"), 0700); err != nil {
		return err
	}
	retained := filepath.Join(bundle, "MacOS", "picfetch")
	source, err := os.Open(executable)
	if err != nil {
		return err
	}
	target, err := os.OpenFile(retained, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		_ = source.Close()
		return err
	}
	digest := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(target, digest), source)
	if err := errors.Join(copyErr, source.Close(), target.Close()); err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>CFBundleExecutable</key><string>picfetch</string><key>CFBundleIdentifier</key><string>%s</string><key>CFBundleName</key><string>PicFetch Explorer Trial</string><key>CFBundlePackageType</key><string>APPL</string><key>NSHighResolutionCapable</key><true/></dict></plist>`, explorertrial.Identity(sessionDir))
	if err := os.WriteFile(filepath.Join(bundle, "Info.plist"), []byte(plist), 0600); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(config.Out, "native.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, log.Close()) }()
	memory, err := os.OpenFile(filepath.Join(config.Out, "memory.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, memory.Close()) }()
	report := nativeReport{ExecutableSHA256: fmt.Sprintf("%x", digest.Sum(nil)), RSSScope: "sum of RSS KiB for live processes in the owned app process group, including analysis workers; sampled, not exact peak", RSSIntervalSeconds: 1, ExitCode: -1}
	start := time.Now()
	defer func() {
		report.Seconds = time.Since(start).Seconds()
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(filepath.Join(config.Out, "runner.json"), data, 0600)
		}
		resultErr = errors.Join(resultErr, err, os.WriteFile(filepath.Join(config.Out, "exit-status.txt"), []byte(strconv.Itoa(report.ExitCode)+"\n"), 0600))
	}()
	cmd := exec.Command("/usr/bin/sandbox-exec", "-p", "(version 1) (allow default) (deny network*)", retained, "--explorer-trial", sessionDir, "--merge=false", "--max-files", strconv.Itoa(math.MaxInt), "--", config.Library)
	cmd.Env = append(os.Environ(), "PICFETCH_SIMILARITY_ASSETS="+config.Assets)
	cmd.Stdout, cmd.Stderr = log, log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Only this Wait goroutine owns the child; every return after Start joins it.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	pgid := cmd.Process.Pid
	encoder := json.NewEncoder(memory)
	sample := func() (int, error) {
		n, rss, err := nativeRSS(pgid)
		if err != nil {
			return n, err
		}
		report.Samples++
		if rss > report.PeakRSSKiB {
			report.PeakRSSKiB = rss
		}
		return n, encoder.Encode(memorySample{Seconds: time.Since(start).Seconds(), Processes: n, RSSKiB: rss})
	}
	_, sampleErr := sample()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var grace *time.Timer
	defer func() {
		if grace != nil {
			grace.Stop()
		}
	}()
	var deadline <-chan time.Time
	canceled := ctx.Done()
	stop := func() {
		// SIGTERM reaches the app first so it cancels and joins its own workers.
		_ = cmd.Process.Signal(syscall.SIGTERM)
		grace = time.NewTimer(10 * time.Second)
		deadline = grace.C
		canceled = nil
	}
	if sampleErr != nil {
		stop()
	}
	_, _ = fmt.Fprintln(output, "Native trial running. Browse the map, then quit PicFetch to finalize collection.")
	var waitErr error
	running := true
	for running {
		select {
		case waitErr = <-exited:
			running = false
		case <-canceled:
			stop()
		case <-deadline:
			report.ForcedStop = true
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
			deadline = nil
		case <-ticker.C:
			if _, err := sample(); err != nil && sampleErr == nil {
				sampleErr = err
				if grace == nil {
					stop()
				}
			}
		}
	}
	report.Exited = true
	report.ExitCode = cmd.ProcessState.ExitCode()
	// A successfully exited app must have reaped every analysis child. Clean up
	// only its owned group if a crash or forced termination left descendants.
	n, err := sample()
	sampleErr = errors.Join(sampleErr, err)
	if n > 0 {
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		waitErr = errors.Join(waitErr, errors.New("native app left live descendants"))
		until := time.Now().Add(5 * time.Second)
		for n > 0 && time.Now().Before(until) {
			<-ticker.C
			n, _, err = nativeRSS(pgid)
			if err != nil {
				sampleErr = errors.Join(sampleErr, err)
				break
			}
		}
		if n > 0 {
			waitErr = errors.Join(waitErr, errors.New("native descendants did not exit"))
		}
	}
	data, readErr := os.ReadFile(filepath.Join(sessionDir, "session.json"))
	var summary explorertrial.Summary
	if readErr == nil {
		readErr = json.Unmarshal(data, &summary)
	}
	report.Collected = waitErr == nil && sampleErr == nil && readErr == nil && ctx.Err() == nil && summary.Collected
	if !report.Collected {
		return errors.Join(errors.New("native collection incomplete"), waitErr, sampleErr, readErr, ctx.Err())
	}
	_, _ = fmt.Fprintln(output, "Native evidence collected. Human usability and full-library qualification remain separate.")
	return nil
}

// ps exposes only numeric process-group, resident-memory and state fields here.
// Neither commands nor source-bearing argv are retained or inspected.
func nativeRSS(pgid int) (int, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "/bin/ps", "-axo", "pgid=,rss=,stat=").Output()
	if err != nil {
		return 0, 0, err
	}
	count := 0
	var total int64
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		group, err := strconv.Atoi(fields[0])
		if err != nil || group != pgid || strings.HasPrefix(fields[2], "Z") {
			continue
		}
		rss, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		count++
		total += rss
	}
	return count, total, nil
}
