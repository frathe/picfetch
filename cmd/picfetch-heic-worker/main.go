// Command picfetch-heic-worker is the minimal disposable HEIC helper. It has no
// GUI, image-format registration, native HEIC codec, or user-facing CLI mode.
package main

import (
	"os"

	"github.com/frathe/picfetch/internal/heicdecode/worker"
)

func main() {
	os.Exit(worker.Main(os.Args[1:]))
}
