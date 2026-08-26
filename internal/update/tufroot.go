package update

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/theupdateframework/go-tuf/v2/metadata/fetcher"
)

// tufRootMinRemaining is how long signed.expires must still be in the
// future for CheckEmbeddedRoot. 60 days matches GitHub TUF-on-CI's
// x-tuf-on-ci-signing-period on the current GitHub root.
const tufRootMinRemaining = 60 * 24 * time.Hour

type tufRootMeta struct {
	Signed struct {
		Expires string `json:"expires"`
	} `json:"signed"`
}

// CheckEmbeddedRoot reports whether root (GitHub TUF root metadata) is
// still valid at now: signed.expires must parse as RFC3339, be strictly
// after now, and be at least 60 days away. No network.
func CheckEmbeddedRoot(root []byte, now time.Time) error {
	exp, err := tufRootExpires(root)
	if err != nil {
		return err
	}
	if !now.Before(exp) {
		return fmt.Errorf("update: tuf root expired on %s", exp.Format(time.RFC3339))
	}
	if exp.Sub(now) < tufRootMinRemaining {
		return fmt.Errorf("update: tuf root expires %s, need at least 60 days remaining", exp.Format(time.RFC3339))
	}
	return nil
}

type ctxDoer struct {
	ctx context.Context
	d   Doer
}

func (c ctxDoer) Do(req *http.Request) (*http.Response, error) {
	return c.d.Do(req.WithContext(c.ctx))
}

// SyncGitHubRoot refreshes dest (a GitHub TUF root.json) from GitHubTUFMirror
// using hc. A newer root is written only after TUF verification against dest's
// keys. Returns changed=true when dest bytes were replaced. An expired latest
// root is refused and dest is left untouched.
func SyncGitHubRoot(ctx context.Context, destPath string, hc Doer) (bool, error) {
	current, err := os.ReadFile(destPath)
	if err != nil {
		return false, fmt.Errorf("update: tuf root: %w", err)
	}
	cache, err := os.MkdirTemp("", "picfetch-tuf-*")
	if err != nil {
		return false, fmt.Errorf("update: tuf root cache: %w", err)
	}
	defer func() { _ = os.RemoveAll(cache) }()

	opts := tuf.DefaultOptions()
	opts.Root = current
	opts.RepositoryBaseURL = GitHubTUFMirror
	opts.CachePath = cache
	f := fetcher.NewDefaultFetcher()
	f.SetHTTPClient(ctxDoer{ctx: ctx, d: hc})
	opts.Fetcher = f

	if _, err := tuf.New(opts); err != nil {
		return false, fmt.Errorf("update: tuf refresh: %w", err)
	}
	latest, err := os.ReadFile(filepath.Join(cache, tuf.URLToPath(GitHubTUFMirror), "root.json"))
	if err != nil {
		return false, fmt.Errorf("update: tuf refreshed root: %w", err)
	}
	if err := tufRootNotExpired(latest, time.Now()); err != nil {
		return false, err
	}
	if bytes.Equal(current, latest) {
		return false, nil
	}
	if err := os.WriteFile(destPath, latest, 0o644); err != nil {
		return false, fmt.Errorf("update: tuf root write: %w", err)
	}
	return true, nil
}

func tufRootExpires(root []byte) (time.Time, error) {
	var meta tufRootMeta
	if err := json.Unmarshal(root, &meta); err != nil {
		return time.Time{}, fmt.Errorf("update: tuf root: %w", err)
	}
	if meta.Signed.Expires == "" {
		return time.Time{}, fmt.Errorf("update: tuf root: missing signed.expires")
	}
	exp, err := time.Parse(time.RFC3339, meta.Signed.Expires)
	if err != nil {
		return time.Time{}, fmt.Errorf("update: tuf root expires: %w", err)
	}
	return exp, nil
}

func tufRootNotExpired(root []byte, now time.Time) error {
	exp, err := tufRootExpires(root)
	if err != nil {
		return err
	}
	if !now.Before(exp) {
		return fmt.Errorf("update: tuf root expired on %s", exp.Format(time.RFC3339))
	}
	return nil
}
