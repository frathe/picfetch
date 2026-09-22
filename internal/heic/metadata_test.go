package heic

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestHEICPrimaryEXIF(t *testing.T) {
	data, err := os.ReadFile("testdata/exif-rotate.heic")
	if err != nil {
		t.Fatal(err)
	}
	tiff := bytes.Index(data, []byte{'I', 'I', 42, 0})
	if tiff < 0 {
		t.Fatal("fixture lost TIFF metadata")
	}
	want := bytes.Clone(data[tiff:])
	if got := primaryEXIF(data); !bytes.Equal(got, want) {
		t.Fatalf("primary EXIF = %x, want %x", got, want)
	}
	for _, tc := range []struct {
		name   string
		change func([]byte)
	}{
		{"unassociated", func(b []byte) { copy(b[bytes.Index(b, []byte("cdsc")):], "free") }},
		{"other image", func(b []byte) { binary.BigEndian.PutUint16(b[bytes.Index(b, []byte("cdsc"))+8:], 3) }},
		{"non EXIF item", func(b []byte) { copy(b[bytes.Index(b, []byte("Exif")):], "mime") }},
		{"invalid TIFF offset", func(b []byte) { binary.BigEndian.PutUint32(b[tiff-4:], ^uint32(0)) }},
		{"missing location", func(b []byte) { copy(b[bytes.Index(b, []byte("iloc")):], "free") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			broken := bytes.Clone(data)
			tc.change(broken)
			if got := primaryEXIF(broken); len(got) != 0 {
				t.Fatalf("unusable/unrelated metadata was returned: %x", got)
			}
		})
	}
	for size := range len(data) {
		if got := primaryEXIF(data[:size]); len(got) != 0 {
			t.Fatalf("truncation at %d returned metadata", size)
		}
	}
}

func TestHEICPrimaryEXIFLocations(t *testing.T) {
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	for _, version := range []byte{0, 1, 2} {
		for _, method := range []uint16{0, 1} {
			if version == 0 && method != 0 {
				continue
			}
			data := metadataFixture(version, method, 0, []uint64{6, 12}, []uint64{6, 12}, append(make([]byte, 10), tiff...))
			if got := primaryEXIF(data); !bytes.Equal(got, tiff) {
				t.Fatalf("version %d method %d multi-extent metadata = %x, want %x", version, method, got, tiff)
			}
		}
	}
	for _, tc := range []struct {
		name    string
		method  uint16
		base    uint64
		offsets []uint64
		lengths []uint64
	}{
		{"external item construction", 2, 0, []uint64{0}, []uint64{18}},
		{"offset overflow", 1, ^uint64(0), []uint64{8}, []uint64{18}},
		{"offset past input", 1, 0, []uint64{200}, []uint64{18}},
		{"length past input", 1, 0, []uint64{0}, []uint64{200}},
		{"metadata budget", 1, 0, []uint64{0}, []uint64{metadataLimit + 1}},
		{"aggregate budget", 1, 0, []uint64{0, 0}, []uint64{metadataLimit / 2, metadataLimit/2 + 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			size := 24
			if tc.name == "metadata budget" || tc.name == "aggregate budget" {
				size = metadataLimit + 1
			}
			payload := make([]byte, size)
			copy(payload[4:], tiff)
			data := metadataFixture(2, tc.method, tc.base, tc.offsets, tc.lengths, payload)
			if got := primaryEXIF(data); len(got) != 0 {
				t.Fatalf("invalid location returned %d metadata bytes", len(got))
			}
		})
	}
}

func TestHEICPrimaryEXIFMalformedTables(t *testing.T) {
	payload := append(make([]byte, 4), []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0}...)
	valid := metadataFixture(2, 1, 0, []uint64{0}, []uint64{uint64(len(payload))}, payload)
	for _, tc := range []struct {
		name   string
		change func([]byte)
	}{
		{"external data reference", func(b []byte) { b[bytes.Index(b, []byte("iloc"))+4+17] = 1 }},
		{"unbounded extent length", func(b []byte) { b[bytes.Index(b, []byte("iloc"))+4+4] &= 0xf0 }},
		{"invalid integer width", func(b []byte) { b[bytes.Index(b, []byte("iloc"))+4+4] = 0x98 }},
		{"reserved method bits", func(b []byte) { b[bytes.Index(b, []byte("iloc"))+4+14] = 0x10 }},
		{"excess item count", func(b []byte) { binary.BigEndian.PutUint32(b[bytes.Index(b, []byte("iloc"))+4+6:], ^uint32(0)) }},
		{"excess extent count", func(b []byte) { binary.BigEndian.PutUint16(b[bytes.Index(b, []byte("iloc"))+4+26:], 65535) }},
		{"protected metadata", func(b []byte) { b[bytes.Index(b, []byte("Exif"))-1] = 1 }},
		{"invalid reference version", func(b []byte) { b[bytes.Index(b, []byte("iref"))+4] = 2 }},
		{"excess reference count", func(b []byte) { binary.BigEndian.PutUint16(b[bytes.Index(b, []byte("cdsc"))+8:], 65535) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := bytes.Clone(valid)
			tc.change(data)
			if got := primaryEXIF(data); len(got) != 0 {
				t.Fatalf("malformed table supplied %d metadata bytes", len(got))
			}
		})
	}
}

