//go:build cgo

package similarity

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	ort "github.com/yalue/onnxruntime_go"
	"golang.org/x/image/draw"
)

type Encoder struct {
	session *ort.AdvancedSession
	input   *ort.Tensor[float32]
	output  *ort.Tensor[float32]
}

func NewEncoder(assets, provider string) (_ *Encoder, err error) {
	// The API opt-out is too late for initialization telemetry in official
	// native builds. Disable the uploader before loading the runtime library.
	if err := os.Setenv("ORT_DISABLE_TELEMETRY", "1"); err != nil {
		return nil, fmt.Errorf("disable runtime telemetry: %w", err)
	}
	asset, err := currentRuntime()
	if err != nil {
		return nil, err
	}
	ort.SetSharedLibraryPath(filepath.Join(assets, asset.directory, asset.library))
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, err
	}
	e := &Encoder{}
	defer func() {
		if err != nil {
			e.Close()
		}
	}()
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	defer func() { _ = options.Destroy() }()
	if err = options.SetIntraOpNumThreads(6); err != nil {
		return nil, err
	}
	if provider == "coreml" {
		err = options.AppendExecutionProviderCoreMLV2(map[string]string{"ModelFormat": "MLProgram", "MLComputeUnits": "ALL", "RequireStaticInputShapes": "1"})
		if err != nil {
			return nil, err
		}
	}
	e.input, err = ort.NewEmptyTensor[float32](ort.NewShape(1, 3, 224, 224))
	if err != nil {
		return nil, err
	}
	e.output, err = ort.NewEmptyTensor[float32](ort.NewShape(1, 768))
	if err != nil {
		return nil, err
	}
	e.session, err = ort.NewAdvancedSession(filepath.Join(assets, "vision_model.onnx"),
		[]string{"pixel_values"}, []string{"pooler_output"}, []ort.Value{e.input}, []ort.Value{e.output}, options)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Encoder) Close() {
	if e.session != nil {
		_ = e.session.Destroy()
	}
	if e.output != nil {
		_ = e.output.Destroy()
	}
	if e.input != nil {
		_ = e.input.Destroy()
	}
	_ = ort.DestroyEnvironment()
}

func (e *Encoder) Encode(ctx context.Context, source image.Image) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The pinned processor specifies bilinear resize to 224x224, rescale by
	// 1/255, and mean/std 0.5. Resize full oriented pixels, never cached thumbs.
	pixels := image.NewRGBA(image.Rect(0, 0, 224, 224))
	draw.Draw(pixels, pixels.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.BiLinear.Scale(pixels, pixels.Bounds(), source, source.Bounds(), draw.Over, nil)
	input := e.input.GetData()
	for y := range 224 {
		for x := range 224 {
			c := pixels.RGBAAt(x, y)
			i := y*224 + x
			input[i] = float32(c.R)/127.5 - 1
			input[224*224+i] = float32(c.G)/127.5 - 1
			input[2*224*224+i] = float32(c.B)/127.5 - 1
		}
	}
	if err := e.session.Run(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	vector := append([]float32(nil), e.output.GetData()...)
	var norm float64
	for _, v := range vector {
		norm += float64(v) * float64(v)
	}
	if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
		return nil, fmt.Errorf("encoder produced an invalid representation")
	}
	for i := range vector {
		vector[i] /= float32(math.Sqrt(norm))
	}
	return vector, nil
}
