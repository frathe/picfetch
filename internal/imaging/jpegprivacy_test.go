package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestJPEGMetadataRemovalPrivacy(t *testing.T) {
	t.Run("EXIF chroma positioning", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		for _, tc := range []struct {
			name      string
			kind      uint16
			count     uint32
			value     uint32
			duplicate bool
			allow     bool
		}{
			{"centered", 3, 1, 1, false, true},
			{"co-sited", 3, 1, 2, false, true},
			{"reserved zero", 3, 1, 0, false, false},
			{"reserved value", 3, 1, 3, false, false},
			{"wrong type", 4, 1, 1, false, false},
			{"empty value", 3, 0, 1, false, false},
			{"multiple values", 3, 2, 1, false, false},
			{"duplicate tag", 3, 1, 1, true, false},
		} {
			for _, orientation := range []uint32{1, 6} {
				t.Run(tc.name+" orientation="+strconv.Itoa(int(orientation)), func(t *testing.T) {
					entries := []tiffEntry{
						{tag: 0x0112, typ: 3, count: 1, value: orientation},
						{tag: 0x0213, typ: tc.kind, count: tc.count, value: tc.value},
					}
					if tc.duplicate {
						entries = append(entries, entries[1])
					}
					tiff := buildIFD0TIFF(t, entries...)
					data := mustInjectRemoval(t, plain, jpegSegmentBytes(0xe1, append([]byte("Exif\x00\x00"), tiff...)), jpegSegmentBytes(0xfe, []byte("private comment")))
					inspection := InspectJPEGMetadata(context.Background(), data)
					path := writeTempFile(t, "chroma-positioning.jpg", data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if !tc.allow {
						if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProcess) || !errors.Is(err, ErrJPEGMetadataProcess) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
							t.Fatalf("chroma-positioning refusal = %+v; %+v, %v", inspection, result, err)
						}
						return
					}
					if inspection.State != JPEGMetadataRemovable || inspection.Err != nil || err != nil || !result.Committed {
						t.Fatalf("centered removal = %+v; %+v, %v", inspection, result, err)
					}
					got := mustRead(t, path)
					if clean := InspectJPEGMetadata(context.Background(), got); clean.State != JPEGMetadataClean || clean.Err != nil {
						t.Fatalf("removed output = %+v", clean)
					}
					if orientation == 1 && tc.value == 1 && !bytes.Equal(got, plain) {
						t.Fatal("centered removal changed the upright primary JPEG")
					}
				})
			}
		}
	})
	t.Run("EXIF color declaration", func(t *testing.T) {
		plain := uitest.EncodeJPEG(t, 16, 12, color.White)
		for _, bigEndian := range []bool{false, true} {
			for _, tc := range []struct {
				name  string
				space uint16
				index string
				allow bool
			}{
				{"sRGB", 1, "R98", true},
				{"Adobe RGB", 0xffff, "R03", true},
				{"uncalibrated", 0xffff, "R98", true},
				{"conflicting interoperability", 1, "R03", true},
				{"reserved color space", 2, "R98", false},
				{"unknown interoperability", 1, "XYZ", false},
			} {
				t.Run(tc.name+" big-endian="+strconv.FormatBool(bigEndian), func(t *testing.T) {
					data := mustInjectRemoval(t, plain, removalColorEXIF(tc.space, tc.index, bigEndian), jpegSegmentBytes(0xfe, []byte("private comment")))
					inspection := InspectJPEGMetadata(context.Background(), data)
					path := writeTempFile(t, "color.jpg", data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if tc.allow {
						if inspection.State != JPEGMetadataRemovable || inspection.Err != nil || err != nil || !result.Committed || !bytes.Equal(mustInjectRemoval(t, plain, removalColorEXIF(tc.space, tc.index, false)), mustRead(t, path)) {
							t.Fatalf("color declaration removal = %+v; %+v, %v", inspection, result, err)
						}
					} else if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProcess) || !errors.Is(err, ErrJPEGMetadataProcess) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
						t.Fatalf("color refusal = %+v; %+v, %v", inspection, result, err)
					}
				})
			}
		}
	})
	t.Run("legal marker fill remains supported", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		frame := bytes.Index(plain, []byte{0xff, 0xc0})
		if frame < 0 {
			t.Fatal("fixture has no baseline frame")
		}
		filled := append(bytes.Clone(plain[:frame]), 0xff)
		filled = append(filled, plain[frame:]...)
		data := mustInjectRemoval(t, filled, jpegSegmentBytes(0xfe, []byte("fixture description")))
		inspection := InspectJPEGMetadata(context.Background(), data)
		if inspection.State != JPEGMetadataRemovable || inspection.Err != nil {
			t.Fatalf("filled-marker inspection = %+v", inspection)
		}
		path := writeTempFile(t, "filled-marker.jpg", data)
		result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
		if err != nil || !result.Committed || !bytes.Equal(filled, mustRead(t, path)) {
			t.Fatalf("filled-marker removal = %+v, %v", result, err)
		}
	})
	for _, name := range []string{"baseline-rgb", "progressive-rgb", "multiscan-rgb", "baseline-gray", "progressive-gray"} {
		t.Run(name+" all scan intervals", func(t *testing.T) {
			plain := removalFixture(t, name+".jpg")
			// Place an ordinary comment before every scan and after the last.
			// These generated JPEGs contain no marker-looking metadata payloads.
			comment := jpegSegmentBytes(0xfe, []byte("PicFetch fixture comment"))
			data := bytes.ReplaceAll(plain, []byte{0xff, 0xda}, append(bytes.Clone(comment), 0xff, 0xda))
			data = bytes.ReplaceAll(data, []byte{0xff, 0xd9}, append(bytes.Clone(comment), 0xff, 0xd9))
			data = mustInjectRemoval(t, data,
				jpegSegmentBytes(0xe1, []byte("http://ns.adobe.com/xap/1.0/\x00<fixture/>")),
				jpegSegmentBytes(0xe2, []byte("MPF\x00fixture secondary-image index")),
				jpegSegmentBytes(0xed, []byte("Photoshop 3.0\x00fixture description")),
				jpegSegmentBytes(0xe0, []byte{'J', 'F', 'X', 'X', 0, 0x13, 1, 1, 12, 34, 56}),
			)
			data = append(data, []byte("fixture trailer")...)
			path := writeTempFile(t, "scans.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if err != nil || !result.Committed {
				t.Fatalf("removal = %+v, %v", result, err)
			}
			if !bytes.Equal(plain, mustRead(t, path)) {
				t.Fatal("primary image differs from independent clean fixture")
			}
		})
	}
	t.Run("baseline removes embedded preview and trailer", func(t *testing.T) {
		plain := uitest.EncodeJPEG(t, 16, 12, color.White)
		// A complete JFIF 1.02 header with a harmless one-pixel RGB preview.
		jfif := []byte{'J', 'F', 'I', 'F', 0, 1, 2, 0, 0, 1, 0, 1, 1, 1, 12, 34, 56}
		data, err := injectJPEGMetadata(plain, [][]byte{jpegSegmentBytes(0xE0, jfif)})
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, []byte("fixture trailer")...)
		path := writeTempFile(t, "preview.jpg", data)
		result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
		if err != nil || !result.Committed {
			t.Fatalf("removal = %+v, %v", result, err)
		}
		jfif = jfif[:14]
		jfif[12], jfif[13] = 0, 0
		want, err := injectJPEGMetadata(plain, [][]byte{jpegSegmentBytes(0xE0, jfif)})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(mustRead(t, path), want) {
			t.Fatal("output must contain only the primary JPEG and thumbnail-free JFIF")
		}
	})
}

