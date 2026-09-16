package imaging

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/heicdecode"
	"github.com/frathe/picfetch/internal/heicdecode/client"
)

// HEICDecoder supplies the application's shared admission owner or an inherited
// analysis connection. It returns only protocol-validated pixels and metadata.
type HEICDecoder func(context.Context, heicdecode.Operation, client.Input) (heicdecode.Response, error)

// Reader is immutable and safe to share. The zero value handles ordinary images
// and refuses HEIC. A configured reader never selects its own executable.
type Reader struct{ heic HEICDecoder }

// Source owns ordinary encoded bytes or already decoded HEIC pixels. HEIC bytes
// never escape admission: only their byte count and digest survive. Callers must
// treat returned pixels as immutable, just like existing shared cache frames.
type Source struct {
	encoded  []byte
	bounds   image.Rectangle
	pixels   image.Image
	metadata Metadata
	size     int64
	digest   [32]byte
}

func NewReader(decode HEICDecoder) Reader { return Reader{heic: decode} }

// Read probes ordinary images without decoding their pixels; HEIC bounds and
// pixels arrive together so a later Decode cannot start another helper job.
func (r Reader) Read(ctx context.Context, u fyne.URI) (*Source, error) {
	return r.read(ctx, u, heicdecode.Decode)
}

func (r Reader) Metadata(ctx context.Context, u fyne.URI) (Metadata, error) {
	info, err := r.InspectMetadata(ctx, u)
	if err != nil {
		return Metadata{}, err
	}
	return info.Values, nil
}

// MetadataInfo includes the editing capability established from the same bytes
// as its values. HEIC metadata never enables the JPEG-only strip operation.
type MetadataInfo struct {
	Values   Metadata
	CanStrip bool
	FileSize int64
}

func (r Reader) InspectMetadata(ctx context.Context, u fyne.URI) (MetadataInfo, error) {
	source, err := r.read(ctx, u, heicdecode.DecodeExif)
	if err != nil {
		return MetadataInfo{}, err
	}
	m := source.Metadata()
	canStrip := source.encoded != nil && CanStripJPEGMetadata(source.encoded) && !m.Empty()
	if err = ctx.Err(); err != nil {
		return MetadataInfo{}, err
	}
	return MetadataInfo{Values: m, CanStrip: canStrip, FileSize: source.size}, nil
}

func (r Reader) CaptureDate(ctx context.Context, u fyne.URI) (time.Time, bool, error) {
	m, err := r.Metadata(ctx, u)
	if err != nil {
		return time.Time{}, false, err
	}
	return m.DateTakenTime, !m.DateTakenTime.IsZero(), nil
}

