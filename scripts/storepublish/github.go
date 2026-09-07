package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const githubPageSize = 100
const githubMaxPages = 10
const receiptTask = "picfetch-store-receipt"
const receiptEnvironment = "microsoft-store"

type githubRun struct {
	ID             int64  `json:"id"`
	Attempt        int    `json:"run_attempt"`
	SHA            string `json:"head_sha"`
	Branch         string `json:"head_branch"`
	Event          string `json:"event"`
	Path           string `json:"path"`
	Conclusion     string `json:"conclusion"`
	HeadRepository struct {
		Name string `json:"full_name"`
	} `json:"head_repository"`
}

type githubArtifact struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Expired bool   `json:"expired"`
	Digest  string `json:"digest"`
	Run     struct {
		ID int64 `json:"id"`
	} `json:"workflow_run"`
}

// api never follows a redirect with GitHub credentials. Error bodies and URLs
// are deliberately excluded from errors because artifact URLs can contain SAS.
func (g githubAPI) api(ctx context.Context, method, path string, body []byte, limit int64) ([]byte, http.Header, int, error) {
	token := g.rt.Env("GH_TOKEN")
	if token == "" {
		token = g.rt.Env("GITHUB_TOKEN")
	}
	if token == "" {
		return nil, nil, 0, fmt.Errorf("GitHub token is missing")
	}
	client := *g.rt.HTTP
	client.Jar = nil
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	for attempt := range 3 {
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(g.rt.GitHubURL, "/")+path, bytes.NewReader(body))
		if err != nil {
			return nil, nil, 0, fmt.Errorf("invalid GitHub request configuration")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		response, err := client.Do(req)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("GitHub request failed; retry the command")
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
		_ = response.Body.Close()
		if method == http.MethodGet && (response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500) && attempt < 2 {
			delay := retryDelay(response.Header.Get("Retry-After"), g.rt.Now(), attempt)
			if err = g.rt.Wait(ctx, delay); err != nil {
				return nil, nil, 0, fmt.Errorf("GitHub retry cancelled")
			}
			continue
		}
		if readErr != nil || int64(len(data)) > limit {
			return nil, nil, 0, fmt.Errorf("GitHub response is unreadable or exceeds the size limit")
		}
		return data, response.Header, response.StatusCode, nil
	}
	return nil, nil, 0, fmt.Errorf("GitHub request exhausted retries")
}

func (g githubAPI) json(ctx context.Context, method, path string, body []byte, result any) error {
	b, _, status, err := g.api(ctx, method, path, body, 8<<20)
	if err != nil {
		return err
	}
	want := http.StatusOK
	if method == http.MethodPost {
		want = http.StatusCreated
	}
	if status != want {
		return fmt.Errorf("GitHub %s failed (HTTP %d)", method, status)
	}
	if err = json.Unmarshal(b, result); err != nil {
		return fmt.Errorf("GitHub returned invalid JSON")
	}
	return nil
}

func (g githubAPI) runs(ctx context.Context, sha string) ([]githubRun, error) {
	var runs []githubRun
	seen := map[int64]bool{}
	for page := 1; page <= githubMaxPages; page++ {
		var result struct {
			Total int         `json:"total_count"`
			Runs  []githubRun `json:"workflow_runs"`
		}
		query := url.Values{"event": {"push"}, "status": {"success"}, "head_sha": {sha}, "per_page": {strconv.Itoa(githubPageSize)}, "page": {strconv.Itoa(page)}}
		path := "/repos/" + repository + "/actions/workflows/microsoft-store.yml/runs?" + query.Encode()
		if err := g.json(ctx, http.MethodGet, path, nil, &result); err != nil {
			return nil, err
		}
		if result.Total > githubPageSize*githubMaxPages || len(result.Runs) > githubPageSize {
			return nil, fmt.Errorf("GitHub run history exceeds the discovery limit")
		}
		for _, run := range result.Runs {
			if run.ID <= 0 || seen[run.ID] {
				return nil, fmt.Errorf("GitHub run history is invalid or changed during discovery")
			}
			seen[run.ID] = true
			runs = append(runs, run)
		}
		if len(result.Runs) < githubPageSize {
			if result.Total > len(runs) {
				return nil, fmt.Errorf("GitHub run history is incomplete")
			}
			return runs, nil
		}
	}
	return nil, fmt.Errorf("GitHub run history exceeds the discovery limit")
}

