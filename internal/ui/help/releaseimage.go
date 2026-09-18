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
	"net/url"
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
	if !releaseImageURL(req.URL) {
		return nil, fmt.Errorf("untrusted release image URL %q", req.URL.Redacted())
	}
	requestClient := *client
	requestClient.CheckRedirect = releaseImageRedirectPolicy
	response, err := requestClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release image: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.Request != nil && !releaseImageURL(response.Request.URL) {
		return nil, fmt.Errorf("release image response came from an untrusted URL")
	}
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

func releaseImageRedirectPolicy(request *http.Request, via []*http.Request) error {
	if len(via) >= 10 || !releaseImageURL(request.URL) {
		return fmt.Errorf("release image redirected outside its HTTPS providers")
	}
	return nil
}

func releaseImageURL(address *url.URL) bool {
	if strings.ToLower(address.Scheme) != "https" || address.User != nil {
		return false
	}
	host := strings.ToLower(address.Hostname())
	return host == "github.com" || host == "githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com")
}
