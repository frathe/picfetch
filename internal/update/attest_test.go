package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type fakeVerifier struct {
	digest []byte
	bundle []byte
	policy VerifyPolicy
	err    error
}

func (f *fakeVerifier) Verify(_ context.Context, digest, bundle []byte, policy VerifyPolicy) error {
	f.digest = append([]byte(nil), digest...)
	f.bundle = append([]byte(nil), bundle...)
	f.policy = policy
	return f.err
}

func TestReleaseAttestationIdentity(t *testing.T) {
	if ReleaseAttestationSAN != "https://dotcom.releases.github.com" {
		t.Errorf("ReleaseAttestationSAN = %q", ReleaseAttestationSAN)
	}
	if GitHubTUFMirror != "https://tuf-repo.github.com" {
		t.Errorf("GitHubTUFMirror = %q", GitHubTUFMirror)
	}
	if releaseAttestationSANRegex != `^https://dotcom\.releases\.github\.com$` {
		t.Errorf("releaseAttestationSANRegex = %q", releaseAttestationSANRegex)
	}
	re := regexp.MustCompile(releaseAttestationSANRegex)
	if !re.MatchString(ReleaseAttestationSAN) {
		t.Error("SAN does not match its anchored regex")
	}
	if re.MatchString("https://github.com/frathe/picfetch/.github/workflows/release.yml@refs/tags/v0.2.6") {
		t.Error("SAN regex must not match release.yml@refs/tags/")
	}
	if re.MatchString("https://token.actions.githubusercontent.com") {
		t.Error("SAN regex must not match Actions OIDC issuer")
	}
}

func TestCheckReleaseStatement(t *testing.T) {
	digest := bytes.Repeat([]byte{0xab}, sha256.Size)
	hexDigest := hex.EncodeToString(digest)
	policy := VerifyPolicy{Tag: "v0.2.6", AssetName: "picfetch-linux-amd64.tar.gz"}
	purl := "pkg:github/frathe/picfetch@v0.2.6"
	releaseV02 := "https://in-toto.io/attestation/release/v0.2"
	releaseV01 := "https://in-toto.io/attestation/release/v0.1"
	otherHex := strings.Repeat("cd", 32)
	zeroHex := strings.Repeat("00", 32)

	goodSubjects := []map[string]any{
		subject("picfetch-macos-arm64.zip", otherHex),
		subject("picfetch-linux-amd64.tar.gz", hexDigest),
	}

	tests := []struct {
		name    string
		stmt    []byte
		wantErr bool
	}{
		{
			name:    "ok",
			stmt:    releaseStatement(t, releaseV02, goodSubjects, "frathe/picfetch", "v0.2.6", purl),
			wantErr: false,
		},
		{
			name:    "ok v0.1 prefix",
			stmt:    releaseStatement(t, releaseV01, goodSubjects, "frathe/picfetch", "v0.2.6", purl),
			wantErr: false,
		},
		{
			name:    "wrong repository",
			stmt:    releaseStatement(t, releaseV02, goodSubjects, "evil/picfetch", "v0.2.6", purl),
			wantErr: true,
		},
		{
			name:    "wrong tag",
			stmt:    releaseStatement(t, releaseV02, goodSubjects, "frathe/picfetch", "v0.2.5", purl),
			wantErr: true,
		},
		{
			name: "digest present under a different subject name",
			stmt: releaseStatement(t, releaseV02, []map[string]any{
				subject("other.zip", hexDigest),
				subject("picfetch-linux-amd64.tar.gz", zeroHex),
			}, "frathe/picfetch", "v0.2.6", purl),
			wantErr: true,
		},
		{
			name: "missing asset name",
			stmt: releaseStatement(t, releaseV02, []map[string]any{
				subject("other.zip", hexDigest),
			}, "frathe/picfetch", "v0.2.6", purl),
			wantErr: true,
		},
		{
			name:    "non-release predicateType",
			stmt:    releaseStatement(t, "https://slsa.dev/provenance/v1", goodSubjects, "frathe/picfetch", "v0.2.6", purl),
			wantErr: true,
		},
		{
			name:    "wrong purl",
			stmt:    releaseStatement(t, releaseV02, goodSubjects, "frathe/picfetch", "v0.2.6", "pkg:github/evil/picfetch@v0.2.6"),
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkReleaseStatement(tc.stmt, digest, policy)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDownload_CallsVerify(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	digestHex := hex.EncodeToString(sum[:])
	bundleJSON := []byte(`{"mediaType":"test-bundle"}`)
	fv := &fakeVerifier{}
	var gotAttest *http.Request
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, func(w http.ResponseWriter, r *http.Request) {
		gotAttest = r.Clone(r.Context())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(attestationsJSON(bundleJSON))
	})
	c := downloadClient(t, srv, fv, t.TempDir())
	rel := Release{
		Version:     "v0.2.6",
		Notes:       "## Fixes\n\n- toast",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: digestHex,
	}
	if _, err := c.Download(context.Background(), rel); err != nil {
		t.Fatal(err)
	}
	if gotAttest == nil {
		t.Fatal("no GET to attestations")
	}
	wantPath := "/repos/frathe/picfetch/attestations/sha256:" + digestHex
	if gotAttest.URL.Path != wantPath {
		t.Errorf("attest path = %q, want %q", gotAttest.URL.Path, wantPath)
	}
	if got := gotAttest.URL.Query().Get("predicate_type"); got != "release" {
		t.Errorf("predicate_type = %q, want release", got)
	}
	if got := gotAttest.Header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Accept = %q", got)
	}
	if got := gotAttest.Header.Get("User-Agent"); got != "picfetch/v0.2.6" {
		t.Errorf("User-Agent = %q, want picfetch/v0.2.6", got)
	}
	if got := gotAttest.Header.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
		t.Errorf("X-GitHub-Api-Version = %q", got)
	}
	if got := gotAttest.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
	if !bytes.Equal(fv.digest, sum[:]) {
		t.Errorf("Verify digest = %x, want %x", fv.digest, sum[:])
	}
	if !bytes.Equal(fv.bundle, bundleJSON) {
		t.Errorf("Verify bundle = %s, want %s", fv.bundle, bundleJSON)
	}
	if fv.policy.Tag != rel.Version {
		t.Errorf("policy.Tag = %q, want %q", fv.policy.Tag, rel.Version)
	}
	if fv.policy.AssetName != rel.AssetName {
		t.Errorf("policy.AssetName = %q, want %q", fv.policy.AssetName, rel.AssetName)
	}
}

