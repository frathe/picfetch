package heicdecode

import (
	"bytes"
	"errors"
	"testing"
)

func TestRequestFraming(t *testing.T) {
	limits := DefaultLimits(4)
	var encoded bytes.Buffer
	if err := WriteRequest(&encoded, Request{Operation: Decode, Input: []byte("data")}, limits); err != nil {
		t.Fatal(err)
	}
	valid := append([]byte(nil), encoded.Bytes()...)
	got, err := ReadRequest(bytes.NewReader(valid), limits)
	if err != nil || got.Operation != Decode || string(got.Input) != "data" {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
	for _, data := range [][]byte{valid[:len(valid)-1], append(append([]byte(nil), valid...), 0), replaceUint16(valid, 6, 99), replaceUint64(valid, 8, 5)} {
		if _, err := ReadRequest(bytes.NewReader(data), limits); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("invalid request accepted: %v", err)
		}
	}
	encoded.Reset()
	if err := WriteRequest(&encoded, Request{Operation: Decode, Input: []byte("excess")}, limits); !errors.Is(err, ErrInvalidRequest) || encoded.Len() != 0 {
		t.Fatalf("oversized request wrote %d bytes: %v", encoded.Len(), err)
	}
}
