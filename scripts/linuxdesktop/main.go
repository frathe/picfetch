// Command linuxdesktop stages Linux desktop integration files without installing
// them or changing the host's MIME associations.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("linuxdesktop", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "repository root containing FyneApp.toml")
	executable := flags.String("executable", "", "bundled executable basename")
	out := flags.String("out", "", "desktop integration staging directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *out == "" || *executable == "" {
		return fmt.Errorf("usage: linuxdesktop -root <repository> -executable <basename> -out <directory>")
	}
	return stage(*root, *executable, *out)
}
