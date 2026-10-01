package imaging

import (
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"os"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/fileaccess"
)

// SourceDigest binds metadata to the exact encoded bytes that supplied pixels.
type SourceDigest [sha256.Size]byte

// SourceVersion additionally binds destructive removal to the inspected inode.
// Its fields are private so callers cannot manufacture a filesystem identity.
type SourceVersion struct {
	digest SourceDigest
	info   os.FileInfo
}

// ErrSourceChanged refuses a mutation whose inspected source has been replaced.
var ErrSourceChanged = errors.New("source changed after inspection")

// Valid reports whether inspection obtained an opened filesystem identity.
func (v SourceVersion) Valid() bool { return v.info != nil && v.info.Mode().IsRegular() }

func (v SourceVersion) matches(data []byte, info os.FileInfo) bool {
	return v.Valid() && info != nil && os.SameFile(v.info, info) && v.digest == sha256.Sum256(data)
}

// ReadAndProbeSnapshot captures identity and bytes from one scoped reader.
// Unavailable descriptor identity leaves ordinary metadata readable but disables
// destructive removal. Probe and input limits are shared with image loading.
func ReadAndProbeSnapshot(ctx context.Context, u fyne.URI) ([]byte, image.Rectangle, SourceVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, image.Rectangle{}, SourceVersion{}, err
	}
	rc, err := fileaccess.Reader(ctx, u)
	if err != nil {
		return nil, image.Rectangle{}, SourceVersion{}, err
	}
	defer func() { _ = rc.Close() }()
	var info os.FileInfo
	if f, ok := rc.(interface{ Stat() (os.FileInfo, error) }); ok {
		info, _ = f.Stat()
	}
	data, err := readMutationBytes(ctx, rc, MaxEncodedBytes())
	if err != nil {
		return nil, image.Rectangle{}, SourceVersion{}, err
	}
	if err := ctx.Err(); err != nil {
		return nil, image.Rectangle{}, SourceVersion{}, err
	}
	version := SourceVersion{digest: sha256.Sum256(data), info: info}
	if f, ok := rc.(interface{ Stat() (os.FileInfo, error) }); ok && info != nil {
		after, statErr := f.Stat()
		if statErr != nil || !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
			return nil, image.Rectangle{}, SourceVersion{}, ErrSourceChanged
		}
	}
	data, bounds, err := probeSource(ctx, data)
	return data, bounds, version, err
}
