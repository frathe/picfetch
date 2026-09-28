package main

import (
	"context"
	"io"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/launch"
)

// startupOps contains external desktop operations for one invocation. The
// entry point and tests share runStartup; no process-wide switches are needed.
type startupOps struct {
	heicWorker       func() bool
	similarityWorker func() bool
	installOpenWith  func()
	cleanup          func()
	capture          func(launch.Options) (launch.Policy, error)
	prerequisites    func(context.Context, launch.Policy) error
	newApp           func(string) (fyne.App, error)
	run              func(fyne.App, []fyne.URI, launch.Options, launch.Policy) error
}

func runStartup(args []string, stdout, stderr io.Writer, ops startupOps) (int, error) {
	paths, opts, exit := launchArgs(args, stdout, stderr)
	if exit >= 0 {
		return exit, nil
	}
	if ops.heicWorker() || ops.similarityWorker() {
		return 0, nil
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
	if err := ops.prerequisites(context.Background(), policy); err != nil {
		return 1, err
	}
	// A relaunch waits for its predecessor before opening app preferences.
	cleanupLaunchPredecessor(policy, ops.cleanup)
	application, err := ops.newApp(policy.ApplicationID())
	if err != nil {
		return 1, err
	}
	if err := ops.run(application, argsToURIs(paths), opts, policy); err != nil {
		return 1, err
	}
	return 0, nil
}
