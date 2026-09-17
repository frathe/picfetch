package help

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	_ "golang.org/x/image/webp"
)

const (
	releaseImageByteLimit  = 8 * 1024 * 1024
	releaseImagePixelLimit = 16 * 1024 * 1024
)

// loadReleaseImage fetches and decodes one bounded image. The caller runs it
// off the UI thread and owns the client's timeout and the request lifetime.
func loadReleaseImage(ctx context.Context, client *http.Client, rawURL string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create release image request: %w", err)
	}
	req.URL.Scheme = strings.ToLower(req.URL.Scheme)
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported release image URL scheme %q", req.URL.Scheme)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release image: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("release image returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > releaseImageByteLimit {
		return nil, fmt.Errorf("release image exceeds size limit of %d bytes", releaseImageByteLimit)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, releaseImageByteLimit+1))
	if err != nil {
		return nil, fmt.Errorf("read release image: %w", err)
	}
	if len(data) > releaseImageByteLimit {
		return nil, fmt.Errorf("release image exceeds size limit of %d bytes", releaseImageByteLimit)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode release image header: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > releaseImagePixelLimit/config.Height {
		return nil, fmt.Errorf("release image dimensions %dx%d exceed pixel limit of %d", config.Width, config.Height, releaseImagePixelLimit)
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode release image: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return decoded, nil
}
