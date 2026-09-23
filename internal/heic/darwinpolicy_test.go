package heic

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestHEICDarwinAlphaPreparation(t *testing.T) {
	for _, bits := range []string{"8", "10"} {
		for _, alpha := range []string{"straight", "premultiplied"} {
			t.Run(alpha+bits, func(t *testing.T) {
				data, err := os.ReadFile("testdata/alpha-" + alpha + bits + ".heic")
				if err != nil {
					t.Fatal(err)
				}
				original := bytes.Clone(data)
				prepared, changed, err := darwinAlphaInput(data)
				if err != nil || changed != (alpha == "premultiplied") {
					t.Fatalf("alpha preparation changed=%t error=%v", changed, err)
				}
				if !bytes.Equal(data, original) || len(prepared) != len(data) {
					t.Fatal("preparation changed source bytes or container offsets")
				}
				media := bytes.Index(data, []byte("mdat"))
				if media < 0 || !bytes.Equal(data[media:], prepared[media:]) {
					t.Fatal("preparation changed the encoded image payload")
				}
				metadata, err := primaryAlpha(prepared)
				if err != nil || metadata.primary != 1 || metadata.item != 2 || metadata.premultiplied {
					t.Fatalf("prepared alpha association = %+v, %v", metadata, err)
				}
				if _, err := inspectContainer(prepared); err != nil {
					t.Fatalf("preparation damaged primary properties: %v", err)
				}
			})
		}
	}
	data, err := os.ReadFile("testdata/alpha-premultiplied8.heic")
	if err != nil {
		t.Fatal(err)
	}
	prem := bytes.Index(data, []byte("prem"))
	binary.BigEndian.PutUint16(data[prem+6:], 65535)
	if _, _, err := darwinAlphaInput(data); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed alpha reference = %v, want invalid", err)
	}
	pixels := []byte{17, 8, 4, 0, 80, 40, 20, 128, 120, 60, 30, 192, 160, 80, 40, 255}
	undoPremultiplication(pixels)
	if !bytes.Equal(pixels, []byte{0, 0, 0, 0, 159, 80, 40, 128, 159, 80, 40, 192, 160, 80, 40, 255}) {
		t.Fatalf("restored straight-alpha channels = %v", pixels)
	}
}

func TestHEICDarwinProfileAdmission(t *testing.T) {
	data := []byte(probe10)
	nclx := bytes.Index(data, []byte("nclx"))
	if nclx < 0 {
		t.Fatal("the project-owned fixture has no nclx profile")
	}
	// The native renderer owns color interpretation, including PQ sources.
	// Color fidelity is best effort and must not become an admission gate.
	binary.BigEndian.PutUint16(data[nclx+6:], 16)
	if err := validateDarwinContainer(data); err != nil {
		t.Fatalf("PQ source must reach ImageIO: %v", err)
	}
}

func TestHEICDarwinDepthAdmission(t *testing.T) {
	for _, fixture := range []string{probe8, probe10} {
		if err := validateDarwinContainer([]byte(fixture)); err != nil {
			t.Fatalf("project-owned 8/10-bit SDR fixture refused: %v", err)
		}
	}
	data := []byte(probe10)
	configuration := bytes.Index(data, []byte("hvcC")) + 4
	data[configuration+17] = data[configuration+17]&0xf8 | 4
	data[configuration+18] = data[configuration+18]&0xf8 | 4
	if err := validateDarwinContainer(data); err != nil {
		t.Fatalf("native renderer must decide 12-bit support: %v", err)
	}
}

func TestHEICDarwinPrimaryPropertyAdmission(t *testing.T) {
	// Item one is AV1 with PQ; item two is HEVC with explicit sRGB. The
	// container declares item two as primary. Non-primary profile properties
	// must neither reject the selected still nor substitute for its profile.
	associations := []byte{0, 0, 0, 0, 0, 0, 0, 2, 0, 1, 1, 1, 0, 2, 2, 2, 3}
	data := darwinPolicyFixture(2, associations)
	if err := validateDarwinContainer(data); err != nil {
		t.Fatalf("designated sRGB HEVC primary was refused: %v", err)
	}
	if err := validateDarwinContainer(darwinPolicyFixture(1, associations)); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("non-HEVC designated primary admission = %v", err)
	}
	t.Run("wrong_property_index", func(t *testing.T) {
		bad := append([]byte(nil), associations...)
		bad[len(bad)-1] = 127
		if err := validateDarwinContainer(darwinPolicyFixture(2, bad)); !errors.Is(err, ErrInvalid) {
			t.Fatalf("out-of-range property association = %v", err)
		}
	})
	t.Run("duplicate_property_index", func(t *testing.T) {
		bad := append([]byte(nil), associations...)
		bad[len(bad)-1] = 2
		if err := validateDarwinContainer(darwinPolicyFixture(2, bad)); !errors.Is(err, ErrInvalid) {
			t.Fatalf("duplicate property association = %v", err)
		}
	})
	t.Run("truncated_association", func(t *testing.T) {
		if err := validateDarwinContainer(darwinPolicyFixture(2, associations[:len(associations)-1])); !errors.Is(err, ErrInvalid) {
			t.Fatalf("truncated property association = %v", err)
		}
	})
}

// darwinPolicyFixture contains no HEVC payload. It exercises container policy,
// never claims to be a decoder fixture, and cannot qualify ImageIO pixels.
func darwinPolicyFixture(primary uint16, associations []byte) []byte {
	box := func(kind string, payload []byte) []byte {
		result := make([]byte, len(payload)+8)
		binary.BigEndian.PutUint32(result, uint32(len(result)))
		copy(result[4:], kind)
		copy(result[8:], payload)
		return result
	}
	item := func(id uint16, kind string) []byte {
		payload := []byte{2, 0, 0, 0, byte(id >> 8), byte(id), 0, 0}
		payload = append(payload, []byte(kind)...)
		payload = append(payload, []byte(fmt.Sprintf("item%d\x00", id))...)
		return box("infe", payload)
	}
	info := []byte{0, 0, 0, 0, 0, 2}
	info = append(info, item(1, "av01")...)
	info = append(info, item(2, "hvc1")...)
	pq := []byte{'n', 'c', 'l', 'x', 0, 1, 0, 16, 0, 1, 0}
	srgb := []byte{'n', 'c', 'l', 'x', 0, 1, 0, 13, 0, 1, 0}
	configuration := make([]byte, 23)
	configuration[0], configuration[17], configuration[18] = 1, 0xf8, 0xf8
	properties := append(box("colr", pq), box("colr", srgb)...)
	properties = append(properties, box("hvcC", configuration)...)
	propertyBox := append(box("ipco", properties), box("ipma", associations)...)
	meta := []byte{0, 0, 0, 0}
	meta = append(meta, box("pitm", []byte{0, 0, 0, 0, byte(primary >> 8), byte(primary)})...)
	meta = append(meta, box("iinf", info)...)
	meta = append(meta, box("iprp", propertyBox)...)
	return append(box("ftyp", []byte("heic\x00\x00\x00\x00heicmif1")), box("meta", meta)...)
}
