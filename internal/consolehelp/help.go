// Package consolehelp presents terminal-only ASCII artwork above command help.
package consolehelp

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// Write prints usage, animating Trane when the output terminal has room.
func Write(out io.Writer, usage string) error {
	file, ok := out.(*os.File)
	if ok && os.Getenv("TERM") != "dumb" && term.IsTerminal(int(file.Fd())) {
		restore, err := enableANSI(file)
		if err == nil {
			defer restore()
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return animate(ctx, out, usage, func() (int, int, error) {
				return term.GetSize(int(file.Fd()))
			})
		}
	}
	_, err := io.WriteString(out, usage)
	return err
}

const turnDuration = 3 * time.Second

func animate(ctx context.Context, out io.Writer, usage string, size func() (int, int, error)) error {
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	return play(ctx, out, usage, size, ticker.C, time.Now())
}

func play(ctx context.Context, out io.Writer, usage string, size func() (int, int, error), ticks <-chan time.Time, start time.Time) (err error) {
	active := false
	final := usage
	defer func() {
		if active {
			_, restoreErr := io.WriteString(out, "\x1b[?25h\x1b[?1049l"+final)
			err = errors.Join(err, restoreErr)
		}
	}()
	elapsed := time.Duration(0)
	lastColumns, lastRows := 0, 0
	for {
		columns, rows, sizeErr := size()
		width, height, text := layout(columns, rows, usage)
		if sizeErr != nil || height == 0 {
			if !active {
				_, err = io.WriteString(out, usage)
			}
			return err
		}
		var frame strings.Builder
		if !active {
			// Mark active before the write, so even a partial write gets cleanup.
			active = true
			frame.WriteString("\x1b[?1049h\x1b[?25l")
		}
		resized := columns != lastColumns || rows != lastRows
		if resized {
			frame.WriteString("\x1b[2J")
		}
		frame.WriteString("\x1b[H")
		progress := elapsed.Seconds() / turnDuration.Seconds()
		// Smooth acceleration and deceleration settle the full turn at the front.
		angle := 360 * progress * progress * (3 - 2*progress)
		frame.WriteString(render(angle, width, height))
		if resized {
			frame.WriteString("\n\n" + centered("Trane / Ctrl+C to skip", width) + "\n\n\n\n\n" + text)
		}
		if _, err = io.WriteString(out, frame.String()); err != nil {
			return err
		}
		lastColumns, lastRows = columns, rows
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticks:
			elapsed = now.Sub(start)
			if elapsed >= turnDuration {
				final = render(0, width, height) + "\n" + wordmark(width) + "\n\n" + text
				return nil
			}
		}
	}
}

// Keep a spare column/row to avoid automatic wrapping or terminal scrolling.
// If artwork cannot fit, ordinary help remains usable in scrollback and pipes.
func layout(columns, rows int, usage string) (width, height int, text string) {
	width = min(columns-1, 120)
	if width < 48 || rows < 16 {
		return 0, 0, usage
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSuffix(usage, "\n"), "\n") {
		for len(line) > width {
			cut := strings.LastIndexByte(line[:width+1], ' ')
			if cut <= 4 {
				cut = width
			}
			lines = append(lines, line[:cut])
			line = "    " + strings.TrimLeft(line[cut:], " ")
		}
		lines = append(lines, line)
	}
	height = min(32, rows-len(lines)-9)
	if height < 10 {
		return 0, 0, usage
	}
	return width, height, strings.Join(lines, "\n") + "\n"
}

func centered(text string, width int) string {
	return strings.Repeat(" ", max(0, (width-len(text))/2)) + text
}

func wordmark(width int) string {
	// Hand-lettered ASCII PicFetch, kept small enough for an ordinary terminal.
	const lettering = ` ____  _      _____    _       _
|  _ \(_) ___|  ___|__| |_ ___| |__
| |_) | |/ __| |_ / _ \ __/ __| '_ \
|  __/| | (__|  _|  __/ || (__| | | |
|_|   |_|\___|_|  \___|\__\___|_| |_|`
	lines := strings.Split(lettering, "\n")
	indent := strings.Repeat(" ", max(0, (width-36)/2))
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}
