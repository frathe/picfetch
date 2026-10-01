package main

import (
	"context"
	"errors"
	"io"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/launch"
)

// startupOps contains external desktop operations for one invocation. The
// entry point and tests share runStartup; no process-wide switches are needed.
type startupOps struct {
	heicWorker       func() bool
	similarityWorker func() bool
	selectResolution func(io.Writer) (launch.Resolution, error)
	installOpenWith  func()
	cleanup          func()
	capture          func(launch.Options) (launch.Policy, error)
	prepare          func(context.Context, launch.Policy) (*launch.Prepared, error)
	newApp           func(string) (fyne.App, error)
	run              func(fyne.App, []fyne.URI, launch.Options, *launch.Prepared) error
}

func runStartup(args []string, stdout, stderr io.Writer, ops startupOps) (code int, err error) {
	paths, opts, exit := launchArgs(args, stdout, stderr)
	if exit >= 0 {
		return exit, nil
	}
	if ops.heicWorker() || ops.similarityWorker() {
		return 0, nil
	}
	if opts.FixedSizeMode && opts.FixedSize == nil {
		selected, selectionErr := ops.selectResolution(stdout)
		if selectionErr != nil {
			return 1, selectionErr
		}
		opts.FixedSize = &selected
	}
	// The native delegate must be installed before Fyne/GLFW initialization.
	ops.installOpenWith()
	policy, err := ops.capture(opts)
	if err != nil {
		return 1, err
	}
	if !policy.Valid() {
		return 1, launch.ErrInvalidPolicy
	}
	prepared, err := ops.prepare(context.Background(), policy)
	// The entry point retains ownership even when construction or Run fails.
	// Run joins evidence producers before returning; Close then flushes evidence
	// before main can call os.Exit, preserving both startup and cleanup errors.
	defer func() {
		if closeErr := prepared.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
			code = 1
		}
	}()
	if err != nil {
		return 1, err
	}
	if prepared.Policy() != policy {
		return 1, launch.ErrInvalidPolicy
	}
	// A relaunch waits for its predecessor before opening app preferences.
	cleanupLaunchPredecessor(policy, ops.cleanup)
	application, err := ops.newApp(policy.ApplicationID())
	if err != nil {
		return 1, err
	}
	if err := ops.run(application, argsToURIs(paths), opts, prepared); err != nil {
		return 1, err
	}
	return 0, nil
}
