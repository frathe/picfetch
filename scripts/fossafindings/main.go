// Command fossafindings retrieves a revision-scoped, read-only FOSSA report.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const project = "git+github.com/frathe/picfetch"

type issue struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	License string `json:"license"`
	Source  struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"source"`
	Projects []struct {
		ID             string `json:"id"`
		RevisionID     string `json:"revisionId"`
		RevisionScanID int    `json:"revisionScanId"`
	} `json:"projects"`
}

type revisionState struct {
	Locator  string `json:"locator"`
	Resolved bool   `json:"resolved"`
	IsSteady bool   `json:"isSteady"`
	IsStale  bool   `json:"is_stale"`
	Error    string `json:"error"`
	ScanID   int    `json:"latestRevisionScanId"`
	Count    int    `json:"unresolved_licensing_issue_count"`
}

type command struct {
	getenv  func(string) string
	github  func(context.Context, string) ([]byte, error)
	client  *http.Client
	baseURL string
	output  io.Writer
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c := command{
		getenv: os.Getenv, github: githubHead,
		client:  &http.Client{Timeout: 45 * time.Second},
		baseURL: "https://app.fossa.com", output: os.Stdout,
	}
	if err := c.run(ctx); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "fossa-findings:", err)
		os.Exit(1)
	}
}

func githubHead(ctx context.Context, pr string) ([]byte, error) {
	args := []string{"pr", "view"}
	if pr != "" {
		args = append(args, pr)
	}
	args = append(args, "--repo", "frathe/picfetch", "--json", "headRefOid")
	return exec.CommandContext(ctx, "gh", args...).Output()
}

func (c command) run(ctx context.Context) error {
	key, err := readKey(c.getenv)
	if err != nil {
		return err
	}
	return c.retrieve(ctx, key)
}

func readKey(getenv func(string) string) (string, error) {
	key := getenv("FOSSA_API_KEY")
	if key == "" {
		path := getenv("FOSSA_ENV_FILE")
		if path == "" {
			path = ".env.local"
		}
		file, err := os.Open(path)
		if err != nil {
			return "", errors.New("set FOSSA_API_KEY or put it in .env.local (override with FOSSA_ENV_FILE)")
		}
		defer func() { _ = file.Close() }()
		scanner := bufio.NewScanner(io.LimitReader(file, 1024*1024))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			line = strings.TrimPrefix(line, "export ")
			name, value, found := strings.Cut(line, "=")
			if !found || strings.TrimSpace(name) != "FOSSA_API_KEY" {
				continue
			}
			if key != "" {
				return "", errors.New("duplicate FOSSA_API_KEY assignment")
			}
			key = strings.TrimSpace(value)
			if len(key) >= 2 && (key[0] == '\'' && key[len(key)-1] == '\'' || key[0] == '"' && key[len(key)-1] == '"') {
				key = key[1 : len(key)-1]
			}
		}
		if scanner.Err() != nil {
			return "", errors.New("cannot parse credential file")
		}
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._~+/=-]+$`).MatchString(key) {
		return "", errors.New("missing or invalid FOSSA_API_KEY; use a literal token, not shell expressions")
	}
	return key, nil
}

