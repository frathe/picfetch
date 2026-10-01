package consolehelp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/launch"
)

func TestOneTurnExitsAndLeavesPortraitAboveHelp(t *testing.T) {
	var out strings.Builder
	start := time.Unix(0, 0)
	ticks := make(chan time.Time, 2)
	ticks <- start.Add(1500 * time.Millisecond)
	ticks <- start.Add(3 * time.Second)
	const usage = "Usage:\n  picfetch [flags]\n"
	if err := play(context.Background(), &out, usage, func() (int, int, error) { return 100, 50, nil }, ticks, start, nil); err != nil {
		t.Fatal(err)
	}
	width, height, text := layout(100, 50, usage)
	if !strings.Contains(out.String(), render(180, width, height)) {
		t.Fatal("turn never displayed the back of the sculpture")
	}
	want := "\x1b[?25h\x1b[?1049l" + render(0, width, height) + "\n" + wordmark(width) + "\n\n" + text
	if !strings.HasSuffix(out.String(), want) {
		t.Fatal("completed turn did not leave front portrait above help")
	}
}

func TestAnimationCancellationRestoresTerminal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var output strings.Builder
	out := callbackWriter{out: &output, after: cancel}
	const usage = "Usage:\n  picfetch [flags]\n"
	if err := animate(ctx, out, usage, func() (int, int, error) { return 100, 50, nil }, nil); err != nil {
		t.Fatal(err)
	}
	result := output.String()
	if !strings.Contains(result, "\x1b[?1049h") || !strings.Contains(result, usage) {
		t.Fatal("animation did not display artwork and help")
	}
	if !strings.HasSuffix(result, "\x1b[?25h\x1b[?1049l"+usage) {
		t.Fatal("cancel did not restore cursor/screen and leave ordinary help")
	}
}

type callbackWriter struct {
	out   *strings.Builder
	after func()
}

func (w callbackWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	w.after()
	return n, err
}