func TestDownload_Attest404(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	})
	stageDir := t.TempDir()
	c := downloadClient(t, srv, &fakeVerifier{}, stageDir)
	_, err := c.Download(context.Background(), Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: hex.EncodeToString(sum[:]),
	})
	if err == nil {
		t.Fatal("want error for attestations HTTP 404")
	}
	assertNoStage(t, stageDir)
}

func TestDownload_EmptyAttestations(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"attestations":[]}`))
	})
	stageDir := t.TempDir()
	c := downloadClient(t, srv, &fakeVerifier{}, stageDir)
	_, err := c.Download(context.Background(), Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: hex.EncodeToString(sum[:]),
	})
	if err == nil {
		t.Fatal("want error for empty attestations array")
	}
	assertNoStage(t, stageDir)
}

func TestDownload_VerifierError(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	fv := &fakeVerifier{err: errors.New("sigstore boom")}
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(attestationsJSON([]byte(`{}`)))
	})
	stageDir := t.TempDir()
	c := downloadClient(t, srv, fv, stageDir)
	_, err := c.Download(context.Background(), Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: hex.EncodeToString(sum[:]),
	})
	if err == nil {
		t.Fatal("want error when Verifier fails")
	}
	assertNoStage(t, stageDir)
}

func TestDownload_NilVerifier(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(attestationsJSON([]byte(`{}`)))
	})
	stageDir := t.TempDir()
	c := downloadClient(t, srv, nil, stageDir)
	_, err := c.Download(context.Background(), Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: hex.EncodeToString(sum[:]),
	})
	if err == nil {
		t.Fatal("want error for missing attestation verifier")
	}
	assertNoStage(t, stageDir)
}

func attestationsJSON(bundle json.RawMessage) []byte {
	b, err := json.Marshal(ghAttestations{
		Attestations: []struct {
			Bundle json.RawMessage `json:"bundle"`
		}{
			{Bundle: bundle},
		},
	})
	if err != nil {
		panic(err)
	}
	return b
}

func subject(name, sha string) map[string]any {
	return map[string]any{
		"name":   name,
		"digest": map[string]any{"sha256": sha},
	}
}

func releaseStatement(t *testing.T, predicateType string, subjects []map[string]any, repo, tag, purl string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"predicateType": predicateType,
		"subject":       subjects,
		"predicate": map[string]any{
			"repository": repo,
			"tag":        tag,
			"purl":       purl,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertNoStage(t *testing.T, stageDir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(stageDir, "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want no usable stage.json, stat err %v", err)
	}
}
