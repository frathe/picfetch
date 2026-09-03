//go:build microsoftstore

package distribution

import "testing"

func TestStoreManaged_MicrosoftStoreBuildIsTrue(t *testing.T) {
	if !StoreManaged {
		t.Error("microsoftstore build does not report Store-managed updates")
	}
}
