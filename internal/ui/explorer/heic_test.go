package explorer_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

type heicBackend struct{}

func (heicBackend) Check(_ context.Context) error { return nil }
func (heicBackend) Read(_ context.Context, _ []byte, _ heic.Request) (heic.Result, error) {
	return heic.Result{}, heic.ErrUnsupported
}

func TestHEICExplorerCapturesAdmission(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	capability := heic.NewCapability(heicBackend{}, heic.Identity{}, heic.Observation{})
	t.Cleanup(func() { capability.Stop(); capability.Wait() })
	<-capability.Check(context.Background())
	entered := make(chan context.Context, 2)
	provider := func(ctx context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
		entered <- ctx
		<-ctx.Done()
		return ctx.Err()
	}
	f := explorer.NewFeature(&featureHost{win: app.NewWindow("Explorer")}, explorer.Options{Analyze: provider, Queue: &uitest.UIQueue{}})
	f.SetHEICCapability(capability)
	t.Cleanup(func() { f.Stop(); f.Settle() })
	before := make(chan struct{})
	f.WaitBefore(before)
	if !f.Open(explorer.OpenRequest{Sources: []string{"/photo.heic"}}) {
		t.Fatal("analysis rejected")
	}
	<-capability.Check(context.Background())
	close(before)
	first := <-entered
	if got := heic.FromContext(first); !got.Available || got.Backend == nil || got.Generation != 1 {
		t.Fatalf("analysis did not capture before waiting for its predecessor: %+v", got)
	}
	<-capability.Check(context.Background())
	if got := heic.FromContext(first); got.Generation != 1 {
		t.Fatalf("support recheck mutated active analysis: %+v", got)
	}
	f.Close()
	f.Settle()
	if first.Err() == nil {
		t.Fatal("close did not cancel captured analysis")
	}
	if !f.Open(explorer.OpenRequest{Sources: []string{"/photo.heic"}}) {
		t.Fatal("new analysis rejected")
	}
	if got := heic.FromContext(<-entered); !got.Available || got.Generation != 3 {
		t.Fatalf("later analysis did not capture new support: %+v", got)
	}
}
