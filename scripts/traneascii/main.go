// Command traneascii previews the same animated help as picfetch --help,
// without starting or linking the desktop application.
package main

import (
	"fmt"
	"os"

	"github.com/frathe/picfetch/internal/consolehelp"
	"github.com/frathe/picfetch/internal/launch"
)

func main() {
	if err := consolehelp.Write(os.Stdout, launch.Usage()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