func TestJPEGMetadataRemovalEntropy(t *testing.T) {
	for _, name := range []string{
		"baseline-rgb", "progressive-rgb", "multiscan-rgb", "baseline-gray", "progressive-gray",
		"entropy-baseline-420-restart1", "entropy-progressive-420", "entropy-progressive-444-restart1",
		"entropy-multiscan-444-restart1", "entropy-progressive-gray-restart1",
	} {
		t.Run(name+" unused scan bytes", func(t *testing.T) {
			plain := removalFixture(t, name+".jpg")
			if inspection := InspectJPEGMetadata(context.Background(), plain); inspection.State != JPEGMetadataClean || inspection.Err != nil {
				t.Fatalf("independent clean fixture = %+v", inspection)
			}
			withComment := mustInjectRemoval(t, plain, jpegSegmentBytes(0xfe, []byte("fixture description")))
			cleanPath := writeTempFile(t, "clean-entropy.jpg", withComment)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(cleanPath))
			if err != nil || !result.Committed || !bytes.Equal(plain, mustRead(t, cleanPath)) {
				t.Fatalf("independent clean output = %+v, %v", result, err)
			}
			for _, at := range removalEntropyBoundaries(t, plain) {
				for _, extra := range [][]byte{{0x42}, {0xff, 0x00}} {
					data := append(bytes.Clone(plain[:at]), extra...)
					data = append(data, plain[at:]...)
					inspection := InspectJPEGMetadata(context.Background(), data)
					path := writeTempFile(t, "unused-scan.jpg", data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataStructure) || !errors.Is(err, ErrJPEGMetadataStructure) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
						t.Fatalf("unused scan bytes refusal = %+v; %+v, %v", inspection, result, err)
					}
				}
			}
		})
	}
	t.Run("cancellation before pixel decoding", func(t *testing.T) {
		data := uitest.EncodeJPEG(t, 2048, 2048, color.White)
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx := &jpegDecodeObserveContext{Context: &groupingCancelContext{Context: base, cancel: cancel, after: 100}}
		inspection := InspectJPEGMetadata(ctx, data)
		if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, context.Canceled) || ctx.decoded {
			t.Fatalf("entropy cancellation = %+v, full decode started=%v", inspection, ctx.decoded)
		}
	})
	t.Run("padding and restart boundaries", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			blocks  int
			restart bool
			entropy []byte
			allow   bool
		}{
			{"one block", 1, false, []byte{0x3f}, true},
			{"restart", 2, true, []byte{0x3f, 0xff, 0xd0, 0x3f}, true},
			{"restart fill", 2, true, []byte{0x3f, 0xff, 0xff, 0xd0, 0x3f}, true},
			{"zero final padding", 1, false, []byte{0x3e}, false},
			{"extra final restart", 1, false, []byte{0x3f, 0xff, 0xd0}, false},
			{"zero restart padding", 2, true, []byte{0x3e, 0xff, 0xd0, 0x3f}, false},
			{"extra restart byte", 2, true, []byte{0x3f, 0x42, 0xff, 0xd0, 0x3f}, false},
			{"wrong restart", 2, true, []byte{0x3f, 0xff, 0xd1, 0x3f}, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				data := removalZeroBlocks(tc.blocks, tc.restart, tc.entropy)
				inspection := InspectJPEGMetadata(context.Background(), data)
				path := writeTempFile(t, "entropy-boundary.jpg", data)
				result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
				if tc.allow {
					if inspection.State != JPEGMetadataClean || inspection.Err != nil || err != nil || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
						t.Fatalf("legal interval = %+v; %+v, %v", inspection, result, err)
					}
				} else if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataStructure) || !errors.Is(err, ErrJPEGMetadataStructure) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
					t.Fatalf("invalid interval refusal = %+v; %+v, %v", inspection, result, err)
				}
			})
		}
	})
}

// Each synthetic gray block has DC category zero and AC EOB, both Huffman
// code 0. Thus 0x3f encodes one block followed by six required one padding bits.
func removalZeroBlocks(blocks int, restart bool, entropy []byte) []byte {
	data := []byte{0xff, 0xd8}
	data = append(data, jpegSegmentBytes(0xdb, append([]byte{0}, bytes.Repeat([]byte{1}, 64)...))...)
	data = append(data, jpegSegmentBytes(0xc0, []byte{8, 0, 8, 0, byte(8 * blocks), 1, 1, 0x11, 0})...)
	var tables []byte
	for _, class := range []byte{0, 0x10} {
		tables = append(tables, class, 1)
		tables = append(tables, make([]byte, 16)...)
	}
	data = append(data, jpegSegmentBytes(0xc4, tables)...)
	if restart {
		data = append(data, jpegSegmentBytes(0xdd, []byte{0, 1})...)
	}
	data = append(data, jpegSegmentBytes(0xda, []byte{1, 1, 0, 0, 63, 0})...)
	data = append(data, entropy...)
	return append(data, 0xff, 0xd9)
}

// Locate intervals in known synthetic fixtures without interpreting their
// coefficients. The operation under test must establish exact consumption.
func removalEntropyBoundaries(t *testing.T, data []byte) []int {
	t.Helper()
	var boundaries []int
	for pos := 2; pos < len(data)-1; {
		if data[pos] != 0xff {
			t.Fatal("fixture marker expected")
		}
		marker := data[pos+1]
		if marker == 0xd9 {
			break
		}
		pos += 2 + int(binary.BigEndian.Uint16(data[pos+2:]))
		if marker != 0xda {
			continue
		}
		for pos < len(data)-1 {
			if data[pos] != 0xff {
				pos++
				continue
			}
			if data[pos+1] == 0 {
				pos += 2
				continue
			}
			boundaries = append(boundaries, pos)
			if data[pos+1] < 0xd0 || data[pos+1] > 0xd7 {
				break
			}
			pos += 2
		}
	}
	return boundaries
}

func removalFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "jpeg-removal", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func removalColorEXIF(space uint16, index string, bigEndian bool) []byte {
	var bo binary.ByteOrder = binary.LittleEndian
	tiff := make([]byte, 74)
	copy(tiff, "II")
	if bigEndian {
		bo = binary.BigEndian
		copy(tiff, "MM")
	}
	bo.PutUint16(tiff[2:4], 42)
	bo.PutUint32(tiff[4:8], 8)
	entry := func(at int, tag, kind uint16, count uint32) {
		bo.PutUint16(tiff[at:at+2], tag)
		bo.PutUint16(tiff[at+2:at+4], kind)
		bo.PutUint32(tiff[at+4:at+8], count)
	}
	bo.PutUint16(tiff[8:10], 1)
	entry(10, 0x8769, 4, 1) // IFD0 -> Exif IFD.
	bo.PutUint32(tiff[18:22], 26)
	bo.PutUint16(tiff[26:28], 2)
	entry(28, 0xa001, 3, 1) // ColorSpace.
	bo.PutUint16(tiff[36:38], space)
	entry(40, 0xa005, 4, 1) // Exif -> Interoperability IFD.
	bo.PutUint32(tiff[48:52], 56)
	bo.PutUint16(tiff[56:58], 1)
	entry(58, 1, 2, 4) // InteroperabilityIndex.
	copy(tiff[66:70], index)
	return jpegSegmentBytes(0xe1, append([]byte("Exif\x00\x00"), tiff...))
}

