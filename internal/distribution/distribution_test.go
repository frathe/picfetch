//go:build !microsoftstore

package distribution

import "testing"

func TestStoreManaged_DefaultBuildIsFalse(t *testing.T) {
	if StoreManaged {
		t.Error("ordinary build reports Store-managed updates")
	}
}
