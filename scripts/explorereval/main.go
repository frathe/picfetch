// Command explorereval evaluates a local content-similarity pipeline before
// its integration into PicFetch. See README.md for setup and evidence.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/frathe/picfetch/internal/similarity"
)

func main() {
	if similarity.WorkerMain() {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	flags := flag.NewFlagSet("explorereval", flag.ContinueOnError)
	flags.SetOutput(output)
	assets := flags.String("assets", ".scratch/visual-similarity-explorer/assets", "directory of pinned local model/runtime assets")
	library := flags.String("library", ".scratch/visual-similarity-explorer/demo", "local input folder")
	out := flags.String("out", ".scratch/visual-similarity-explorer/evidence/run", "new local evidence directory (must not exist)")
	worker := flags.Bool("worker", false, "internal: process within the offline sandbox")
	trial := flags.String("trial", "smoke", "bounded smoke or production throughput trial")
	automatic := flags.Bool("automatic", false, "throughput: publish a map every 30 processed sources")
	provider := flags.String("provider", "cpu", "cpu or coreml execution provider")
	probe := flags.Bool("probe", false, "verify actual TCP/UDP denial, without reading images")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *probe {
		return similarity.VerifyOffline(ctx)
	}
	if *trial != "smoke" && *trial != "throughput" {
		return fmt.Errorf("only smoke and throughput trials are supported; full-library qualification remains pending")
	}
	if *trial == "throughput" && (*provider != "cpu" || *worker) {
		return fmt.Errorf("throughput uses the production CPU client and its own offline worker")
	}
	if *automatic && *trial != "throughput" {
		return fmt.Errorf("automatic publication is only configurable for throughput")
	}
	if *provider != "cpu" && *provider != "coreml" {
		return fmt.Errorf("provider must be cpu or coreml")
	}
	if err := similarity.VerifyAssets(ctx, *assets); err != nil {
		return err
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("this experiment requires an Apple Silicon Mac")
	}
	config := configuration{Assets: *assets, Library: *library, Out: *out, Provider: *provider}
	for _, path := range []*string{&config.Assets, &config.Library, &config.Out} {
		absolute, err := filepath.Abs(*path)
		if err != nil {
			return err
		}
		*path = absolute
	}
	if *trial == "throughput" {
		return profile(ctx, config, *automatic, output)
	}
	if *worker {
		return evaluate(ctx, config, output)
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1) (allow default) (deny network*)", executable,
		"-worker", "-assets", config.Assets, "-library", config.Library, "-out", config.Out, "-provider", config.Provider)
	cmd.Stdout, cmd.Stderr = output, output
	err = cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