func TestJPEGMetadataRemovalFidelity(t *testing.T) {
	t.Run("orientation preserves JFIF pixel aspect", func(t *testing.T) {
		for _, orientation := range []uint16{2, 6} {
			plain := uitest.EncodeJPEG(t, 16, 12, color.White)
			jfif := []byte{'J', 'F', 'I', 'F', 0, 1, 2, 0, 0, 2, 0, 1, 0, 0}
			data := mustInjectRemoval(t, plain, jpegSegmentBytes(0xe0, jfif), wrapAsAPP1(buildExifSegment(t, orientation, false)), jpegSegmentBytes(0xfe, []byte("private comment")))
			path := writeTempFile(t, "aspect.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if err != nil || !result.Committed {
				t.Fatalf("orientation = %+v, %v", result, err)
			}
			wantPrefix := append([]byte{0xff, 0xd8}, jpegSegmentBytes(0xe0, jfif)...)
			if !bytes.HasPrefix(mustRead(t, path), wantPrefix) {
				t.Fatalf("orientation %d lost or mis-rotated the JFIF pixel aspect", orientation)
			}
		}
	})
	for _, model := range []string{"rgb", "gray"} {
		processes := []string{"baseline", "progressive"}
		if model == "rgb" {
			processes = append(processes, "multiscan")
		}
		for _, process := range processes {
			for _, version := range []string{"v2", "v4"} {
				for orientation := 2; orientation <= 8; orientation++ {
					t.Run(process+" "+model+" "+version+" orientation "+strconv.Itoa(orientation), func(t *testing.T) {
						plain := removalFixture(t, process+"-"+model+".jpg")
						profile := removalFixture(t, model+"-"+version+".icc")
						segments := profileSegments(profile)
						segments = append(segments, wrapAsAPP1(buildExifSegment(t, uint16(orientation), false)))
						path := writeTempFile(t, "oriented.jpg", mustInjectRemoval(t, plain, segments...))
						result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
						if err != nil || !result.Committed {
							t.Fatalf("oriented removal = %+v, %v", result, err)
						}
						got := mustRead(t, path)
						inspection := InspectJPEGMetadata(context.Background(), got)
						if inspection.State != JPEGMetadataClean {
							t.Fatalf("orientation output is not qualified/clean: %+v", inspection)
						}
						before, err := jpeg.Decode(bytes.NewReader(plain))
						if err != nil {
							t.Fatal(err)
						}
						after, err := jpeg.Decode(bytes.NewReader(got))
						if err != nil {
							t.Fatal(err)
						}
						if jpegEXIFOrientation(got) != orientation || !reflect.DeepEqual(before, after) {
							t.Fatal("orientation or exact decoded samples changed")
						}
						after = ApplyOrientation(after, orientation)
						width, height := 16, 12
						if orientation >= 5 {
							width, height = 12, 16
						}
						if after.Bounds().Dx() != width || after.Bounds().Dy() != height {
							t.Fatalf("orientation bounds %v", after.Bounds())
						}
						if !reflect.DeepEqual(ApplyOrientation(before, orientation), after) {
							t.Fatal("displayed pixels changed")
						}
						if !bytes.Equal(profileTestTags(t, profile)["wtpt"], profileTestTags(t, readRemovalProfile(t, got))["wtpt"]) {
							t.Fatal("orientation changed profile transform")
						}
					})
				}
			}
		}
	}
	for _, name := range []string{"baseline-rgb", "progressive-rgb", "multiscan-rgb", "baseline-gray", "progressive-gray"} {
		t.Run(name, func(t *testing.T) {
			plain := removalFixture(t, name+".jpg")
			before, err := jpeg.Decode(bytes.NewReader(plain))
			if err != nil {
				t.Fatal(err)
			}
			data := mustInjectRemoval(t, plain, jpegSegmentBytes(0xfe, []byte("fixture")))
			path := writeTempFile(t, "fidelity.jpg", data)
			if _, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path)); err != nil {
				t.Fatal(err)
			}
			got := mustRead(t, path)
			after, err := jpeg.Decode(bytes.NewReader(got))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(plain, got) || !reflect.DeepEqual(before, after) {
				t.Fatal("upright coded bytes or decoded samples changed")
			}
		})
	}
}

