package ui

import (
	"context"

	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
)

// imageServices owns one admission lane for this viewer and its analysis
// descendants. Readers are immutable dependencies installed before any work.
type imageServices struct {
	owner                  *heicclient.Client
	foreground, background imaging.Reader
	startupError           error
}

func installedImageServices(prefs preferences.State, executable, privateDir string) imageServices {
	if !prefs.ExperimentalHEIC {
		return imageServices{}
	}
	owner, err := heicclient.OpenInstalled(context.Background(), executable, privateDir, heicdecode.DefaultLimits(int64(prefs.MaxFileSizeMB)*1024*1024))
	if err != nil {
		return imageServices{startupError: err}
	}
	return newImageServices(owner)
}

func (s imageServices) unavailable() bool {
	return s.startupError != nil || (s.owner != nil && s.owner.Unavailable())
}

func newImageServices(owner *heicclient.Client) imageServices {
	if owner == nil {
		return imageServices{}
	}
	return imageServices{owner: owner, foreground: imaging.NewReader(owner.Do), background: imaging.NewReader(func(ctx context.Context, op heicdecode.Operation, input heicclient.Input) (heicdecode.Response, error) {
		return owner.DoWithPriority(ctx, heicclient.Background, op, input)
	})}
}

func (s imageServices) Stop() {
	if s.owner != nil {
		s.owner.Stop()
	}
}
func (s imageServices) Wait() {
	if s.owner != nil {
		s.owner.Wait()
	}
}
