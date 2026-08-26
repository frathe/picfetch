package update

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

// httpTUF is a minimal consistent-snapshot TUF repo served over httptest.
// It is enough to exercise SyncGitHubRoot; it is not a full TUF implementation.
type httpTUF struct {
	t         *testing.T
	keys      map[string]ed25519.PrivateKey
	root      *metadata.Metadata[metadata.RootType]
	targets   *metadata.Metadata[metadata.TargetsType]
	snapshot  *metadata.Metadata[metadata.SnapshotType]
	timestamp *metadata.Metadata[metadata.TimestampType]
	roots     map[int64][]byte
	unsigned  map[string][]byte // extra paths, e.g. unsigned 2.root.json
}

func newHTTPTUF(t *testing.T, expires time.Time) *httpTUF {
	t.Helper()
	r := &httpTUF{
		t:        t,
		keys:     make(map[string]ed25519.PrivateKey),
		roots:    make(map[int64][]byte),
		unsigned: make(map[string][]byte),
	}
	r.targets = metadata.Targets(expires)
	r.snapshot = metadata.Snapshot(expires)
	r.timestamp = metadata.Timestamp(expires)
	r.root = metadata.Root(expires)

	for _, name := range []string{metadata.TARGETS, metadata.SNAPSHOT, metadata.TIMESTAMP, metadata.ROOT} {
		_, private, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		r.keys[name] = private
		key, err := metadata.KeyFromPublicKey(private.Public())
		if err != nil {
			t.Fatal(err)
		}
		if err := r.root.Signed.AddKey(key, name); err != nil {
			t.Fatal(err)
		}
	}
	r.signAll()
	r.storeRoot()
	return r
}

func (r *httpTUF) signer(role string) signature.Signer {
	r.t.Helper()
	s, err := signature.LoadSigner(r.keys[role], crypto.Hash(0))
	if err != nil {
		r.t.Fatal(err)
	}
	return s
}

func (r *httpTUF) signAll() {
	r.t.Helper()
	r.root.ClearSignatures()
	if _, err := r.root.Sign(r.signer(metadata.ROOT)); err != nil {
		r.t.Fatal(err)
	}
	r.targets.ClearSignatures()
	if _, err := r.targets.Sign(r.signer(metadata.TARGETS)); err != nil {
		r.t.Fatal(err)
	}
	r.snapshot.ClearSignatures()
	if _, err := r.snapshot.Sign(r.signer(metadata.SNAPSHOT)); err != nil {
		r.t.Fatal(err)
	}
	r.timestamp.ClearSignatures()
	if _, err := r.timestamp.Sign(r.signer(metadata.TIMESTAMP)); err != nil {
		r.t.Fatal(err)
	}
}

func (r *httpTUF) storeRoot() {
	r.t.Helper()
	b, err := r.root.ToBytes(false)
	if err != nil {
		r.t.Fatal(err)
	}
	r.roots[r.root.Signed.Version] = b
}

func (r *httpTUF) rootBytes(version int64) []byte {
	b, ok := r.roots[version]
	if !ok {
		r.t.Fatalf("no root version %d", version)
	}
	return b
}

func (r *httpTUF) rotateRoot(expires time.Time) {
	r.t.Helper()
	r.root.ClearSignatures()
	r.root.Signed.Version++
	r.root.Signed.Expires = expires
	if _, err := r.root.Sign(r.signer(metadata.ROOT)); err != nil {
		r.t.Fatal(err)
	}
	r.storeRoot()
}

func (r *httpTUF) serve() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p := req.URL.Path
		if b, ok := r.unsigned[p]; ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(b)
			return
		}
		b, err := r.file(p)
		if err != nil {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
}

func (r *httpTUF) file(p string) ([]byte, error) {
	if p == "/timestamp.json" {
		return r.timestamp.ToBytes(false)
	}
	base := path.Base(p)
	if !strings.HasSuffix(base, ".json") {
		return nil, fmt.Errorf("not found")
	}
	name := strings.TrimSuffix(base, ".json")
	before, after, ok := strings.Cut(name, ".")
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	ver, err := strconv.ParseInt(before, 10, 64)
	if err != nil {
		return nil, err
	}
	role := after
	switch role {
	case metadata.ROOT:
		b, ok := r.roots[ver]
		if !ok {
			return nil, fmt.Errorf("not found")
		}
		return b, nil
	case metadata.SNAPSHOT:
		if r.snapshot.Signed.Version != ver {
			return nil, fmt.Errorf("not found")
		}
		return r.snapshot.ToBytes(false)
	case metadata.TARGETS:
		if r.targets.Signed.Version != ver {
			return nil, fmt.Errorf("not found")
		}
		return r.targets.ToBytes(false)
	}
	return nil, fmt.Errorf("not found")
}

// rewriteHost sends TUF requests aimed at GitHubTUFMirror to srv instead.
type rewriteHost struct {
	srv   *httptest.Server
	inner Doer
}

func (r rewriteHost) Do(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(r.srv.URL)
	if err != nil {
		return nil, err
	}
	c := req.Clone(req.Context())
	c.URL.Scheme = u.Scheme
	c.URL.Host = u.Host
	c.Host = u.Host
	return r.inner.Do(c)
}