func TestJPEGMetadataRemovalRefusal(t *testing.T) {
	plain := uitest.EncodeJPEG(t, 16, 12, color.White)
	t.Run("undeclared component interpretation", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			ids   []byte
			allow bool
		}{
			{"YCbCr identifiers", []byte{1, 2, 3}, true},
			{"RGB identifiers", []byte("RGB"), true},
			{"unqualified identifiers", []byte("ABC"), false},
			{"unqualified identifier order", []byte{3, 2, 1}, false},
		} {
			for _, orientation := range []uint16{1, 6} {
				t.Run(tc.name+" orientation="+strconv.Itoa(int(orientation)), func(t *testing.T) {
					components := bytes.Clone(plain)
					frame := bytes.Index(components, []byte{0xff, 0xc0})
					scan := bytes.Index(components, []byte{0xff, 0xda})
					if frame < 0 || scan < 0 {
						t.Fatal("fixture has no baseline frame or scan")
					}
					for i, id := range tc.ids {
						components[frame+10+3*i] = id
						components[scan+5+2*i] = id
					}
					data := mustInjectRemoval(t, components, wrapAsAPP1(buildExifSegment(t, orientation, false)), jpegSegmentBytes(0xfe, []byte("private comment")))
					inspection := InspectJPEGMetadata(context.Background(), data)
					path := writeTempFile(t, "component-interpretation.jpg", data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if tc.allow {
						if inspection.State != JPEGMetadataRemovable || inspection.Err != nil || err != nil || !result.Committed {
							t.Fatalf("qualified interpretation = %+v; %+v, %v", inspection, result, err)
						}
					} else if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProcess) || !errors.Is(err, ErrJPEGMetadataProcess) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
						t.Fatalf("interpretation refusal = %+v; %+v, %v", inspection, result, err)
					}
				})
			}
		}
	})
	for _, tc := range []struct {
		name   string
		tag    uint16
		values []uint32
	}{
		{"transfer function", 0x012d, nil},
		{"white point", 0x013e, []uint32{3127, 10000, 3290, 10000}},
		{"primary chromaticities", 0x013f, []uint32{64, 100, 33, 100, 30, 100, 60, 100, 15, 100, 6, 100}},
		{"YCbCr coefficients", 0x0211, []uint32{299, 1000, 587, 1000, 114, 1000}},
		{"reference black white", 0x0214, []uint32{0, 1, 255, 1, 128, 1, 255, 1, 128, 1, 255, 1}},
		{"gamma", 0xa500, []uint32{22, 10}},
	} {
		t.Run("explicit EXIF "+tc.name, func(t *testing.T) {
			var values []byte
			kind, count := uint16(5), uint32(len(tc.values)/2)
			for _, value := range tc.values {
				values = binary.LittleEndian.AppendUint32(values, value)
			}
			if tc.tag == 0x012d {
				kind, count = 3, 768
				for i := range count {
					values = binary.LittleEndian.AppendUint16(values, uint16(i%256)*257)
				}
			}
			valueOffset := uint32(26)
			if tc.tag == 0xa500 {
				valueOffset = 44
			}
			tiff := buildIFD0TIFF(t, tiffEntry{tag: tc.tag, typ: kind, count: count, value: valueOffset})
			if tc.tag == 0xa500 {
				root := buildIFD0TIFF(t, tiffEntry{tag: 0x8769, typ: 4, count: 1, value: 26})
				tiff = append(root, tiff[8:]...)
			}
			tiff = append(tiff, values...)
			data := mustInjectRemoval(t, plain, jpegSegmentBytes(0xe1, append([]byte("Exif\x00\x00"), tiff...)), jpegSegmentBytes(0xfe, []byte("private comment")))
			inspection := InspectJPEGMetadata(context.Background(), data)
			path := writeTempFile(t, "color-transform.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProcess) || !errors.Is(err, ErrJPEGMetadataProcess) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
				t.Fatalf("EXIF transform refusal = %+v; %+v, %v", inspection, result, err)
			}
		})
	}
	t.Run("inspection has a separate working-memory limit", func(t *testing.T) {
		// A real, ordinary JPEG with constant pixels needs little encoded
		// storage; the test generator does not allocate its full pixel plane.
		var encoded bytes.Buffer
		large := removalUniformImage{Uniform: image.NewUniform(color.White), bounds: image.Rect(0, 0, 10000, 5000)}
		if err := jpeg.Encode(&encoded, large, nil); err != nil {
			t.Fatal(err)
		}
		ctx := &jpegDecodeObserveContext{Context: context.Background()}
		data := mustInjectRemoval(t, encoded.Bytes(), jpegSegmentBytes(0xee, []byte{'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 0}))
		inspection := InspectJPEGMetadata(ctx, data)
		if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataMemory) || ctx.decoded {
			t.Fatalf("memory admission = %+v, full decode started=%v", inspection, ctx.decoded)
		}
		path := writeTempFile(t, "large-photo.jpg", data)
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !errors.Is(err, ErrJPEGMetadataMemory) || result.Committed || ctx.decoded || !bytes.Equal(data, mustRead(t, path)) {
			t.Fatalf("memory refusal = %+v, %v, full decode started=%v", result, err, ctx.decoded)
		}
	})
	t.Run("conflicting Adobe and RGB component declarations", func(t *testing.T) {
		components := bytes.Clone(plain)
		frame := bytes.Index(components, []byte{0xff, 0xc0})
		scan := bytes.Index(components, []byte{0xff, 0xda})
		if frame < 0 || scan < 0 {
			t.Fatal("fixture has no baseline frame or scan")
		}
		for i, id := range []byte{'R', 'G', 'B'} {
			components[frame+10+3*i] = id
			components[scan+5+2*i] = id
		}
		adobeYCbCr := jpegSegmentBytes(0xee, []byte{'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 1})
		for _, orientation := range []uint16{1, 6} {
			data := mustInjectRemoval(t, components, adobeYCbCr, wrapAsAPP1(buildExifSegment(t, orientation, false)))
			path := writeTempFile(t, "conflicting-colors.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if !errors.Is(err, ErrJPEGMetadataProcess) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
				t.Fatalf("orientation %d: conflicting color declarations = %+v, %v", orientation, result, err)
			}
			inspection := InspectJPEGMetadata(context.Background(), data)
			if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProcess) {
				t.Fatalf("orientation %d: conflicting color inspection = %+v", orientation, inspection)
			}
		}
	})
	// Conflicting recognized color declarations remain uncertain even in order.
	adobe := jpegSegmentBytes(0xee, []byte{'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 0})
	jfif := jpegSegmentBytes(0xe0, []byte{'J', 'F', 'I', 'F', 0, 1, 2, 0, 0, 1, 0, 1, 0, 0})
	t.Run("misplaced JFIF declaration", func(t *testing.T) {
		// The standard-library fixture starts with a quantization table.
		afterTable := 4 + int(binary.BigEndian.Uint16(plain[4:6]))
		beforeScan := bytes.Index(plain, []byte{0xff, 0xda})
		for _, at := range []int{afterTable, beforeScan} {
			data := append(bytes.Clone(plain[:at]), jfif...)
			data = append(data, plain[at:]...)
			inspection := InspectJPEGMetadata(context.Background(), data)
			if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataStructure) {
				t.Fatalf("misplaced JFIF inspection = %+v", inspection)
			}
			path := writeTempFile(t, "misplaced-jfif.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if !errors.Is(err, ErrJPEGMetadataStructure) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
				t.Fatalf("misplaced JFIF removal = %+v, %v", result, err)
			}
		}
	})
	for name, data := range map[string][]byte{
		"conflicting color declarations": mustInjectRemoval(t, plain, jfif, adobe),
		"unsupported SPIFF declaration":  mustInjectRemoval(t, plain, jpegSegmentBytes(0xe8, []byte("SPIFF\x00"))),
		"missing end":                    plain[:len(plain)-2],
		"incomplete image":               {0xff, 0xd8, 0xff, 0xd9},
		"invalid profile":                mustInjectRemoval(t, plain, jpegSegmentBytes(0xe2, []byte("ICC_PROFILE\x00\x01\x01fixture"))),
		"uncertain orientation":          mustInjectRemoval(t, plain, jpegSegmentBytes(0xe1, []byte("Exif\x00\x00fixture"))),
		"non JPEG":                       []byte("fixture"),
	} {
		t.Run(name, func(t *testing.T) {
			path := writeTempFile(t, "refused.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if err == nil || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
				t.Fatalf("refusal must preserve source: result=%+v error=%v", result, err)
			}
			inspection := InspectJPEGMetadata(context.Background(), data)
			if inspection.State != JPEGMetadataUnsupported || inspection.Err == nil || CanStripJPEGMetadata(data) {
				t.Fatalf("uncertain input must not be presented as clean/removable: %+v", inspection)
			}
		})
	}
	t.Run("cancellation leaves source", func(t *testing.T) {
		data := append(plain, []byte("fixture trailer")...)
		path := writeTempFile(t, "cancel.jpg", data)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !errors.Is(err, context.Canceled) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
			t.Fatalf("cancellation = %+v, %v", result, err)
		}
	})
	t.Run("cancellation during pixel validation", func(t *testing.T) {
		data := mustInjectRemoval(t, uitest.EncodeJPEG(t, 512, 512, color.White), wrapAsAPP1(buildExifSegment(t, 6, false)))
		data = append(data, []byte("private preview")...)
		path := writeTempFile(t, "cancel-validation.jpg", data)
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx := &jpegDecodeObserveContext{Context: base, cancel: cancel}
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !ctx.decoded || !errors.Is(err, context.Canceled) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
			t.Fatalf("pixel-validation cancellation = %+v, %v", result, err)
		}
	})
	t.Run("cancellation during output header validation", func(t *testing.T) {
		data := append(bytes.Clone(plain), []byte("fixture trailer")...)
		path := writeTempFile(t, "cancel-output-header.jpg", data)
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx := &jpegConfigCancelContext{Context: base, cancel: cancel}
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !ctx.observed || !errors.Is(err, context.Canceled) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
			t.Fatalf("output-header cancellation observed=%v result=%+v error=%v", ctx.observed, result, err)
		}
	})
}

// Cancel at output validation's standard-library DecodeConfig boundary,
// following admission's first DecodeConfig. No private production hook is used.
type jpegConfigCancelContext struct {
	context.Context
	cancel   context.CancelFunc
	inside   bool
	configs  int
	observed bool
}

func (c *jpegConfigCancelContext) Err() error {
	var callers [32]uintptr
	frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
	inside := false
	for {
		frame, more := frames.Next()
		if frame.Function == "image/jpeg.DecodeConfig" {
			inside = true
			break
		}
		if !more {
			break
		}
	}
	if inside && !c.inside {
		c.configs++
		if c.configs == 2 {
			c.observed = true
			c.cancel()
		}
	}
	c.inside = inside
	return c.Context.Err()
}

type removalUniformImage struct {
	*image.Uniform
	bounds image.Rectangle
}

