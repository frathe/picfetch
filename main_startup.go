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
	identity         func(context.Context, launch.Options) (string, error)
	newApp           func(string) (fyne.App, error)
	run              func(fyne.App, []fyne.URI, launch.Options) error
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
	// A relaunch waits for its predecessor before opening app preferences.
	cleanupLaunchPredecessor(opts, ops.cleanup)
	identity, err := ops.identity(context.Background(), opts)
	if err != nil {
		return 1, err
	}
	application, err := ops.newApp(identity)
	if err != nil {
		return 1, err
	}
	if err := ops.run(application, argsToURIs(paths), opts); err != nil {
		return 1, err
	}
	return 0, nil
}
