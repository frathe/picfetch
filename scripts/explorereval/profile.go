package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
)

// profileEvent deliberately excludes source identities, pixels and vectors.
type profileEvent struct {
	Pass                      string
	ReceivedSeconds           float64
	Total, Successful, Failed int
	Reused                    int
	Stage                     string
	OfflineVerified, Complete bool
	Measurements              similarity.Measurements
}

type profilePass struct {
	profileEvent
	FirstMapSeconds, WorkerExitedSeconds float64
	Cohorts, Unassigned                  int
}

type throughputProfile struct {
	ModelRevision, ExecutableSHA256, InputSHA256 string
	GOOS, GOARCH, GoVersion                      string
	Provider                                     string
	Automatic                                    bool
	Available, Sampled                           int
	ScanSeconds                                  float64
	CacheBytes                                   int64
	Passes                                       []profilePass
}

func profile(ctx context.Context, config configuration, automatic bool, output io.Writer) error {
	scanStart := time.Now()
	files, err := scanLibrary(ctx, config.Library)
	if err != nil {
		return err
	}
	report := throughputProfile{
		ModelRevision: similarity.ModelRevision, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		GoVersion: runtime.Version(), Provider: "cpu", Automatic: automatic, Available: len(files),
	}
	files = smokeSample(files)
	report.Sampled, report.ScanSeconds = len(files), time.Since(scanStart).Seconds()
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.Path()
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	binary, err := os.Open(executable)
	if err != nil {
		return err
	}
	digest := sha256.New()
	_, hashErr := io.Copy(digest, binary)
	closeErr := binary.Close()
	if hashErr != nil {
		return hashErr
	}
	if closeErr != nil {
		return closeErr
	}
	report.ExecutableSHA256 = fmt.Sprintf("%x", digest.Sum(nil))
	if err := os.MkdirAll(filepath.Dir(config.Out), 0700); err != nil {
		return err
	}
	if err := os.Mkdir(config.Out, 0700); err != nil {
		return fmt.Errorf("evidence directory must be new: %w", err)
	}
	trace, err := os.OpenFile(filepath.Join(config.Out, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = trace.Close() }()
	cache, err := os.MkdirTemp("", "picfetch-profile-cache-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cache) }()
	if err := favstore.Save(cache, "Profile", files); err != nil {
		return err
	}
	client := similarity.Client{Assets: config.Assets, FavoritesDir: cache}
	for _, name := range []string{"cold", "warm"} {
		pass, identity, err := profileAnalysis(ctx, client, paths, automatic, name, trace, output)
		if err != nil {
			return err
		}
		if name == "cold" {
			report.InputSHA256 = identity
		} else if identity != report.InputSHA256 || pass.Successful != report.Passes[0].Successful || pass.Reused != pass.Successful || pass.Measurements.InferenceAttempts != 0 {
			return fmt.Errorf("sources changed or warm analysis did not reuse every successful source")
		}
		report.Passes = append(report.Passes, pass)
	}
	if err := filepath.WalkDir(cache, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("cache size: %s: %w", path, err)
		}
		report.CacheBytes += info.Size()
		return nil
	}); err != nil {
		return err
	}
	if err := os.RemoveAll(cache); err != nil {
		return err
	}
	if err := trace.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// A completed summary exists only after both workers exit and private cache
	// cleanup succeeds. Partial event traces remain useful after cancellation.
	part := filepath.Join(config.Out, "profile.json.part")
	if err := writeJSON(part, report); err != nil {
		return err
	}
	return os.Rename(part, filepath.Join(config.Out, "profile.json"))
}

func profileAnalysis(ctx context.Context, client similarity.Client, paths []string, automatic bool, name string, trace, output io.Writer) (profilePass, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	controls := make(chan similarity.Control, 1)
	controls <- similarity.Control{Automatic: automatic}
	close(controls)
	start := time.Now()
	var pass profilePass
	var final similarity.Event
	var deliveryErr error
	encoder := json.NewEncoder(trace)
	err := client.Analyze(ctx, paths, controls, func(event similarity.Event) {
		if deliveryErr != nil {
			return
		}
		record := profileEvent{
			Pass: name, ReceivedSeconds: time.Since(start).Seconds(), Total: event.Total,
			Successful: event.Successful, Failed: event.Failed, Reused: event.Reused,
			Stage: event.Stage, Complete: event.Complete, OfflineVerified: event.OfflineVerified,
			Measurements: event.Measurements,
		}
		deliveryErr = encoder.Encode(record)
		if deliveryErr == nil {
			_, deliveryErr = fmt.Fprintf(output, "%s processed %d/%d: ready=%d failed=%d reused=%d stage=%s\n", name, event.Successful+event.Failed, event.Total, event.Successful, event.Failed, event.Reused, event.Stage)
		}
		if deliveryErr != nil {
			cancel()
			return
		}
		if event.Measurements.Publications > 0 && pass.FirstMapSeconds == 0 {
			pass.FirstMapSeconds = record.ReceivedSeconds
		}
		if event.Complete {
			final, pass.profileEvent = event, record
		}
	})
	pass.WorkerExitedSeconds = time.Since(start).Seconds()
	if deliveryErr != nil {
		return pass, "", deliveryErr
	}
	if err != nil {
		return pass, "", err
	}
	if !final.Complete || final.OfflineVerified != similarity.EnforcesNetworkIsolation() || final.Total != len(paths) || len(final.Items) != len(paths) || final.Successful+final.Failed != len(paths) || final.CacheWarning != "" || final.Measurements.Publications == 0 {
		return pass, "", fmt.Errorf("production profile lacks complete source/cache/publication or network-policy accounting")
	}
	digest := sha256.New()
	identities := json.NewEncoder(digest)
	cohorts := map[string]bool{}
	for i, item := range final.Items {
		if item.Path != paths[i] {
			return pass, "", fmt.Errorf("production profile changed input identities")
		}
		// Hash identities for repeatability without recording paths or pixels.
		if err := identities.Encode(struct {
			Path, SHA256, Error string
			Size, ModifiedNS    int64
		}{item.Path, item.SHA256, item.Error, item.Size, item.ModifiedNS}); err != nil {
			return pass, "", err
		}
		if item.Error == "" {
			if item.Cohort == "unassigned" {
				pass.Unassigned++
			} else {
				cohorts[item.Cohort] = true
			}
		}
	}
	pass.Cohorts = len(cohorts)
	return pass, fmt.Sprintf("%x", digest.Sum(nil)), nil
}