// metadataFixture deliberately uses non-first primary and metadata item IDs,
// 64-bit extents, and version-dependent IDs; it contains no coded pixels.
func metadataFixture(version byte, method uint16, base uint64, offsets, lengths []uint64, payload []byte) []byte {
	box := func(kind string, data []byte) []byte {
		result := binary.BigEndian.AppendUint32(nil, uint32(8+len(data)))
		result = append(result, kind...)
		return append(result, data...)
	}
	primary, exif := uint32(3), uint32(7)
	if version == 2 {
		primary, exif = 0x10003, 0x10007
	}
	item := func(id uint32, kind string) []byte {
		entry := binary.BigEndian.AppendUint32([]byte{3, 0, 0, 0}, id)
		entry = append(entry, 0, 0)
		return box("infe", append(entry, []byte(kind+"\x00")...))
	}
	info := append([]byte{1, 0, 0, 0, 0, 0, 0, 3}, item(1, "hvc1")...)
	info = append(info, item(primary, "hvc1")...)
	info = append(info, item(exif, "Exif")...)
	meta := append([]byte{0, 0, 0, 0}, box("pitm", binary.BigEndian.AppendUint32([]byte{1, 0, 0, 0}, primary))...)
	meta = append(meta, box("iinf", info)...)
	reference := binary.BigEndian.AppendUint32(nil, exif)
	reference = append(reference, 0, 1)
	reference = binary.BigEndian.AppendUint32(reference, primary)
	meta = append(meta, box("iref", append([]byte{1, 0, 0, 0}, box("cdsc", reference)...))...)
	locations := []byte{version, 0, 0, 0, 0x88, 0x80}
	if version == 2 {
		locations = append(locations, 0, 0, 0, 1)
		locations = binary.BigEndian.AppendUint32(locations, exif)
	} else {
		locations = append(locations, 0, 1, 0, 7)
	}
	if version != 0 {
		locations = binary.BigEndian.AppendUint16(locations, method)
	}
	locations = append(locations, 0, 0) // local data reference
	basePosition := len(locations)
	locations = binary.BigEndian.AppendUint64(locations, base)
	locations = binary.BigEndian.AppendUint16(locations, uint16(len(offsets)))
	for i, offset := range offsets {
		locations = binary.BigEndian.AppendUint64(locations, offset)
		locations = binary.BigEndian.AppendUint64(locations, lengths[i])
	}
	if method == 0 {
		// Absolute file offsets address the following mdat payload.
		binary.BigEndian.PutUint64(locations[basePosition:], uint64(8+len(meta)+8+len(locations)+8)+base)
	}
	meta = append(meta, box("iloc", locations)...)
	if method == 0 {
		return append(box("meta", meta), box("mdat", payload)...)
	}
	meta = append(meta, box("idat", payload)...)
	return box("meta", meta)
}

func FuzzHEICPrimaryEXIF(f *testing.F) {
	data, err := os.ReadFile("testdata/exif-rotate.heic")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(data)
	tiff := bytes.Index(data, []byte{'I', 'I', 42, 0})
	if tiff < 0 {
		f.Fatal("fixture lost TIFF metadata")
	}
	payload := append(make([]byte, 4), data[tiff:]...)
	f.Add(metadataFixture(2, 1, 0, []uint64{0}, []uint64{uint64(len(payload))}, payload))
	f.Fuzz(func(t *testing.T, data []byte) {
		if got := primaryEXIF(data); len(got) > metadataLimit {
			t.Fatalf("metadata exceeded the worker bound: %d", len(got))
		}
	})
}
