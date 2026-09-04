package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMakeValidateAMPDoesNotExecuteOutputPathAsShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Makefile target uses a POSIX shell on supported maintainer systems")
	}
	repo := repositoryRoot(t)
	marker := filepath.Join(t.TempDir(), "shell-injection-marker")
	maliciousOutput := "docs; touch " + marker + "; #"
	cmd := exec.Command("make", "validate-amp", "SITE_OUTPUT_DIR="+maliciousOutput)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "npm_config_offline=true")
	_, _ = cmd.CombinedOutput()
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("validate-amp executed SITE_OUTPUT_DIR as shell syntax")
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect shell-injection marker: %v", err)
	}
}

func TestMakeBuildDoesNotExecuteOutputPathAsShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Makefile target uses a POSIX shell on supported maintainer systems")
	}
	repo := repositoryRoot(t)
	fixtureRoot := t.TempDir()
	marker := filepath.Join(fixtureRoot, "shell-injection-marker")
	maliciousOutput := filepath.Join(fixtureRoot, "site") + `"; touch ` + marker + `; #`
	cmd := exec.Command("make", "build",
		"SITE_OUTPUT_DIR="+maliciousOutput,
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	_, _ = cmd.CombinedOutput()
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("make build executed SITE_OUTPUT_DIR as shell syntax")
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect shell-injection marker: %v", err)
	}
}
