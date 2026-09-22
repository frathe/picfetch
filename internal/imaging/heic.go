package imaging

import (
	"context"
	"fmt"
	"image"

	"github.com/frathe/picfetch/internal/heic"
)

// MaxImagePixels is the canonical decode ceiling passed to private workers.
func MaxImagePixels() int64 { return maxImagePixels }

// ReadMetadataContext reads metadata using the operation's captured capability.
func ReadMetadataContext(ctx context.Context, data []byte) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if heic.IsData(data) {
		result, err := readHEIC(ctx, data, false)
		if err != nil {
			return Metadata{}, err
		}
		return parseExifMetadata(result.EXIF), nil
	}
	return ReadMetadata(data), nil
}

func readHEIC(ctx context.Context, data []byte, pixels bool) (heic.Result, error) {
	if err := ctx.Err(); err != nil {
		return heic.Result{}, err
	}
	snapshot := heic.FromContext(ctx)
	if !snapshot.Available || snapshot.Backend == nil {
		return heic.Result{}, heic.ErrUnavailable
	}
	if int64(len(data)) > MaxEncodedBytes() {
		return heic.Result{}, &InputTooLargeError{limit: MaxEncodedBytes()}
	}
	result, err := snapshot.Backend.Read(ctx, data, heic.Request{Pixels: pixels, MaxEncodedBytes: MaxEncodedBytes(), MaxPixels: maxImagePixels})
	if ctx.Err() != nil {
		return heic.Result{}, ctx.Err()
	}
	if err != nil {
		return heic.Result{}, err
	}
	if err := checkDimensions(result.Width, result.Height); err != nil {
		return heic.Result{}, fmt.Errorf("%w: %v", heic.ErrProtocol, err)
	}
	if len(result.EXIF) > 1024*1024 || len(result.Provider) > 4096 {
		return heic.Result{}, fmt.Errorf("%w: oversized metadata", heic.ErrProtocol)
	}
	if pixels {
		if result.Stride != result.Width*4 || int64(len(result.Pixels)) != int64(result.Stride)*int64(result.Height) {
			return heic.Result{}, fmt.Errorf("%w: pixel layout", heic.ErrProtocol)
		}
	} else if len(result.Pixels) != 0 || (result.Stride != 0 && result.Stride != result.Width*4) {
		return heic.Result{}, fmt.Errorf("%w: unexpected probe pixels", heic.ErrProtocol)
	}
	return result, nil
}

func decodeHEIC(ctx context.Context, data []byte) (*LoadedImage, error) {
	result, err := readHEIC(ctx, data, true)
	if err != nil {
		return nil, err
	}
	pixels := &image.NRGBA{Pix: result.Pixels, Stride: result.Stride, Rect: image.Rect(0, 0, result.Width, result.Height)}
	return &LoadedImage{Frames: []image.Image{pixels}, HasEXIF: len(result.EXIF) != 0}, nil
}
