// Command macstorestage verifies and extracts a pinned ONNX archive before signing.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/frathe/picfetch/internal/similarity"
)

func main() {
	arch := flag.String("arch", "", "Go architecture: arm64 or amd64")
	archive := flag.String("archive", "", "pinned upstream ONNX archive")
	out := flag.String("out", "", "new staging directory")
	flag.Parse()
	if err := similarity.StageMacRuntime(context.Background(), *arch, *archive, *out); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "macstorestage: %v\n", err)
		os.Exit(1)
	}
}
