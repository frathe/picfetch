package grid

import (
	"testing"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestSetUIQueue_NilRestoresFyneQueue(t *testing.T) {
	g := newOverview(t, hostWith(t, "a.jpg"))
	g.SetUIQueue(nil)
	if _, ok := g.ui.(fyneQueue); !ok {
		t.Fatalf("SetUIQueue(nil) ui type = %T, want fyneQueue", g.ui)
	}
	g.SetUIQueue(&uitest.UIQueue{})
	if _, ok := g.ui.(*uitest.UIQueue); !ok {
		t.Fatalf("SetUIQueue(&uitest.UIQueue{}) ui type = %T, want *uitest.UIQueue", g.ui)
	}
}
