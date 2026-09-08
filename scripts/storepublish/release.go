package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const repository = "frathe/picfetch"
const productID = "9P0DM0KTH01K"
const bundleName = "picfetch-microsoft-store.msixbundle"
const identityName = "OpenSourceDeveloperFloria.PicFetch"
const publisherName = "CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F"
const maxBundleBytes = 512 << 20

type release struct {
	Schema        int    `json:"schema"`
	Repository    string `json:"repository"`
	Tag           string `json:"tag"`
	Commit        string `json:"commit"`
	Version       string `json:"version"`
	BundleVersion string `json:"bundle_version"`
	RunID         int64  `json:"run_id"`
	Attempt       int    `json:"run_attempt"`
	Event         string `json:"event"`
	Ref           string `json:"ref"`
	BundleSHA     string `json:"bundle_sha256"`
	WACKSHA       string `json:"wack_sha256"`
	Notes         string `json:"notes"`
	ArtifactID    int64  `json:"artifact_id,omitempty"`
	// Local paths are reconstructed from the verified artifact on every invocation.
	Directory string `json:"-"`
}

type noteRelease struct{ Version, Notes string }

type manifestIdentity struct {
	Name         string `xml:"Name,attr"`
	Publisher    string `xml:"Publisher,attr"`
	Version      string `xml:"Version,attr"`
	Architecture string `xml:"ProcessorArchitecture,attr"`
}

