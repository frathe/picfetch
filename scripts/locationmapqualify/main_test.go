package main

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeCLIRequiresExplicitPathsAndFiniteTimeout(t *testing.T) {
	var output bytes.Buffer
	options, err := parseNativeRun([]string{"-images", "fixture", "-evidence", "new-results", "-binary", "app", "-helper", "capture", "-timeout", "2m"}, &output)
	if err != nil || options.images != "fixture" || options.evidence != "new-results" || options.timeout != 2*time.Minute {
		t.Fatalf("explicit native options: %+v, %v", options, err)
	}
	for _, args := range [][]string{nil, {"-images", "fixture"}, {"-images", "fixture", "-evidence", "out", "-binary", "app", "-helper", "capture", "-timeout", "0"}} {
		if _, err := parseNativeRun(args, &output); err == nil {
			t.Fatalf("incomplete/unbounded run accepted: %v", args)
		}
	}
}

func TestCheckerCLIUsesBinaryIdentityAndPropagatesFailure(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "test-build")
	if err := os.WriteFile(binary, []byte("checker fixture, not an application"), 0o600); err != nil {
		t.Fatal(err)
	}
	build, err := binaryIdentity(binary)
	if err != nil {
		t.Fatal(err)
	}
	report := validReport(3)
	report.BuildID = build
	writePNG(t, filepath.Join(dir, "before.png"), color.RGBA{R: 1, A: 255})
	writePNG(t, filepath.Join(dir, "after.png"), color.RGBA{R: 2, A: 255})
	writeEvidence(t, dir, report)
	args := []string{"check", "-evidence", dir, "-images", "3", "-binary", binary}
	var output bytes.Buffer
	if err := runCLI(args, &output); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binary, []byte("changed build"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runCLI(args, &output); err == nil {
		t.Fatal("wrong actual binary accepted")
	}
	if err := runCLI([]string{"check", "-evidence", dir}, &output); err == nil {
		t.Fatal("missing expected count/build accepted")
	}
	if err := runCLI([]string{"unknown"}, &output); err == nil {
		t.Fatal("unknown mode accepted")
	}
}
