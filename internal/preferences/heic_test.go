package preferences

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/heic"
)

func TestHEICCapabilityPreferences(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	if got := LoadHEICObservation(app); got != (heic.Observation{}) {
		t.Fatalf("fresh observation = %+v", got)
	}
	for _, available := range []bool{true, false} {
		want := heic.Observation{
			Identity:  heic.Identity{OS: "linux", Architecture: "amd64", OSVersion: "6.8", Revision: heic.Revision},
			CheckedAt: time.Unix(100, 0).UTC(), Available: available,
		}
		SaveHEICObservation(app, want)
		Save(app, State{})
		if got := LoadHEICObservation(app); got != want {
			t.Fatalf("round trip after stale settings save = %+v, want %+v", got, want)
		}
		SaveHEICObservation(app, heic.Observation{})
		if got := LoadHEICObservation(app); got != want {
			t.Fatalf("incomplete check overwrote observation: %+v", got)
		}
	}
}
