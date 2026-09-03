// Command msixstage prepares one architecture-specific directory for
// MakeAppx.exe. It deliberately stops before creating or signing an MSIX: the
// Windows SDK owns package-schema validation and GitHub Actions runs it on a
// Windows host.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var opts stageOptions
	flag.StringVar(&opts.Root, "root", ".", "repository root containing FyneApp.toml and assets/appIcon.png")
	flag.StringVar(&opts.Arch, "arch", "", "Go architecture: amd64 or arm64")
	flag.StringVar(&opts.Executable, "exe", "", "Windows executable to stage")
	flag.StringVar(&opts.Out, "out", "", "output staging directory")
	flag.Parse()

	if err := stage(opts); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "msixstage: %v\n", err)
		os.Exit(1)
	}
}
