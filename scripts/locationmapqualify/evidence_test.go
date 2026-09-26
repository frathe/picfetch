package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureBuild = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func validReport(images int) Report {
	gestures := make([]Gesture, 40)
	for i := range gestures {
		kind := "pan"
		if i%2 == 1 {
			kind = "zoom"
		}
		gestures[i] = Gesture{Kind: kind, InputNS: int64(i+1) * 1_000_000_000, VisibleNS: int64(i+1)*1_000_000_000 + 100_000_000, Before: "before.png", After: "after.png", Identified: true}
	}
	return Report{
		Schema: 1, BuildID: fixtureBuild, Native: true, Observation: "macos-screen-capture",
		Protocol: "synthetic checker fixture only; not native qualification",
		Images:   images, Formats: map[string]int{"jpeg": images}, Hardware: "test Mac", Storage: "test SSD", Complete: true,
		Stages:   []Stage{{Kind: "cold", StartNS: 1, EndNS: 1_000_000_001, PreparationNS: 500_000_000, ScanNS: 500_000_000, Complete: true}, {Kind: "warm", StartNS: 2_000_000_000, EndNS: 3_000_000_000, PreparationNS: 500_000_000, ScanNS: 500_000_000, Complete: true}},
		Gestures: gestures, Cancellations: []Cancellation{{InputNS: 1, VisibleNS: 250_000_001, Before: "before.png", After: "after.png", Complete: true}},
		Memory: []MemorySample{{AtNS: 1, RSSBytes: 100}, {AtNS: 60_000_000_001, RSSBytes: 200}},
	}
}

func TestCheckReportRejectsUnidentifiedGestures(t *testing.T) {
	report := validReport(10_000)
	report.Gestures[0].Identified = false
	if err := CheckReport(report, 10_000, fixtureBuild); err == nil {
		t.Fatal("an unrelated changed frame qualified as gesture timing")
	}
}

func writePNG(t *testing.T, path string, shade color.RGBA) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, shade)
	img.SetRGBA(1, 0, color.RGBA{R: 50, A: 255})
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func writeEvidence(t *testing.T, dir string, report Report) {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReportQualifiesTenThousandAtBoundary(t *testing.T) {
	report := validReport(10_000)
	if err := CheckReport(report, 10_000, fixtureBuild); err != nil {
		t.Fatal(err)
	}
	for i := range report.Gestures[:2] {
		report.Gestures[i].VisibleNS++
	}
	if err := CheckReport(report, 10_000, fixtureBuild); err != nil {
		t.Fatalf("95%% should pass: %v", err)
	}
	report.Gestures[2].VisibleNS++
	if err := CheckReport(report, 10_000, fixtureBuild); err == nil {
		t.Fatal("below 95% accepted")
	}
}

func TestCheckEvidenceLoadsValidPNGs(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "before.png"), color.RGBA{R: 1, A: 255})
	writePNG(t, filepath.Join(dir, "after.png"), color.RGBA{R: 2, A: 255})
	writeEvidence(t, dir, validReport(10_000))
	if _, err := CheckEvidence(dir, 10_000, fixtureBuild); err != nil {
		t.Fatal(err)
	}
}

func TestCheckEvidenceRejectsMissingReport(t *testing.T) {
	if _, err := CheckEvidence(t.TempDir(), 10_000, fixtureBuild); err == nil {
		t.Fatal("missing report accepted")
	}
}

func TestCheckReportRejectsPartialAndWrongIdentity(t *testing.T) {
	cases := map[string]func(*Report){
		"partial":           func(r *Report) { r.Complete = false },
		"non native":        func(r *Report) { r.Native = false },
		"wrong count":       func(r *Report) { r.Images = 9_999 },
		"wrong build":       func(r *Report) { r.BuildID = strings.Repeat("a", 64) },
		"invalid build":     func(r *Report) { r.BuildID = strings.Repeat("A", 64) },
		"wrong observation": func(r *Report) { r.Observation = "timer" },
		"missing hardware":  func(r *Report) { r.Hardware = " " },
		"format mismatch":   func(r *Report) { r.Formats["jpeg"]-- },
		"negative format":   func(r *Report) { r.Formats["png"] = -1 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := validReport(10_000)
			change(&r)
			if err := CheckReport(r, 10_000, fixtureBuild); err == nil {
				t.Fatal("invalid report accepted")
			}
		})
	}
}

func TestCheckReportRejectsInvalidStages(t *testing.T) {
	cases := map[string]func(*Report){
		"missing warm":              func(r *Report) { r.Stages = r.Stages[:1] },
		"interrupted":               func(r *Report) { r.Stages[1].Complete = false },
		"unknown kind":              func(r *Report) { r.Stages[0].Kind = "cached" },
		"unordered timestamps":      func(r *Report) { r.Stages[0].EndNS = r.Stages[0].StartNS },
		"negative duration":         func(r *Report) { r.Stages[0].PreparationNS = -1 },
		"duration exceeds interval": func(r *Report) { r.Stages[0].ScanNS++ },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := validReport(10_000)
			change(&r)
			if err := CheckReport(r, 10_000, fixtureBuild); err == nil {
				t.Fatal("invalid stage accepted")
			}
		})
	}
}