func (g githubAPI) artifact(ctx context.Context, runID int64) (githubArtifact, error) {
	var selected githubArtifact
	seen := map[int64]bool{}
	count := 0
	for page := 1; page <= githubMaxPages; page++ {
		var result struct {
			Total     int              `json:"total_count"`
			Artifacts []githubArtifact `json:"artifacts"`
		}
		path := fmt.Sprintf("/repos/%s/actions/runs/%d/artifacts?per_page=%d&page=%d", repository, runID, githubPageSize, page)
		if err := g.json(ctx, http.MethodGet, path, nil, &result); err != nil {
			return selected, err
		}
		if result.Total > githubPageSize*githubMaxPages || len(result.Artifacts) > githubPageSize {
			return selected, fmt.Errorf("GitHub artifact history exceeds the discovery limit")
		}
		for _, artifact := range result.Artifacts {
			if artifact.ID <= 0 || seen[artifact.ID] {
				return selected, fmt.Errorf("GitHub artifact history is invalid or changed during discovery")
			}
			seen[artifact.ID] = true
			count++
			if artifact.Name != "picfetch-microsoft-store" {
				continue
			}
			if artifact.Expired || artifact.Run.ID != runID || selected.ID != 0 {
				return selected, fmt.Errorf("Store artifact is expired, ambiguous or belongs to another run")
			}
			selected = artifact
		}
		if len(result.Artifacts) < githubPageSize {
			if result.Total > count {
				return selected, fmt.Errorf("GitHub artifact history is incomplete")
			}
			if selected.ID == 0 {
				return selected, fmt.Errorf("validated Store artifact is missing for run %d", runID)
			}
			return selected, nil
		}
	}
	return selected, fmt.Errorf("GitHub artifact history exceeds the discovery limit")
}

func (g githubAPI) download(ctx context.Context, artifact githubArtifact) ([]byte, error) {
	const limit = maxBundleBytes + 16<<20
	path := fmt.Sprintf("/repos/%s/actions/artifacts/%d/zip", repository, artifact.ID)
	b, headers, status, err := g.api(ctx, http.MethodGet, path, nil, limit)
	if err != nil {
		return nil, err
	}
	client := *g.rt.HTTP
	client.Jar = nil
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	for redirects := 0; status >= 300 && status < 400 && redirects < 5; redirects++ {
		target, parseErr := url.Parse(headers.Get("Location"))
		if parseErr != nil || target.Host == "" || target.User != nil || target.Fragment != "" {
			return nil, fmt.Errorf("GitHub artifact redirect is invalid")
		}
		ip := net.ParseIP(target.Hostname())
		local := target.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
		if target.Scheme != "https" && !(target.Scheme == "http" && local && strings.HasPrefix(g.rt.GitHubURL, "http://")) {
			return nil, fmt.Errorf("GitHub artifact redirect requires HTTPS")
		}
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
		if requestErr != nil {
			return nil, fmt.Errorf("GitHub artifact redirect is invalid")
		}
		// A fresh request intentionally carries no API Authorization header, even
		// for a same-host redirect, because downloads use their own signed URLs.
		response, requestErr := client.Do(req)
		if requestErr != nil {
			return nil, fmt.Errorf("GitHub artifact download failed; retry the command")
		}
		b, err = io.ReadAll(io.LimitReader(response.Body, limit+1))
		_ = response.Body.Close()
		if err != nil || int64(len(b)) > limit {
			return nil, fmt.Errorf("GitHub artifact exceeds the download limit or could not be read")
		}
		status, headers = response.StatusCode, response.Header
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GitHub artifact download failed (HTTP %d)", status)
	}
	if artifact.Digest != "" && artifact.Digest != "sha256:"+digest(b) {
		return nil, fmt.Errorf("GitHub artifact digest mismatch")
	}
	return b, nil
}