func (i removalUniformImage) Bounds() image.Rectangle { return i.bounds }

type jpegDecodeObserveContext struct {
	context.Context
	decoded bool
	cancel  context.CancelFunc
}

func (c *jpegDecodeObserveContext) Err() error {
	var callers [32]uintptr
	frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
	for {
		frame, more := frames.Next()
		if frame.Function == "image/jpeg.Decode" {
			c.decoded = true
			if c.cancel != nil {
				c.cancel()
			}
			return c.Context.Err()
		}
		if !more {
			break
		}
	}

	return c.Context.Err()
}

func mustInjectRemoval(t *testing.T, plain []byte, segments ...[]byte) []byte {
	t.Helper()
	at := 2
	// Keep independently generated JFIF fixtures in their required leading
	// position while adding the metadata under test after that declaration.
	if len(plain) >= 11 && bytes.Equal(plain[2:4], []byte{0xff, 0xe0}) && bytes.Equal(plain[6:11], []byte("JFIF\x00")) {
		at += 2 + int(binary.BigEndian.Uint16(plain[4:6]))
	}
	out := bytes.Clone(plain[:at])
	for _, segment := range segments {
		out = append(out, segment...)
	}
	return append(out, plain[at:]...)
}

func TestJPEGMetadataRemovalInspection(t *testing.T) {
	t.Run("retained declaration fill is not metadata", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		for _, tc := range []struct {
			name     string
			marker   byte
			segments [][]byte
		}{
			{"JFIF", 0xe0, nil},
			{"Adobe", 0xee, [][]byte{jpegSegmentBytes(0xee, []byte{'A', 'd', 'o', 'b', 'e', 0, 100, 0, 0, 0, 0, 1})}},
			{"ICC", 0xe2, profileSegments(removalFixture(t, "rgb-v4.icc"))},
		} {
			t.Run(tc.name, func(t *testing.T) {
				path := writeTempFile(t, "normalized.jpg", mustInjectRemoval(t, plain, tc.segments...))
				if _, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path)); err != nil {
					t.Fatal(err)
				}
				clean := mustRead(t, path)
				filled := fillRemovalMarker(t, clean, tc.marker)
				for _, metadata := range []bool{false, true} {
					data, state := filled, JPEGMetadataClean
					if metadata {
						data = fillRemovalMarker(t, mustInjectRemoval(t, clean, jpegSegmentBytes(0xfe, []byte("fixture description"))), tc.marker)
						state = JPEGMetadataRemovable
					}
					inspection := InspectJPEGMetadata(context.Background(), data)
					path := writeTempFile(t, "filled-declaration.jpg", data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if inspection.State != state || inspection.Err != nil || err != nil || result.Committed != metadata || !bytes.Equal(filled, mustRead(t, path)) {
						t.Fatalf("declaration fill (metadata=%v) = %+v; %+v, %v", metadata, inspection, result, err)
					}
				}
			})
		}
	})
	t.Run("legal EOI fill preserves clean and removable sources", func(t *testing.T) {
		plain := uitest.EncodeJPEG(t, 16, 12, color.White)
		filled := append(bytes.Clone(plain[:len(plain)-2]), 0xff, 0xff, 0xd9)
		for _, metadata := range []bool{false, true} {
			data := filled
			state := JPEGMetadataClean
			if metadata {
				data = mustInjectRemoval(t, filled, jpegSegmentBytes(0xfe, []byte("fixture description")))
				state = JPEGMetadataRemovable
			}
			inspection := InspectJPEGMetadata(context.Background(), data)
			path := writeTempFile(t, "filled-end.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if inspection.State != state || inspection.Err != nil || err != nil || result.Committed != metadata || !bytes.Equal(filled, mustRead(t, path)) {
				t.Fatalf("EOI fill (metadata=%v) = %+v; %+v, %v", metadata, inspection, result, err)
			}
		}
	})
	plain := uitest.EncodeJPEG(t, 16, 12, color.White)
	data := append(bytes.Clone(plain), []byte("fixture trailer")...)
	path := writeTempFile(t, "repeat.jpg", data)
	u := storage.NewFileURI(path)
	if got := InspectJPEGMetadata(context.Background(), data); got.State != JPEGMetadataRemovable || got.Err != nil {
		t.Fatalf("inspection = %+v", got)
	}
	first, err := StripJPEGMetadataContext(context.Background(), u)
	if err != nil || !first.Committed {
		t.Fatalf("first = %+v, %v", first, err)
	}
	got := mustRead(t, path)
	if !bytes.Equal(plain, got) {
		t.Fatal("unexpected retained bytes")
	}
	if inspection := InspectJPEGMetadata(context.Background(), got); inspection.State != JPEGMetadataClean || inspection.Err != nil {
		t.Fatalf("clean inspection = %+v", inspection)
	}
	second, err := StripJPEGMetadataContext(context.Background(), u)
	if err != nil || second.Committed || !bytes.Equal(got, mustRead(t, path)) {
		t.Fatalf("second = %+v, %v", second, err)
	}
}

