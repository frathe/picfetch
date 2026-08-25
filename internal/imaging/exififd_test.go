package imaging

import (
	"encoding/binary"
	"testing"
)

func TestRationalsValue(t *testing.T) {
	bo := binary.LittleEndian

	rationals := func(pairs ...uint32) []byte {
		b := make([]byte, 0, len(pairs)*4)
		for _, v := range pairs {
			b = binary.LittleEndian.AppendUint32(b, v)
		}
		return b
	}

	t.Run("three rationals", func(t *testing.T) {
		got, ok := rationalsValue(bo, 5, rationals(1, 2, 3, 4, 10, 4), 3)

		if !ok || len(got) != 3 || got[0] != 0.5 || got[1] != 0.75 || got[2] != 2.5 {
			t.Errorf("rationalsValue() = %v, %v, want [0.5 0.75 2.5], true", got, ok)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		if _, ok := rationalsValue(bo, 4, rationals(1, 2, 3, 4, 5, 6), 3); ok {
			t.Error("rationalsValue() ok = true for a LONG entry, want false")
		}
	})

	t.Run("truncated", func(t *testing.T) {
		if _, ok := rationalsValue(bo, 5, rationals(1, 2, 3, 4), 3); ok {
			t.Error("rationalsValue() ok = true for two rationals, want false")
		}
	})

	t.Run("zero denominator", func(t *testing.T) {
		if _, ok := rationalsValue(bo, 5, rationals(1, 0, 3, 4, 5, 6), 3); ok {
			t.Error("rationalsValue() ok = true for a zero denominator, want false")
		}
	})
}
