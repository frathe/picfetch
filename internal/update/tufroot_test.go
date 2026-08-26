package update

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckEmbeddedRoot_Expired(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	root := tufRootJSON(now.Add(-time.Hour))
	if err := CheckEmbeddedRoot(root, now); err == nil {
		t.Fatal("expired root must fail")
	}
}

func TestCheckEmbeddedRoot_ExpiresIn59Days(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	root := tufRootJSON(now.Add(59 * 24 * time.Hour))
	if err := CheckEmbeddedRoot(root, now); err == nil {
		t.Fatal("root with fewer than 60 days remaining must fail")
	}
}

func TestCheckEmbeddedRoot_ExpiresIn61Days(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	root := tufRootJSON(now.Add(61 * 24 * time.Hour))
	if err := CheckEmbeddedRoot(root, now); err != nil {
		t.Fatalf("root with 61 days remaining: %v", err)
	}
}

func TestCheckEmbeddedRoot_MissingExpires(t *testing.T) {
	if err := CheckEmbeddedRoot([]byte(`{"signed":{}}`), time.Now()); err == nil {
		t.Fatal("missing signed.expires must fail")
	}
}

func TestCheckEmbeddedRoot_BadJSON(t *testing.T) {
	if err := CheckEmbeddedRoot([]byte(`{`), time.Now()); err == nil {
		t.Fatal("invalid JSON must fail")
	}
}

func tufRootJSON(expires time.Time) []byte {
	return []byte(fmt.Sprintf(`{"signed":{"expires":%q}}`, expires.UTC().Format(time.RFC3339)))
}

func TestSyncGitHubRoot_UnsignedNewerLeftUnchanged(t *testing.T) {
	expires := time.Now().UTC().Add(120 * 24 * time.Hour)
	repo := newHTTPTUF(t, expires)
	srv := repo.serve()
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "root.json")
	v1 := repo.rootBytes(1)
	if err := os.WriteFile(dest, v1, 0o644); err != nil {
		t.Fatal(err)
	}
	repo.unsigned["/2.root.json"] = []byte(`{"signed":{"_type":"root","version":2}}`)

	changed, err := SyncGitHubRoot(context.Background(), dest, rewriteHost{srv: srv, inner: srv.Client()})
	if err == nil {
		t.Fatal("unsigned N+1 root must fail")
	}
	if changed {
		t.Error("changed = true on failed sync")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, v1) {
		t.Fatal("dest must be unchanged after a failed sync")
	}
}

func TestSyncGitHubRoot_WrongKeyNewerLeftUnchanged(t *testing.T) {
	expires := time.Now().UTC().Add(120 * 24 * time.Hour)
	repo := newHTTPTUF(t, expires)
	other := newHTTPTUF(t, expires)
	srv := repo.serve()
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "root.json")
	v1 := repo.rootBytes(1)
	if err := os.WriteFile(dest, v1, 0o644); err != nil {
		t.Fatal(err)
	}
	repo.unsigned["/2.root.json"] = other.rootBytes(1)

	changed, err := SyncGitHubRoot(context.Background(), dest, rewriteHost{srv: srv, inner: srv.Client()})
	if err == nil {
		t.Fatal("root signed by a different key must fail")
	}
	if changed {
		t.Error("changed = true on failed sync")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, v1) {
		t.Fatal("dest must be unchanged after a failed sync")
	}
}

func TestSyncGitHubRoot_VerifiedNewerReplacesDest(t *testing.T) {
	expires := time.Now().UTC().Add(120 * 24 * time.Hour)
	repo := newHTTPTUF(t, expires)
	repo.rotateRoot(expires.Add(24 * time.Hour))
	srv := repo.serve()
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "root.json")
	if err := os.WriteFile(dest, repo.rootBytes(1), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := SyncGitHubRoot(context.Background(), dest, rewriteHost{srv: srv, inner: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, repo.rootBytes(2)) {
		t.Fatalf("dest is not the verified v2 root")
	}
}

func TestSyncGitHubRoot_AlreadyLatestUnchanged(t *testing.T) {
	expires := time.Now().UTC().Add(120 * 24 * time.Hour)
	repo := newHTTPTUF(t, expires)
	srv := repo.serve()
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "root.json")
	v1 := repo.rootBytes(1)
	if err := os.WriteFile(dest, v1, 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := SyncGitHubRoot(context.Background(), dest, rewriteHost{srv: srv, inner: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("changed = true when dest is already the latest root")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, v1) {
		t.Fatal("dest bytes must be unchanged")
	}
}

func TestSyncGitHubRoot_RefusesExpiredLatest(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(120 * 24 * time.Hour)
	repo := newHTTPTUF(t, future)
	repo.rotateRoot(past)
	srv := repo.serve()
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "root.json")
	v1 := repo.rootBytes(1)
	if err := os.WriteFile(dest, v1, 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := SyncGitHubRoot(context.Background(), dest, rewriteHost{srv: srv, inner: srv.Client()})
	if err == nil {
		t.Fatal("expired latest root must not be written")
	}
	if changed {
		t.Error("changed = true on failed sync")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, v1) {
		t.Fatal("dest must be unchanged when the latest root is expired")
	}
}
