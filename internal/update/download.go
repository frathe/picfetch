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
	stageJSONName           = "stage.json"
	maxArchiveBytes         = 200 << 20
	unknownProgressByteStep = 1 << 20
	lastIntermediatePercent = 99
)

// DownloadProgress reports bytes read from the release archive. Total is
// non-positive when the response did not declare a size.
type DownloadProgress struct {
	Downloaded int64
	Total      int64
}

type stageFile struct {
	Version               string `json:"version"`
	Notes                 string `json:"notes"`
	BinaryPath            string `json:"binaryPath"`
	PlistPath             string `json:"plistPath,omitempty"`
	VerifiedAssetName     string `json:"verifiedAssetName,omitempty"`
	VerifiedArchiveDigest string `json:"verifiedArchiveDigest,omitempty"`
	VerifiedBinaryDigest  string `json:"verifiedBinaryDigest,omitempty"`
	VerifiedPlistDigest   string `json:"verifiedPlistDigest,omitempty"`
	VerifiedGOOS          string `json:"verifiedGoos,omitempty"`
	VerifiedGOARCH        string `json:"verifiedGoarch,omitempty"`
}

// Download fetches rel's archive, SHA-256s it, compares to AssetDigest when
// set, verifies a GitHub immutable release attestation, extracts, and writes
// Stage under StageDir.
func (c *Client) Download(ctx context.Context, rel Release) (st Stage, err error) {
	return c.DownloadWithProgress(ctx, rel, nil)
}

// DownloadWithProgress is Download with synchronous progress notifications
// for the release archive. A nil progress callback is safe.
func (c *Client) DownloadWithProgress(ctx context.Context, rel Release, progress func(DownloadProgress)) (st Stage, err error) {
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

	data, err := c.fetchArchiveBytes(ctx, rel.AssetURL, "picfetch/"+rel.Version, progress)
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
	binaryDigest, err := fileSHA256(bin)
	if err != nil {
		return Stage{}, err
	}
	plistDigest := ""
	if plist != "" {
		plistDigest, err = fileSHA256(plist)
		if err != nil {
			return Stage{}, err
		}
	}
	st = Stage{
		Version:    rel.Version,
		Notes:      rel.Notes,
		BinaryPath: bin,
		PlistPath:  plist,
		verification: stageVerification{
			AssetName:     rel.AssetName,
			ArchiveDigest: hex.EncodeToString(sum[:]),
			BinaryDigest:  binaryDigest,
			PlistDigest:   plistDigest,
			GOOS:          c.cfg.GOOS,
			GOARCH:        c.cfg.GOARCH,
		},
	}
	if err := SaveStage(c.cfg.StageDir, st); err != nil {
		return Stage{}, err
	}
	return st, nil
}

func (c *Client) fetchArchiveBytes(ctx context.Context, rawURL, userAgent string, progress func(DownloadProgress)) ([]byte, error) {
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
	return readArchive(resp.Body, resp.ContentLength, maxArchiveBytes, progress)
}

func readArchive(r io.Reader, total, limit int64, progress func(DownloadProgress)) ([]byte, error) {
	reporter := newDownloadProgressReporter(total, progress)
	data, err := io.ReadAll(io.LimitReader(&progressReader{reader: r, reporter: reporter}, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("archive exceeds %d bytes", limit)
	}
	reporter.finish(int64(len(data)))
	return data, nil
}

type progressReader struct {
	reader   io.Reader
	reporter *downloadProgressReporter
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.reporter.advance(int64(n))
	}
	return n, err
}

type downloadProgressReporter struct {
	progress       func(DownloadProgress)
	total          int64
	downloaded     int64
	nextPercent    int64
	nextUnknown    int64
	lastDownloaded int64
}

func newDownloadProgressReporter(total int64, progress func(DownloadProgress)) *downloadProgressReporter {
	r := &downloadProgressReporter{
		progress:    progress,
		total:       total,
		nextPercent: 1,
		nextUnknown: unknownProgressByteStep,
	}
	r.emit(0)
	return r
}

func (r *downloadProgressReporter) advance(n int64) {
	r.downloaded += n
	if r.progress == nil {
		return
	}
	if r.total <= 0 {
		for r.downloaded >= r.nextUnknown {
			r.emit(r.nextUnknown)
			r.nextUnknown += unknownProgressByteStep
		}
		return
	}
	for r.nextPercent <= lastIntermediatePercent {
		threshold := percentageThreshold(r.total, r.nextPercent)
		if threshold >= r.total {
			return
		}
		if threshold <= r.lastDownloaded {
			r.nextPercent++
			continue
		}
		if r.downloaded < threshold {
			return
		}
		r.nextPercent++
		r.emit(threshold)
	}
}

