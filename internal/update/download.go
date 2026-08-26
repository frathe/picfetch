package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	stageJSONName   = "stage.json"
	maxArchiveBytes = 200 << 20
)

type stageFile struct {
	Version    string `json:"version"`
	Notes      string `json:"notes"`
	BinaryPath string `json:"binaryPath"`
	PlistPath  string `json:"plistPath,omitempty"`
}

// Download fetches rel's archive, SHA-256s it, compares to AssetDigest when
// set, verifies a GitHub immutable release attestation, extracts, and writes
// Stage under StageDir.
func (c *Client) Download(ctx context.Context, rel Release) (st Stage, err error) {
	if c.cfg.StageDir == "" {
		return Stage{}, fmt.Errorf("update: empty StageDir")
	}
	if err := RemoveStage(c.cfg.StageDir); err != nil {
		return Stage{}, err
	}
	defer func() {
		if err != nil {
			_ = RemoveStage(c.cfg.StageDir)
		}
	}()

	data, err := c.fetchBytes(ctx, rel.AssetURL, "picfetch/"+rel.Version)
	if err != nil {
		return Stage{}, err
	}
	sum := sha256.Sum256(data)
	if rel.AssetDigest != "" {
		if err := VerifyHash(data, rel.AssetDigest); err != nil {
			return Stage{}, err
		}
	}
	if c.cfg.Verify == nil {
		return Stage{}, errors.New("update: missing attestation verifier")
	}
	bundleJSON, err := c.fetchReleaseAttestation(ctx, hex.EncodeToString(sum[:]), "picfetch/"+rel.Version)
	if err != nil {
		return Stage{}, err
	}
	if err := c.cfg.Verify.Verify(ctx, sum[:], bundleJSON, VerifyPolicy{
		Tag:       rel.Version,
		AssetName: rel.AssetName,
	}); err != nil {
		return Stage{}, err
	}

	tmp, err := writeTempArchive(rel.AssetName, data)
	if err != nil {
		return Stage{}, err
	}
	defer func() { _ = os.Remove(tmp) }()

	if err := os.MkdirAll(c.cfg.StageDir, 0o700); err != nil {
		return Stage{}, err
	}
	bin, plist, err := extract(ctx, tmp, c.cfg.StageDir)
	if err != nil {
		return Stage{}, err
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		return Stage{}, err
	}
	if plist != "" {
		plist, err = filepath.Abs(plist)
		if err != nil {
			return Stage{}, err
		}
	}
	st = Stage{
		Version:    rel.Version,
		Notes:      rel.Notes,
		BinaryPath: bin,
		PlistPath:  plist,
	}
	if err := SaveStage(c.cfg.StageDir, st); err != nil {
		return Stage{}, err
	}
	return st, nil
}

func (c *Client) fetchBytes(ctx context.Context, rawURL, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxArchiveBytes {
		return nil, fmt.Errorf("archive exceeds %d bytes", maxArchiveBytes)
	}
	return data, nil
}

func writeTempArchive(assetName string, data []byte) (string, error) {
	f, err := os.CreateTemp("", "picfetch-update-*"+archiveSuffix(assetName))
	if err != nil {
		return "", err
	}
	name := f.Name()
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if closeErr != nil {
		_ = os.Remove(name)
		return "", closeErr
	}
	return name, nil
}

func archiveSuffix(name string) string {
	if strings.HasSuffix(name, ".tar.gz") {
		return ".tar.gz"
	}
	return filepath.Ext(name)
}

func SaveStage(dir string, s Stage) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	bin, err := absOrEmpty(s.BinaryPath)
	if err != nil {
		return err
	}
	plist, err := absOrEmpty(s.PlistPath)
	if err != nil {
		return err
	}
	data, err := json.Marshal(stageFile{
		Version:    s.Version,
		Notes:      s.Notes,
		BinaryPath: bin,
		PlistPath:  plist,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stageJSONName), data, 0o600)
}

func LoadStage(dir string) (Stage, error) {
	data, err := os.ReadFile(filepath.Join(dir, stageJSONName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Stage{}, os.ErrNotExist
		}
		return Stage{}, err
	}
	var sf stageFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return Stage{}, err
	}
	return Stage{
		Version:    sf.Version,
		Notes:      sf.Notes,
		BinaryPath: sf.BinaryPath,
		PlistPath:  sf.PlistPath,
	}, nil
}

func RemoveStage(dir string) error {
	if dir == "" {
		return fmt.Errorf("update: empty stage dir")
	}
	return os.RemoveAll(dir)
}

func absOrEmpty(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	return filepath.Abs(path)
}
