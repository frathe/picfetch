//go:build darwin && appleappstore && !microsoftstore

package distribution_test

import (
	"testing"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/launch"
)

func TestAppleStoreLaunchRefusesSelfUpdates(t *testing.T) {
	if !distribution.AppleAppStore {
		t.Fatal("Apple build does not identify its store")
	}
	policy, err := launch.NewPolicy(launch.Options{}, "io.github.frathe.picfetch", distribution.StoreManaged)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Updates().Allowed() {
		t.Fatal("Mac App Store launch admits GitHub self-updates")
	}
}
