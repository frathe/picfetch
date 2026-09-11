package similarity

import (
	"image"
	"path/filepath"
	"strings"

	"github.com/frathe/picfetch/internal/imaging"
)

func imageFacts(path string, data []byte, bounds image.Rectangle) ImageFacts {
	metadata := imaging.ReadMetadata(data)
	facts := ImageFacts{Version: FactsVersion, Width: bounds.Dx(), Height: bounds.Dy(),
		Format: strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")),
		Make:   strings.TrimSpace(metadata.Make), Model: strings.TrimSpace(metadata.Model)}
	if facts.Format == "jpeg" {
		facts.Format = "jpg"
	}
	if facts.Format == "tif" {
		facts.Format = "tiff"
	}
	if !metadata.DateTakenTime.IsZero() {
		facts.CaptureDate = metadata.DateTakenTime.Format("2006-01-02")
	}
	return facts
}
