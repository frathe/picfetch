package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"
)

func testZIP(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, data := range entries {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestExtractStoreArtifact(t *testing.T) {
	type entry struct {
		name string
		data string
		mode os.FileMode
	}
	valid := []entry{
		{bundleName, "bundle bytes for archive checksum validation", 0o600},
		{"wack-report.xml", "report", 0o600},
		{"store-release.json", "record", 0o600},
	}
	archive := func(t *testing.T, entries []entry) []byte {
		t.Helper()
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for _, e := range entries {
			header := &zip.FileHeader{Name: e.name, Method: zip.Store}
			header.SetMode(e.mode)
			w, err := z.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.WriteString(w, e.data); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	t.Run("exact regular files", func(t *testing.T) {
		dir := t.TempDir()
		if err := extractStoreArtifact(archive(t, valid), dir); err != nil {
			t.Fatal(err)
		}
		for _, e := range valid {
			b, err := os.ReadFile(filepath.Join(dir, e.name))
			if err != nil || string(b) != e.data {
				t.Fatalf("staged %s = %q, %v", e.name, b, err)
			}
		}
	})
	for _, path := range []string{"../outside", "nested/../../outside", "/outside", `..\outside`, `C:\outside`, "./store-release.json", "nested/store-release.json", "unexpected"} {
		t.Run("path "+path, func(t *testing.T) {
			entries := append([]entry(nil), valid...)
			entries[2].name = path
			if err := extractStoreArtifact(archive(t, entries), t.TempDir()); err == nil {
				t.Fatalf("accepted untrusted archive path %q", path)
			}
		})
	}
	for _, mode := range []os.FileMode{os.ModeSymlink | 0o777, os.ModeDir | 0o700, os.ModeNamedPipe | 0o600} {
		t.Run("archive type "+mode.String(), func(t *testing.T) {
			entries := append([]entry(nil), valid...)
			entries[2].mode = mode
			if err := extractStoreArtifact(archive(t, entries), t.TempDir()); err == nil {
				t.Fatalf("accepted non-regular archive entry %v", mode)
			}
		})
	}
	t.Run("duplicate", func(t *testing.T) {
		entries := append([]entry(nil), valid...)
		entries[2] = entries[1]
		if err := extractStoreArtifact(archive(t, entries), t.TempDir()); err == nil {
			t.Fatal("accepted duplicate entry with missing release record")
		}
	})
	t.Run("extra", func(t *testing.T) {
		entries := append(append([]entry(nil), valid...), entry{"extra", "extra", 0o600})
		if err := extractStoreArtifact(archive(t, entries), t.TempDir()); err == nil {
			t.Fatal("accepted extra archive entry")
		}
	})
	t.Run("oversized", func(t *testing.T) {
		entries := append([]entry(nil), valid...)
		entries[2].data = strings.Repeat("x", (2<<20)+1)
		if err := extractStoreArtifact(archive(t, entries), t.TempDir()); err == nil {
			t.Fatal("accepted oversized release record")
		}
	})
	t.Run("corrupt checksum", func(t *testing.T) {
		data := archive(t, valid)
		i := bytes.Index(data, []byte(valid[0].data))
		if i < 0 {
			t.Fatal("bundle payload missing from test archive")
		}
		data[i] ^= 1
		if err := extractStoreArtifact(data, t.TempDir()); err == nil {
			t.Fatal("accepted archive entry with corrupt checksum")
		}
	})
	for _, e := range valid {
		for _, symlink := range []bool{false, true} {
			t.Run(fmt.Sprintf("existing %s symlink=%t", e.name, symlink), func(t *testing.T) {
				dir := t.TempDir()
				target := filepath.Join(dir, e.name)
				if symlink {
					target = filepath.Join(t.TempDir(), "outside")
				}
				if err := os.WriteFile(target, []byte("untouched"), 0o600); err != nil {
					t.Fatal(err)
				}
				if symlink {
					if err := os.Symlink(target, filepath.Join(dir, e.name)); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
				}
				err := extractStoreArtifact(archive(t, valid), dir)
				b, readErr := os.ReadFile(target)
				if readErr != nil || string(b) != "untouched" {
					t.Errorf("overwrote existing target: %q, %v", b, readErr)
				}
				if err == nil {
					t.Fatal("accepted an existing extraction destination")
				}
			})
		}
	}
}

// The fake services persist across command invocations, just as GitHub receipts
// and Partner Center do across disposable Actions runners.
type storeHarness struct {
	t                                          *testing.T
	mu                                         sync.Mutex
	dir                                        string
	rt                                         runtime
	out                                        bytes.Buffer
	server                                     *httptest.Server
	published, draft                           object
	journal                                    []object
	artifact, bundle                           []byte
	state, fail, pending, tag                  string
	creates, updates, uploads, commits, tokens int
	waits                                      []time.Duration
	drift                                      bool
}

func newStoreHarness(t *testing.T) *storeHarness {
	t.Helper()
	dir, rt := recordFixture(t)
	h := &storeHarness{t: t, dir: dir, rt: rt, state: "PendingCommit", tag: "v1.0.3"}
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	h.bundle = read(bundleName)
	h.artifact = testZIP(t, map[string][]byte{bundleName: h.bundle, "wack-report.xml": read("wack-report.xml"), "store-release.json": read("store-release.json")})
	var err error
	h.published, err = parseObject(read("published.json"))
	if err != nil {
		t.Fatal(err)
	}
	h.server = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.server.Close)
	h.rt.HTTP = h.server.Client()
	h.rt.StoreURL = h.server.URL + "/store"
	h.rt.TokenURL = h.server.URL + "/token"
	h.rt.GitHubURL = h.server.URL
	h.rt.Out = &h.out
	h.rt.Env = func(k string) string {
		switch k {
		case "MSSTORE_TENANT_ID", "MSSTORE_CLIENT_ID", "MSSTORE_CLIENT_SECRET", "GH_TOKEN":
			return "private-token"
		case "PICFETCH_STORE_SERIALIZED":
			return "1"
		}
		return ""
	}
	h.rt.Wait = func(_ context.Context, d time.Duration) error { h.waits = append(h.waits, d); return nil }
	h.rt.Git = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "for-each-ref"):
			return []byte("v1.0.2\nv1.0.3\n"), nil
		case strings.HasPrefix(joined, "merge-base"):
			return nil, nil
		case strings.Contains(joined, "FyneApp.toml"):
			return read("FyneApp.toml"), nil
		case strings.Contains(joined, "release-notes.md"):
			return read(".github/release-notes.md"), nil
		case strings.HasPrefix(joined, "rev-parse"):
			return []byte(strings.Repeat("a", 40)), nil
		default:
			return nil, fmt.Errorf("unexpected git: %v", args)
		}
	}
	return h
}
func (h *storeHarness) serve(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	write := func(v any) {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			h.t.Error(err)
		}
	}
	decode := func() object {
		var v object
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			h.t.Error(err)
		}
		return v
	}
	base := "/store/applications/" + productID
	runData := object{"id": 123, "run_attempt": 1, "head_sha": strings.Repeat("a", 40), "head_branch": h.tag, "event": "push", "path": ".github/workflows/microsoft-store.yml", "conclusion": "success", "head_repository": object{"full_name": repository}}
	artifactData := object{"id": 456, "name": "picfetch-microsoft-store", "expired": false, "digest": "sha256:" + digest(h.artifact), "workflow_run": object{"id": 123}}
	if r.URL.Path != "/token" && r.URL.Path != "/blob" && r.Header.Get("Authorization") == "" {
		h.t.Error("API request missing authorization")
	}
	switch {
	case r.URL.Path == "/token":
		h.tokens++
		if err := r.ParseForm(); err != nil {
			h.t.Error(err)
		}
		if r.Form.Get("resource") != "https://manage.devcenter.microsoft.com" || r.Form.Get("grant_type") != "client_credentials" {
			h.t.Error("wrong OAuth contract")
		}
		write(object{"access_token": "private-store-token", "expires_in": 3600})
	case r.URL.Path == base:
		if h.fail == "unavailable" {
			w.WriteHeader(503)
			_, _ = w.Write([]byte("private-error-body"))
			return
		}
		if h.fail == "forbidden" {
			w.WriteHeader(403)
			_, _ = w.Write([]byte("private-error-body"))
			return
		}
		if h.fail == "throttle" {
			h.fail = ""
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(429)
			return
		}
		if h.fail == "unauthorized" {
			h.fail = ""
			w.WriteHeader(401)
			return
		}
		app := object{"id": productID, "lastPublishedApplicationSubmission": object{"id": stringField(h.published, "id")}}
		if h.pending != "" {
			app["pendingApplicationSubmission"] = object{"id": h.pending}
		}
		write(app)
	case r.URL.Path == base+"/submissions/published-102":
		write(h.published)
	case r.URL.Path == base+"/submissions" && r.Method == http.MethodPost:
		h.creates++
		if len(h.journal) == 0 || stringField(asObject(h.journal[len(h.journal)-1]["payload"]), "phase") != "creating" {
			h.t.Error("create had no durable intent")
		}
		h.draft, _ = parseObject(mustJSON(h.t, h.published))
		h.draft["id"] = "submission-103"
		h.draft["fileUploadUrl"] = h.server.URL + "/blob?sig=private-sas"
		h.pending = "submission-103"
		if h.fail == "create" {
			w.WriteHeader(503)
			return
		}
		write(h.draft)
	case r.URL.Path == base+"/submissions/submission-103" && r.Method == http.MethodPut:
		h.updates++
		h.draft = decode()
		if h.fail == "update" {
			w.WriteHeader(503)
			return
		}
		write(h.draft)
	case r.URL.Path == base+"/submissions/submission-103":
		if h.drift {
			h.draft["pricing"] = object{"priceId": "Paid"}
		}
		write(h.draft)
	case r.URL.Path == base+"/submissions/submission-103/status":
		write(object{"status": h.state})
	case r.URL.Path == base+"/submissions/submission-103/commit":
		h.commits++
		if stringField(asObject(h.journal[len(h.journal)-1]["payload"]), "phase") != "committing" {
			h.t.Error("commit had no durable intent")
		}
		h.state = "CommitStarted"
		if h.fail == "commit" {
			w.WriteHeader(503)
			return
		}
		write(object{"status": h.state})
	case r.URL.Path == "/blob":
		h.uploads++
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			h.t.Error("credential leaked to blob upload")
		}
		if r.Method != "PUT" || r.Header.Get("x-ms-blob-type") != "BlockBlob" || r.ContentLength <= 0 {
			h.t.Error("incorrect Azure upload contract")
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			h.t.Error(err)
		}
		z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			h.t.Error(err)
			return
		}
		inner, err := zipEntry(z, bundleName, maxBundleBytes)
		if err != nil || len(z.File) != 1 || !bytes.Equal(inner, h.bundle) {
			h.t.Error("upload changed the validated bundle")
		}
		if h.fail == "upload" {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(201)
	case strings.HasSuffix(r.URL.Path, "/microsoft-store.yml/runs"):
		write(object{"total_count": 1, "workflow_runs": []any{runData}})
	case strings.HasSuffix(r.URL.Path, "/actions/runs/123/attempts/1"):
		write(runData)
	case strings.HasSuffix(r.URL.Path, "/actions/runs/123/artifacts"):
		write(object{"total_count": 1, "artifacts": []any{artifactData}})
	case strings.HasSuffix(r.URL.Path, "/actions/artifacts/456"):
		write(artifactData)
	case strings.HasSuffix(r.URL.Path, "/actions/artifacts/456/zip"):
		if h.fail == "expired" {
			w.WriteHeader(410)
			return
		}
		_, _ = w.Write(h.artifact)
	case r.URL.Path == "/repos/"+repository+"/deployments":
		if r.Method == http.MethodPost {
			v := decode()
			v["id"] = len(h.journal) + 1
			if v["auto_merge"] != false || v["task"] != receiptTask || v["environment"] != receiptEnvironment {
				h.t.Error("incorrect journal deployment")
			}
			if strings.Contains(string(mustJSON(h.t, v)), "private-") {
				h.t.Error("secret in receipt")
			}
			h.journal = append(h.journal, v)
			w.WriteHeader(201)
			write(object{"id": len(h.journal)})
			return
		}
		// GitHub normally lists newest first; exercise a deliberately unsorted page.
		write(h.journal)
	default:
		h.t.Errorf("unexpected service request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}
}
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func (h *storeHarness) command(command string) error {
	return h.invoke(command, "v1.0.3")
}

func TestStorePublishApproval(t *testing.T) {
	t.Run("reconcile rejects absent receipt before Store access", func(t *testing.T) {
		h := newStoreHarness(t)
		err := drive(context.Background(), h.rt, options{stateDir: h.dir}, approval{Mode: "reconcile", ReceiptSHA: receiptDigest(nil)})
		if err == nil || !strings.Contains(err.Error(), "receipt to reconcile") {
			t.Fatalf("missing receipt was not rejected: %v", err)
		}
		if h.tokens+h.creates+h.updates+h.uploads+h.commits+len(h.journal) != 0 {
			t.Fatal("missing receipt reached Microsoft or mutated state")
		}
	})
	for _, change := range []string{"digest", "notes", "receipt", "base", "expired", "tag", "artifact", "mode"} {
		t.Run("reject changed "+change, func(t *testing.T) {
			h := newStoreHarness(t)
			path := filepath.Join(h.dir, "reviewed.json")
			if err := run(context.Background(), []string{"prepare", "--tag", "v1.0.3", "--out", path}, h.rt); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sha, mode := digest(b), "submit"
			switch change {
			case "digest":
				sha = strings.Repeat("0", 64)
			case "notes":
				writeTestFile(t, path, bytes.ReplaceAll(b, []byte("Open two photos together."), []byte("Unreviewed replacement text.")))
			case "receipt":
				if err = h.command("submit"); err != nil {
					t.Fatal(err)
				}
			case "base":
				h.published["applicationPackages"] = []any{object{"version": "1.0.4.0", "fileStatus": "Uploaded"}}
			case "expired":
				h.fail = "expired"
			case "tag":
				old := h.rt.Git
				h.rt.Git = func(ctx context.Context, root string, args ...string) ([]byte, error) {
					if args[0] == "rev-parse" {
						return []byte(strings.Repeat("b", 40)), nil
					}
					return old(ctx, root, args...)
				}
			case "artifact":
				h.artifact = testZIP(t, map[string][]byte{bundleName: []byte("newer rebuilt bundle")})
			case "mode":
				mode = "reconcile"
			}
			before := h.creates + h.updates + h.uploads + h.commits + len(h.journal)
			err = run(context.Background(), []string{mode, "--approval", path, "--approval-sha256", sha, "--state-dir", h.dir}, h.rt)
			if err == nil {
				t.Fatal("changed approval or selected release was accepted")
			}
			if h.creates+h.updates+h.uploads+h.commits+len(h.journal) != before {
				t.Fatal("stale approval mutated Store or receipt state")
			}
		})
	}
	t.Run("triggering run cannot select another build", func(t *testing.T) {
		h := newStoreHarness(t)
		path := filepath.Join(h.dir, "reviewed.json")
		if err := run(context.Background(), []string{"prepare", "--run-id", "999", "--out", path}, h.rt); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("different producer run was selected for approval")
		}
		if err := run(context.Background(), []string{"prepare", "--run-id", "123", "--out", path}, h.rt); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatal("triggering producer was not prepared")
		}
	})
	t.Run("mutation requires frozen approval", func(t *testing.T) {
		h := newStoreHarness(t)
		err := run(context.Background(), []string{"submit", "--tag", "v1.0.3", "--state-dir", h.dir}, h.rt)
		if err == nil || !strings.Contains(err.Error(), "approval") {
			t.Fatalf("missing approval was not rejected: %v", err)
		}
		if h.creates+h.updates+h.uploads+h.commits+len(h.journal)+h.tokens != 0 {
			t.Fatal("unapproved command accessed Microsoft or wrote a receipt")
		}
	})
	t.Run("prepare freezes reviewable release without Store access", func(t *testing.T) {
		h := newStoreHarness(t)
		path := filepath.Join(h.dir, "approval.json")
		if err := run(context.Background(), []string{"prepare", "--tag", "v1.0.3", "--out", path}, h.rt); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"tag":"v1.0.3"`, `"artifact_id":456`, `"run_id":123`, "Open two photos together.", `"base_version":"1.0.2"`} {
			if !bytes.Contains(b, []byte(want)) {
				t.Fatalf("approval omits %s: %s", want, b)
			}
		}
		if h.tokens+h.creates+h.updates+h.uploads+h.commits+len(h.journal) != 0 {
			t.Fatal("preparation accessed protected Microsoft credentials or mutated services")
		}
		// Discovery after approval must be unnecessary, including a newer rebuild.
		old := h.rt.Git
		h.rt.Git = func(ctx context.Context, root string, args ...string) ([]byte, error) {
			if args[0] == "for-each-ref" {
				return nil, fmt.Errorf("rediscovered releases after approval")
			}
			return old(ctx, root, args...)
		}
		if err = run(context.Background(), []string{"submit", "--approval", path, "--approval-sha256", digest(b), "--state-dir", h.dir}, h.rt); err != nil {
			t.Fatal(err)
		}
		if h.creates != 1 || h.commits != 1 {
			t.Fatal("approved release was not submitted")
		}
	})
}
func TestStorePublishNoteProvenance(t *testing.T) {
	for _, tc := range []struct {
		name       string
		recordCRLF bool
		taggedCRLF bool
		change     string
	}{
		{name: "LF notes"},
		{name: "Windows artifact", recordCRLF: true},
		{name: "CRLF Git blob", taggedCRLF: true},
		{name: "CRLF on both sides", recordCRLF: true, taggedCRLF: true},
		{name: "changed content", recordCRLF: true, change: "content"},
		{name: "changed whitespace", recordCRLF: true, change: "space"},
		{name: "bare carriage return", recordCRLF: true, change: "carriage return"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newStoreHarness(t)
			rec, err := loadRelease(filepath.Join(h.dir, "store-release.json"))
			if err != nil {
				t.Fatal(err)
			}
			taggedNotes := rec.Notes
			if tc.taggedCRLF {
				taggedNotes = strings.ReplaceAll(taggedNotes, "\n", "\r\n")
			}
			if tc.recordCRLF {
				rec.Notes = strings.ReplaceAll(rec.Notes, "\n", "\r\n")
			}
			switch tc.change {
			case "content":
				rec.Notes = strings.ReplaceAll(rec.Notes, "two photos", "other photos")
			case "space":
				rec.Notes = strings.ReplaceAll(rec.Notes, "together.", "together. ")
			case "carriage return":
				rec.Notes += "\r"
			}
			wack, err := os.ReadFile(filepath.Join(h.dir, "wack-report.xml"))
			if err != nil {
				t.Fatal(err)
			}
			h.artifact = testZIP(t, map[string][]byte{bundleName: h.bundle, "wack-report.xml": wack, "store-release.json": mustJSON(t, rec)})
			oldGit := h.rt.Git
			h.rt.Git = func(ctx context.Context, root string, args ...string) ([]byte, error) {
				if args[0] == "show" && strings.HasSuffix(args[1], ":.github/release-notes.md") {
					return []byte(taggedNotes), nil
				}
				return oldGit(ctx, root, args...)
			}
			path := filepath.Join(h.dir, "approval.json")
			err = run(context.Background(), []string{"prepare", "--run-id", "123", "--out", path}, h.rt)
			if h.tokens+h.creates+h.updates+h.uploads+h.commits+len(h.journal) != 0 {
				t.Fatal("preparation accessed Microsoft or mutated services")
			}
			if tc.change != "" {
				if err == nil || !strings.Contains(err.Error(), "tagged release notes disagree with Store artifact") {
					t.Fatalf("changed notes were not rejected by provenance check: %v", err)
				}
				if _, err = os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("changed notes produced an approval")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(b, []byte("Open two photos together.")) {
				t.Fatal("approval omitted the release note")
			}
			if err = run(context.Background(), []string{"submit", "--approval", path, "--approval-sha256", digest(b), "--state-dir", h.dir}, h.rt); err != nil {
				t.Fatal(err)
			}
			if h.creates != 1 || h.commits != 1 {
				t.Fatal("approved artifact was not submitted")
			}
		})
	}
}

func (h *storeHarness) invoke(command, tag string) error {
	h.mu.Lock()
	h.out.Reset()
	h.mu.Unlock()
	args := []string{command, "--tag", tag, "--root", h.dir, "--state-dir", h.dir}
	if command == "submit" || command == "reconcile" {
		path := filepath.Join(h.dir, "approval.json")
		if err := run(context.Background(), []string{"prepare", "--mode", command, "--tag", tag, "--root", h.dir, "--out", path}, h.rt); err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		args = append(args, "--approval", path, "--approval-sha256", digest(b))
		h.out.Reset()
	}
	err := run(context.Background(), args, h.rt)
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.Contains(h.out.String(), "private-") || err != nil && strings.Contains(err.Error(), "private-") {
		h.t.Error("credentials leaked to command output")
	}
	return err
}
func TestStorePublishSubmission(t *testing.T) {
	h := newStoreHarness(t)
	if err := h.command("preview"); err != nil {
		t.Fatal(err)
	}
	if err := h.command("check"); err != nil {
		t.Fatal(err)
	}
	if h.creates+h.updates+h.uploads+h.commits+len(h.journal) != 0 {
		t.Fatal("read-only command mutated services")
	}
	if err := h.command("submit"); err != nil {
		t.Fatal(err)
	}
	if h.creates != 1 || h.updates != 1 || h.uploads != 1 || h.commits != 1 {
		t.Fatalf("incorrect submission counts: create=%d update=%d upload=%d commit=%d", h.creates, h.updates, h.uploads, h.commits)
	}
	if !strings.Contains(h.out.String(), `"state":"CommitStarted"`) {
		t.Fatal("submission claimed publication prematurely")
	}
	de := asObject(asObject(asObject(h.draft["listings"])["de-de"])["baseListing"])
	if stringField(de, "description") != "Deutsche Beschreibung" || !strings.Contains(stringField(de, "releaseNotes"), "Open two photos together.") || h.draft["targetPublishMode"] != "Immediate" {
		t.Fatal("submission changed protected metadata or notes")
	}
	checkpoints := len(h.journal)
	for range 3 {
		if err := h.command("reconcile"); err != nil {
			t.Fatal(err)
		}
	}
	if h.commits != 1 || h.creates != 1 {
		t.Fatal("polling repeated a mutation")
	}
	if len(h.journal) != checkpoints {
		t.Fatal("unchanged certification polls keep growing durable journal")
	}
	h.state = "Published"
	h.draft["applicationPackages"] = []any{object{"version": "1.0.3.0", "fileStatus": "Uploaded"}}
	h.published = h.draft
	h.pending = ""
	if err := h.command("reconcile"); err != nil {
		t.Fatal(err)
	}
	if err := h.command("submit"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), `"state":"Published"`) || h.creates != 1 {
		t.Fatal("published rerun was not a no-op")
	}
}
func TestStorePublishRecovery(t *testing.T) {
	for _, stage := range []string{"create", "update", "upload", "commit"} {
		t.Run(stage, func(t *testing.T) {
			h := newStoreHarness(t)
			h.fail = stage
			if err := h.command("submit"); err == nil {
				t.Fatal("injected outage was ignored")
			}
			h.fail = ""
			err := h.command("reconcile")
			if stage == "create" {
				if err == nil || !strings.Contains(err.Error(), "ambiguous") {
					t.Fatal("lost creation outcome was not preserved")
				}
				if h.creates != 1 || h.commits != 0 {
					t.Fatal("unknown draft was adopted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if h.creates != 1 || h.commits != 1 {
				t.Fatal("recovery repeated a non-idempotent mutation")
			}
		})
	}
	t.Run("expired original artifact", func(t *testing.T) {
		h := newStoreHarness(t)
		h.fail = "upload"
		if h.command("submit") == nil {
			t.Fatal("expected outage")
		}
		h.fail = "expired"
		if h.command("reconcile") == nil {
			t.Fatal("missing original artifact was replaced")
		}
		if h.commits != 0 {
			t.Fatal("committed without original artifact")
		}
	})
	t.Run("metadata drift before upload", func(t *testing.T) {
		h := newStoreHarness(t)
		h.drift = true
		if h.command("submit") == nil {
			t.Fatal("concurrent Partner Center edit was overwritten")
		}
		if h.uploads+h.commits != 0 {
			t.Fatal("upload proceeded after protected metadata changed")
		}
	})
	t.Run("unknown pending draft", func(t *testing.T) {
		h := newStoreHarness(t)
		h.pending = "manual-draft"
		if h.command("submit") == nil {
			t.Fatal("unknown draft was adopted")
		}
		if h.creates+h.updates+h.uploads+h.commits != 0 {
			t.Fatal("manual draft was mutated")
		}
	})
}
func TestStorePublishStatus(t *testing.T) {
	t.Run("retries", testStorePublishRetries)
	for _, state := range []string{"Certification", "Release", "Publishing", "CertificationFailed", "PublishFailed", "PendingPublication", "UnrecognizedFutureState", "PendingCommit"} {
		t.Run(state, func(t *testing.T) {
			h := newStoreHarness(t)
			if err := h.command("submit"); err != nil {
				t.Fatal(err)
			}
			h.state = state
			err := h.command("reconcile")
			processing := state == "Certification" || state == "Release" || state == "Publishing"
			if (err == nil) != processing {
				t.Fatalf("incorrect handling of %s: %v", state, err)
			}
			if h.creates != 1 || h.commits != 1 {
				t.Fatal("status reconciliation resubmitted a package")
			}
		})
	}
}
func testStorePublishRetries(t *testing.T) {
	t.Run("bounded outage", func(t *testing.T) {
		h := newStoreHarness(t)
		h.fail = "unavailable"
		if h.command("check") == nil {
			t.Fatal("outage was ignored")
		}
		if len(h.waits) != 2 {
			t.Fatalf("retry count was not bounded: %v", h.waits)
		}
	})
	t.Run("access denied", func(t *testing.T) {
		h := newStoreHarness(t)
		h.fail = "forbidden"
		if h.command("check") == nil {
			t.Fatal("access denial was ignored")
		}
		if len(h.waits) != 0 {
			t.Fatal("permanent authorization failure was retried")
		}
	})
	t.Run("cancel retry", func(t *testing.T) {
		h := newStoreHarness(t)
		h.fail = "throttle"
		h.rt.Wait = func(_ context.Context, _ time.Duration) error { return context.Canceled }
		if err := h.command("check"); err == nil || !strings.Contains(err.Error(), "canceled") {
			t.Fatal("retry ignored cancellation")
		}
	})
	t.Run("token expires", func(t *testing.T) {
		h := newStoreHarness(t)
		now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
		h.rt.Now = func() time.Time { now = now.Add(time.Hour); return now }
		if err := h.command("check"); err != nil {
			t.Fatal(err)
		}
		if h.tokens != 2 {
			t.Fatal("cached token was used after expiry")
		}
	})
	t.Run("Retry After", func(t *testing.T) {
		h := newStoreHarness(t)
		h.fail = "throttle"
		if err := h.command("check"); err != nil {
			t.Fatal(err)
		}
		if len(h.waits) != 1 || h.waits[0] < 120*time.Second {
			t.Fatalf("server throttle was shortened: %v", h.waits)
		}
	})
	t.Run("token refresh", func(t *testing.T) {
		h := newStoreHarness(t)
		h.fail = "unauthorized"
		if err := h.command("check"); err != nil {
			t.Fatal(err)
		}
		if h.tokens != 2 {
			t.Fatal("expired token was not refreshed")
		}
	})
}

func TestStorePublishBundleVersions(t *testing.T) {
	h := newStoreHarness(t)
	h.published["applicationPackages"] = []any{object{"version": "2026.905.1829.0", "fileStatus": "Uploaded"}}
	if err := h.command("check"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), `"published_version":"1.0.2"`) {
		t.Fatalf("bundle timestamp was mistaken for the app version: %s", h.out.String())
	}
	// A future unknown Store version must be diagnosed, never reported as no update.
	h.published["applicationPackages"] = []any{object{"version": "2026.909.1000.0", "fileStatus": "Uploaded"}}
	if h.command("reconcile") == nil {
		t.Fatal("unmapped Store version was silently treated as newer than every release")
	}
	t.Run("recorded outer version after publication", func(t *testing.T) {
		h := newStoreHarness(t)
		h.published["applicationPackages"] = []any{object{"version": "2026.905.1829.0", "fileStatus": "Uploaded"}}
		outer := "2026.908.1000.0"
		z, err := zip.NewReader(bytes.NewReader(h.bundle), int64(len(h.bundle)))
		if err != nil {
			t.Fatal(err)
		}
		entries := map[string][]byte{}
		for _, f := range z.File {
			b, err := zipEntry(z, f.Name, maxBundleBytes)
			if err != nil {
				t.Fatal(err)
			}
			if f.Name == "AppxMetadata/AppxBundleManifest.xml" {
				b = bytes.Replace(b, []byte(`Version="1.0.3.0"`), []byte(`Version="`+outer+`"`), 1)
			}
			entries[f.Name] = b
		}
		h.bundle = testZIP(t, entries)
		writeTestFile(t, filepath.Join(h.dir, bundleName), h.bundle)
		_, env := testRelease(t)
		if err := run(context.Background(), []string{"record", "--root", h.dir, "--bundle", filepath.Join(h.dir, bundleName), "--wack", filepath.Join(h.dir, "wack-report.xml"), "--out", filepath.Join(h.dir, "store-release.json")}, runtime{Env: func(k string) string { return env[k] }}); err != nil {
			t.Fatal(err)
		}
		rec, err := os.ReadFile(filepath.Join(h.dir, "store-release.json"))
		if err != nil {
			t.Fatal(err)
		}
		wack, err := os.ReadFile(filepath.Join(h.dir, "wack-report.xml"))
		if err != nil {
			t.Fatal(err)
		}
		h.artifact = testZIP(t, map[string][]byte{bundleName: h.bundle, "wack-report.xml": wack, "store-release.json": rec})
		if err := h.command("submit"); err != nil {
			t.Fatal(err)
		}
		h.state = "Published"
		h.draft["applicationPackages"] = []any{object{"version": outer, "fileStatus": "Uploaded"}}
		h.published = h.draft
		h.pending = ""
		if err := h.command("reconcile"); err != nil {
			t.Fatal(err)
		}
		if err := h.command("check"); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(h.out.String(), `"published_version":"1.0.3"`) {
			t.Fatal("receipt failed to bind outer version to public release")
		}
	})
}

func TestStorePublishReconcile(t *testing.T) {
	h := newStoreHarness(t)
	if err := h.command("submit"); err != nil {
		t.Fatal(err)
	}
	// Two new tags arrive while the original submission is in certification.
	oldGit := h.rt.Git
	nextNotes := "### Features\n\n- Newest zoom improvement.\n\n**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.0.4...v1.0.5\n"
	middleNotes := "### Fixes\n\n- Skipped release fixed rotation.\n\n**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.0.3...v1.0.4\n"
	newestDiscovered := false
	h.rt.Git = func(ctx context.Context, root string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "rev-parse --verify refs/tags/v1.0.5") {
			newestDiscovered = true
		}
		if strings.Contains(joined, "rev-parse --verify refs/tags/v1.0.4") && !newestDiscovered {
			return nil, fmt.Errorf("older build looked up before newest release")
		}
		switch {
		case strings.HasPrefix(joined, "for-each-ref"):
			return []byte("v1.0.3\nv1.0.4\nv1.0.5\n"), nil
		case strings.Contains(joined, "refs/tags/v1.0.4:"):
			return []byte(middleNotes), nil
		case strings.Contains(joined, "release-notes.md"):
			return []byte(nextNotes), nil
		case strings.Contains(joined, "FyneApp.toml"):
			return []byte("Version = \"1.0.5\"\n"), nil
		default:
			return oldGit(ctx, root, args...)
		}
	}
	h.state = "Certification"
	if err := h.invoke("reconcile", ""); err != nil {
		t.Fatal(err)
	}
	if h.creates != 1 {
		t.Fatal("new release displaced active certification")
	}
	h.state = "Published"
	h.draft["applicationPackages"] = []any{object{"version": "1.0.3.0", "fileStatus": "Uploaded"}}
	h.published = h.draft
	h.pending = ""
	// Only the newest artifact is retained; the skipped tag provides notes.
	entries := map[string][]byte{}
	z, err := zip.NewReader(bytes.NewReader(h.bundle), int64(len(h.bundle)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		b, err := zipEntry(z, f.Name, maxBundleBytes)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f.Name, ".msix") {
			inner, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := zipEntry(inner, "AppxManifest.xml", 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			b = testZIP(t, map[string][]byte{"AppxManifest.xml": bytes.ReplaceAll(manifest, []byte("1.0.3.0"), []byte("1.0.5.0"))})
		} else {
			b = bytes.ReplaceAll(b, []byte("1.0.3.0"), []byte("1.0.5.0"))
		}
		entries[f.Name] = b
	}
	h.bundle = testZIP(t, entries)
	rec, err := loadRelease(filepath.Join(h.dir, "store-release.json"))
	if err != nil {
		t.Fatal(err)
	}
	rec.Version = "1.0.5"
	rec.BundleVersion = "1.0.5.0"
	rec.Tag = "v1.0.5"
	rec.Ref = "refs/tags/v1.0.5"
	rec.Notes = nextNotes
	rec.BundleSHA = digest(h.bundle)
	h.tag = rec.Tag
	wack, err := os.ReadFile(filepath.Join(h.dir, "wack-report.xml"))
	if err != nil {
		t.Fatal(err)
	}
	h.artifact = testZIP(t, map[string][]byte{bundleName: h.bundle, "wack-report.xml": wack, "store-release.json": mustJSON(t, rec)})
	h.out.Reset()
	if err := h.invoke("reconcile", ""); err != nil {
		t.Fatal(err)
	}
	if h.creates != 1 || !strings.Contains(h.out.String(), `"version":"1.0.3"`) {
		t.Fatal("reconciliation submitted a waiting release under an older approval")
	}
	if err := h.invoke("submit", ""); err != nil {
		t.Fatal(err)
	}
	if h.creates != 2 || !strings.Contains(h.out.String(), `"version":"1.0.5"`) {
		t.Fatal("separate approval did not submit newest waiting release")
	}
	notes := stringField(asObject(asObject(asObject(h.draft["listings"])["en-us"])["baseListing"]), "releaseNotes")
	for _, want := range []string{"Newest zoom", "Skipped release fixed rotation", "v1.0.3...v1.0.5"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("notes omit %q: %s", want, notes)
		}
	}
}

func testStorePublishGuards(t *testing.T) {
	for _, guard := range []string{"serialization", "lock", "ancestry", "moved tag", "artifact traversal", "artifact tampering"} {
		t.Run(guard, func(t *testing.T) {
			h := newStoreHarness(t)
			switch guard {
			case "serialization":
				old := h.rt.Env
				h.rt.Env = func(k string) string {
					if k == "PICFETCH_STORE_SERIALIZED" {
						return ""
					}
					return old(k)
				}
			case "lock":
				if err := os.Mkdir(filepath.Join(h.dir, "picfetch-store.lock"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "ancestry":
				old := h.rt.Git
				h.rt.Git = func(ctx context.Context, root string, args ...string) ([]byte, error) {
					if args[0] == "merge-base" {
						return nil, fmt.Errorf("not ancestor")
					}
					return old(ctx, root, args...)
				}
			case "moved tag":
				if err := h.command("submit"); err != nil {
					t.Fatal(err)
				}
				old := h.rt.Git
				h.rt.Git = func(ctx context.Context, root string, args ...string) ([]byte, error) {
					if args[0] == "rev-parse" {
						return []byte(strings.Repeat("b", 40)), nil
					}
					return old(ctx, root, args...)
				}
			case "artifact traversal":
				h.artifact = testZIP(t, map[string][]byte{"../escape": []byte("bad")})
			case "artifact tampering":
				z, err := zip.NewReader(bytes.NewReader(h.artifact), int64(len(h.artifact)))
				if err != nil {
					t.Fatal(err)
				}
				rec, err := zipEntry(z, "store-release.json", 2<<20)
				if err != nil {
					t.Fatal(err)
				}
				wack, err := zipEntry(z, "wack-report.xml", 8<<20)
				if err != nil {
					t.Fatal(err)
				}
				h.artifact = testZIP(t, map[string][]byte{"store-release.json": rec, "wack-report.xml": wack, bundleName: []byte("tampered")})
			}
			before := h.creates + h.updates + h.uploads + h.commits
			mode := "submit"
			if guard == "moved tag" {
				mode = "reconcile"
			}
			if h.command(mode) == nil {
				t.Fatalf("%s guard did not reject", guard)
			}
			if h.creates+h.updates+h.uploads+h.commits != before {
				t.Fatal("rejected input caused a Store mutation")
			}
		})
	}
}

func testRelease(t *testing.T) (string, map[string]string) {
	t.Helper()
	dir := t.TempDir()
	entries := map[string][]byte{}
	for _, arch := range []string{"x64", "arm64"} {
		manifest := fmt.Sprintf(`<Package><Identity Name="OpenSourceDeveloperFloria.PicFetch" Publisher="CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F" Version="1.0.3.0" ProcessorArchitecture="%s"/></Package>`, arch)
		entries["picfetch-"+arch+".msix"] = testZIP(t, map[string][]byte{"AppxManifest.xml": []byte(manifest)})
	}
	entries["AppxMetadata/AppxBundleManifest.xml"] = []byte(`<Bundle><Identity Name="OpenSourceDeveloperFloria.PicFetch" Publisher="CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F" Version="1.0.3.0"/><Packages><Package Type="application" FileName="picfetch-x64.msix" Architecture="x64" Version="1.0.3.0"/><Package Type="application" FileName="picfetch-arm64.msix" Architecture="arm64" Version="1.0.3.0"/></Packages></Bundle>`)
	writeTestFile(t, filepath.Join(dir, "picfetch-microsoft-store.msixbundle"), testZIP(t, entries))
	writeTestFile(t, filepath.Join(dir, "wack-report.xml"), []byte("<REPORT><RESULT>PASS</RESULT></REPORT>"))
	writeTestFile(t, filepath.Join(dir, "FyneApp.toml"), []byte("[Details]\nVersion = \"1.0.3\"\nBuild = 454\n"))
	writeTestFile(t, filepath.Join(dir, ".github/release-notes.md"), []byte("## What's Changed\n\n### New Features\n\n- Open **two photos** together.\n\n### Internal\n\n- Refactor test helpers.\n\n**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.0.2...v1.0.3\n"))
	return dir, map[string]string{"GITHUB_REPOSITORY": "frathe/picfetch", "GITHUB_REF": "refs/tags/v1.0.3", "GITHUB_SHA": strings.Repeat("a", 40), "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "1", "GITHUB_EVENT_NAME": "push"}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStorePublishReadOnly(t *testing.T) {
	t.Run("Preview", func(t *testing.T) {
		dir, env := testRelease(t)
		var out bytes.Buffer
		rt := runtime{Out: &out, Env: func(k string) string { return env[k] }}
		record := filepath.Join(dir, "store-release.json")
		if err := run(context.Background(), []string{"record", "--root", dir, "--bundle", filepath.Join(dir, "picfetch-microsoft-store.msixbundle"), "--wack", filepath.Join(dir, "wack-report.xml"), "--out", record}, rt); err != nil {
			t.Fatal(err)
		}
		snapshot := filepath.Join(dir, "published.json")
		writeTestFile(t, snapshot, []byte(`{"id":"published-102","targetPublishMode":"Manual","pricing":{"priceId":"Free"},"applicationPackages":[{"version":"1.0.2.0"}],"listings":{"en-us":{"baseListing":{"description":"English description","releaseNotes":"Old"}},"de-de":{"baseListing":{"description":"Deutsche Beschreibung","releaseNotes":"Alt"},"platformOverrides":{"Windows81":{"releaseNotes":"Old override"}}}}}`))
		out.Reset()
		if err := run(context.Background(), []string{"preview", "--candidate", record, "--snapshot", snapshot}, rt); err != nil {
			t.Fatal(err)
		}
		var result struct {
			State      string
			Notes      string
			Submission map[string]any
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.State != "preview" || !strings.Contains(result.Notes, "Open two photos together.") || strings.Contains(result.Notes, "test helpers") {
			t.Fatalf("unexpected preview: %s", out.Bytes())
		}
		if result.Submission["targetPublishMode"] != "Immediate" {
			t.Fatal("preview did not select automatic publication")
		}
		listings := result.Submission["listings"].(map[string]any)
		de := listings["de-de"].(map[string]any)["baseListing"].(map[string]any)
		if de["description"] != "Deutsche Beschreibung" || de["releaseNotes"] != result.Notes {
			t.Fatal("localized description or notes were not preserved correctly")
		}
	})
}

func recordFixture(t *testing.T) (string, runtime) {
	t.Helper()
	dir, env := testRelease(t)
	rt := runtime{Out: io.Discard, Env: func(k string) string { return env[k] }}
	err := run(context.Background(), []string{"record", "--root", dir, "--bundle", filepath.Join(dir, bundleName), "--wack", filepath.Join(dir, "wack-report.xml"), "--out", filepath.Join(dir, "store-release.json")}, rt)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "published.json"), []byte(`{"id":"published-102","targetPublishMode":"Manual","pricing":{"priceId":"Free"},"applicationPackages":[{"version":"1.0.2.0","fileName":"old.msixbundle","fileStatus":"Uploaded","id":"old-package"}],"listings":{"en-us":{"baseListing":{"description":"English description","releaseNotes":"Old"}},"de-de":{"baseListing":{"description":"Deutsche Beschreibung","releaseNotes":"Alt"}}}}`))
	return dir, rt
}
func previewArgs(dir string) []string {
	return []string{"preview", "--candidate", filepath.Join(dir, "store-release.json"), "--snapshot", filepath.Join(dir, "published.json")}
}
func amendRecord(t *testing.T, dir string, update func(map[string]any)) {
	t.Helper()
	path := filepath.Join(dir, "store-release.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	update(m)
	b, err = json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, b)
}
func TestStorePublishAdmission(t *testing.T) {
	t.Run("live guards", testStorePublishGuards)
	for _, tc := range []struct {
		name, key string
		value     any
	}{
		{"fork", "repository", "fork/picfetch"}, {"branch", "ref", "refs/heads/main"}, {"pull request", "event", "pull_request"}, {"prerelease", "tag", "v1.0.3-rc1"}, {"wrong tag", "tag", "v1.0.4"}, {"invalid commit", "commit", "bad"}, {"missing run", "run_id", 0}, {"tampered bundle", "bundle_sha256", "bad"}, {"tampered report", "wack_sha256", "bad"}, {"stale notes", "notes", "old notes"}, {"old version", "version", "1.0.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, rt := recordFixture(t)
			amendRecord(t, dir, func(m map[string]any) { m[tc.key] = tc.value })
			if err := run(context.Background(), previewArgs(dir), rt); err == nil {
				t.Fatal("invalid candidate admitted")
			}
		})
	}
	t.Run("missing bundle", func(t *testing.T) {
		dir, rt := recordFixture(t)
		if err := os.Remove(filepath.Join(dir, bundleName)); err != nil {
			t.Fatal(err)
		}
		if err := run(context.Background(), previewArgs(dir), rt); err == nil {
			t.Fatal("missing bundle admitted")
		}
	})
}
func TestStorePublishNotes(t *testing.T) {
	t.Run("scope and unicode", func(t *testing.T) {
		dir, rt := recordFixture(t)
		notes := "## What's Changed\n\n### New Features\n\n- Windows and Linux can compare images.\n\n- On Linux set wallpaper.\n\n- macOS: Native menu.\n\n- Shared **zoom** " + strings.Repeat("😀", 800) + ".\n\n- Open [two photos](https://example.com) with `Ctrl+D`.\n\n### Internal\n\n- Credentials refactor.\n\n**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.0.2...v1.0.3\n"
		amendRecord(t, dir, func(m map[string]any) { m["notes"] = notes })
		var out bytes.Buffer
		rt.Out = &out
		if err := run(context.Background(), previewArgs(dir), rt); err != nil {
			t.Fatal(err)
		}
		var result struct{ Notes string }
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"Version 1.0.3", "Windows and Linux can compare images.", "Open two photos with Ctrl+D.", "v1.0.2...v1.0.3"} {
			if !strings.Contains(result.Notes, want) {
				t.Fatalf("missing %q: %s", want, result.Notes)
			}
		}
		for _, unwanted := range []string{"set wallpaper", "Native menu", "Credentials", "😀", "**", "`"} {
			if strings.Contains(result.Notes, unwanted) {
				t.Fatalf("unexpected %q in notes", unwanted)
			}
		}
		if len(utf16.Encode([]rune(result.Notes))) > 1500 {
			t.Fatal("notes exceed Store limit")
		}
	})
	t.Run("missing intermediate notes", func(t *testing.T) {
		dir, rt := recordFixture(t)
		amendRecord(t, dir, func(m map[string]any) { m["notes"] = strings.Replace(m["notes"].(string), "v1.0.2...", "v1.0.1...", 1) })
		if err := run(context.Background(), previewArgs(dir), rt); err == nil {
			t.Fatal("incomplete note range was accepted")
		}
	})
}

