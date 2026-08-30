package update

import (
	"net/url"
	"testing"
	"time"
)

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
		ok                 bool
	}{
		{"darwin", "arm64", "picfetch-macos-arm64.zip", true},
		{"darwin", "amd64", "picfetch-macos-x86_64.zip", true},
		{"windows", "amd64", "picfetch-windows-amd64.zip", true},
		{"windows", "arm64", "picfetch-windows-arm64.zip", true},
		{"linux", "amd64", "picfetch-linux-amd64.tar.gz", true},
		{"linux", "arm64", "picfetch-linux-arm64.tar.gz", true},
		{"freebsd", "amd64", "", false},
		{"linux", "386", "", false},
	}
	for _, tc := range tests {
		got, ok := AssetName(tc.goos, tc.goarch)
		if got != tc.want || ok != tc.ok {
			t.Errorf("AssetName(%q, %q) = %q, %v; want %q, %v",
				tc.goos, tc.goarch, got, ok, tc.want, tc.ok)
		}
	}
}

func TestNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"0.2.5", "v0.2.6", true},
		{"v0.2.5", "v0.2.5", false},
		{"0.2.6", "v0.2.5", false},
		{"", "v0.2.6", false},
		{"0.2.5", "not-a-version", false},
		{"0.2.5", "v0.2.6-rc.1", false}, // prerelease latest is not newer-stable
	}
	for _, tc := range tests {
		if got := Newer(tc.current, tc.latest); got != tc.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestDue(t *testing.T) {
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.Local)
	if !Due("", now) {
		t.Error("empty last day must be due (first check after enabling)")
	}
	if Due("2026-08-26", now) {
		t.Error("same local calendar day must not be due")
	}
	if !Due("2026-08-25", now) {
		t.Error("previous local calendar day must be due")
	}
}

func TestDayStringUsesLocalDate(t *testing.T) {
	loc := time.FixedZone("test", 2*60*60)
	ts := time.Date(2026, 8, 26, 1, 0, 0, 0, loc)
	if got := DayString(ts); got != "2026-08-26" {
		t.Errorf("DayString = %q, want 2026-08-26 in the timestamp's location", got)
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"0.2.5", "v0.2.5"},
		{"v0.2.5", "v0.2.5"},
		{"  0.2.5  ", "v0.2.5"},
		{"1.0", "v1.0.0"},
		{"", ""},
		{"not-a-version", ""},
		{"v0.2.6-rc.1", ""},
	}
	for _, tc := range tests {
		if got := NormalizeVersion(tc.in); got != tc.want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDownloadPageURL_IsAParseableLinkToTheProjectSite(t *testing.T) {
	// The failure dialog parses this constant at the moment the user asks
	// for the download page, so a typo would surface as a dead button
	// rather than a build error.
	u, err := url.Parse(DownloadPageURL)
	if err != nil {
		t.Fatalf("url.Parse(%q) = %v", DownloadPageURL, err)
	}
	if u.Scheme != "https" {
		t.Errorf("scheme = %q, want https", u.Scheme)
	}
	if want := RepoOwner + ".github.io"; u.Host != want {
		t.Errorf("host = %q, want %q", u.Host, want)
	}
	if want := "/" + RepoName + "/"; u.Path != want {
		t.Errorf("path = %q, want %q", u.Path, want)
	}
	if u.Fragment != "downloads" {
		t.Errorf("fragment = %q, want downloads - the link has to land on the download list", u.Fragment)
	}
}