func TestCheckReportRejectsInvalidMeasurements(t *testing.T) {
	cases := map[string]func(*Report){
		"too few gestures": func(r *Report) { r.Gestures = r.Gestures[:39] },
		"no zoom": func(r *Report) {
			for i := range r.Gestures {
				r.Gestures[i].Kind = "pan"
			}
		},
		"skipped gesture":          func(r *Report) { r.Gestures[0].Skipped = true },
		"bad gesture time":         func(r *Report) { r.Gestures[0].VisibleNS = r.Gestures[0].InputNS },
		"missing gesture artifact": func(r *Report) { r.Gestures[0].After = "" },
		"incomplete cancellation":  func(r *Report) { r.Cancellations[0].Complete = false },
		"bad cancellation time":    func(r *Report) { r.Cancellations[0].InputNS = 0 },
		"slow cancellation":        func(r *Report) { r.Cancellations[0].VisibleNS++ },
		"too few memory samples":   func(r *Report) { r.Memory = r.Memory[:1] },
		"unordered memory":         func(r *Report) { r.Memory[1].AtNS = r.Memory[0].AtNS },
		"nonpositive RSS":          func(r *Report) { r.Memory[0].RSSBytes = 0 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := validReport(10_000)
			change(&r)
			if err := CheckReport(r, 10_000, fixtureBuild); err == nil {
				t.Fatal("invalid measurement accepted")
			}
		})
	}
}

func TestCheckReportThirtyThousandRequiresHumanAndSustainedMemory(t *testing.T) {
	report := validReport(30_000)
	report.OpenCloseCycles = 3
	report.VerdictBy = "Ronin"
	report.Verdict = "pass"
	for i := range report.Gestures {
		report.Gestures[i].VisibleNS += 2_000_000_000
	}
	report.Cancellations[0].VisibleNS += 1_000_000_000
	if err := CheckReport(report, 30_000, fixtureBuild); err != nil {
		t.Fatalf("30k should have no 10k latency gate: %v", err)
	}
	cases := map[string]func(*Report){
		"cycles":      func(r *Report) { r.OpenCloseCycles = 2 },
		"memory span": func(r *Report) { r.Memory[1].AtNS-- },
		"reviewer":    func(r *Report) { r.VerdictBy = "other" },
		"verdict":     func(r *Report) { r.Verdict = "fail" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r := report
			r.Memory = append([]MemorySample(nil), report.Memory...)
			change(&r)
			if err := CheckReport(r, 30_000, fixtureBuild); err == nil {
				t.Fatal("unqualified 30k accepted")
			}
		})
	}
}

func TestCheckReportSmokeCannotClaimLargerCount(t *testing.T) {
	report := validReport(500)
	if err := CheckReport(report, 500, fixtureBuild); err != nil {
		t.Fatal(err)
	}
	if err := CheckReport(report, 10_000, fixtureBuild); err == nil {
		t.Fatal("smoke reported as 10k")
	}
	if err := CheckReport(report, 30_000, fixtureBuild); err == nil {
		t.Fatal("smoke reported as 30k")
	}
}

func TestCheckEvidenceRejectsBadArtifactsAndJSON(t *testing.T) {
	cases := map[string]func(*testing.T, string, *Report){
		"missing PNG": func(_ *testing.T, _ string, r *Report) { r.Gestures[0].After = "missing.png" },
		"corrupt PNG": func(t *testing.T, dir string, r *Report) {
			if err := os.WriteFile(filepath.Join(dir, "corrupt.png"), []byte("not PNG"), 0600); err != nil {
				t.Fatal(err)
			}
			r.Gestures[0].After = "corrupt.png"
		},
		"same pixels":     func(_ *testing.T, _ string, r *Report) { r.Gestures[0].After = "before.png" },
		"parent escape":   func(_ *testing.T, _ string, r *Report) { r.Gestures[0].After = "../outside.png" },
		"absolute escape": func(_ *testing.T, dir string, r *Report) { r.Gestures[0].After = filepath.Join(dir, "after.png") },
		"symlink escape": func(t *testing.T, dir string, r *Report) {
			outside := filepath.Join(t.TempDir(), "outside.png")
			writePNG(t, outside, color.RGBA{R: 2, A: 255})
			if err := os.Symlink(outside, filepath.Join(dir, "link.png")); err != nil {
				t.Fatal(err)
			}
			r.Gestures[0].After = "link.png"
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writePNG(t, filepath.Join(dir, "before.png"), color.RGBA{R: 1, A: 255})
			writePNG(t, filepath.Join(dir, "after.png"), color.RGBA{R: 2, A: 255})
			report := validReport(10_000)
			change(t, dir, &report)
			writeEvidence(t, dir, report)
			if _, err := CheckEvidence(dir, 10_000, fixtureBuild); err == nil {
				t.Fatal("invalid artifact accepted")
			}
		})
	}
	t.Run("unknown JSON field", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "report.json"), []byte(`{"extra":true}`), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := CheckEvidence(dir, 10_000, fixtureBuild); err == nil {
			t.Fatal("unknown field accepted")
		}
	})
	t.Run("oversized report", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "report.json"), make([]byte, maxReportBytes+1), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := CheckEvidence(dir, 10_000, fixtureBuild); err == nil {
			t.Fatal("oversized report accepted")
		}
	})
}
