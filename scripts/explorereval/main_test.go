package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("PICFETCH_EXPLORER_TEST_PROCESS") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestEvaluationCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, nil, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled evaluation returned %v; want context.Canceled", err)
	}
}

func TestEvaluationRejectsUnverifiedModel(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "vision_model.onnx"), []byte("not the pinned model"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"-assets", root, "-library", root}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("evaluation with changed model returned %v; want checksum rejection", err)
	}
}

func TestEvaluationRequiresLocalAssets(t *testing.T) {
	root := t.TempDir()
	err := run(context.Background(), []string{"-assets", filepath.Join(root, "missing"), "-library", root}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "assets") {
		t.Fatalf("evaluation without local assets returned %v; want an assets error", err)
	}
}
