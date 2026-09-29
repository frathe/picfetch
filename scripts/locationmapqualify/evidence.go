package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
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
	Kind       string           `json:"kind"`
	InputNS    int64            `json:"input_ns"`
	VisibleNS  int64            `json:"visible_ns"`
	Before     string           `json:"before"`
	After      string           `json:"after"`
	Skipped    bool             `json:"skipped"`
	Identified bool             `json:"identified"`
	Transform  *VisualTransform `json:"transform,omitempty"`
}

// VisualTransform records independently matched screen pixels, in capture-pixel
// coordinates. It is a consistency witness, not a provenance attestation.
type VisualTransform struct {
	Method  string  `json:"method"`
	Scale   float64 `json:"scale"`
	DX      float64 `json:"dx"`
	DY      float64 `json:"dy"`
	Matches int     `json:"matches"`
	Tested  int     `json:"tested"`
}

func (v *VisualTransform) validFor(kind string) bool {
	if v == nil || v.Method != "patch-grid-v1" || v.Matches < 8 || v.Tested < v.Matches || v.Tested > 1024 || float64(v.Matches)/float64(v.Tested) < 0.65 {
		return false
	}
	if math.IsNaN(v.DX) || math.IsNaN(v.DY) || math.Abs(v.DX) > 4096 || math.Abs(v.DY) > 4096 {
		return false
	}
	switch kind {
	case "pan":
		return v.Scale == 1 && math.Abs(v.DX) >= 18 && math.Abs(v.DX) <= 166 && math.Abs(v.DY) <= 14
	case "zoom":
		if v.Scale != 2 && v.Scale != 0.5 {
			return false
		}
		// The observer captures 1200x800 pixels. Its centered zoom search
		// spans +/-12 and +/-20 four-pixel samples, plus 1.5 for refinement.
		centerDX, centerDY := (1-v.Scale)*1200/2, (1-v.Scale)*800/2
		return math.Abs(v.DX-centerDX) <= 54 && math.Abs(v.DY-centerDY) <= 86
	default:
		return false
	}
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
	if report.Schema != 2 || !validBuildID(report.BuildID) || report.BuildID != expectedBuild {
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
		if !gesture.Transform.validFor(gesture.Kind) {
			return fmt.Errorf("gesture %d has no valid visual transform witness", i)
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