func (r Reader) read(ctx context.Context, u fyne.URI, op heicdecode.Operation) (*Source, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ext := strings.ToLower(u.Extension())
	isolated := ext == ".heic" || ext == ".heif" || ext == ".heics" || ext == ".heifs"
	if isolated && r.heic == nil {
		return nil, client.ErrUnavailable
	}
	file, err := storage.Reader(u)
	if err != nil {
		return nil, err
	}
	var once sync.Once
	closeFile := func() { once.Do(func() { _ = file.Close() }) }
	joinClose := closeSourceOnCancel(ctx, closeFile)
	defer func() { closeFile(); joinClose() }()
	limit := MaxEncodedBytes()
	var prefix []byte
	if !isolated {
		prefix, err = sourcePrefix(ctxReader{ctx: ctx, r: file}, limit)
		if err != nil {
			return nil, sourceReadError(ctx, err)
		}
		isolated = routeSource(prefix) == sourceIsolated
	}
	read := func(ctx context.Context, maxBytes int64) ([]byte, error) {
		join := closeSourceOnCancel(ctx, closeFile)
		defer join()
		return readSourceBytes(ctx, io.MultiReader(bytes.NewReader(prefix), file), min(limit, maxBytes))
	}
	if isolated {
		if r.heic == nil {
			return nil, client.ErrUnavailable
		}
		source := &Source{}
		result, err := r.heic(ctx, op, func(ctx context.Context, maxBytes int64) ([]byte, error) {
			data, err := read(ctx, maxBytes)
			if err == nil {
				source.size = int64(len(data))
				source.digest = sha256.Sum256(data)
			}
			return data, err
		})
		if err != nil {
			return nil, sourceReadError(ctx, err)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if op == heicdecode.Decode {
			if result.Image == nil || result.Image.Bounds() != image.Rect(0, 0, result.Config.Width, result.Config.Height) {
				return nil, heicdecode.ErrInvalidResponse
			}
			source.pixels, source.bounds = result.Image, result.Image.Bounds()
		}
		if m := result.Metadata; m != nil {
			source.metadata = metadataFromISOBMFFExif(m.Make, m.Model, m.ExposureTime, m.FNumber, m.ISOSpeed, m.FocalLength, m.DateTimeOriginal, m.DateTime, m.GPSLatitude, m.GPSLongitude)
		}
		return source, nil
	}
	data, err := read(ctx, limit)
	if err != nil {
		return nil, err
	}
	source := &Source{encoded: data, size: int64(len(data))}
	if op == heicdecode.Decode {
		_, source.bounds, err = probeEncoded(data)
		if err != nil {
			return nil, err
		}
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *Source) Bounds() image.Rectangle { return s.bounds }

func (s *Source) SHA256() [32]byte {
	if s.encoded != nil {
		return sha256.Sum256(s.encoded)
	}
	return s.digest
}

func (s *Source) Metadata() Metadata {
	if s.encoded != nil {
		return ReadMetadata(s.encoded)
	}
	return s.metadata
}

func (s *Source) Decode(ctx context.Context, maxAnimBytes int64) (*LoadedImage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.pixels != nil {
		// Container transforms have already been applied by the guest. Exif
		// orientation is descriptive metadata, never a second pixel transform.
		return &LoadedImage{Frames: []image.Image{s.pixels}, FileSize: s.size, HasEXIF: !s.metadata.Empty()}, nil
	}
	return DecodeRecord(ctx, s.encoded, maxAnimBytes)
}

func (r Reader) Thumbnail(ctx context.Context, u fyne.URI, maxEdge int) (image.Image, image.Rectangle, error) {
	if maxEdge <= 0 {
		return nil, image.Rectangle{}, fmt.Errorf("thumbnail edge must be positive: %d", maxEdge)
	}
	source, err := r.Read(ctx, u)
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	pixels, err := source.thumbnail(ctx, maxEdge)
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	return pixels, source.bounds, nil
}

func (s *Source) thumbnail(ctx context.Context, maxEdge int) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var pixels image.Image
	var err error
	if s.pixels != nil {
		pixels = scaleToFit(s.pixels, maxEdge)
	} else {
		pixels, err = decodeThumbnailAtEdge(ctx, s.encoded, s.bounds, maxEdge)
	}
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	return pixels, err
}

func (r Reader) Preview(ctx context.Context, u fyne.URI, maxEdge int, animationBytes int64) (*LoadedImage, error) {
	if maxEdge <= 0 {
		return nil, fmt.Errorf("preview edge must be positive: %d", maxEdge)
	}
	source, err := r.Read(ctx, u)
	if err != nil {
		return nil, err
	}
	if source.pixels == nil {
		return decodePreview(ctx, source.encoded, source.bounds, maxEdge, animationBytes)
	}
	pixels, err := source.thumbnail(ctx, maxEdge)
	if err != nil {
		return nil, err
	}
	return &LoadedImage{Frames: []image.Image{pixels}}, nil
}

func readSourceBytes(ctx context.Context, input io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 || limit == math.MaxInt64 {
		return nil, &InputTooLargeError{limit: limit}
	}
	data, err := io.ReadAll(io.LimitReader(ctxReader{ctx: ctx, r: input}, limit+1))
	if err = sourceReadError(ctx, err); err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, &InputTooLargeError{limit: limit}
	}
	return data, nil
}

func sourceReadError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func closeSourceOnCancel(ctx context.Context, closeFile func()) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(done); closeFile() })
	return func() {
		if !stop() {
			<-done
		}
	}
}
