package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/frathe/picfetch/internal/locationtrial"
)

const (
	maxReportBytes    = 2 * 1024 * 1024
	maxArtifactBytes  = 32 * 1024 * 1024
	maxArtifactPixels = 64 * 1024 * 1024
)

type Report struct {
	Schema          int            `json:"schema"`
	BuildID         string         `json:"build_id"`
	Observation     string         `json:"observation"`
	Protocol        string         `json:"protocol"`
	Failure         string         `json:"failure,omitempty"`
	Images          int            `json:"images"`
	Formats         map[string]int `json:"formats"`
	Hardware        string         `json:"hardware"`
	Storage         string         `json:"storage"`
	Stages          []Stage        `json:"stages"`
	Gestures        []Gesture      `json:"gestures"`
	Cancellations   []Cancellation `json:"cancellations"`
	Memory          []MemorySample `json:"memory"`
	OpenCloseCycles int            `json:"open_close_cycles"`
	VerdictBy       string         `json:"verdict_by"`
	Verdict         string         `json:"verdict"`
	Native          bool           `json:"native"`
	Complete        bool           `json:"complete"`
}

type Stage = locationtrial.Stage

type Gesture struct {
	Kind       string `json:"kind"`
	InputNS    int64  `json:"input_ns"`
	VisibleNS  int64  `json:"visible_ns"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Skipped    bool   `json:"skipped"`
	Identified bool   `json:"identified"`
}

type Cancellation struct {
	InputNS   int64  `json:"input_ns"`
	VisibleNS int64  `json:"visible_ns"`
	Before    string `json:"before"`
	After     string `json:"after"`
	Complete  bool   `json:"complete"`
}

type MemorySample struct {
	AtNS     int64 `json:"at_ns"`
	RSSBytes int64 `json:"rss_bytes"`
}

func CheckReport(report Report, expectedImages int, expectedBuild string) error {
	if expectedImages <= 0 || !validBuildID(expectedBuild) {
		return errors.New("invalid expected count or build ID")
	}
	if report.Schema != 1 || !validBuildID(report.BuildID) || report.BuildID != expectedBuild {
		return errors.New("invalid schema or build ID")
	}
	if !report.Native || !report.Complete || report.Failure != "" || strings.TrimSpace(report.Protocol) == "" || report.Observation != "macos-screen-capture" {
		return errors.New("incomplete or non-native screen capture report")
	}
	if report.Images != expectedImages || strings.TrimSpace(report.Hardware) == "" || strings.TrimSpace(report.Storage) == "" {
		return errors.New("image count or hardware/storage identity mismatch")
	}
	if len(report.Formats) == 0 {
		return errors.New("missing image formats")
	}
	formats := 0
	for name, count := range report.Formats {
		if strings.TrimSpace(name) == "" || count < 0 || count > expectedImages-formats {
			return fmt.Errorf("invalid format count for %q", name)
		}
		formats += count
	}
	if formats != expectedImages {
		return fmt.Errorf("format total %d differs from %d images", formats, expectedImages)
	}
	cold, warm := false, false
	for i, stage := range report.Stages {
		if !stage.Complete || stage.StartNS <= 0 || stage.EndNS <= stage.StartNS {
			return fmt.Errorf("stage %d is incomplete or has invalid timestamps", i)
		}
		duration := stage.EndNS - stage.StartNS
		if duration <= 0 || stage.PreparationNS < 0 || stage.ScanNS < 0 || stage.PreparationNS > duration || stage.ScanNS > duration-stage.PreparationNS {
			return fmt.Errorf("stage %d has invalid durations", i)
		}
		switch stage.Kind {
		case "cold":
			cold = true
		case "warm":
			warm = true
		default:
			return fmt.Errorf("stage %d has invalid kind %q", i, stage.Kind)
		}
	}
	if !cold || !warm {
		return errors.New("both cold and warm stages are required")
	}
	if len(report.Gestures) < 40 {
		return errors.New("fewer than 40 gestures")
	}
	pan, zoom, fast := false, false, 0
	for i, gesture := range report.Gestures {
		if !gesture.Identified || gesture.Skipped || gesture.InputNS <= 0 || gesture.VisibleNS <= gesture.InputNS || gesture.Before == "" || gesture.After == "" {
			return fmt.Errorf("gesture %d is skipped or invalid", i)
		}
		switch gesture.Kind {
		case "pan":
			pan = true
		case "zoom":
			zoom = true
		default:
			return fmt.Errorf("gesture %d has invalid kind %q", i, gesture.Kind)
		}
		if gesture.VisibleNS-gesture.InputNS <= 100_000_000 {
			fast++
		}
	}
	if !pan || !zoom {
		return errors.New("both pan and zoom gestures are required")
	}
	if len(report.Cancellations) == 0 {
		return errors.New("cancellation feedback is required")
	}
	for i, cancellation := range report.Cancellations {
		if !cancellation.Complete || cancellation.InputNS <= 0 || cancellation.VisibleNS <= cancellation.InputNS || cancellation.Before == "" || cancellation.After == "" {
			return fmt.Errorf("cancellation %d is incomplete or invalid", i)
		}
		if expectedImages == 10_000 && cancellation.VisibleNS-cancellation.InputNS > 250_000_000 {
			return fmt.Errorf("cancellation %d exceeds 250ms", i)
		}
	}
	if len(report.Memory) < 2 {
		return errors.New("at least two memory samples are required")
	}
	last := int64(0)
	for i, sample := range report.Memory {
		if sample.AtNS <= last || sample.RSSBytes <= 0 {
			return fmt.Errorf("memory sample %d is invalid or unordered", i)
		}
		last = sample.AtNS
	}
	if expectedImages == 10_000 && fast < (len(report.Gestures)*95+99)/100 {
		return fmt.Errorf("only %d/%d gestures reached visible output within 100ms", fast, len(report.Gestures))
	}
	if expectedImages == 30_000 {
		if report.OpenCloseCycles < 3 || report.Memory[len(report.Memory)-1].AtNS-report.Memory[0].AtNS < 60_000_000_000 || report.VerdictBy != "Ronin" || report.Verdict != "pass" {
			return errors.New("30k requires three open/close cycles, 60s memory, and Ronin's passing verdict")
		}
	}
	return nil
}

func validBuildID(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			if digit < 'a' || digit > 'f' {
				return false
			}
		}
	}
	return true
}

func CheckEvidence(dir string, expectedImages int, expectedBuild string) (Report, error) {
	var report Report
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return report, fmt.Errorf("evidence directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return report, errors.New("evidence directory is not a directory")
	}
	data, err := boundedFile(filepath.Join(root, "report.json"), maxReportBytes)
	if err != nil {
		return report, fmt.Errorf("report.json: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return report, fmt.Errorf("report.json: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return report, errors.New("report.json contains trailing data")
	}
	if err := CheckReport(report, expectedImages, expectedBuild); err != nil {
		return report, err
	}
	for i, gesture := range report.Gestures {
		if err := checkImagePair(root, gesture.Before, gesture.After); err != nil {
			return report, fmt.Errorf("gesture %d: %w", i, err)
		}
	}
	for i, cancellation := range report.Cancellations {
		if err := checkImagePair(root, cancellation.Before, cancellation.After); err != nil {
			return report, fmt.Errorf("cancellation %d: %w", i, err)
		}
	}
	return report, nil
}

func boundedFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("file exceeds size limit")
	}
	return data, nil
}

func artifactPath(root, name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", errors.New("artifact path must be relative")
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact escapes evidence directory")
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", errors.New("artifact symlink escapes evidence directory")
	}
	return resolved, nil
}

func readPNG(root, name string) (image.Image, error) {
	path, err := artifactPath(root, name)
	if err != nil {
		return nil, err
	}
	data, err := boundedFile(path, maxArtifactBytes)
	if err != nil {
		return nil, err
	}
	if len(data) < 8 || !bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")) {
		return nil, errors.New("artifact is not PNG")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxArtifactPixels/config.Height {
		return nil, errors.New("PNG exceeds pixel limit")
	}
	return png.Decode(bytes.NewReader(data))
}

func checkImagePair(root, before, after string) error {
	first, err := readPNG(root, before)
	if err != nil {
		return fmt.Errorf("before %q: %w", before, err)
	}
	second, err := readPNG(root, after)
	if err != nil {
		return fmt.Errorf("after %q: %w", after, err)
	}
	if first.Bounds().Dx() != second.Bounds().Dx() || first.Bounds().Dy() != second.Bounds().Dy() {
		return nil
	}
	for y := 0; y < first.Bounds().Dy(); y++ {
		for x := 0; x < first.Bounds().Dx(); x++ {
			ar, ag, ab, aa := first.At(first.Bounds().Min.X+x, first.Bounds().Min.Y+y).RGBA()
			br, bg, bb, ba := second.At(second.Bounds().Min.X+x, second.Bounds().Min.Y+y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				return nil
			}
		}
	}
	return errors.New("before and after PNG pixels are identical")
}
