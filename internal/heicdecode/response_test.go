package heicdecode

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"testing"
)

func TestResponseOperationsAndFailures(t *testing.T) {
	limits := DefaultLimits(0)
	for _, op := range []Operation{Decode, DecodeConfig, DecodeExif} {
		result := Response{Metadata: &Metadata{Orientation: 6, Make: "Camera"}}
		if op != DecodeExif {
			result.Config = image.Config{Width: 1, Height: 1, ColorModel: color.NRGBA64Model}
		}
		if op == Decode {
			result.Image = &image.NRGBA64{Pix: []byte{1, 2, 3, 4, 5, 6, 7, 8}, Stride: 8, Rect: image.Rect(0, 0, 1, 1)}
		}
		var encoded bytes.Buffer
		if err := WriteResponse(&encoded, op, result, limits); err != nil {
			t.Fatal(err)
		}
		got, err := ReadResponse(&encoded, op, limits)
		if err != nil || got.Metadata.Make != "Camera" || (got.Image == nil) != (op != Decode) {
			t.Fatalf("operation %d = %+v, %v", op, got, err)
		}
	}
	var encoded bytes.Buffer
	if err := WriteFailure(&encoded, StatusUnsupported, "unsupported", limits); err != nil {
		t.Fatal(err)
	}
	_, err := ReadResponse(&encoded, Decode, limits)
	var failure *Failure
	if !errors.As(err, &failure) || failure.Status != StatusUnsupported {
		t.Fatalf("failure = %v", err)
	}
}
