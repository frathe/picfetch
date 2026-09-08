package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf16"
)

type object map[string]any

func parseObject(b []byte) (object, error) {
	var o object
	err := json.Unmarshal(b, &o)
	if err == nil && o == nil {
		err = fmt.Errorf("expected JSON object")
	}
	return o, err
}
func stringField(o object, key string) string { s, _ := o[key].(string); return s }
func asObject(v any) object {
	o, _ := v.(map[string]any)
	if o == nil {
		o, _ = v.(object)
	}
	return o
}
func packageVersion(o object) (string, error) {
	packages, ok := o["applicationPackages"].([]any)
	if !ok {
		return "", fmt.Errorf("published submission has no packages")
	}
	latest := ""
	for _, v := range packages {
		p := asObject(v)
		if stringField(p, "fileStatus") == "PendingDelete" {
			continue
		}
		version, err := publicVersion(stringField(p, "version"))
		if err != nil {
			return "", err
		}
		if latest == "" {
			latest = version
		} else {
			cmp, err := compareVersion(version, latest)
			if err != nil {
				return "", err
			}
			if cmp > 0 {
				latest = version
			}
		}
	}
	if latest == "" {
		return "", fmt.Errorf("published submission has no package versions")
	}
	return latest, nil
}

func publishedVersion(o object, r *receipt) (string, error) {
	actual, err := packageVersion(o)
	if err != nil {
		return "", err
	}
	id := stringField(o, "id")
	if r != nil {
		if id == r.SubmissionID {
			outer, err := publicVersion(r.Release.BundleVersion)
			if err != nil {
				return "", err
			}
			if actual != outer && actual != r.Release.Version {
				return "", fmt.Errorf("published package version differs from the recorded bundle")
			}
			return r.Release.Version, nil
		}
		if id == r.BaseSubmission && actual == r.BasePackageVersion {
			return r.BaseVersion, nil
		}
	}
	// Bootstrap only the user-confirmed accepted 1.0.2 release. The outer version
	// is from the actual artifact of Store producer run 33983485123; MakeAppx
	// generated it independently of the inner application's 1.0.2.0 version.
	if actual == "1.0.2" || actual == "2026.905.1829" {
		return "1.0.2", nil
	}
	return "", fmt.Errorf("published package version %s.0 is not bound to a release receipt; inspect the published submission", actual)
}
func plainText(s string) string {
	s = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
	s = strings.NewReplacer("**", "", "__", "", "`", "", "*", "", "\\", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
func noteEntries(notes string) []string {
	var entries []string
	var current []string
	skipDepth := 0
	fenced := false
	flush := func() {
		if len(current) > 0 {
			s := plainText(strings.Join(current, " "))
			if s != "" && !platformOnly(s) {
				entries = append(entries, s)
			}
			current = nil
		}
	}
	for line := range strings.SplitSeq(strings.ReplaceAll(notes, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "```") {
			flush()
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if strings.HasPrefix(line, "**Full Changelog**:") {
			flush()
			continue
		}
		if strings.HasPrefix(line, "#") {
			flush()
			depth := len(line) - len(strings.TrimLeft(line, "#"))
			title := plainText(strings.TrimSpace(line[depth:]))
			if skipDepth > 0 && depth <= skipDepth {
				skipDepth = 0
			}
			if strings.EqualFold(title, "Internal") || platformOnly(title) {
				skipDepth = depth
			}
			continue
		}
		if skipDepth > 0 {
			continue
		}
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			flush()
			line = line[2:]
		}
		current = append(current, line)
	}
	flush()
	return entries
}
func platformOnly(s string) bool {
	lower := strings.ToLower(s)
	if regexp.MustCompile(`\bwindows\b|\bcross.platform\b|\ball platforms\b`).MatchString(lower) {
		return false
	}
	return regexp.MustCompile(`^(?:on\s+)?\[?(?:linux|macos|mac os|mac)(?:\]|:|\s|$)`).MatchString(lower)
}
func generateNotes(base, target string, items []noteRelease) (string, error) {
	if _, err := versionParts(base); err != nil {
		return "", err
	}
	if _, err := versionParts(target); err != nil {
		return "", err
	}
	items = append([]noteRelease(nil), items...)
	for _, r := range items {
		if err := validateNotes(r.Version, r.Notes); err != nil {
			return "", err
		}
	}
	sort.Slice(items, func(i, j int) bool { n, _ := compareVersion(items[i].Version, items[j].Version); return n > 0 })
	if len(items) == 0 || items[0].Version != target {
		return "", fmt.Errorf("target release notes are missing")
	}
	// Each release links to its predecessor; all changes since the Store base
	// must be represented, including versions skipped by the Store queue.
	expected := target
	for _, item := range items {
		cmp, _ := compareVersion(item.Version, base)
		if cmp <= 0 {
			continue
		}
		if item.Version != expected {
			return "", fmt.Errorf("release-note range has a gap or duplicate")
		}
		match := regexp.MustCompile(`compare/v([0-9]+\.[0-9]+\.[0-9]+)\.\.\.v`).FindStringSubmatch(item.Notes)
		if len(match) != 2 {
			return "", fmt.Errorf("release-note predecessor is missing")
		}
		expected = match[1]
	}
	if expected != base {
		return "", fmt.Errorf("release-note range does not reach the published Store version")
	}
	tail := "\n\nFull changes: https://github.com/" + repository + "/compare/v" + base + "...v" + target
	result := "Version " + target
	for _, r := range items {
		cmp, err := compareVersion(r.Version, base)
		if err != nil {
			return "", err
		}
		if cmp <= 0 {
			continue
		}
		for _, entry := range noteEntries(r.Notes) {
			next := result + "\n- " + entry
			if len(utf16.Encode([]rune(next+tail))) <= 1500 {
				result = next
			}
		}
	}
	return result + tail, nil
}
func prepareSubmission(source object, notes string) (object, error) {
	data, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	out, err := parseObject(data)
	if err != nil {
		return nil, err
	}
	listings := asObject(out["listings"])
	if len(listings) == 0 {
		return nil, fmt.Errorf("published Store listings are missing")
	}
	for _, v := range listings {
		listing := asObject(v)
		base := asObject(listing["baseListing"])
		if base == nil {
			return nil, fmt.Errorf("invalid Store listing")
		}
		base["releaseNotes"] = notes
		for _, override := range asObject(listing["platformOverrides"]) {
			o := asObject(override)
			if o == nil {
				return nil, fmt.Errorf("invalid listing override")
			}
			if _, ok := o["releaseNotes"]; ok {
				o["releaseNotes"] = notes
			}
		}
	}
	old, ok := out["applicationPackages"].([]any)
	if !ok {
		return nil, fmt.Errorf("submission packages are missing")
	}
	for _, v := range old {
		p := asObject(v)
		if p == nil {
			return nil, fmt.Errorf("invalid package metadata")
		}
		p["fileStatus"] = "PendingDelete"
	}
	out["applicationPackages"] = append(old, object{"fileName": bundleName, "fileStatus": "PendingUpload", "minimumDirectXVersion": "None", "minimumSystemRam": "None"})
	out["targetPublishMode"] = "Immediate"
	if options := asObject(out["packageDeliveryOptions"]); options != nil {
		if rollout := asObject(options["packageRollout"]); rollout != nil {
			rollout["isPackageRollout"] = false
		}
	}
	return out, nil
}
func metadataDigest(o object) (string, error) {
	// Provider-generated status, package ingestion fields and upload URLs change
	// independently of the listing and product metadata we preserve.
	stable := object{}
	for k, v := range o {
		switch k {
		case "id", "status", "statusDetails", "fileUploadUrl", "applicationPackages", "friendlyName":
			continue
		}
		stable[k] = v
	}
	b, err := json.Marshal(stable)
	return digest(b), err
}
