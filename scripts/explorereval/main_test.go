package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv("PICFETCH_NATIVE_FIXTURE"); mode != "" {
		stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		_, opts, err := launch.Parse(os.Args[1:])
		if err == nil {
			err = similarity.VerifyOffline(context.Background())
		}
		if err != nil {
			os.Exit(2)
		}
		s, err := explorertrial.New(opts.ExplorerTrial)
		if err != nil {
			os.Exit(3)
		}
		n := s.Begin([]string{"fixture"})
		if mode == "cancel" {
			<-stopped.Done()
			s.Exited(n, stopped.Err())
			_ = s.Close()
			os.Exit(1)
		}
		event := similarity.Event{Total: 1, Successful: 1, OfflineVerified: true, Complete: true, Items: []similarity.Item{{Path: "fixture"}}}
		eventID := s.Received(n, event)
		s.Applied(n, eventID, event, time.Now(), time.Now())
		s.Exited(n, nil)
		if err := s.Close(); err != nil {
			os.Exit(4)
		}
		os.Exit(0)
	}

	if similarity.WorkerMain() {
		return
	}
	if os.Getenv("PICFETCH_EXPLORER_TEST_PROCESS") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestEvaluationFailedProbeTiming(t *testing.T) {
	read := false
	file := uitest.ReaderURI(storage.NewFileURI(filepath.Join(t.TempDir(), "broken.jpg")), func() (io.ReadCloser, error) {
		read = true
		return nil, io.ErrUnexpectedEOF
	})
	var result evaluation
	var entry item
	loaded, err := result.decode(context.Background(), file, &entry)
	if err == nil || !read || loaded != nil {
		t.Fatalf("premise: failed probe was not exercised: read=%v err=%v", read, err)
	}
	if result.DecodeSeconds <= 0 || result.EncodeSeconds != 0 {
		t.Fatalf("failed probe timing: decode=%v encode=%v", result.DecodeSeconds, result.EncodeSeconds)
	}
}

func TestEvaluationReportClusteringProvenance(t *testing.T) {
	for _, tc := range []struct{ goos, arch, binding, native string }{
		{"darwin", "amd64", "v1.25.0", "1.23.2"},
		{"darwin", "arm64", "v1.36.0", "1.29.0"},
		{"linux", "amd64", "v1.36.0", "1.29.0"},
		{"windows", "arm64", "v1.36.0", "1.29.0"},
	} {
		binding, native := reportRuntimeVersions(tc.goos, tc.arch)
		if binding != tc.binding || native != tc.native {
			t.Errorf("%s/%s reports %s/%s, want %s/%s", tc.goos, tc.arch, binding, native, tc.binding, tc.native)
		}
	}
	out := t.TempDir()
	result := evaluation{Config: configuration{Out: out}, Items: []item{{Path: filepath.Join(out, "a.jpg"), Cohort: "a", Position: []float32{0, 0}}}}
	if err := writeReview(result, result.Items); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "pipeline-evaluation.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PhotoPrism", "c48d23f6b03c25fc19d376d789fac56c32a26fdb", "minPts=3", "minimum cohort 4"} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("report missing active clustering provenance %q", want)
		}
	}
	if bytes.Contains(data, []byte("alDuncanson/latent")) {
		t.Fatal("report identifies the removed clustering dependency")
	}
	binding, native := "v1.36.0", "1.29.0"
	if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
		binding, native = "v1.25.0", "1.23.2"
	}
	if !bytes.Contains(data, []byte("binding "+binding+"; native ONNX Runtime "+native)) {
		t.Fatal("emitted report lacks the build-selected runtime versions")
	}
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

func TestNativeTrialRequiresExecutable(t *testing.T) {
	err := run(context.Background(), []string{"-trial", "library"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "native executable") {
		t.Fatalf("library trial: %v", err)
	}
}

func TestTrialPlatformAdmission(t *testing.T) {
	for _, trial := range []string{"throughput", "smoke", "library"} {
		t.Run(trial, func(t *testing.T) {
			want := (runtime.GOOS == "darwin" || trial == "throughput") && similarity.SupportedPlatform()
			if err := checkTrialPlatform(trial); (err == nil) != want {
				t.Fatalf("%s on %s/%s: %v; admitted=%v", trial, runtime.GOOS, runtime.GOARCH, err, want)
			}
		})
	}
}
