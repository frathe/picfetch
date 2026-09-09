package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"

	ort "github.com/yalue/onnxruntime_go"
	"golang.org/x/image/draw"
)

type encoder struct {
	session *ort.AdvancedSession
	input   *ort.Tensor[float32]
	output  *ort.Tensor[float32]
}

func newEncoder(config configuration) (_ *encoder, err error) {
	ort.SetSharedLibraryPath(filepath.Join(config.Assets, runtimeLibrary))
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, err
	}
	e := &encoder{}
	defer func() {
		if err != nil {
			e.close()
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
	if config.Provider == "coreml" {
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
	e.session, err = ort.NewAdvancedSession(filepath.Join(config.Assets, "vision_model.onnx"),
		[]string{"pixel_values"}, []string{"pooler_output"}, []ort.Value{e.input}, []ort.Value{e.output}, options)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *encoder) close() {
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

func (e *encoder) encode(ctx context.Context, source image.Image) ([]float32, error) {
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
