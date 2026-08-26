package update

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const githubAPIVersion = "2022-11-28"

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Body       string    `json:"body"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

// Check reports a Release when an update should be downloaded, or (nil, nil)
// when none. Errors are network/API/parse failures.
func (c *Client) Check(ctx context.Context, currentVersion string) (*Release, error) {
	url := c.cfg.BaseURL + "/repos/" + RepoOwner + "/" + RepoName + "/releases/latest"
	req, err := c.newGitHubRequest(ctx, url, "picfetch/"+currentVersion)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github latest release: %s", resp.Status)
	}

	var gh ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, fmt.Errorf("github latest release: %w", err)
	}
	if gh.Draft || gh.Prerelease || !Newer(currentVersion, gh.TagName) {
		return nil, nil
	}

	want, ok := AssetName(c.cfg.GOOS, c.cfg.GOARCH)
	if !ok {
		return nil, nil
	}
	var asset *ghAsset
	for i := range gh.Assets {
		if gh.Assets[i].Name == want {
			asset = &gh.Assets[i]
			break
		}
	}
	if asset == nil {
		return nil, fmt.Errorf("no asset %q in release %s", want, gh.TagName)
	}
	digest, err := parseAssetDigest(asset.Digest)
	if err != nil {
		return nil, err
	}
	return &Release{
		Version:     NormalizeVersion(gh.TagName),
		Notes:       gh.Body,
		AssetName:   asset.Name,
		AssetURL:    asset.BrowserDownloadURL,
		AssetDigest: digest,
	}, nil
}

type ghAttestations struct {
	Attestations []struct {
		Bundle json.RawMessage `json:"bundle"`
	} `json:"attestations"`
}

func (c *Client) fetchReleaseAttestation(ctx context.Context, digestHex, userAgent string) ([]byte, error) {
	rawURL := c.cfg.BaseURL + "/repos/" + RepoOwner + "/" + RepoName + "/attestations/sha256:" + digestHex + "?predicate_type=release"
	req, err := c.newGitHubRequest(ctx, rawURL, userAgent)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github attestations: %s", resp.Status)
	}
	var gh ghAttestations
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, fmt.Errorf("github attestations: %w", err)
	}
	if len(gh.Attestations) == 0 {
		return nil, fmt.Errorf("github attestations: empty")
	}
	if len(gh.Attestations[0].Bundle) == 0 {
		return nil, fmt.Errorf("github attestations: missing bundle")
	}
	return gh.Attestations[0].Bundle, nil
}

func (c *Client) newGitHubRequest(ctx context.Context, rawURL, userAgent string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.cfg.HTTP == nil {
		return nil, fmt.Errorf("update: nil HTTP client")
	}
	return c.cfg.HTTP.Do(req)
}

func parseAssetDigest(digest string) (string, error) {
	if digest == "" {
		return "", nil
	}
	hexPart := strings.TrimPrefix(digest, "sha256:")
	if hexPart == digest {
		return "", fmt.Errorf("unsupported asset digest %q", digest)
	}
	hexPart = strings.ToLower(hexPart)
	decoded, err := hex.DecodeString(hexPart)
	if err != nil || len(decoded) != 32 {
		return "", fmt.Errorf("unsupported asset digest %q", digest)
	}
	return hexPart, nil
}