func extractStoreArtifact(data []byte, dir string) error {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("GitHub artifact is not a ZIP archive")
	}
	outputs := []struct {
		name  string
		limit int64
	}{{bundleName, maxBundleBytes}, {"wack-report.xml", 8 << 20}, {"store-release.json", 2 << 20}}
	if len(z.File) != len(outputs) {
		return fmt.Errorf("Store artifact must contain exactly the bundle, WACK report and release record")
	}
	for _, f := range z.File {
		if !f.Mode().IsRegular() {
			return fmt.Errorf("Store artifact contains an unexpected path or file type")
		}
	}
	// The three required unique entries account for the entire archive. Output
	// paths use only these trusted names, never a name supplied by the archive.
	for _, output := range outputs {
		b, err := zipEntry(z, output.name, output.limit)
		if err != nil {
			return fmt.Errorf("Store artifact contains an invalid or oversized file")
		}
		// Callers provide a fresh private staging directory. Exclusive creation
		// also refuses existing files or symlinks instead of overwriting them.
		file, err := os.OpenFile(filepath.Join(dir, output.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("could not stage Store artifact")
		}
		_, writeErr := file.Write(b)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			return fmt.Errorf("could not stage Store artifact")
		}
	}
	return nil
}

func (g githubAPI) candidates(ctx context.Context, root, tag, base string) (_ []release, err error) {
	return g.candidatesForRun(ctx, root, tag, base, 0)
}

func (g githubAPI) candidatesForRun(ctx context.Context, root, tag, base string, runID int64) (_ []release, err error) {
	if _, err = versionParts(base); err != nil {
		return nil, err
	}
	if tag != "" {
		if !strings.HasPrefix(tag, "v") {
			return nil, fmt.Errorf("requested release must be a stable vMAJOR.MINOR.PATCH tag")
		}
		if _, err = versionParts(strings.TrimPrefix(tag, "v")); err != nil {
			return nil, err
		}
	}
	tags, err := g.rt.Git(ctx, root, "for-each-ref", "--format=%(refname:strip=2)", "refs/tags/v*")
	if err != nil || len(tags) > 1<<20 {
		return nil, fmt.Errorf("could not enumerate release tags")
	}
	var eligible []string
	for candidateTag := range strings.FieldsSeq(string(tags)) {
		if tag != "" && candidateTag != tag || !strings.HasPrefix(candidateTag, "v") {
			continue
		}
		comparison, versionErr := compareVersion(strings.TrimPrefix(candidateTag, "v"), base)
		if versionErr == nil && comparison > 0 {
			eligible = append(eligible, candidateTag)
		}
	}
	if len(eligible) > 64 {
		return nil, fmt.Errorf("more than 64 Store releases are waiting; discovery limit exceeded")
	}
	sort.Slice(eligible, func(i, j int) bool { n, _ := compareVersion(eligible[i][1:], eligible[j][1:]); return n > 0 })
	// Inspect newest first. Skipped releases contribute tagged notes, so their
	// older binary artifacts need not remain available.
	for _, candidateTag := range eligible {
		commit, gitErr := g.rt.Git(ctx, root, "rev-parse", "--verify", "refs/tags/"+candidateTag+"^{commit}")
		sha := strings.TrimSpace(string(commit))
		if gitErr != nil || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(sha) {
			return nil, fmt.Errorf("could not resolve immutable release commit for %s", candidateTag)
		}
		if _, gitErr = g.rt.Git(ctx, root, "merge-base", "--is-ancestor", sha, "origin/main"); gitErr != nil {
			return nil, fmt.Errorf("release %s is not reachable from origin/main", candidateTag)
		}
		runs, fetchErr := g.runs(ctx, sha)
		if fetchErr != nil {
			return nil, fetchErr
		}
		var selected githubRun
		for _, run := range runs {
			if runID != 0 && run.ID != runID {
				continue
			}
			if run.SHA != sha || run.HeadRepository.Name != repository || run.Path != ".github/workflows/microsoft-store.yml" || run.Conclusion != "success" || run.Event != "push" || run.Attempt <= 0 || (run.Branch != "" && run.Branch != candidateTag) {
				continue
			}
			if run.ID > selected.ID {
				selected = run
			}
		}
		if selected.ID == 0 {
			continue
		}
		artifact, fetchErr := g.artifact(ctx, selected.ID)
		if fetchErr != nil {
			return nil, fetchErr
		}
		data, fetchErr := g.download(ctx, artifact)
		if fetchErr != nil {
			return nil, fetchErr
		}
		dir, stageErr := os.MkdirTemp(g.rt.Scratch, "picfetch-store-")
		if stageErr != nil {
			return nil, fmt.Errorf("could not create Store artifact directory")
		}
		r, stageErr := g.verifyArtifact(ctx, root, dir, data, candidateTag, sha, base, selected, artifact.ID)
		if stageErr != nil {
			_ = os.RemoveAll(dir)
			return nil, stageErr
		}
		return []release{r}, nil
	}
	if tag != "" {
		return nil, fmt.Errorf("no newer validated Store build exists for %s", tag)
	}
	return nil, nil
}

