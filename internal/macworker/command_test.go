package macworker

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCommandUsesOnlyBundledBrokerAndMode(t *testing.T) {
	for _, mode := range []Mode{HEIC, Similarity} {
		executable := filepath.Join(t.TempDir(), "PicFetch.app", "Contents", "MacOS", "picfetch")
		cmd := Command(context.Background(), executable, mode)
		broker := filepath.Join(filepath.Dir(executable), "picfetch-worker-client")
		want := []string{broker, string(mode)}
		if !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("arguments = %v, want %v", cmd.Args, want)
		}
		if cmd.Path != broker {
			t.Fatal(cmd.Path)
		}
	}
}
