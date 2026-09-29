//go:build microsoftstore && !appleappstore

package distribution

import "testing"

func TestStoreManaged_StoreBuildIsTrue(t *testing.T) {
	if !StoreManaged {
		t.Error("Store build does not report Store-managed updates")
	}
	if AppleAppStore {
		t.Error("Microsoft Store build reports Apple distribution")
	}
}
