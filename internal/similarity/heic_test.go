package similarity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
)

func TestAnalysisWorkersUseHEICOwner(t *testing.T) {
	for _, search := range []bool{false, true} {
		for _, cancelAfterDelivery := range []bool{false, true} {
			t.Run(fmt.Sprintf("search=%v/cancel=%v", search, cancelAfterDelivery), func(t *testing.T) {
				runAnalysisHEICOwner(t, search, cancelAfterDelivery)
			})
		}
	}
}

func runAnalysisHEICOwner(t *testing.T, search, cancelAfterDelivery bool) {
	t.Helper()
	// A mismatched pin exercises the owner's typed refusal before source input.
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := heicclient.New(heicclient.Config{Executable: executable, SHA256: [32]byte{1}, Limits: heicdecode.DefaultLimits(0)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Stop(); owner.Wait() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestHEICAnalysisHelper$", "-test.timeout=8s")
	mode := "exit"
	if cancelAfterDelivery {
		mode = "wait"
	}
	cmd.Env = append(os.Environ(), "PICFETCH_OWNED_HEIC_ANALYSIS="+mode)
	req := request{}
	client := Client{HEIC: owner}
	seen := false
	observe := func(delivered bool) {
		seen = delivered
		if cancelAfterDelivery {
			cancel()
		}
	}
	if search {
		req.Search = &SearchRequest{SessionID: 17}
		err = client.searchCommand(ctx, cmd, req, nil, func(event SearchEvent) { observe(event.Kind == SearchReady) })
	} else {
		err = client.analyzeCommand(ctx, cmd, req, nil, func(event Event) { observe(event.Complete) })
	}
	if !seen || (!cancelAfterDelivery && err != nil) || (cancelAfterDelivery && !errors.Is(err, context.Canceled)) {
		t.Fatalf("shared owner result: seen=%v, %v", seen, err)
	}
	if cmd.ProcessState == nil {
		t.Fatal("analysis return preceded process join")
	}
}

func TestHEICAnalysisHelper(_ *testing.T) {
	if os.Getenv("PICFETCH_OWNED_HEIC_ANALYSIS") == "" {
		return
	}
	if err := ownedHEICAnalysis(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func ownedHEICAnalysis() error {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		return err
	}
	if req.HEIC == nil {
		return fmt.Errorf("analysis did not receive the owner's pipe handles")
	}
	cleanup, err := openWorkerSource(&req)
	if err != nil {
		return err
	}
	defer cleanup()
	_ = RegisterLocalFiles()
	// The HEIC suffix forces dispatch; the owner refuses before invoking
	// the bulk input callback, so no encoded file content is needed.
	file, err := os.CreateTemp("", "picfetch-owned-analysis-*.heic")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if err = file.Close(); err != nil {
		return err
	}
	_, err = req.reader.Read(context.Background(), storage.NewFileURI(file.Name()))
	var failure *heicdecode.Failure
	if !errors.As(err, &failure) || failure.Status != heicdecode.StatusUnavailable {
		return fmt.Errorf("expected owner refusal, got %v", err)
	}
	if req.Search != nil {
		err = json.NewEncoder(os.Stdout).Encode(SearchEvent{SessionID: req.Search.SessionID, Revision: 1, Kind: SearchReady})
	} else {
		err = json.NewEncoder(os.Stdout).Encode(Event{Complete: true})
	}
	if err == nil && os.Getenv("PICFETCH_OWNED_HEIC_ANALYSIS") == "wait" {
		_, err = io.Copy(io.Discard, os.Stdin)
	}
	return err
}

func TestAnalysisHEICAttachmentStartFailureJoins(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := heicclient.New(heicclient.Config{Executable: executable, SHA256: [32]byte{1}, Limits: heicdecode.DefaultLimits(0)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Stop(); owner.Wait() }()
	client := Client{HEIC: owner}
	for _, search := range []bool{false, true} {
		for range 10 {
			cmd := exec.Command(filepath.Join(t.TempDir(), "picfetch-owned-missing-executable"))
			if search {
				err = client.searchCommand(context.Background(), cmd, request{Search: &SearchRequest{}}, nil, func(_ SearchEvent) {})
			} else {
				err = client.analyzeCommand(context.Background(), cmd, request{}, nil, func(_ Event) {})
			}
			if err == nil || errors.Is(err, heicclient.ErrBusy) {
				t.Fatalf("start failure leaked service admission: %v", err)
			}
		}
	}
}

func TestSearchPreviewUsesInjectedHEICReader(t *testing.T) {
	_ = RegisterLocalFiles()
	data := []byte("owned HEIC source callback")
	path := filepath.Join(t.TempDir(), "photo.heic")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	reader := imaging.NewReader(func(ctx context.Context, op heicdecode.Operation, input heicclient.Input) (heicdecode.Response, error) {
		calls++
		if op != heicdecode.Decode {
			t.Fatalf("preview operation = %v", op)
		}
		got, err := input(ctx, 128)
		if err != nil {
			return heicdecode.Response{}, err
		}
		if !bytes.Equal(got, data) {
			return heicdecode.Response{}, io.ErrUnexpectedEOF
		}
		return heicdecode.Response{Image: image.NewNRGBA64(image.Rect(0, 0, 4, 6)), Config: image.Config{Width: 4, Height: 6}, Metadata: &heicdecode.Metadata{Make: "Owned camera", DateTimeOriginal: "2026:09:16 12:34:56"}}, nil
	})
	p := searchPreparer{reader: reader, versions: map[string]os.FileInfo{path: info}}
	item := Item{Path: path, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}
	got, err := p.completePreview(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	pixels, err := jpeg.Decode(bytes.NewReader(got.Preview))
	if err != nil || pixels.Bounds().Size() != image.Pt(4, 6) || calls != 1 {
		t.Fatalf("preview decoder: calls=%d, %v", calls, err)
	}
}
