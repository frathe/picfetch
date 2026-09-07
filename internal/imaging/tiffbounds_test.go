package imaging

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestTIFFReadersRejectOutOfBoundsIFDs(t *testing.T) {
	for _, order := range []struct {
		name string
		mark string
		bo   binary.ByteOrder
	}{
		{"little-endian", "II", binary.LittleEndian},
		{"big-endian", "MM", binary.BigEndian},
	} {
		t.Run(order.name, func(t *testing.T) {
			for _, offset := range []uint32{8, 9, 0xfffffffe, 0xffffffff} {
				data := make([]byte, 8)
				copy(data, order.mark)
				order.bo.PutUint16(data[2:4], 42)
				order.bo.PutUint32(data[4:8], offset)
				t.Run(fmt.Sprintf("offset-%08x", offset), func(t *testing.T) {
					testTIFFReaders(t, data)
				})
			}
		})
	}
}

func TestTIFFReadersTolerateMalformedEntries(t *testing.T) {
	for _, bigEndian := range []bool{false, true} {
		for _, input := range tiffReaderCases(bigEndian) {
			t.Run(fmt.Sprintf("big-endian-%v/%s", bigEndian, input.name), func(t *testing.T) {
				testTIFFReaders(t, input.data)
				if t.Failed() {
					return
				}
				if got := ReadMetadata(input.data).Make; got != input.make {
					t.Errorf("camera make = %q, want %q", got, input.make)
				}
				if got := parseExifOrientation(append([]byte("Exif\x00\x00"), input.data...)); got != input.orientation {
					t.Errorf("orientation = %d, want %d", got, input.orientation)
				}
			})
		}
	}
}

type tiffReaderCase struct {
	name        string
	data        []byte
	make        string
	orientation int
}

func tiffReaderCases(bigEndian bool) []tiffReaderCase {
	var bo binary.ByteOrder = binary.LittleEndian
	mark := "II"
	if bigEndian {
		bo, mark = binary.BigEndian, "MM"
	}
	base := make([]byte, 50)
	copy(base, mark)
	bo.PutUint16(base[2:4], 42)
	bo.PutUint32(base[4:8], 8)
	bo.PutUint16(base[8:10], 3)
	bo.PutUint16(base[10:12], 0x010f) // Make, four inline ASCII bytes.
	bo.PutUint16(base[12:14], 2)
	bo.PutUint32(base[14:18], 4)
	copy(base[18:22], "Cam\x00")
	bo.PutUint16(base[22:24], exifIFDPointer)
	bo.PutUint16(base[24:26], 4)
	bo.PutUint32(base[26:30], 1)
	bo.PutUint16(base[34:36], 0x0112) // Orientation, one inline SHORT.
	bo.PutUint16(base[36:38], 3)
	bo.PutUint32(base[38:42], 1)
	bo.PutUint16(base[42:44], 6)
	cases := []tiffReaderCase{{"valid", base, "Cam", 6}}
	add := func(name, make string, orientation int, mutate func([]byte)) {
		data := append([]byte(nil), base...)
		mutate(data)
		cases = append(cases, tiffReaderCase{name, data, make, orientation})
	}
	add("max-entry-count", "Cam", 6, func(data []byte) { bo.PutUint16(data[8:10], 0xffff) })
	add("overflow-value-count", "", 6, func(data []byte) { bo.PutUint32(data[14:18], 0xffffffff) })
	add("overflow-value-offset", "", 6, func(data []byte) {
		bo.PutUint32(data[14:18], 8)
		bo.PutUint32(data[18:22], 0xffffffff)
	})
	add("truncated-value", "", 6, func(data []byte) {
		bo.PutUint32(data[14:18], 8)
		bo.PutUint32(data[18:22], 49)
	})
	for _, tag := range []uint16{exifIFDPointer, gpsIFDPointer, tagSubIFDs} {
		add(fmt.Sprintf("overflow-sub-ifd-%04x", tag), "Cam", 6, func(data []byte) {
			bo.PutUint16(data[22:24], tag)
			bo.PutUint32(data[30:34], 0xffffffff)
		})
	}
	add("overflow-next-ifd", "Cam", 6, func(data []byte) { bo.PutUint32(data[46:50], 0xffffffff) })
	add("cyclic-next-ifd", "Cam", 6, func(data []byte) { bo.PutUint32(data[46:50], 8) })
	add("root-count-straddles-end", "", 0, func(data []byte) { bo.PutUint32(data[4:8], 49) })
	cases = append(cases,
		tiffReaderCase{"truncated-entry", base[:45], "Cam", 0},
		tiffReaderCase{"truncated-next-ifd", base[:48], "Cam", 6},
	)
	return cases
}

func FuzzTIFFReaders(f *testing.F) {
	for _, bigEndian := range []bool{false, true} {
		for _, input := range tiffReaderCases(bigEndian) {
			f.Add(input.data)
		}
	}
	f.Add([]byte{'I', 'I', 42, 0, 255, 255, 255, 255})
	f.Fuzz(func(t *testing.T, data []byte) {
		// Bound work to TIFF reader inputs, excluding unrelated container decoders.
		if len(data) > 1<<16 {
			t.Skip()
		}
		if _, ok := tiffOrder(data); !ok {
			return
		}
		ReadMetadata(data)
		parseExifOrientation(append([]byte("Exif\x00\x00"), data...))
		embeddedJPEGPreview(data)
	})
}

func testTIFFReaders(t *testing.T, data []byte) {
	t.Helper()
	for _, reader := range []struct {
		name string
		read func()
	}{
		{"metadata", func() { ReadMetadata(data) }},
		{"orientation", func() { parseExifOrientation(append([]byte("Exif\x00\x00"), data...)) }},
		{"raw-preview", func() { embeddedJPEGPreview(data) }},
	} {
		t.Run(reader.name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Errorf("TIFF reader panicked on malformed spans: %v", err)
				}
			}()
			reader.read()
		})
	}
}
