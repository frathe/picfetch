package explorer_test

import (
	"context"
	"image/color"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFeatureSuspendPreservesFrozenSearchOrigin(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	queue := &uitest.UIQueue{}
	published := make(chan struct{})
	preview := uitest.EncodeJPEG(t, 32, 24, color.White)
	f := explorer.NewFeature(host, explorer.Options{App: app, Queue: queue, Analyze: func(ctx context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		emit(similarity.Event{Total: 1, Successful: 1, Items: []similarity.Item{{Path: "/a.jpg", Cohort: "group", Preview: preview}}})
		close(published)
		<-ctx.Done()
		emit(similarity.Event{Total: 1, Complete: true})
		return ctx.Err()
	}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	f.Open(explorer.OpenRequest{Sources: []string{"/a.jpg"}})
	<-published
	queue.Drain()
	if !f.State().HasMap {
		t.Fatal("premise: map was not presented")
	}
	done := f.Suspend()
	if f.State().SessionCurrent {
		t.Fatal("suspension left worker delivery current")
	}
	<-done
	f.Settle()
	if !f.State().HasMap || !f.State().HasSources || f.State().SessionCurrent {
		t.Fatalf("suspension lost origin or retained delivery: %+v", f.State())
	}
}
