package heic

import (
	"encoding/binary"
	"testing"
)

func TestHEICDispatchBrands(t *testing.T) {
	for _, tc := range []struct {
		brands []string
		want   bool
	}{
		{[]string{"heic"}, true},
		{[]string{"mif1"}, true},
		{[]string{"mif1", "avif"}, false},
		{[]string{"avif", "mif1", "heic"}, false},
		{[]string{"msf1"}, true},
		{[]string{"isom"}, false},
	} {
		data := make([]byte, 12+4*len(tc.brands))
		binary.BigEndian.PutUint32(data, uint32(len(data)))
		copy(data[4:], "ftyp")
		copy(data[8:], tc.brands[0])
		for i, brand := range tc.brands[1:] {
			copy(data[16+i*4:], brand)
		}
		if got := IsData(data); got != tc.want {
			t.Errorf("brands %v: dispatch=%v, want %v", tc.brands, got, tc.want)
		}
	}
}
