package imaging

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"testing"
	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestAnimatedPreview(t *testing.T) {
	u := storage.NewFileURI(writeTempFile(t, "partial.gif", uitest.EncodePartialFrameGIF(t)))
	p, err := LoadAnimatedPreviewContext(context.Background(), u, 512, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Frames) != 2 || p.Frames[0].Bounds().Size() != image.Pt(80, 40) || p.Frames[1].Bounds().Size() != image.Pt(80, 40) {
		t.Fatalf("animation canvas/frames: %+v", p)
	}
	r, _, b, a := p.Frames[0].At(35, 15).RGBA()
	if r == 0 || b != 0 || a == 0 {
		t.Fatal("first frame lost its offset red patch")
	}
	_, _, _, a = p.Frames[0].At(0, 0).RGBA()
	if a != 0 {
		t.Fatal("first frame lost transparency")
	}
	r, _, b, a = p.Frames[1].At(35, 15).RGBA()
	if r != 0 || b == 0 || a == 0 {
		t.Fatal("second frame was not composited independently")
	}
	if len(p.Delays) != 2 || p.Delays[0] != 50*time.Millisecond || p.Delays[1] != 50*time.Millisecond {
		t.Fatalf("delays: %v", p.Delays)
	}
}

func TestAnimatedPreviewBudget(t *testing.T) {
	u := storage.NewFileURI(writeTempFile(t, "budget.gif", uitest.EncodeAnimatedGIF(t, 160, 80, []color.Color{color.Black, color.White}, []int{1, 0})))
	const budget = 2 * 40 * 20 * 4
	p, err := LoadAnimatedPreviewContext(context.Background(), u, 512, budget)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Frames) != 2 || p.AnimationTruncated {
		t.Fatalf("budget should reduce resolution and preserve animation: %+v", p)
	}
	if p.DecodedBytes() > budget || p.Frames[0].Bounds().Size() != image.Pt(40, 20) {
		t.Fatalf("preview bounds %v, retained bytes %d", p.Frames[0].Bounds(), p.DecodedBytes())
	}
	if p.Delays[0] != 10*time.Millisecond || p.Delays[1] != 100*time.Millisecond {
		t.Fatalf("source/zero delays: %v", p.Delays)
	}
}

func TestAnimatedPreviewFallback(t *testing.T) {
	for _, tc := range []struct {
		name        string
		count, w, h int
		budget      int64
		truncated   bool
	}{
		{"disabled", 2, 80, 40, 0, false},
		{"too_small", 2, 80, 40, 7, true},
		{"native_decode_limit", 60, 2048, 2048, 16 << 20, true},
		{"frame_count_limit", 4097, 1, 1, 16 << 20, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame := solidFrame(image.Rect(0, 0, 1, 1), color.Palette{color.White, color.Black}, color.White)
			frames, delays := make([]*image.Paletted, tc.count), make([]int, tc.count)
			for i := range frames {
				frames[i], delays[i] = frame, 1
			}
			u := storage.NewFileURI(writeTempFile(t, "fallback.gif", buildGIF(t, tc.w, tc.h, frames, delays, nil)))
			p, err := LoadAnimatedPreviewContext(context.Background(), u, 512, tc.budget)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Frames) != 1 || p.AnimationTruncated != tc.truncated || len(p.Delays) != 0 {
				t.Fatalf("fallback: %d frames, %d delays, truncated=%v", len(p.Frames), len(p.Delays), p.AnimationTruncated)
			}
			if b := p.Frames[0].Bounds(); b.Dx() > 512 || b.Dy() > 512 {
				t.Fatalf("unbounded static fallback: %v", b)
			}
		})
	}
}

func TestAnimatedPreviewDisposalPrevious(t *testing.T) {
	palette := color.Palette{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}, color.White}
	frames := []*image.Paletted{
		solidFrame(image.Rect(0, 0, 20, 10), palette, palette[0]),
		solidFrame(image.Rect(0, 0, 10, 10), palette, palette[1]),
		solidFrame(image.Rect(10, 0, 20, 10), palette, palette[2]),
	}
	u := storage.NewFileURI(writeTempFile(t, "previous.gif", buildGIF(t, 20, 10, frames, []int{1, 1, 1}, []byte{gif.DisposalNone, gif.DisposalPrevious, gif.DisposalNone})))
	p, err := LoadAnimatedPreviewContext(context.Background(), u, 10, 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Frames) != 3 {
		t.Fatalf("frames: %d", len(p.Frames))
	}
	for i, blue := range []bool{false, true, false} {
		r, _, b, _ := p.Frames[i].At(1, 1).RGBA()
		if (b > r) != blue || r+b == 0 {
			t.Fatalf("frame %d lost restored canvas: r=%d b=%d", i, r, b)
		}
	}
}

func TestAnimatedPreviewCancelled(t *testing.T) {
	u := storage.NewFileURI(writeTempFile(t, "cancelled.gif", uitest.EncodePartialFrameGIF(t)))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, err := LoadAnimatedPreviewContext(ctx, u, 512, 16<<20)
	if p != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled preview: %v, %v", p, err)
	}
}
