// Command storepublish prepares and submits validated Microsoft Store updates.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type runtime struct {
	HTTP      *http.Client
	Now       func() time.Time
	Wait      func(context.Context, time.Duration) error
	Git       gitRunner
	GitHubURL string
	StoreURL  string
	TokenURL  string
	Scratch   string

	Out io.Writer
	Env func(string) string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 12*time.Minute)
	defer cancel()
	if err := run(ctx, os.Args[1:], runtime{Out: os.Stdout, Env: os.Getenv}); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, rt runtime) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: storepublish record|prepare|preview|check|submit|reconcile")
	}
	if rt.Out == nil {
		rt.Out = io.Discard
	}
	if rt.Env == nil {
		rt.Env = func(_ string) string { return "" }
	}
	rt = normalizeRuntime(rt)
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "trusted repository checkout")
	bundle := flags.String("bundle", "", "validated bundle")
	wack := flags.String("wack", "", "WACK report")
	out := flags.String("out", "", "record path")
	candidate := flags.String("candidate", "", "offline release record")
	snapshot := flags.String("snapshot", "", "offline published submission")
	tag := flags.String("tag", "", "specific stable release")
	state := flags.String("state-dir", os.TempDir(), "local exclusive-claim directory")
	mode := flags.String("mode", "submit", "operation to prepare for approval")
	runID := flags.Int64("run-id", 0, "specific successful producer run")
	approvalFile := flags.String("approval", "", "frozen approval artifact")
	approvalSHA := flags.String("approval-sha256", "", "approval digest from preparation job")
	if err := flags.Parse(args[1:]); err != nil {
		return fmt.Errorf("invalid command arguments")
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *tag != "" {
		if !strings.HasPrefix(*tag, "v") {
			return fmt.Errorf("tag must be vMAJOR.MINOR.PATCH")
		}
		if _, err := versionParts(strings.TrimPrefix(*tag, "v")); err != nil {
			return err
		}
	}
	if args[0] == "record" {
		return recordRelease(*root, *bundle, *wack, *out, rt.Env)
	}
	scratch, err := os.MkdirTemp("", "picfetch-store-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	rt.Scratch = scratch
	o := options{root: *root, tag: *tag, candidate: *candidate, snapshot: *snapshot, stateDir: *state, runID: *runID}
	switch args[0] {
	case "prepare":
		return prepareApproval(ctx, rt, o, *mode, *out)
	case "preview":
		return previewRelease(ctx, rt, o)
	case "check":
		return checkStore(ctx, rt)
	case "submit", "reconcile":
		p, err := loadApproval(*approvalFile, *approvalSHA, args[0])
		if err != nil {
			return err
		}
		if (o.tag != "" && o.tag != p.Release.Tag) || o.runID != 0 {
			return fmt.Errorf("submission selection must match its frozen approval")
		}
		o.tag = p.Release.Tag
		return drive(ctx, rt, o, p)
	default:
		return fmt.Errorf("unknown Store command")
	}
}
