package update

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyHash(t *testing.T) {
	sum := sha256.Sum256([]byte("hello"))
	hexSum := hex.EncodeToString(sum[:])
	if err := VerifyHash([]byte("hello"), hexSum); err != nil {
		t.Fatal(err)
	}
	if err := VerifyHash([]byte("hello"), "00"+hexSum[2:]); err == nil {
		t.Fatal("want mismatch error")
	}
}

func TestVerifyHash_InvalidExpected(t *testing.T) {
	if err := VerifyHash([]byte("hello"), "zz"); err == nil {
		t.Fatal("want error for non-hex expected digest")
	}
	if err := VerifyHash([]byte("hello"), "abcd"); err == nil {
		t.Fatal("want error for wrong-length expected digest")
	}
}
