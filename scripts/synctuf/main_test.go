package main

import (
	"strings"
	"testing"
)

func TestRun_RequiresRepoRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	err := run([]string{"--check"})
	if err == nil || err.Error() != "run from the repository root" {
		t.Fatalf("got %v, want run from the repository root", err)
	}
}

func TestRun_Usage(t *testing.T) {
	err := run(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("got %v, want usage error", err)
	}
}