func TestJPEGMetadataRemovalProfiles(t *testing.T) {
	t.Run("standard optional camera profile fields", func(t *testing.T) {
		for _, version := range []string{"v2", "v4"} {
			base := removalFixture(t, "rgb-"+version+".icc")
			luminance := make([]byte, 20)
			copy(luminance, "XYZ ")
			binary.BigEndian.PutUint32(luminance[12:16], 80*65536)
			measurement := make([]byte, 36)
			copy(measurement, "meas")
			binary.BigEndian.PutUint32(measurement[8:12], 1)
			binary.BigEndian.PutUint32(measurement[24:28], 2)
			binary.BigEndian.PutUint32(measurement[28:32], 65536)
			binary.BigEndian.PutUint32(measurement[32:36], 2)
			technology := []byte("sig \x00\x00\x00\x00CRT ")
			tags := map[string][]byte{
				"lumi": luminance, "meas": measurement, "tech": technology,
				"vued": profileTestTags(t, base)["desc"],
			}
			for _, field := range []string{"attributes", "lumi", "meas", "tech", "vued", "all"} {
				t.Run(version+"/"+field, func(t *testing.T) {
					extra := make(map[string][]byte)
					for name, value := range tags {
						if field == name || field == "all" {
							extra[name] = value
						}
					}
					profile := removalProfileWithTags(t, base, extra)
					if field == "attributes" || field == "all" {
						binary.BigEndian.PutUint64(profile[56:64], 0x5052495600000005)
					}
					plain := removalFixture(t, "baseline-rgb.jpg")
					data := mustInjectRemoval(t, plain, profileSegments(profile)...)
					path := writeTempFile(t, "camera-profile.jpg", data)
					inspection := InspectJPEGMetadata(context.Background(), data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if inspection.State != JPEGMetadataRemovable || inspection.Err != nil || err != nil || !result.Committed {
						t.Fatalf("standard camera profile = %+v; %+v, %v", inspection, result, err)
					}
					output := mustRead(t, path)
					clean, primary := removalProfileParts(t, output)
					if !bytes.Equal(primary, plain) {
						t.Fatal("image bytes changed")
					}
					before, after := profileTestTags(t, profile), profileTestTags(t, clean)
					for name, value := range before {
						switch name {
						case "desc", "cprt", "dmnd", "dmdd", "vued":
						default:
							if !bytes.Equal(value, after[name]) {
								t.Fatalf("numerical/enumerated field %s changed", name)
							}
						}
					}
					if after["vued"] != nil || bytes.Contains(clean, []byte("PRIV")) || binary.BigEndian.Uint64(clean[56:64]) != binary.BigEndian.Uint64(profile[56:64])&15 {
						t.Fatal("private profile fields survived or standard media attributes changed")
					}
					if again := InspectJPEGMetadata(context.Background(), output); again.State != JPEGMetadataClean || again.Err != nil {
						t.Fatalf("output is not clean: %+v", again)
					}
				})
			}
		}
	})
	t.Run("sRGB EXIF with sRGB ICC", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		segments := append(profileSegments(removalFixture(t, "rgb-v4.icc")), removalColorEXIF(1, "R98", false))
		data := mustInjectRemoval(t, plain, segments...)
		inspection := InspectJPEGMetadata(context.Background(), data)
		if inspection.State != JPEGMetadataRemovable || inspection.Err != nil {
			t.Fatalf("ordinary sRGB JPEG cannot remove metadata: %+v", inspection)
		}
	})
	t.Run("malformed optional profile fields leave source untouched", func(t *testing.T) {
		base := removalFixture(t, "rgb-v2.icc")
		plain := removalFixture(t, "baseline-rgb.jpg")
		for _, tc := range []struct {
			name, tag, kind string
			size, offset    int
			value           uint32
		}{
			{"luminance length", "lumi", "XYZ ", 24, 12, 65536},
			{"luminance type", "lumi", "text", 20, 12, 65536},
			{"luminance X", "lumi", "XYZ ", 20, 8, 1},
			{"negative luminance", "lumi", "XYZ ", 20, 12, 0xffffffff},
			{"luminance Z", "lumi", "XYZ ", 20, 16, 1},
			{"measurement length", "meas", "meas", 40, 8, 1},
			{"measurement type", "meas", "text", 36, 8, 1},
			{"measurement reserved", "meas", "meas", 36, 4, 1},
			{"observer enum", "meas", "meas", 36, 8, 3},
			{"geometry enum", "meas", "meas", 36, 24, 3},
			{"flare range", "meas", "meas", 36, 28, 65537},
			{"illuminant enum", "meas", "meas", 36, 32, 9},
			{"technology length", "tech", "sig ", 16, 8, 0x43525420},
			{"technology type", "tech", "text", 12, 8, 0x43525420},
			{"unknown technology", "tech", "sig ", 12, 8, 0x50524956},
			{"view description", "vued", "desc", 12, 8, 0xffffffff},
		} {
			t.Run(tc.name, func(t *testing.T) {
				value := make([]byte, tc.size)
				copy(value, tc.kind)
				binary.BigEndian.PutUint32(value[tc.offset:], tc.value)
				profile := removalProfileWithTags(t, base, map[string][]byte{tc.tag: value})
				data := mustInjectRemoval(t, plain, profileSegments(profile)...)
				path := writeTempFile(t, "refused-profile.jpg", data)
				inspection := InspectJPEGMetadata(context.Background(), data)
				result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
				if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProfile) || !errors.Is(err, ErrJPEGMetadataProfile) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
					t.Fatalf("malformed profile refusal = %+v; %+v, %v", inspection, result, err)
				}
			})
		}
	})
	t.Run("filled JFIF retains normalized ICC through orientation", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		profile := removalFixture(t, "rgb-v4.icc")
		for _, orientation := range []uint16{1, 6} {
			segments := append(profileSegments(profile), wrapAsAPP1(buildExifSegment(t, orientation, false)))
			data := fillRemovalMarker(t, mustInjectRemoval(t, plain, segments...), 0xe0)
			path := writeTempFile(t, "filled-profile.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if err != nil || !result.Committed {
				t.Fatalf("filled profile orientation %d = %+v, %v", orientation, result, err)
			}
			output := mustRead(t, path)
			if got := InspectJPEGMetadata(context.Background(), output); got.State != JPEGMetadataClean || got.Err != nil {
				t.Fatalf("filled profile output = %+v", got)
			}
			// The fixture tag table is independent of marker fill and of
			// the tolerant export reader's handling of filled JFIF.
			start := bytes.Index(output, []byte("ICC_PROFILE\x00\x01\x01"))
			if start < 0 {
				t.Fatal("normalized profile was lost")
			}
			p := output[start+14:]
			p = p[:binary.BigEndian.Uint32(p[:4])]
			if !bytes.Equal(profileTestTags(t, profile)["rTRC"], profileTestTags(t, p)["rTRC"]) {
				t.Fatal("normalized profile transform changed")
			}
		}
	})
	t.Run("explicit EXIF color with a different ICC is preserved", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		profile := removalFixture(t, "rgb-v4.icc")
		// Derive a qualified, non-sRGB matrix by changing the red primary.
		red := profileTestTags(t, profile)["rXYZ"]
		binary.BigEndian.PutUint32(red[8:12], binary.BigEndian.Uint32(red[8:12])+2048)
		if got := InspectJPEGMetadata(context.Background(), mustInjectRemoval(t, plain, profileSegments(profile)...)); got.State != JPEGMetadataRemovable || got.Err != nil {
			t.Fatalf("independent ICC qualification = %+v", got)
		}
		for _, exifFirst := range []bool{false, true} {
			segments := profileSegments(profile)
			exif := removalColorEXIF(1, "R98", false)
			if exifFirst {
				segments = append([][]byte{exif}, segments...)
			} else {
				segments = append(segments, exif)
			}
			data := mustInjectRemoval(t, plain, segments...)
			inspection := InspectJPEGMetadata(context.Background(), data)
			path := writeTempFile(t, "conflicting-color.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if inspection.State != JPEGMetadataRemovable || inspection.Err != nil || err != nil || !result.Committed {
				t.Fatalf("EXIF/ICC preservation (EXIF first=%v) = %+v; %+v, %v", exifFirst, inspection, result, err)
			}
			got := mustRead(t, path)
			if (bytes.Index(got, []byte("Exif\x00\x00")) < bytes.Index(got, []byte("ICC_PROFILE\x00"))) != exifFirst {
				t.Fatal("relative order of rendering declarations changed")
			}
			if values := removalRenderingValues(t, got); values[2] != 1 || values[3] != binary.LittleEndian.Uint32([]byte("R98\x00")) {
				t.Fatalf("explicit EXIF color changed: %v", values)
			}
			if !bytes.Equal(profileTestTags(t, readRemovalProfile(t, got))["rXYZ"], red) {
				t.Fatal("retained ICC transform changed")
			}
		}
	})
	t.Run("chunks across scans assemble in sequence order", func(t *testing.T) {
		plain := removalFixture(t, "progressive-rgb.jpg")
		profile := removalFixture(t, "rgb-v4.icc")
		middle := len(profile) / 2
		last := jpegSegmentBytes(0xe2, append([]byte("ICC_PROFILE\x00\x02\x02"), profile[middle:]...))
		first := jpegSegmentBytes(0xe2, append([]byte("ICC_PROFILE\x00\x01\x02"), profile[:middle]...))
		data := mustInjectRemoval(t, plain, last)
		data = append(append(bytes.Clone(data[:len(data)-2]), first...), 0xff, 0xd9)
		path := writeTempFile(t, "chunks.jpg", data)
		result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
		if err != nil || !result.Committed {
			t.Fatalf("assembled removal = %+v, %v", result, err)
		}
		gotProfile, gotPrimary := removalProfileParts(t, mustRead(t, path))
		if !bytes.Equal(plain, gotPrimary) || !bytes.Equal(profileTestTags(t, profile)["rTRC"], profileTestTags(t, gotProfile)["rTRC"]) {
			t.Fatal("assembled output changed image or transform")
		}
	})
	t.Run("unqualified profiles and incomplete assembly leave sources untouched", func(t *testing.T) {
		plain := removalFixture(t, "baseline-rgb.jpg")
		profile := removalFixture(t, "rgb-v4.icc")
		unknown := bytes.Clone(profile)
		copy(unknown[132:136], "zzzz")
		badVersion := bytes.Clone(profile)
		badVersion[9] = 0x1f
		reservedAttributes := bytes.Clone(profile)
		binary.BigEndian.PutUint32(reservedAttributes[60:64], 16)
		wrongModel := removalFixture(t, "gray-v4.icc")
		for name, segments := range map[string][][]byte{
			"unknown transform":         profileSegments(unknown),
			"invalid version":           profileSegments(badVersion),
			"reserved media attributes": profileSegments(reservedAttributes),
			"model mismatch":            profileSegments(wrongModel),
			"duplicate chunks":          {profileSegments(profile)[0], profileSegments(profile)[0]},
			"missing chunk":             {jpegSegmentBytes(0xe2, append([]byte("ICC_PROFILE\x00\x01\x02"), profile...))},
		} {
			t.Run(name, func(t *testing.T) {
				data := mustInjectRemoval(t, plain, segments...)
				path := writeTempFile(t, "refused.jpg", data)
				result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
				if !errors.Is(err, ErrJPEGMetadataProfile) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
					t.Fatalf("profile refusal = %+v, %v", result, err)
				}
			})
		}
	})
	for _, model := range []string{"rgb", "gray"} {
		for _, version := range []string{"v2", "v4"} {
			name := model + "-" + version
			processes := []string{"baseline", "progressive"}
			if model == "rgb" {
				processes = append(processes, "multiscan")
			}
			for _, class := range []string{"mntr", "scnr"} {
				for _, process := range processes {
					t.Run(name+" "+class+" "+process, func(t *testing.T) {
						profile := removalFixture(t, name+".icc")
						copy(profile[12:16], class)
						plain := removalFixture(t, process+"-"+model+".jpg")
						data := mustInjectRemoval(t, plain, profileSegments(profile)...)
						path := writeTempFile(t, "profile.jpg", data)
						result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
						if err != nil || !result.Committed {
							t.Fatalf("profile removal = %+v, %v", result, err)
						}
						got := mustRead(t, path)
						cleanProfile, primary := removalProfileParts(t, got)
						if !bytes.Equal(plain, primary) {
							t.Fatal("profile removal changed primary coded data or retained other metadata")
						}
						beforePixels, err := jpeg.Decode(bytes.NewReader(plain))
						if err != nil {
							t.Fatal(err)
						}
						afterPixels, err := jpeg.Decode(bytes.NewReader(got))
						if err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(beforePixels, afterPixels) {
							t.Fatal("profile removal changed decoded pixels")
						}
						beforeTags, afterTags := profileTestTags(t, profile), profileTestTags(t, cleanProfile)
						for _, tag := range []string{"rXYZ", "gXYZ", "bXYZ", "rTRC", "gTRC", "bTRC", "kTRC", "wtpt", "chad", "chrm"} {
							if !bytes.Equal(beforeTags[tag], afterTags[tag]) {
								t.Fatalf("color-transform tag %q changed", tag)
							}
						}
						if bytes.Contains(cleanProfile, []byte("PicFetch")) || bytes.Contains(cleanProfile, []byte{0, 'P', 0, 'i', 0, 'c', 0, 'F'}) {
							t.Fatal("original profile description survived")
						}
						if inspection := InspectJPEGMetadata(context.Background(), got); inspection.State != JPEGMetadataClean {
							t.Fatalf("normalized profile not clean: %+v", inspection)
						}
						again, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
						if err != nil || again.Committed || !bytes.Equal(got, mustRead(t, path)) {
							t.Fatalf("repeat = %+v, %v", again, err)
						}
						if dir := os.Getenv("PICFETCH_ICC_EVIDENCE"); dir != "" {
							if err := os.MkdirAll(dir, 0o700); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(dir, name+"-"+class+"-source.icc"), profile, 0o600); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(dir, name+"-"+class+".icc"), cleanProfile, 0o600); err != nil {
								t.Fatal(err)
							}
						}
					})
				}
			}
		}
	}
}