func (g githubAPI) verifyArtifact(ctx context.Context, root, dir string, data []byte, tag, sha, base string, run githubRun, artifactID int64) (release, error) {
	var r release
	if err := extractStoreArtifact(data, dir); err != nil {
		return r, err
	}
	r, err := loadRelease(filepath.Join(dir, "store-release.json"))
	if err != nil {
		return r, fmt.Errorf("Store release record is unreadable")
	}
	if r.Tag != tag || r.Commit != sha || r.RunID != run.ID || r.Attempt != run.Attempt || (r.ArtifactID != 0 && r.ArtifactID != artifactID) {
		return r, fmt.Errorf("Store artifact provenance does not match its tag and producing run")
	}
	if err = admit(r, base); err != nil {
		return r, err
	}
	app, err := g.rt.Git(ctx, root, "show", sha+":FyneApp.toml")
	if err != nil || len(app) > 1<<20 {
		return r, fmt.Errorf("tagged application version is unavailable")
	}
	version, err := appVersion(app)
	if err != nil || version != r.Version {
		return r, fmt.Errorf("tagged application version disagrees with Store artifact")
	}
	notes, err := g.rt.Git(ctx, root, "show", sha+":.github/release-notes.md")
	if err != nil || len(notes) > 1<<20 || string(notes) != r.Notes {
		return r, fmt.Errorf("tagged release notes disagree with Store artifact")
	}
	r.ArtifactID = artifactID
	return r, nil
}

// restore is deliberately tied to the receipt's run attempt and artifact ID.
// A later successful rebuild of the same tag cannot replace an in-flight upload.
func (g githubAPI) restore(ctx context.Context, root string, original release) (release, error) {
	var restored release
	if original.RunID <= 0 || original.Attempt <= 0 || original.ArtifactID <= 0 || original.Tag != "v"+original.Version {
		return restored, fmt.Errorf("Store receipt lacks original build identity")
	}
	if _, err := versionParts(original.Version); err != nil {
		return restored, fmt.Errorf("Store receipt has an invalid release version")
	}
	commit, err := g.rt.Git(ctx, root, "rev-parse", "--verify", "refs/tags/"+original.Tag+"^{commit}")
	if err != nil || strings.TrimSpace(string(commit)) != original.Commit {
		return restored, fmt.Errorf("receipt release tag has moved or is unavailable")
	}
	if _, err = g.rt.Git(ctx, root, "merge-base", "--is-ancestor", original.Commit, "origin/main"); err != nil {
		return restored, fmt.Errorf("receipt release is not reachable from origin/main")
	}
	var run githubRun
	path := fmt.Sprintf("/repos/%s/actions/runs/%d/attempts/%d", repository, original.RunID, original.Attempt)
	if err = g.json(ctx, http.MethodGet, path, nil, &run); err != nil {
		return restored, err
	}
	if run.ID != original.RunID || run.Attempt != original.Attempt || run.SHA != original.Commit || run.HeadRepository.Name != repository || run.Path != ".github/workflows/microsoft-store.yml" || run.Event != "push" || run.Conclusion != "success" || (run.Branch != "" && run.Branch != original.Tag) {
		return restored, fmt.Errorf("original Store build no longer matches receipt provenance")
	}
	var artifact githubArtifact
	path = fmt.Sprintf("/repos/%s/actions/artifacts/%d", repository, original.ArtifactID)
	if err = g.json(ctx, http.MethodGet, path, nil, &artifact); err != nil {
		return restored, err
	}
	if artifact.ID != original.ArtifactID || artifact.Run.ID != original.RunID || artifact.Name != "picfetch-microsoft-store" || artifact.Expired {
		return restored, fmt.Errorf("original Store artifact is missing, expired or no longer matches the receipt")
	}
	data, err := g.download(ctx, artifact)
	if err != nil {
		return restored, err
	}
	dir, err := os.MkdirTemp(g.rt.Scratch, "picfetch-store-")
	if err != nil {
		return restored, fmt.Errorf("could not create Store artifact directory")
	}
	// 1.0.2 was published before automation; its receipts can only describe
	// subsequent releases. The lifecycle separately checks the published base.
	restored, err = g.verifyArtifact(ctx, root, dir, data, original.Tag, original.Commit, "1.0.2", run, artifact.ID)
	if err != nil {
		_ = os.RemoveAll(dir)
		return restored, err
	}
	expected, actual := original, restored
	expected.Directory, actual.Directory = "", ""
	expected.Notes, actual.Notes = "", ""
	if expected != actual {
		_ = os.RemoveAll(dir)
		return release{}, fmt.Errorf("original Store artifact identity or digest changed from the receipt")
	}
	return restored, nil
}

