package ui

import (
	"context"

	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
)

// imageServices owns one admission lane for this viewer and its analysis
// descendants. Readers are immutable dependencies installed before any work.
type imageServices struct {
	owner                  *heicclient.Client
	foreground, background imaging.Reader
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
