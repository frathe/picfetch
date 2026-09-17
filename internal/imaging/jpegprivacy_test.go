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
	"strconv"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestJPEGMetadataRemovalPrivacy(t *testing.T) {
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
				{"Adobe RGB", 0xffff, "R03", false},
				{"uncalibrated", 0xffff, "R98", false},
				{"conflicting interoperability", 1, "R03", false},
			} {
				t.Run(tc.name+" big-endian="+strconv.FormatBool(bigEndian), func(t *testing.T) {
					data := mustInjectRemoval(t, plain, removalColorEXIF(tc.space, tc.index, bigEndian))
					inspection := InspectJPEGMetadata(context.Background(), data)
					path := writeTempFile(t, "color.jpg", data)
					result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
					if tc.allow {
						if inspection.State != JPEGMetadataRemovable || inspection.Err != nil || err != nil || !result.Committed || !bytes.Equal(plain, mustRead(t, path)) {
							t.Fatalf("sRGB removal = %+v; %+v, %v", inspection, result, err)
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
			data := mustInjectRemoval(t, plain, jpegSegmentBytes(0xe0, jfif), wrapAsAPP1(buildExifSegment(t, orientation, false)))
			path := writeTempFile(t, "aspect.jpg", data)
			result, err := StripJPEGMetadataContext(context.Background(), storage.NewFileURI(path))
			if err != nil || !result.Committed {
				t.Fatalf("orientation = %+v, %v", result, err)
			}
			if orientation == 6 {
				jfif[9], jfif[11] = 1, 2
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
						width, height := 16, 12
						if orientation >= 5 {
							width, height = 12, 16
						}
						if after.Bounds().Dx() != width || after.Bounds().Dy() != height {
							t.Fatalf("orientation bounds %v", after.Bounds())
						}
						// Compare each output sample against the independently mapped
						// source coordinate with JPEG's lossy tolerance.
						var total uint64
						for y := 0; y < height; y++ {
							for x := 0; x < width; x++ {
								sx, sy := x, y
								switch orientation {
								case 2:
									sx, sy = 15-x, y
								case 3:
									sx, sy = 15-x, 11-y
								case 4:
									sx, sy = x, 11-y
								case 5:
									sx, sy = y, x
								case 6:
									sx, sy = y, 11-x
								case 7:
									sx, sy = 15-y, 11-x
								case 8:
									sx, sy = 15-y, x
								}
								r, g, b, _ := before.At(sx, sy).RGBA()
								rr, gg, bb, _ := after.At(x, y).RGBA()
								for _, pair := range [][2]uint32{{r, rr}, {g, gg}, {b, bb}} {
									if pair[0] > pair[1] {
										total += uint64(pair[0] - pair[1])
									} else {
										total += uint64(pair[1] - pair[0])
									}
								}
							}
						}
						if float64(total)/(16*12*3*257) > 12 {
							t.Fatal("orientation exceeded 12/255 mean JPEG sample tolerance")
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
			data := mustInjectRemoval(t, plain, jpegSegmentBytes(0xe1, append([]byte("Exif\x00\x00"), tiff...)))
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
		inspection := InspectJPEGMetadata(ctx, encoded.Bytes())
		if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataMemory) || ctx.decoded {
			t.Fatalf("memory admission = %+v, full decode started=%v", inspection, ctx.decoded)
		}
		path := writeTempFile(t, "large-photo.jpg", encoded.Bytes())
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !errors.Is(err, ErrJPEGMetadataMemory) || result.Committed || ctx.decoded || !bytes.Equal(encoded.Bytes(), mustRead(t, path)) {
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
	t.Run("cancellation during pixel orientation", func(t *testing.T) {
		data := mustInjectRemoval(t, uitest.EncodeJPEG(t, 512, 512, color.White), wrapAsAPP1(buildExifSegment(t, 6, false)))
		path := writeTempFile(t, "cancel-orientation.jpg", data)
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		// Give reading/parsing the small encoded fixture ample work, then
		// cancel only if the pixel-heavy transformation observes the context.
		ctx := &groupingCancelContext{Context: base, cancel: cancel, after: 100}
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !errors.Is(err, context.Canceled) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
			t.Fatalf("pixel-work cancellation = %+v, %v", result, err)
		}
	})
	t.Run("cancellation during JPEG encoding", func(t *testing.T) {
		data := mustInjectRemoval(t, uitest.EncodeJPEG(t, 512, 512, color.White), wrapAsAPP1(buildExifSegment(t, 6, false)))
		path := writeTempFile(t, "cancel-encoding.jpg", data)
		base, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx := &jpegEncodeCancelContext{Context: base, cancel: cancel}
		result, err := StripJPEGMetadataContext(ctx, storage.NewFileURI(path))
		if !ctx.observed || !errors.Is(err, context.Canceled) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
			t.Fatalf("encoder cancellation observed=%v result=%+v error=%v", ctx.observed, result, err)
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

// Cancellation is injected at the standard-library encoding boundary so this
// public mutation test needs neither timing assumptions nor a production hook.
type jpegEncodeCancelContext struct {
	context.Context
	cancel   context.CancelFunc
	observed bool
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
}

func (c *jpegDecodeObserveContext) Err() error {
	var callers [32]uintptr
	frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
	for {
		frame, more := frames.Next()
		if frame.Function == "image/jpeg.Decode" {
			c.decoded = true
			break
		}
		if !more {
			break
		}
	}
	return c.Context.Err()
}

func (c *jpegEncodeCancelContext) Err() error {
	var callers [32]uintptr
	frames := runtime.CallersFrames(callers[:runtime.Callers(2, callers[:])])
	for {
		frame, more := frames.Next()
		if frame.Function == "image/jpeg.Encode" {
			c.observed = true
			c.cancel()
			break
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
	t.Run("explicit EXIF color with ICC is refused", func(t *testing.T) {
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
			if inspection.State != JPEGMetadataUnsupported || !errors.Is(inspection.Err, ErrJPEGMetadataProcess) || !errors.Is(err, ErrJPEGMetadataProcess) || result.Committed || !bytes.Equal(data, mustRead(t, path)) {
				t.Fatalf("EXIF/ICC refusal (EXIF first=%v) = %+v; %+v, %v", exifFirst, inspection, result, err)
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
		wrongModel := removalFixture(t, "gray-v4.icc")
		for name, segments := range map[string][][]byte{
			"unknown transform": profileSegments(unknown),
			"invalid version":   profileSegments(badVersion),
			"model mismatch":    profileSegments(wrongModel),
			"duplicate chunks":  {profileSegments(profile)[0], profileSegments(profile)[0]},
			"missing chunk":     {jpegSegmentBytes(0xe2, append([]byte("ICC_PROFILE\x00\x01\x02"), profile...))},
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