func (g githubAPI) loadReceipt(ctx context.Context) (*receipt, error) {
	var latest struct {
		ID      int64           `json:"id"`
		Payload json.RawMessage `json:"payload"`
	}
	seen := map[int64]bool{}
	// Receipts do not expire. Traverse the journal within the command deadline,
	// without imposing the artifact discovery limit on years of release history.
	for page := 1; ; page++ {
		var records []struct {
			ID          int64           `json:"id"`
			Task        string          `json:"task"`
			Environment string          `json:"environment"`
			Payload     json.RawMessage `json:"payload"`
		}
		query := url.Values{"task": {receiptTask}, "environment": {receiptEnvironment}, "per_page": {strconv.Itoa(githubPageSize)}, "page": {strconv.Itoa(page)}}
		if err := g.json(ctx, http.MethodGet, "/repos/"+repository+"/deployments?"+query.Encode(), nil, &records); err != nil {
			return nil, err
		}
		if len(records) > githubPageSize {
			return nil, fmt.Errorf("GitHub receipt history exceeds the page limit")
		}
		for _, record := range records {
			if record.ID <= 0 || seen[record.ID] || (record.Task != "" && record.Task != receiptTask) || (record.Environment != "" && record.Environment != receiptEnvironment) {
				return nil, fmt.Errorf("GitHub receipt history is invalid or changed during discovery")
			}
			seen[record.ID] = true
			if record.ID > latest.ID {
				latest.ID, latest.Payload = record.ID, record.Payload
			}
		}
		if len(records) < githubPageSize {
			if latest.ID == 0 {
				return nil, nil
			}
			var r receipt
			payload := latest.Payload
			if len(payload) > 0 && payload[0] == '"' {
				var encoded string
				if err := json.Unmarshal(payload, &encoded); err != nil {
					return nil, fmt.Errorf("GitHub receipt payload is invalid")
				}
				payload = []byte(encoded)
			}
			if err := json.Unmarshal(payload, &r); err != nil || r.Schema != 1 {
				return nil, fmt.Errorf("GitHub receipt has an invalid or unsupported schema")
			}
			return &r, nil
		}
	}
}

func (g githubAPI) saveReceipt(ctx context.Context, r *receipt) error {
	if r == nil || r.Schema != 1 {
		return fmt.Errorf("cannot persist an invalid Store receipt")
	}
	payload := *r
	payload.Release.Directory = ""
	payload.Release.Notes = ""
	body, err := encode(struct {
		Ref              string   `json:"ref"`
		Task             string   `json:"task"`
		Environment      string   `json:"environment"`
		AutoMerge        bool     `json:"auto_merge"`
		RequiredContexts []string `json:"required_contexts"`
		Payload          receipt  `json:"payload"`
	}{"main", receiptTask, receiptEnvironment, false, []string{}, payload})
	if err != nil {
		return fmt.Errorf("could not encode Store receipt")
	}
	var saved struct {
		ID int64 `json:"id"`
	}
	if err = g.json(ctx, http.MethodPost, "/repos/"+repository+"/deployments", body, &saved); err != nil {
		return err
	}
	if saved.ID <= 0 {
		return fmt.Errorf("GitHub did not confirm persistence of the Store receipt")
	}
	return nil
}
