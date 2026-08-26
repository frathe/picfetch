package update

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

func VerifyHash(data []byte, wantHex string) error {
	sum := sha256.Sum256(data)
	want, err := hex.DecodeString(strings.TrimSpace(wantHex))
	if err != nil {
		return fmt.Errorf("invalid checksum: %w", err)
	}
	if len(want) != sha256.Size || subtle.ConstantTimeCompare(sum[:], want) != 1 {
		return fmt.Errorf("checksum mismatch")
	}
	return nil
}
