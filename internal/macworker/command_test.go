package macworker

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCommandUsesOnlyBundledBrokerAndMode(t *testing.T) {
	for _, mode := range []Mode{HEIC, Similarity} {
		cmd := Command(context.Background(), "/Applications/PicFetch.app/Contents/MacOS/picfetch", mode)
		want := []string{"/Applications/PicFetch.app/Contents/MacOS/picfetch-worker-client", string(mode)}
		if !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("arguments = %v, want %v", cmd.Args, want)
		}
		if cmd.Path != filepath.Join("/Applications/PicFetch.app/Contents/MacOS", "picfetch-worker-client") {
			t.Fatal(cmd.Path)
		}
	}
}
