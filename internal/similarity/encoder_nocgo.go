//go:build !cgo

package similarity

import (
	"context"
	"fmt"
	"image"
)

type Encoder struct{}

func NewEncoder(_, _ string) (*Encoder, error) {
	return nil, fmt.Errorf("local similarity inference requires a build with cgo enabled")
}

func (_ *Encoder) Encode(ctx context.Context, _ image.Image) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("local similarity inference requires a build with cgo enabled")
}

func (_ *Encoder) Close() {}
