package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_RequiresRepoRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	err := run([]string{"--prev", "0.1.0", "--next", "0.1.1"})
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

func TestRun_WriteAndClearDone(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.WriteFile(todosPath, []byte(sampleTodos), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "notes.md")
	if err := run([]string{"--prev", "0.2.7", "--next", "0.2.8", "--write", out, "--clear-done"}); err != nil {
		t.Fatal(err)
	}
	notes, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notes), "selection rectangle") {
		t.Fatalf("notes missing item: %s", notes)
	}
	if !strings.Contains(string(notes), "compare/v0.2.7...v0.2.8") {
		t.Fatalf("notes missing changelog: %s", notes)
	}
	todos, err := os.ReadFile(todosPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(todos), "selection rectangle") {
		t.Fatalf("Done items not cleared: %s", todos)
	}
	if !strings.Contains(string(todos), "- leftover work") {
		t.Fatalf("TODO section lost: %s", todos)
	}
}