func versionParts(v string) ([3]int, error) {
	var result [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return result, fmt.Errorf("version must be canonical MAJOR.MINOR.PATCH")
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || strconv.Itoa(n) != p || n < 0 || n > 65535 || (i == 0 && n == 0) {
			return result, fmt.Errorf("version is outside the Store range")
		}
		result[i] = n
	}
	return result, nil
}
func compareVersion(a, b string) (int, error) {
	x, err := versionParts(a)
	if err != nil {
		return 0, err
	}
	y, err := versionParts(b)
	if err != nil {
		return 0, err
	}
	for i := range x {
		if x[i] < y[i] {
			return -1, nil
		}
		if x[i] > y[i] {
			return 1, nil
		}
	}
	return 0, nil
}
func publicVersion(v string) (string, error) {
	if !strings.HasSuffix(v, ".0") {
		return "", fmt.Errorf("Store package revision must be zero")
	}
	v = strings.TrimSuffix(v, ".0")
	_, err := versionParts(v)
	return v, err
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err = io.Copy(h, io.LimitReader(f, maxBundleBytes+1)); err != nil {
		return "", err
	}
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > maxBundleBytes {
		return "", fmt.Errorf("artifact exceeds size limit")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func readLimited(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(b)) > limit {
		return nil, fmt.Errorf("input exceeds size limit")
	}
	return b, err
}
func zipEntry(z *zip.Reader, name string, limit int64) ([]byte, error) {
	var found *zip.File
	for _, f := range z.File {
		if f.Name == name {
			if found != nil {
				return nil, fmt.Errorf("duplicate archive entry")
			}
			found = f
		}
	}
	if found == nil || found.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("missing or oversized archive entry %s", name)
	}
	r, err := found.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err == nil && int64(len(b)) > limit {
		return nil, fmt.Errorf("archive entry exceeds size limit")
	}
	return b, err
}
func inspectBundle(path string) (string, string, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return "", "", fmt.Errorf("invalid MSIX bundle: %w", err)
	}
	defer func() { _ = z.Close() }()
	data, err := zipEntry(&z.Reader, "AppxMetadata/AppxBundleManifest.xml", 1<<20)
	if err != nil {
		return "", "", err
	}
	var m struct {
		Identity manifestIdentity `xml:"Identity"`
		Packages []struct {
			Type    string `xml:"Type,attr"`
			File    string `xml:"FileName,attr"`
			Arch    string `xml:"Architecture,attr"`
			Version string `xml:"Version,attr"`
		} `xml:"Packages>Package"`
	}
	if err = xml.Unmarshal(data, &m); err != nil {
		return "", "", err
	}
	if m.Identity.Name != identityName || m.Identity.Publisher != publisherName {
		return "", "", fmt.Errorf("bundle has the wrong Store identity")
	}
	arches := map[string]bool{}
	version := ""
	for _, p := range m.Packages {
		if p.Type != "application" || (p.Arch != "x64" && p.Arch != "arm64") || arches[p.Arch] || filepath.Base(p.File) != p.File {
			return "", "", fmt.Errorf("unexpected bundle package")
		}
		data, err = zipEntry(&z.Reader, p.File, maxBundleBytes)
		if err != nil {
			return "", "", err
		}
		inner, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return "", "", err
		}
		manifest, err := zipEntry(inner, "AppxManifest.xml", 1<<20)
		if err != nil {
			return "", "", err
		}
		var app struct {
			Identity manifestIdentity `xml:"Identity"`
		}
		if err = xml.Unmarshal(manifest, &app); err != nil {
			return "", "", err
		}
		id := app.Identity
		if id.Name != identityName || id.Publisher != publisherName || id.Architecture != p.Arch || id.Version != p.Version {
			return "", "", fmt.Errorf("inner package identity does not match bundle")
		}
		v, err := publicVersion(id.Version)
		if err != nil {
			return "", "", err
		}
		if version != "" && version != v {
			return "", "", fmt.Errorf("package versions disagree")
		}
		version = v
		arches[p.Arch] = true
	}
	if len(arches) != 2 {
		return "", "", fmt.Errorf("bundle must contain x64 and ARM64")
	}
	if _, err = publicVersion(m.Identity.Version); err != nil {
		return "", "", fmt.Errorf("invalid outer bundle version: %w", err)
	}
	return version, m.Identity.Version, nil
}
func appVersion(data []byte) (string, error) {
	matches := regexp.MustCompile(`(?m)^Version\s*=\s*"([^"]+)"\s*$`).FindAllSubmatch(data, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("application version is missing or ambiguous")
	}
	v := string(matches[0][1])
	_, err := versionParts(v)
	return v, err
}
func validateNotes(version, notes string) error {
	pattern := `(?m)^\*\*Full Changelog\*\*: https://github\.com/frathe/picfetch/compare/(v[0-9]+\.[0-9]+\.[0-9]+)\.\.\.(v[0-9]+\.[0-9]+\.[0-9]+)\s*$`
	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(notes, -1)
	if len(matches) != 1 || matches[0][2] != "v"+version {
		return fmt.Errorf("missing or stale release notes for %s", version)
	}
	n, err := compareVersion(strings.TrimPrefix(matches[0][1], "v"), version)
	if err != nil || n >= 0 {
		return fmt.Errorf("invalid release-note version range")
	}
	return nil
}
func recordRelease(root, bundle, wack, out string, env func(string) string) error {
	version, bundleVersion, err := inspectBundle(bundle)
	if err != nil {
		return err
	}
	data, err := readLimited(filepath.Join(root, "FyneApp.toml"), 1<<20)
	if err != nil {
		return err
	}
	app, err := appVersion(data)
	if err != nil {
		return err
	}
	if app != version {
		return fmt.Errorf("application and package versions disagree")
	}
	notes, err := readLimited(filepath.Join(root, ".github/release-notes.md"), 1<<20)
	if err != nil {
		return err
	}
	if err = validateNotes(version, string(notes)); err != nil {
		return err
	}
	sha, err := fileDigest(bundle)
	if err != nil {
		return err
	}
	wackSHA, err := fileDigest(wack)
	if err != nil {
		return err
	}
	runID, err := strconv.ParseInt(env("GITHUB_RUN_ID"), 10, 64)
	if err != nil || runID <= 0 {
		return fmt.Errorf("invalid producing run ID")
	}
	attempt, err := strconv.Atoi(env("GITHUB_RUN_ATTEMPT"))
	if err != nil || attempt <= 0 {
		return fmt.Errorf("invalid producing run attempt")
	}
	r := release{Schema: 1, Repository: env("GITHUB_REPOSITORY"), Tag: "v" + version, Commit: env("GITHUB_SHA"), Version: version, BundleVersion: bundleVersion, RunID: runID, Attempt: attempt, Event: env("GITHUB_EVENT_NAME"), Ref: env("GITHUB_REF"), BundleSHA: sha, WACKSHA: wackSHA, Notes: string(notes)}
	if err = validReleaseProvenance(r); err != nil {
		return err
	}
	return writeJSON(out, r)
}
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}
func loadRelease(path string) (release, error) {
	var r release
	b, err := readLimited(path, 2<<20)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(b, &r); err != nil {
		return r, err
	}
	r.Directory = filepath.Dir(path)
	return r, nil
}
func validReleaseProvenance(r release) error {
	if r.Schema != 1 || r.Repository != repository || r.Event != "push" || r.Ref != "refs/tags/"+r.Tag || r.Tag != "v"+r.Version || r.RunID <= 0 || r.Attempt <= 0 || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(r.Commit) {
		return fmt.Errorf("release provenance is not an eligible stable tag")
	}
	_, err := versionParts(r.Version)
	return err
}
func admit(r release, base string) error {
	if err := validReleaseProvenance(r); err != nil {
		return err
	}
	n, err := compareVersion(r.Version, base)
	if err != nil {
		return err
	}
	if n <= 0 {
		return fmt.Errorf("release is not newer than the published Store version")
	}
	if err = validateNotes(r.Version, r.Notes); err != nil {
		return err
	}
	sha, err := fileDigest(filepath.Join(r.Directory, bundleName))
	if err != nil {
		return err
	}
	if sha != r.BundleSHA {
		return fmt.Errorf("bundle digest mismatch")
	}
	wack, err := fileDigest(filepath.Join(r.Directory, "wack-report.xml"))
	if err != nil {
		return err
	}
	if wack != r.WACKSHA {
		return fmt.Errorf("WACK report digest mismatch")
	}
	v, bundleVersion, err := inspectBundle(filepath.Join(r.Directory, bundleName))
	if err != nil {
		return err
	}
	if v != r.Version || bundleVersion != r.BundleVersion {
		return fmt.Errorf("bundle and release versions disagree")
	}
	return nil
}
