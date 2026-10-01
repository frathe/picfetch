package consolehelp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"github.com/frathe/picfetch/internal/launch"
)

// SelectResolution offers a keyboard picker before creating the desktop app.
// A missing terminal is an error; scripts can supply an explicit flag value.
func SelectResolution(in *os.File, out io.Writer) (size launch.Resolution, err error) {
	output, ok := out.(*os.File)
	if !ok || !term.IsTerminal(int(in.Fd())) || !term.IsTerminal(int(output.Fd())) || os.Getenv("TERM") == "dumb" {
		return size, errors.New("fixed-size-mode needs an interactive terminal; use -fixed-size-mode=1280x800 for a scripted launch")
	}
	// Cover setup and cleanup too: default signal handling must never terminate
	// the process while its terminal is still raw.
	notices := make(chan os.Signal, 2)
	signal.Notify(notices, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(notices)
	restoreANSI, err := enableANSI(output)
	if err != nil {
		return size, err
	}
	defer restoreANSI()
	state, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return size, err
	}
	defer func() { err = errors.Join(err, term.Restore(int(in.Fd()), state)) }()
	defer func() {
		_, restoreErr := io.WriteString(out, "\x1b[?25h\x1b[?1049l")
		err = errors.Join(err, restoreErr)
	}()
	if _, err = io.WriteString(out, "\x1b[?1049h\x1b[?25l"); err != nil {
		return size, err
	}
	size, err = pickResolution(out, func() (byte, bool, error) { return readTerminalByte(in) }, notices)
	return size, err
}

func pickResolution(out io.Writer, read func() (byte, bool, error), notices <-chan os.Signal) (launch.Resolution, error) {
	sizes := launch.ScreenshotResolutions()
	selected, escape := 0, 0
	draw := func() error {
		if _, err := io.WriteString(out, "\x1b[H\x1b[2JChoose screenshot size (Up/Down, Return; Ctrl+C cancels):\r\n\r\n"); err != nil {
			return err
		}
		for i, size := range sizes {
			marker := "  "
			if i == selected {
				marker = "> "
			}
			if _, err := fmt.Fprintf(out, "%s%d x %dpx\r\n", marker, size.Width, size.Height); err != nil {
				return err
			}
		}
		return nil
	}
	if err := draw(); err != nil {
		return launch.Resolution{}, err
	}
	for {
		select {
		case <-notices:
			return launch.Resolution{}, errors.New("fixed-size selection interrupted")
		default:
		}
		key, ready, err := read()
		if err != nil {
			return launch.Resolution{}, fmt.Errorf("fixed-size selection: %w", err)
		}
		if !ready {
			continue
		}
		if key == 3 || key == 4 {
			return launch.Resolution{}, errors.New("fixed-size selection cancelled")
		}
		if escape == 1 {
			if key == '[' || key == 'O' {
				escape = 2
				continue
			}
			escape = 0
		} else if escape == 2 {
			escape = 0
			switch key {
			case 'A':
				selected = (selected + len(sizes) - 1) % len(sizes)
			case 'B':
				selected = (selected + 1) % len(sizes)
			default:
				continue
			}
			if err := draw(); err != nil {
				return launch.Resolution{}, err
			}
			continue
		}
		switch key {
		case 27:
			escape = 1
		case '\r', '\n':
			return sizes[selected], nil
		}
	}
}