func TestJPEGMetadataRemovalCameraDeclarations(t *testing.T) {
	for _, name := range []string{"baseline-rgb", "progressive-rgb", "multiscan-rgb"} {
		for orientation := uint16(1); orientation <= 8; orientation++ {
			for _, positioning := range []uint32{1, 2} {
				for _, space := range []uint16{1, 0xffff} {
					for _, version := range []string{"none", "v2", "v4"} {
						t.Run(name+"/orientation="+strconv.Itoa(int(orientation))+"/positioning="+strconv.Itoa(int(positioning))+"/space="+strconv.Itoa(int(space))+"/"+version, func(t *testing.T) {
							plain := removalFixture(t, name+".jpg")
							index := "R98"
							if space == 0xffff {
								index = "R03"
							}
							colorTIFF := removalColorEXIF(space, index, false)[10:]
							root := buildIFD0TIFF(t,
								tiffEntry{tag: 0x0112, typ: 3, count: 1, value: uint32(orientation)},
								tiffEntry{tag: 0x0213, typ: 3, count: 1, value: positioning},
								tiffEntry{tag: 0x8769, typ: 4, count: 1, value: 50})
							tiff := append(root, colorTIFF[26:]...)
							binary.LittleEndian.PutUint32(tiff[72:76], 80)
							tiff = append(tiff, []byte("private camera identity and thumbnail")...)
							segments := [][]byte{jpegSegmentBytes(0xe1, append([]byte("Exif\x00\x00"), tiff...))}
							if version != "none" {
								segments = append(segments, profileSegments(removalFixture(t, "rgb-"+version+".icc"))...)
							}
							data := append(mustInjectRemoval(t, plain, segments...), []byte("private trailing preview")...)
							path := writeTempFile(t, "camera.jpg", data)
							if inspection := InspectJPEGMetadata(context.Background(), data); inspection.State != JPEGMetadataRemovable || inspection.Err != nil {
								t.Fatalf("camera JPEG has no removal action: %+v", inspection)
							}
							result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
							if err != nil || !result.Committed {
								t.Fatalf("camera removal = %+v, %v", result, err)
							}
							got := mustRead(t, path)
							if jpegEXIFOrientation(got) != int(orientation) || bytes.Contains(got, []byte("private")) || !ReadMetadata(got).Empty() {
								t.Fatal("rendering orientation or privacy was lost")
							}
							wantRendering := [4]uint32{uint32(orientation), positioning, uint32(space), binary.LittleEndian.Uint32(append([]byte(index), 0))}
							if values := removalRenderingValues(t, got); values != wantRendering {
								t.Fatalf("rendering declarations = %v, want %v", values, wantRendering)
							}
							if !bytes.Equal(removalImageBytes(t, got), plain) {
								t.Fatal("encoded primary image data changed")
							}
							before, err := jpeg.Decode(bytes.NewReader(data))
							if err != nil {
								t.Fatal(err)
							}
							after, err := jpeg.Decode(bytes.NewReader(got))
							if err != nil || !reflect.DeepEqual(before, after) {
								t.Fatalf("camera pixels changed: %v", err)
							}
							if inspection := InspectJPEGMetadata(context.Background(), got); inspection.State != JPEGMetadataClean || inspection.Err != nil {
								t.Fatalf("sanitized camera JPEG is not clean: %+v", inspection)
							}
							again, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
							if err != nil || again.Committed || !bytes.Equal(got, mustRead(t, path)) {
								t.Fatalf("repeated removal = %+v, %v", again, err)
							}
						})
					}
				}
			}
		}
	}
}

