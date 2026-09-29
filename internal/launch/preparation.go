package launch

import (
	"context"
	"errors"
	"sync"

	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/locationtrial"
	"github.com/frathe/picfetch/internal/similarity"
)

// PreparationOptions supplies one launch's external acquisition operations.
// Nil operations use production behavior.
type PreparationOptions struct {
	VerifyOffline func(context.Context) error
	NewExplorer   func(string) (*explorertrial.Session, error)
	NewLocation   func(string) (*locationtrial.Recorder, error)
}

// Prepared owns any resources acquired for an isolated trial.
type Prepared struct {
	policy    Policy
	explorer  *explorertrial.Session
	location  *locationtrial.Recorder
	closeOnce sync.Once
	closeErr  error
}

// Prepare acquires one launch's trial resources before application construction.
func Prepare(ctx context.Context, policy Policy, options PreparationOptions) (*Prepared, error) {
	if !policy.Valid() {
		return nil, ErrInvalidPolicy
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	prepared := &Prepared{policy: policy}
	switch policy.Purpose() {
	case Ordinary:
		return prepared, nil
	case ExplorerTrial:
		verify := options.VerifyOffline
		if verify == nil {
			verify = similarity.VerifyOffline
		}
		if err := verify(ctx); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		newExplorer := options.NewExplorer
		if newExplorer == nil {
			newExplorer = explorertrial.New
		}
		var err error
		prepared.explorer, err = newExplorer(policy.TrialDir())
		if err != nil {
			return nil, errors.Join(err, prepared.Close())
		}
		if prepared.explorer == nil {
			return nil, errors.New("launch: Explorer trial acquisition returned no session")
		}
	case LocationMapTrial:
		newLocation := options.NewLocation
		if newLocation == nil {
			newLocation = locationtrial.New
		}
		var err error
		prepared.location, err = newLocation(policy.TrialDir())
		if err != nil {
			return nil, errors.Join(err, prepared.Close())
		}
		if prepared.location == nil {
			return nil, errors.New("launch: Location Map trial acquisition returned no recorder")
		}
	default:
		return nil, ErrInvalidPolicy
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, prepared.Close())
	}
	return prepared, nil
}

// Policy returns the captured, resource-free launch decision.
func (p *Prepared) Policy() Policy {
	if p == nil {
		return Policy{}
	}
	return p.policy
}

// ExplorerTrial lends the Explorer evidence session to the UI.
func (p *Prepared) ExplorerTrial() *explorertrial.Session {
	if p == nil {
		return nil
	}
	return p.explorer
}

// LocationMapTrial lends the Location Map recorder to the UI.
func (p *Prepared) LocationMapTrial() *locationtrial.Recorder {
	if p == nil {
		return nil
	}
	return p.location
}

// Close finalizes any trial evidence after its producers have stopped.
func (p *Prepared) Close() error {
	if p == nil {
		return nil
	}
	p.closeOnce.Do(func() {
		var explorerErr, locationErr error
		if p.explorer != nil {
			explorerErr = p.explorer.Close()
		}
		if p.location != nil {
			p.location.Stop()
			locationErr = p.location.Wait()
		}
		p.closeErr = errors.Join(explorerErr, locationErr)
	})
	return p.closeErr
}