func TestStorePublishGitHub(t *testing.T) {
	dir, rt := recordFixture(t)
	record, err := os.ReadFile(filepath.Join(dir, "store-release.json"))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(filepath.Join(dir, bundleName))
	if err != nil {
		t.Fatal(err)
	}
	wack, err := os.ReadFile(filepath.Join(dir, "wack-report.xml"))
	if err != nil {
		t.Fatal(err)
	}
	artifact := testZIP(t, map[string][]byte{"store-release.json": record, bundleName: bundle, "wack-report.xml": wack})
	published, err := os.ReadFile(filepath.Join(dir, "published.json"))
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/token":
			if _, err := fmt.Fprint(w, `{"access_token":"store-token","expires_in":3600}`); err != nil {
				t.Error(err)
			}
		case r.URL.Path == "/repos/frathe/picfetch/deployments":
			if _, err := fmt.Fprint(w, `[]`); err != nil {
				t.Error(err)
			}
		case r.URL.Path == "/store/applications/"+productID:
			if _, err := fmt.Fprintf(w, `{"id":"%s","lastPublishedApplicationSubmission":{"id":"published-102"}}`, productID); err != nil {
				t.Error(err)
			}
		case r.URL.Path == "/store/applications/"+productID+"/submissions/published-102":
			_, _ = w.Write(published)
		case strings.HasSuffix(r.URL.Path, "/microsoft-store.yml/runs"):
			if r.URL.Query().Get("page") != "" && r.URL.Query().Get("page") != "1" {
				if _, err := fmt.Fprint(w, `{"workflow_runs":[]}`); err != nil {
					t.Error(err)
				}
				return
			}
			if _, err := fmt.Fprintf(w, `{"total_count":1,"workflow_runs":[{"id":123,"run_attempt":1,"head_sha":"%s","event":"push","path":".github/workflows/microsoft-store.yml","conclusion":"success","head_repository":{"full_name":"frathe/picfetch"}}]}`, strings.Repeat("a", 40)); err != nil {
				t.Error(err)
			}
		case r.URL.Path == "/repos/frathe/picfetch/actions/runs/123/artifacts":
			if _, err := fmt.Fprintf(w, `{"total_count":1,"artifacts":[{"id":456,"name":"picfetch-microsoft-store","expired":false,"digest":"sha256:%s","workflow_run":{"id":123}}]}`, digest(artifact)); err != nil {
				t.Error(err)
			}
		case r.URL.Path == "/repos/frathe/picfetch/actions/artifacts/456/zip":
			_, _ = w.Write(artifact)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	rt.HTTP = server.Client()
	rt.StoreURL = server.URL + "/store"
	rt.TokenURL = server.URL + "/token"
	rt.GitHubURL = server.URL
	rt.Env = func(k string) string {
		switch k {
		case "MSSTORE_TENANT_ID", "MSSTORE_CLIENT_ID", "MSSTORE_CLIENT_SECRET", "GH_TOKEN":
			return "test-credential"
		}
		return ""
	}
	rt.Git = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "for-each-ref"), strings.HasPrefix(joined, "tag "):
			return []byte("v1.0.2\nv1.0.3\n"), nil
		case strings.HasPrefix(joined, "merge-base"):
			return nil, nil
		case strings.Contains(joined, "FyneApp.toml"):
			return os.ReadFile(filepath.Join(dir, "FyneApp.toml"))
		case strings.Contains(joined, "release-notes.md"):
			return os.ReadFile(filepath.Join(dir, ".github/release-notes.md"))
		case strings.HasPrefix(joined, "rev-parse"):
			return []byte(strings.Repeat("a", 40) + "\n"), nil
		default:
			return nil, fmt.Errorf("unexpected git arguments %q", args)
		}
	}
	var out bytes.Buffer
	rt.Out = &out
	if err := run(context.Background(), []string{"preview", "--tag", "v1.0.3", "--root", dir}, rt); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Open two photos together.") {
		t.Fatalf("preview did not consume tagged artifact: %s", out.String())
	}
}