func (r *downloadProgressReporter) finish(downloaded int64) {
	r.downloaded = downloaded
	if r.progress != nil {
		r.progress(DownloadProgress{Downloaded: downloaded, Total: r.total})
	}
}

func (r *downloadProgressReporter) emit(downloaded int64) {
	if r.progress == nil {
		return
	}
	r.lastDownloaded = downloaded
	r.progress(DownloadProgress{Downloaded: downloaded, Total: r.total})
}

func percentageThreshold(total, percent int64) int64 {
	quotient, remainder := total/100, total%100
	return quotient*percent + (remainder*percent+99)/100
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
		Version:               s.Version,
		Notes:                 s.Notes,
		BinaryPath:            bin,
		PlistPath:             plist,
		VerifiedAssetName:     s.verification.AssetName,
		VerifiedArchiveDigest: s.verification.ArchiveDigest,
		VerifiedBinaryDigest:  s.verification.BinaryDigest,
		VerifiedPlistDigest:   s.verification.PlistDigest,
		VerifiedGOOS:          s.verification.GOOS,
		VerifiedGOARCH:        s.verification.GOARCH,
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
		verification: stageVerification{
			AssetName:     sf.VerifiedAssetName,
			ArchiveDigest: sf.VerifiedArchiveDigest,
			BinaryDigest:  sf.VerifiedBinaryDigest,
			PlistDigest:   sf.VerifiedPlistDigest,
			GOOS:          sf.VerifiedGOOS,
			GOARCH:        sf.VerifiedGOARCH,
		},
	}, nil
}

// ValidateStage proves that s carries provenance written by Download after
// release verification and that its extracted files have not changed since.
// It deliberately rejects Stage values saved by other callers without that
// provenance, so an unsealed stage.json cannot become Ready or be applied
// merely because it names a regular file.
func ValidateStage(s Stage) error {
	v := s.verification
	if v.AssetName == "" || v.GOOS == "" || v.GOARCH == "" || !validSHA256(v.ArchiveDigest) || !validSHA256(v.BinaryDigest) {
		return errors.New("update: staged update lacks verified provenance")
	}
	if s.BinaryPath == "" {
		return errors.New("update: staged binary path is empty")
	}
	if err := verifyFileSHA256(s.BinaryPath, v.BinaryDigest); err != nil {
		return fmt.Errorf("update: staged binary verification: %w", err)
	}
	if s.PlistPath == "" {
		if v.PlistDigest != "" {
			return errors.New("update: staged plist provenance has no file")
		}
		return nil
	}
	if !validSHA256(v.PlistDigest) {
		return errors.New("update: staged plist lacks verified provenance")
	}
	if err := verifyFileSHA256(s.PlistPath, v.PlistDigest); err != nil {
		return fmt.Errorf("update: staged plist verification: %w", err)
	}
	return nil
}

// ValidateStageForPlatform adds the install-time platform boundary to
// ValidateStage. A verified cache copied from another target must never be
// applied merely because its paths and version are otherwise usable.
func ValidateStageForPlatform(s Stage, goos, goarch string) error {
	if err := ValidateStage(s); err != nil {
		return err
	}
	if s.verification.GOOS != goos || s.verification.GOARCH != goarch {
		return fmt.Errorf("update: staged update targets %s/%s, want %s/%s", s.verification.GOOS, s.verification.GOARCH, goos, goarch)
	}
	wantAsset, ok := AssetName(goos, goarch)
	if !ok || s.verification.AssetName != wantAsset {
		return fmt.Errorf("update: staged asset %q does not match %s/%s", s.verification.AssetName, goos, goarch)
	}
	return nil
}

// StageMatchesRelease reports whether s is an unchanged, previously verified
// stage for rel. When GitHub supplies an archive digest, it must still match
// the digest recorded by Download after reading that archive.
func StageMatchesRelease(s Stage, rel Release) bool {
	wantVersion := NormalizeVersion(rel.Version)
	if wantVersion == "" || NormalizeVersion(s.Version) != wantVersion ||
		s.verification.AssetName != rel.AssetName ||
		(rel.AssetDigest != "" && (!validSHA256(rel.AssetDigest) || !strings.EqualFold(s.verification.ArchiveDigest, rel.AssetDigest))) {
		return false
	}
	return ValidateStage(s) == nil
}

func fileSHA256(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%q is not a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyFileSHA256(path, want string) error {
	got, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(got, want) {
		return errors.New("SHA-256 mismatch")
	}
	return nil
}

func validSHA256(digest string) bool {
	decoded, err := hex.DecodeString(digest)
	return err == nil && len(decoded) == sha256.Size
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