// Read the rebuilt declarations through the existing TIFF consumer, separately
// from the removal parser. Unknown output tags are a privacy failure.
func removalRenderingValues(t *testing.T, data []byte) [4]uint32 {
	t.Helper()
	values := [4]uint32{1, 1, 0, 0}
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker != 0xe1 {
			return true
		}
		if len(payload) < 14 || !bytes.HasPrefix(payload, []byte("Exif\x00\x00")) || len(payload) > 104 {
			t.Fatal("unexpected retained EXIF payload")
		}
		tiff := payload[6:]
		bo, ok := tiffOrder(tiff)
		if !ok {
			t.Fatal("invalid output TIFF")
		}
		offsets := [3]uint32{bo.Uint32(tiff[4:8])}
		for level := range offsets {
			if offsets[level] == 0 {
				continue
			}
			walkIFD(tiff, bo, offsets[level], func(tag, kind uint16, value []byte) {
				switch {
				case level == 0 && tag == 0x0112 && kind == 3:
					values[0] = uint32(bo.Uint16(value))
				case level == 0 && tag == 0x0213 && kind == 3:
					values[1] = uint32(bo.Uint16(value))
				case level == 1 && tag == 0xa001 && kind == 3:
					values[2] = uint32(bo.Uint16(value))
				case level == 2 && tag == 1 && kind == 2:
					values[3] = binary.LittleEndian.Uint32(value)
				case level == 0 && tag == 0x8769 && kind == 4, level == 1 && tag == 0xa005 && kind == 4:
					offsets[level+1] = bo.Uint32(value)
				default:
					t.Fatalf("unexpected output tag %#x in directory %d", tag, level)
				}
			})
		}
		return true
	})
	return values
}

func removalImageBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	out := bytes.Clone(data[:2])
	for pos := 2; pos+4 <= len(data); {
		if data[pos] != 0xff {
			t.Fatal("invalid output header")
		}
		marker := data[pos+1]
		if marker == 0xda {
			return append(out, data[pos:]...)
		}
		length := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if length < 2 || pos+2+length > len(data) {
			t.Fatal("invalid output segment")
		}
		if marker != 0xe1 && marker != 0xe2 {
			out = append(out, data[pos:pos+2+length]...)
		}
		pos += 2 + length
	}
	t.Fatal("output contains no scan")
	return nil
}

func TestJPEGMetadataRemovalPhotoMemory(t *testing.T) {
	// Admission inspects headers only. Use independent baseline/progressive
	// headers at camera dimensions; their tiny scans are deliberately not decoded.
	for _, name := range []string{"baseline-rgb.jpg", "entropy-progressive-420.jpg"} {
		for _, tc := range []struct {
			width, height uint16
			allow         bool
		}{
			{6000, 4000, true},
			{18000, 10000, false},
		} {
			t.Run(name+"/"+strconv.Itoa(int(tc.width)), func(t *testing.T) {
				data := removalFixture(t, name)
				marker := byte(0xc0)
				if name != "baseline-rgb.jpg" {
					marker = 0xc2
				}
				at := bytes.Index(data, []byte{0xff, marker})
				if at < 0 {
					t.Fatal("fixture has no frame")
				}
				binary.BigEndian.PutUint16(data[at+5:at+7], tc.height)
				binary.BigEndian.PutUint16(data[at+7:at+9], tc.width)
				// Reserve the encoded storage of a normal 10 MiB camera file.
				data = append(data, make([]byte, 10*1024*1024-len(data))...)
				ctx := &jpegDecodeObserveContext{Context: context.Background()}
				memory, err := jpegRemovalAdmission(ctx, data)
				if ctx.decoded || tc.allow && (err != nil || memory.working > jpegRemovalWorkingBytes) || !tc.allow && !errors.Is(err, ErrJPEGMetadataMemory) {
					t.Fatalf("photo admission: %+v, %v, decode=%v", memory, err, ctx.decoded)
				}
			})
		}
	}
}

func profileSegments(profile []byte) [][]byte {
	return [][]byte{jpegSegmentBytes(0xe2, append([]byte("ICC_PROFILE\x00\x01\x01"), profile...))}
}

func readRemovalProfile(t *testing.T, data []byte) []byte {
	t.Helper()
	profile, _ := removalProfileParts(t, data)
	return profile
}

func removalProfileParts(t *testing.T, data []byte) ([]byte, []byte) {
	t.Helper()
	var result []byte
	primary := bytes.Clone(data[:2])
	for pos := 2; pos+4 <= len(data); {
		if data[pos] != 0xff {
			t.Fatal("invalid output header")
		}
		marker := data[pos+1]
		if marker == 0xda {
			primary = append(primary, data[pos:]...)
			break
		}
		n := int(binary.BigEndian.Uint16(data[pos+2:]))
		if n < 2 || pos+2+n > len(data) {
			t.Fatal("invalid output segment")
		}
		payload := data[pos+4 : pos+2+n]
		if marker == 0xe2 {
			if len(payload) < 14 || !bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00")) {
				t.Fatal("unexpected APP2 retained")
			}
			result = append(result, payload[14:]...)
		} else {
			primary = append(primary, data[pos:pos+2+n]...)
		}
		pos += 2 + n
	}
	if len(result) == 0 {
		t.Fatal("no retained profile")
	}
	return result, primary
}

func profileTestTags(t *testing.T, p []byte) map[string][]byte {
	t.Helper()
	if len(p) < 132 || int(binary.BigEndian.Uint32(p)) != len(p) {
		t.Fatal("invalid profile size")
	}
	n := int(binary.BigEndian.Uint32(p[128:]))
	if n > (len(p)-132)/12 {
		t.Fatal("invalid tag table")
	}
	tags := make(map[string][]byte)
	for i := 0; i < n; i++ {
		e := p[132+i*12 : 144+i*12]
		start, size := int(binary.BigEndian.Uint32(e[4:])), int(binary.BigEndian.Uint32(e[8:]))
		if start < 132+n*12 || size < 8 || start > len(p)-size {
			t.Fatal("invalid tag range")
		}
		tags[string(e[:4])] = p[start : start+size]
	}
	return tags
}

func removalProfileWithTags(t *testing.T, source []byte, extra map[string][]byte) []byte {
	t.Helper()
	tags := profileTestTags(t, source)
	for name, value := range extra {
		tags[name] = value
	}
	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}
	slices.Sort(names)
	profile := make([]byte, 132+12*len(names))
	copy(profile[:128], source[:128])
	binary.BigEndian.PutUint32(profile[128:132], uint32(len(names)))
	for i, name := range names {
		entry := profile[132+12*i : 144+12*i]
		copy(entry[:4], name)
		binary.BigEndian.PutUint32(entry[4:8], uint32(len(profile)))
		binary.BigEndian.PutUint32(entry[8:12], uint32(len(tags[name])))
		profile = append(profile, tags[name]...)
		for len(profile)%4 != 0 {
			profile = append(profile, 0)
		}
	}
	binary.BigEndian.PutUint32(profile[:4], uint32(len(profile)))
	return profile
}

func fillRemovalMarker(t *testing.T, data []byte, marker byte) []byte {
	t.Helper()
	pos := bytes.Index(data, []byte{0xff, marker})
	if pos < 0 {
		t.Fatalf("fixture has no marker %#x", marker)
	}
	out := append(bytes.Clone(data[:pos]), 0xff)
	return append(out, data[pos:]...)
}
