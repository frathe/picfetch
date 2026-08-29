// Package update talks to GitHub Releases, verifies GitHub immutable
// release attestations, and replaces the running binary. It has no Fyne types.
package update

import (
	"net/http"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	RepoOwner             = "frathe"
	RepoName              = "picfetch"
	APIHost               = "https://api.github.com"
	ReleaseAttestationSAN = "https://dotcom.releases.github.com"
	GitHubTUFMirror       = "https://tuf-repo.github.com"
)

type Config struct {
	BaseURL  string // default APIHost, httptest in tests
	HTTP     Doer
	Now      func() time.Time
	Verify   Verifier // required; Download fails closed if nil
	StageDir string
	GOOS     string // default runtime.GOOS
	GOARCH   string
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Release struct {
	Version     string // canonical vX.Y.Z
	Notes       string // GitHub body, may be empty
	AssetName   string
	AssetURL    string
	AssetDigest string // hex from API digest without "sha256:" prefix; empty if omitted
}

type Stage struct {
	Version    string
	Notes      string
	BinaryPath string
	PlistPath  string // darwin extracted Info.plist; empty otherwise

	// verification is written only by Client.Download after the release
	// archive has passed its digest and Sigstore checks. Keeping it
	// unexported prevents callers from accidentally turning an arbitrary
	// Stage literal into trusted update provenance.
	verification stageVerification
}

type stageVerification struct {
	AssetName     string
	ArchiveDigest string
	BinaryDigest  string
	PlistDigest   string
	GOOS          string
	GOARCH        string
}

type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = APIHost
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	if cfg.GOARCH == "" {
		cfg.GOARCH = runtime.GOARCH
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Client{cfg: cfg}
}

// Now returns the client's clock (Config.Now), used by UI glue for Due /
// DayString without reaching into the unexported cfg field.
func (c *Client) Now() time.Time {
	return c.cfg.Now()
}

func AssetName(goos, goarch string) (string, bool) {
	switch goos + "/" + goarch {
	case "darwin/arm64":
		return "picfetch-macos-arm64.zip", true
	case "darwin/amd64":
		return "picfetch-macos-x86_64.zip", true
	case "windows/amd64":
		return "picfetch-windows-amd64.zip", true
	case "windows/arm64":
		return "picfetch-windows-arm64.zip", true
	case "linux/amd64":
		return "picfetch-linux-amd64.tar.gz", true
	case "linux/arm64":
		return "picfetch-linux-arm64.tar.gz", true
	default:
		return "", false
	}
}

// NormalizeVersion returns a canonical semver (leading v) or "".
func NormalizeVersion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] != 'v' {
		s = "v" + s
	}
	if !semver.IsValid(s) {
		return ""
	}
	if semver.Prerelease(s) != "" {
		return ""
	}
	return semver.Canonical(s)
}

// Newer reports whether latest is a stable semver strictly greater than current.
func Newer(current, latest string) bool {
	c := NormalizeVersion(current)
	l := NormalizeVersion(latest)
	if c == "" || l == "" {
		return false
	}
	return semver.Compare(l, c) > 0
}

func DayString(t time.Time) string {
	// Format uses t's Location. Production passes time.Now() (local).
	// Tests pass a time constructed in a fixed zone so the day is deterministic.
	return t.Format("2006-01-02")
}

// Due reports whether a check should run: never checked, or last check was
// a previous local calendar day. lastDay is DayString's format or empty.
func Due(lastDay string, now time.Time) bool {
	today := DayString(now)
	return lastDay == "" || lastDay != today
}
