package similarity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"path/filepath"
)

const maximumAnalysisRecordBytes = 1024 * 1024

func decodeRepresentation(reader io.Reader) (Item, error) {
	decoder := json.NewDecoder(io.LimitReader(reader, maximumAnalysisRecordBytes+1))
	var entry cachedRepresentation
	if err := decoder.Decode(&entry); err != nil {
		return Item{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Item{}, fmt.Errorf("analysis record has trailing data")
	}
	item := entry.Item
	if entry.Version != RepresentationVersion || !filepath.IsAbs(item.Path) || filepath.Clean(item.Path) != item.Path || item.Error != "" {
		return Item{}, fmt.Errorf("incompatible analysis record")
	}
	if _, valid := searchEmbeddingNorm(item.Embedding); !valid {
		return Item{}, fmt.Errorf("invalid analysis vector")
	}
	hash, err := hex.DecodeString(item.SHA256)
	if err != nil || len(hash) != sha256.Size {
		return Item{}, fmt.Errorf("invalid analysis digest")
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(item.Preview))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 160 || config.Height > 160 {
		return Item{}, fmt.Errorf("invalid analysis preview")
	}
	item.Cohort, item.Position, item.Thumbnail = "", nil, ""
	item.Tags = nil
	return item, nil
}
