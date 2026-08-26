package update

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

//go:embed embed/tuf-repo.github.com/root.json
var githubTUFRoot []byte

const (
	releaseAttestationSANRegex = `^https://dotcom\.releases\.github\.com$`
	releasePredicatePrefix     = "https://in-toto.io/attestation/release/"
)

type VerifyPolicy struct {
	Tag       string // canonical vX.Y.Z
	AssetName string
}

type Verifier interface {
	Verify(ctx context.Context, digest, bundle []byte, policy VerifyPolicy) error
}

type sigstoreVerifier struct {
	inner *verify.Verifier
}

func NewSigstoreVerifier() (Verifier, error) {
	opts := tuf.DefaultOptions()
	opts.Root = githubTUFRoot
	opts.RepositoryBaseURL = GitHubTUFMirror
	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		cacheDir = os.TempDir()
	}
	opts.CachePath = filepath.Join(cacheDir, "picfetch", "tuf")
	client, err := tuf.New(opts)
	if err != nil {
		return nil, fmt.Errorf("update: tuf client: %w", err)
	}
	trusted, err := root.GetTrustedRoot(client)
	if err != nil {
		return nil, fmt.Errorf("update: trusted root: %w", err)
	}
	inner, err := verify.NewVerifier(trusted, verify.WithSignedTimestamps(1))
	if err != nil {
		return nil, fmt.Errorf("update: sigstore verifier: %w", err)
	}
	return &sigstoreVerifier{inner: inner}, nil
}

func (v *sigstoreVerifier) Verify(_ context.Context, digest, bundleJSON []byte, policy VerifyPolicy) error {
	entity := &bundle.Bundle{}
	if err := entity.UnmarshalJSON(bundleJSON); err != nil {
		return fmt.Errorf("update: attestation bundle: %w", err)
	}
	sanMatcher, err := verify.NewSANMatcher("", releaseAttestationSANRegex)
	if err != nil {
		return fmt.Errorf("update: SAN matcher: %w", err)
	}
	issuerMatcher, err := verify.NewIssuerMatcher("", ".*")
	if err != nil {
		return fmt.Errorf("update: issuer matcher: %w", err)
	}
	certID, err := verify.NewCertificateIdentity(sanMatcher, issuerMatcher, certificate.Extensions{})
	if err != nil {
		return fmt.Errorf("update: certificate identity: %w", err)
	}
	_, err = v.inner.Verify(entity, verify.NewPolicy(
		verify.WithArtifactDigest("sha256", digest),
		verify.WithCertificateIdentity(certID),
	))
	if err != nil {
		return fmt.Errorf("update: verify attestation: %w", err)
	}
	env, err := entity.Envelope()
	if err != nil {
		return fmt.Errorf("update: attestation envelope: %w", err)
	}
	payload, err := env.DecodeB64Payload()
	if err != nil {
		return fmt.Errorf("update: attestation payload: %w", err)
	}
	return checkReleaseStatement(payload, digest, policy)
}

type releasePredicate struct {
	PredicateType string `json:"predicateType"`
	Subject       []struct {
		Name   string            `json:"name"`
		Digest map[string]string `json:"digest"`
	} `json:"subject"`
	Predicate struct {
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
		Purl       string `json:"purl"`
	} `json:"predicate"`
}

func checkReleaseStatement(stmt []byte, digest []byte, policy VerifyPolicy) error {
	var s releasePredicate
	if err := json.Unmarshal(stmt, &s); err != nil {
		return fmt.Errorf("release statement: %w", err)
	}
	if !strings.HasPrefix(s.PredicateType, releasePredicatePrefix) {
		return fmt.Errorf("release statement: predicateType %q", s.PredicateType)
	}
	wantHex := hex.EncodeToString(digest)
	matched := false
	for _, sub := range s.Subject {
		if sub.Name != policy.AssetName {
			continue
		}
		got := strings.ToLower(strings.TrimPrefix(sub.Digest["sha256"], "sha256:"))
		if len(got) == sha256.Size*2 && subtle.ConstantTimeCompare([]byte(got), []byte(wantHex)) == 1 {
			matched = true
			break
		}
	}
	if !matched {
		return fmt.Errorf("release statement: no subject %q with matching sha256", policy.AssetName)
	}
	repo := RepoOwner + "/" + RepoName
	if s.Predicate.Repository != repo {
		return fmt.Errorf("release statement: repository %q, want %q", s.Predicate.Repository, repo)
	}
	if s.Predicate.Tag != policy.Tag {
		return fmt.Errorf("release statement: tag %q, want %q", s.Predicate.Tag, policy.Tag)
	}
	wantPurl := "pkg:github/" + repo + "@" + policy.Tag
	if s.Predicate.Purl != wantPurl {
		return fmt.Errorf("release statement: purl %q, want %q", s.Predicate.Purl, wantPurl)
	}
	return nil
}
