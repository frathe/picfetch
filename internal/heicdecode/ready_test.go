package heicdecode

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestReadinessReportsActualMemoryControl(t *testing.T) {
	for _, native := range []int64{0, 2 * 1024 * 1024 * 1024} {
		var stream bytes.Buffer
		want := Ready{WASMMemoryBytes: 1024 * 1024 * 1024, NativeMemoryBytes: native}
		if err := WriteReady(&stream, want); err != nil {
			t.Fatal(err)
		}
		stream.WriteString("following response")
		got, err := ReadReady(&stream)
		if err != nil || got != want || stream.String() != "following response" {
			t.Fatalf("readiness = %+v, %v, remaining=%q", got, err, stream.String())
		}
	}
	for _, memory := range []uint64{0, 1, 1024*1024*1024 + 65536, 1 << 63} {
		var frame [24]byte
		copy(frame[:8], "PHREADY2")
		binary.LittleEndian.PutUint64(frame[8:16], memory)
		if _, err := ReadReady(bytes.NewReader(frame[:])); err == nil {
			t.Fatalf("accepted invalid memory control %d", memory)
		}
	}
}