func TestSmallTerminalKeepsCompleteHelp(t *testing.T) {
	var out strings.Builder
	const usage = "Usage:\n  picfetch [flags]\n"
	if err := animate(context.Background(), &out, usage, func() (int, int, error) { return 20, 10, nil }, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != usage {
		t.Fatalf("small terminal must get unchanged help, got %q", out.String())
	}
}

func TestResizeAndOutputFailureRestoreTerminal(t *testing.T) {
	const usage = "Usage:\n  picfetch [flags]\n"
	start := time.Unix(0, 0)
	t.Run("resize", func(t *testing.T) {
		var out strings.Builder
		ticks := make(chan time.Time, 1)
		ticks <- start.Add(time.Second)
		calls := 0
		size := func() (int, int, error) {
			calls++
			if calls > 1 {
				return 40, 10, nil
			}
			return 100, 50, nil
		}
		if err := play(context.Background(), &out, usage, size, ticks, start, nil); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(out.String(), "\x1b[?25h\x1b[?1049l"+usage) {
			t.Fatal("shrinking the terminal lost help or terminal restoration")
		}
	})
	t.Run("partial_write", func(t *testing.T) {
		out := &partialWriter{}
		err := animate(context.Background(), out, usage, func() (int, int, error) { return 100, 50, nil }, nil)
		if !errors.Is(err, errOutput) || !strings.Contains(out.String(), "\x1b[?25h\x1b[?1049l") {
			t.Fatalf("partial write: error=%v, cleanup missing=%v", err, out.String())
		}
	})
}

var errOutput = errors.New("output failed")

type partialWriter struct{ strings.Builder }

func (w *partialWriter) Write(p []byte) (int, error) {
	if w.Len() == 0 {
		n, _ := w.Builder.Write(p[:5])
		return n, errOutput
	}
	return w.Builder.Write(p)
}

// Prevent io.WriteString from selecting the embedded Builder's WriteString.
func (w *partialWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func TestLayoutPreservesHelpAndFits(t *testing.T) {
	usage := "Usage:\n  picfetch [flags]\n\n  --sort=MODE  " + strings.Repeat("description ", 18) + "\n"
	width, height, wrapped := layout(80, 60, usage)
	if width == 0 || height < 10 {
		t.Fatal("large terminal unexpectedly declined animation")
	}
	if strings.Join(strings.Fields(wrapped), " ") != strings.Join(strings.Fields(usage), " ") {
		t.Fatal("wrapping changed help content")
	}
	for _, line := range strings.Split(wrapped, "\n") {
		if len(line) >= 80 {
			t.Fatalf("help line reaches terminal wrap column: %q", line)
		}
	}
	if height+strings.Count(wrapped, "\n")+9 > 60 {
		t.Fatal("art, branding and help exceed the viewport")
	}
}

func TestWritePlainHelp(t *testing.T) {
	var out strings.Builder
	const usage = "Usage:\n  picfetch [flags]\n"
	if err := Write(&out, usage); err != nil || out.String() != usage {
		t.Fatalf("Write = %q, %v; want unchanged usage", out.String(), err)
	}
	want := errors.New("closed output")
	if err := Write(failingWriter{want}, usage); !errors.Is(err, want) {
		t.Fatalf("Write error = %v; want %v", err, want)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write(_ []byte) (int, error) { return 0, w.err }

func TestTurntable(t *testing.T) {
	front := render(0, 80, 32)
	if front == "" || strings.TrimSpace(front) == "" {
		t.Fatal("front view has no sculpture")
	}
	if front != render(360, 80, 32) {
		t.Fatal("one full turn must return to exactly the first frame")
	}
	views := make(map[string]bool)
	for _, angle := range []float64{0, 45, 90, 180, 270} {
		frame := render(angle, 80, 32)
		rows := strings.Split(frame, "\n")
		if len(rows) != 32 {
			t.Fatalf("angle %v: got %d rows", angle, len(rows))
		}
		for _, row := range rows {
			if len(row) != 80 {
				t.Fatalf("angle %v: row width %d", angle, len(row))
			}
			for _, c := range row {
				if c < ' ' || c > '~' {
					t.Fatalf("art contains non-ASCII character %q", c)
				}
			}
		}
		views[frame] = true
	}
	if len(views) != 5 {
		t.Fatal("front, angled, side and back views must be distinct")
	}
}

func TestAnimationSignalsRestoreTerminal(t *testing.T) {
	for _, notice := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(notice.String(), func(t *testing.T) {
			var out strings.Builder
			notices := make(chan os.Signal, 1)
			notices <- notice
			const usage = "Usage: picfetch\n"
			err := play(context.Background(), &out, usage, func() (int, int, error) {
				return 100, 50, nil
			}, nil, time.Unix(0, 0), notices)
			if (err == nil) != (notice == os.Interrupt) {
				t.Fatalf("signal %v returned %v; only Ctrl+C is successful", notice, err)
			}
			if !strings.HasSuffix(out.String(), "\x1b[?25h\x1b[?1049l"+usage) {
				t.Fatal("signal did not restore cursor/screen and ordinary help")
			}
		})
	}
}

func TestAnimationSignalsDuringRestoration(t *testing.T) {
	for _, tc := range []struct {
		name    string
		notices []os.Signal
		failed  bool
	}{
		{name: "interrupt_skips", notices: []os.Signal{os.Interrupt}},
		{name: "termination_fails", notices: []os.Signal{syscall.SIGTERM}, failed: true},
		{name: "interrupt_then_termination_fails", notices: []os.Signal{os.Interrupt, syscall.SIGTERM}, failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output strings.Builder
			notices := make(chan os.Signal, len(tc.notices))
			out := callbackWriter{out: &output, after: func() {
				if strings.Contains(output.String(), "\x1b[?1049l") {
					for _, notice := range tc.notices {
						notices <- notice
					}
				}
			}}
			start := time.Unix(0, 0)
			ticks := make(chan time.Time, 1)
			ticks <- start.Add(turnDuration)
			const usage = "Usage: picfetch\n"
			err := play(context.Background(), out, usage, func() (int, int, error) {
				return 100, 50, nil
			}, ticks, start, notices)
			if (err != nil) != tc.failed {
				t.Fatalf("cleanup signals %v: error=%v; want failed=%v", tc.notices, err, tc.failed)
			}
			if !strings.Contains(output.String(), "\x1b[?25h\x1b[?1049l") || !strings.HasSuffix(output.String(), usage) {
				t.Fatal("cleanup signal lost restored cursor/screen or final help")
			}
		})
	}
}

func TestResolutionPicker(t *testing.T) {
	for _, tc := range []struct {
		keys      string
		index     int
		cancelled bool
	}{
		{keys: "\r", index: 0}, {keys: "\x1b[B\n", index: 1},
		{keys: "\x1b[B\x1b[B\r", index: 2}, {keys: "\x1b[A\r", index: 3},
		{keys: "\x1bOB\x1bOA\r", index: 0}, {keys: "\x03", cancelled: true},
		{keys: "\x04", cancelled: true},
	} {
		t.Run(fmt.Sprintf("%q", tc.keys), func(t *testing.T) {
			var out bytes.Buffer
			input := strings.NewReader(tc.keys)
			size, err := pickResolution(&out, func() (byte, bool, error) { b, err := input.ReadByte(); return b, err == nil, err }, nil)
			if tc.cancelled {
				if err == nil {
					t.Fatal("cancelled picker launched")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if want := launch.ScreenshotResolutions()[tc.index]; size != want {
				t.Fatalf("size=%v want=%v", size, want)
			}
			for _, label := range []string{"1280 x 800px", "1440 x 900px", "2560 x 1600px", "2880 x 1800px"} {
				if !strings.Contains(out.String(), label) {
					t.Fatalf("missing %s", label)
				}
			}
		})
	}
}

func TestResolutionPickerDoesNotLoseSignalAfterSelection(t *testing.T) {
	var out bytes.Buffer
	notices := make(chan os.Signal, 1)
	_, err := pickResolution(&out, func() (byte, bool, error) { notices <- syscall.SIGTERM; return '\r', true, nil }, notices)
	if err == nil {
		t.Fatal("Return swallowed an arriving SIGTERM and allowed GUI startup")
	}
}

func TestResolutionSignalDuringCleanupPreventsStartup(t *testing.T) {
	for _, notice := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		notices := make(chan os.Signal, 1)
		notices <- notice
		if err := finishResolutionSignals(notices); err == nil {
			t.Fatalf("cleanup swallowed %v", notice)
		}
		if err := finishResolutionSignals(notices); err != nil {
			t.Fatalf("already-consumed signal remained: %v", err)
		}
	}
}