func (c command) retrieve(ctx context.Context, key string) (resultErr error) {
	defer func() {
		if resultErr != nil {
			resultErr = errors.New(strings.ReplaceAll(resultErr.Error(), key, "[REDACTED]"))
		}
	}()
	sha := c.getenv("FOSSA_REVISION")
	pr := c.getenv("PR")
	if sha != "" && pr != "" {
		return errors.New("use PR or FOSSA_REVISION, not both")
	}
	if pr != "" && !regexp.MustCompile(`^[1-9][0-9]*$`).MatchString(pr) {
		return errors.New("PR must be a positive number")
	}
	if sha == "" {
		raw, err := c.github(ctx, pr)
		if err != nil {
			return errors.New("cannot read PR head; authenticate gh or set FOSSA_REVISION")
		}
		var head struct{ HeadRefOid string }
		if err := json.Unmarshal(raw, &head); err != nil {
			return errors.New("invalid GitHub PR response")
		}
		sha = head.HeadRefOid
	}
	if !regexp.MustCompile(`^[0-9a-fA-F]{40}$`).MatchString(sha) {
		return errors.New("expected a full 40-character revision SHA")
	}
	sha = strings.ToLower(sha)
	client := *c.client
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	get := func(path string, query url.Values) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+query.Encode(), nil)
		if err != nil {
			return nil, errors.New("cannot construct FOSSA request")
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Accept", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return nil, errors.New("FOSSA request failed (network or timeout); report incomplete")
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("FOSSA HTTP %d; report incomplete (202 means analysis pending)", res.StatusCode)
		}
		const maxResponse = 32 * 1024 * 1024
		raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponse+1))
		if err != nil || len(raw) > maxResponse {
			return nil, errors.New("FOSSA response unreadable or too large; report incomplete")
		}
		if !json.Valid(raw) {
			return nil, errors.New("invalid FOSSA JSON; report incomplete")
		}
		// Normalize JSON escapes before redaction; a reflected token must not
		// survive as escaped characters. Keep JSON numbers exact in evidence.
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, errors.New("cannot decode FOSSA response")
		}
		normalized, err := json.Marshal(value)
		if err != nil {
			return nil, errors.New("cannot normalize FOSSA response")
		}
		return []byte(strings.ReplaceAll(string(normalized), key, "[REDACTED]")), nil
	}
	revisionPath := "/api/revisions/" + url.PathEscape(project+"$"+sha)
	revisionRaw, err := get(revisionPath, nil)
	if err != nil {
		return err
	}
	// A missing or null count must not look like a confirmed zero.
	state := revisionState{Count: -1}
	if err := json.Unmarshal(revisionRaw, &state); err != nil {
		return errors.New("invalid revision response")
	}
	if state.Locator != project+"$"+sha || !state.Resolved || !state.IsSteady || state.IsStale || state.Error != "" || state.ScanID <= 0 || state.Count < 0 {
		return errors.New("revision analysis is not ready; report incomplete")
	}
	var all []json.RawMessage
	var parsed []issue
	pages := make(map[string][]byte)
	seen := make(map[int]bool)
	for page := 1; ; page++ {
		if page > 1000 {
			return errors.New("pagination limit reached; report incomplete")
		}
		query := url.Values{
			"category": {"licensing"}, "status": {"active"}, "scope[type]": {"project"},
			"scope[id]": {project}, "scope[revision]": {sha},
			"scope[revisionScanId]": {strconv.Itoa(state.ScanID)},
			"sort":                  {"created_at_asc"}, "page": {strconv.Itoa(page)}, "count": {"100"},
		}
		raw, err := get("/api/v2/issues", query)
		if err != nil {
			return err
		}
		var response struct {
			Issues []json.RawMessage `json:"issues"`
		}
		if err := json.Unmarshal(raw, &response); err != nil || response.Issues == nil {
			return errors.New("missing or invalid issues array; report incomplete")
		}
		pages[fmt.Sprintf("issues-page-%04d.json", page)] = raw
		if len(response.Issues) == 0 {
			break
		}
		for _, item := range response.Issues {
			var finding issue
			if err := json.Unmarshal(item, &finding); err != nil || finding.ID <= 0 || finding.Source.ID == "" {
				return errors.New("invalid issue; report incomplete")
			}
			if seen[finding.ID] {
				return errors.New("repeated issue across pages; report incomplete")
			}
			matched := false
			for _, affected := range finding.Projects {
				if affected.ID == project && affected.RevisionID == project+"$"+sha && affected.RevisionScanID == state.ScanID {
					matched = true
				}
			}
			if !matched {
				return errors.New("issue does not match requested revision/scan; report incomplete")
			}
			seen[finding.ID] = true
			all = append(all, item)
			parsed = append(parsed, finding)
		}
	}
	if len(all) != state.Count {
		return errors.New("licensing issue count differs from revision summary; report incomplete, retry")
	}
	finalRaw, err := get(revisionPath, nil)
	if err != nil {
		return err
	}
	final := revisionState{Count: -1}
	if err := json.Unmarshal(finalRaw, &final); err != nil || final != state {
		return errors.New("revision scan changed during retrieval; retry")
	}
	dir := c.getenv("FOSSA_OUTPUT_DIR")
	if dir == "" {
		dir = ".scratch/fossa-findings"
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	dir, err = os.MkdirTemp(dir, time.Now().UTC().Format("20060102T150405Z")+"-"+sha[:12]+"-")
	if err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	pages["revision.json"] = revisionRaw
	for name, raw := range pages {
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			return errors.New("cannot save raw evidence; report incomplete")
		}
	}
	if all == nil {
		all = []json.RawMessage{}
	}
	report := struct {
		Project     string            `json:"project"`
		Revision    string            `json:"revision"`
		ScanID      int               `json:"scanId"`
		RetrievedAt string            `json:"retrievedAt"`
		Issues      []json.RawMessage `json:"issues"`
	}{project, sha, state.ScanID, time.Now().UTC().Format(time.RFC3339), all}
	reportRaw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return errors.New("cannot encode report")
	}
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].ID < parsed[j].ID })
	var summary strings.Builder
	_, _ = fmt.Fprintf(&summary, "# FOSSA licensing findings\n\nRevision: %s\n\nScan: %d\n\n%d active licensing findings. Retrieval success is not CI approval.\n\n", sha, state.ScanID, len(parsed))
	for _, item := range parsed {
		_, _ = fmt.Fprintf(&summary, "- %d: %s; %s; %s @ %s\n", item.ID, item.Type, item.License, item.Source.Name, item.Source.Version)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte(summary.String()), 0600); err != nil {
		return errors.New("cannot save summary; report incomplete")
	}
	// Publish last and atomically: report.json is the completion marker.
	reportTemp := filepath.Join(dir, ".report.json.tmp")
	if err := os.WriteFile(reportTemp, reportRaw, 0600); err != nil {
		return errors.New("cannot save report; report incomplete")
	}
	if err := os.Rename(reportTemp, filepath.Join(dir, "report.json")); err != nil {
		return errors.New("cannot publish report; report incomplete")
	}
	_, err = fmt.Fprintf(c.output, "%s\nSaved report: %s\n", summary.String(), dir)
	return err
}
