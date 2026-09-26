package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testRevision = "1111111111111111111111111111111111111111"
const testProject = "git+github.com/frathe/picfetch"

func TestFindingsLatestPRAndAllPages(t *testing.T) {
	env := map[string]string{
		"PR": "61", "FOSSA_API_KEY": "test-secret", "FOSSA_OUTPUT_DIR": t.TempDir(),
	}
	var output bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("expected authenticated GET, got %s", r.Method)
		}
		if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
			_, _ = fmt.Fprintf(w, `{"locator":%q,"isSteady":true,"resolved":true,"latestRevisionScanId":42,"unresolved_licensing_issue_count":2}`, testProject+"$"+testRevision)
			return
		}
		query := r.URL.Query()
		if query.Get("scope[revision]") != testRevision || query.Get("scope[id]") != testProject || query.Get("scope[revisionScanId]") != "42" || query.Get("status") != "active" || query.Get("category") != "licensing" || query.Get("count") != "100" {
			t.Errorf("incorrect issue scope: %v", query)
		}
		switch query.Get("page") {
		case "1", "2":
			_, _ = fmt.Fprintf(w, `{"issues":[{"id":%s,"type":"policy_flag","license":"MPL-2.0","source":{"id":"go+example.test/module$v1.0.0","name":"example.test/module","version":"v1.0.0"},"projects":[{"id":%q,"revisionId":%q,"revisionScanId":42}]}]}`, query.Get("page"), testProject, testProject+"$"+testRevision)
		case "3":
			_, _ = w.Write([]byte(`{"issues":[]}`))
		default:
			t.Errorf("unexpected page %q", query.Get("page"))
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	c := command{
		getenv: func(key string) string { return env[key] },
		github: func(_ context.Context, pr string) ([]byte, error) {
			if pr != "61" {
				t.Fatalf("PR = %q", pr)
			}
			return []byte(`{"headRefOid":"` + testRevision + `"}`), nil
		},
		client: server.Client(), baseURL: server.URL, output: &output,
	}
	if err := c.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "2 active licensing findings") || !strings.Contains(output.String(), testRevision) || strings.Contains(output.String(), "test-secret") {
		t.Fatalf("unexpected summary: %s", &output)
	}
	reports, err := filepath.Glob(filepath.Join(env["FOSSA_OUTPUT_DIR"], "*", "report.json"))
	if err != nil || len(reports) != 1 {
		t.Fatalf("reports = %v; %v", reports, err)
	}
	data, err := os.ReadFile(reports[0])
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Revision string
		ScanID   int
		Issues   []struct{ ID int }
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Revision != testRevision || report.ScanID != 42 || len(report.Issues) != 2 || report.Issues[1].ID != 2 {
		t.Fatalf("incomplete report: %s", data)
	}
	pages, err := filepath.Glob(filepath.Join(filepath.Dir(reports[0]), "issues-page-*.json"))
	if err != nil || len(pages) != 3 {
		t.Fatalf("raw pages = %v; %v", pages, err)
	}
}

func findingsCommand(t *testing.T, env map[string]string, handler http.HandlerFunc) (command, *bytes.Buffer) {
	t.Helper()
	if env["FOSSA_OUTPUT_DIR"] == "" {
		env["FOSSA_OUTPUT_DIR"] = t.TempDir()
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	output := new(bytes.Buffer)
	return command{
		getenv: func(key string) string { return env[key] },
		github: func(_ context.Context, _ string) ([]byte, error) {
			t.Error("unexpected GitHub request")
			return nil, fmt.Errorf("unexpected GitHub request")
		},
		client: server.Client(), baseURL: server.URL, output: output,
	}, output
}

func emptyFindings(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
		_, _ = fmt.Fprintf(w, `{"locator":%q,"isSteady":true,"resolved":true,"latestRevisionScanId":42,"unresolved_licensing_issue_count":0}`, testProject+"$"+testRevision)
	} else {
		_, _ = w.Write([]byte(`{"issues":[]}`))
	}
}

func TestFindingsReadsDotenvAsData(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".env.local")
	sentinel := filepath.Join(dir, "must-not-exist")
	if err := os.WriteFile(file, []byte("OTHER=$(touch "+sentinel+")\nexport FOSSA_API_KEY='file-secret'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"FOSSA_REVISION": testRevision, "FOSSA_ENV_FILE": file}
	c, output := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer file-secret" {
			t.Error("dotenv key not used")
		}
		emptyFindings(w, r)
	})
	if err := c.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "0 active licensing findings") {
		t.Fatalf("summary = %s", output)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("dotenv was executed: %v", err)
	}
	// An environment credential takes precedence, even if the file is absent.
	env["FOSSA_API_KEY"] = "environment-secret"
	env["FOSSA_ENV_FILE"] = filepath.Join(dir, "absent")
	c, _ = findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer environment-secret" {
			t.Error("environment key not used")
		}
		emptyFindings(w, r)
	})
	if err := c.run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFindingsRejectsIncompleteReports(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"pending", `{"issues":[],"message":"test-secret"}`, "HTTP 202", 202},
		{"forbidden", `{"message":"test-secret"}`, "HTTP 403", 403},
		{"invalid", `not JSON test-secret`, "invalid FOSSA JSON", 200},
		{"missing array", `{}`, "issues array", 200},
		{"null array", `{"issues":null}`, "issues array", 200},
		{"wrong revision", `{"issues":[{"id":1,"source":{"id":"go+example.test/m$v1"},"projects":[]}]}`, "revision/scan", 200},
		{"repeated page", fmt.Sprintf(`{"issues":[{"id":1,"source":{"id":"go+example.test/m$v1"},"projects":[{"id":%q,"revisionId":%q,"revisionScanId":42}]}]}`, testProject, testProject+"$"+testRevision), "repeated issue", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"FOSSA_API_KEY": "test-secret", "FOSSA_REVISION": testRevision}
			c, output := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
					emptyFindings(w, r)
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			err := c.run(context.Background())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v; want %q", err, tc.want)
			}
			if strings.Contains(err.Error()+output.String(), "test-secret") {
				t.Fatal("credential exposed")
			}
			reports, globErr := filepath.Glob(filepath.Join(env["FOSSA_OUTPUT_DIR"], "*", "report.json"))
			if globErr != nil || len(reports) != 0 {
				t.Fatalf("partial report marked complete: %v; %v", reports, globErr)
			}
		})
	}
}

func TestFindingsDoesNotFollowRedirects(t *testing.T) {
	// Even same-origin redirects must not forward authentication or change scope.
	env := map[string]string{"FOSSA_API_KEY": "test-secret", "FOSSA_REVISION": testRevision}
	c, _ := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect-target" {
			t.Error("followed redirect with credential")
			emptyFindings(w, r)
			return
		}
		http.Redirect(w, r, "/redirect-target", http.StatusFound)
	})
	if err := c.run(context.Background()); err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("error = %v", err)
	}
}

func TestFindingsRejectsUnreadyOrChangingScan(t *testing.T) {
	for _, tc := range []struct {
		name, revision string
		change         bool
	}{
		{"pending", `{"locator":%q,"isSteady":false,"resolved":false,"latestRevisionScanId":42,"unresolved_licensing_issue_count":0}`, false},
		{"stale", `{"locator":%q,"isSteady":true,"resolved":true,"is_stale":true,"latestRevisionScanId":42,"unresolved_licensing_issue_count":0}`, false},
		{"no scan", `{"locator":%q,"isSteady":true,"resolved":true,"unresolved_licensing_issue_count":0}`, false},
		{"error", `{"locator":%q,"isSteady":true,"resolved":true,"latestRevisionScanId":42,"error":"test-secret","unresolved_licensing_issue_count":0}`, false},
		{"new scan", `{"locator":%q,"isSteady":true,"resolved":true,"latestRevisionScanId":43,"unresolved_licensing_issue_count":0}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			env := map[string]string{"FOSSA_API_KEY": "test-secret", "FOSSA_REVISION": testRevision}
			c, output := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
					reads++
					if !tc.change || reads > 1 {
						_, _ = fmt.Fprintf(w, tc.revision, testProject+"$"+testRevision)
						return
					}
				}
				emptyFindings(w, r)
			})
			err := c.run(context.Background())
			if err == nil || strings.Contains(err.Error()+output.String(), "test-secret") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestFindingsRejectsUnsafeInputs(t *testing.T) {
	for _, values := range []map[string]string{
		{"PR": "61", "FOSSA_REVISION": testRevision},
		{"PR": "$(touch sentinel)"},
		{"FOSSA_REVISION": "main"},
		{"FOSSA_API_KEY": "$(touch sentinel)"},
	} {
		if values["FOSSA_API_KEY"] == "" {
			values["FOSSA_API_KEY"] = "test-secret"
		}
		c, _ := findingsCommand(t, values, func(_ http.ResponseWriter, _ *http.Request) { t.Error("unexpected API request") })
		if err := c.run(context.Background()); err == nil {
			t.Fatal("unsafe input accepted")
		}
	}
}

func TestFindingsRejectsTruncatedIssueList(t *testing.T) {
	env := map[string]string{"FOSSA_API_KEY": "test-secret", "FOSSA_REVISION": testRevision}
	c, _ := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
			_, _ = fmt.Fprintf(w, `{"locator":%q,"isSteady":true,"resolved":true,"latestRevisionScanId":42,"unresolved_licensing_issue_count":1}`, testProject+"$"+testRevision)
			return
		}
		emptyFindings(w, r)
	})
	if err := c.run(context.Background()); err == nil || !strings.Contains(err.Error(), "count") {
		t.Fatalf("error = %v", err)
	}
}

func TestFindingsRejectsMissingRevisionCount(t *testing.T) {
	for _, tc := range []struct {
		name, count string
		read        int
	}{
		{"initial omitted", "", 1},
		{"initial null", `,"unresolved_licensing_issue_count":null`, 1},
		{"final omitted", "", 2},
		{"final null", `,"unresolved_licensing_issue_count":null`, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			env := map[string]string{"FOSSA_API_KEY": "test-secret", "FOSSA_REVISION": testRevision}
			c, output := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
					reads++
					if reads == tc.read {
						_, _ = fmt.Fprintf(w, `{"locator":%q,"isSteady":true,"resolved":true,"latestRevisionScanId":42%s}`, testProject+"$"+testRevision, tc.count)
						return
					}
				}
				emptyFindings(w, r)
			})
			if err := c.run(context.Background()); err == nil {
				t.Fatalf("missing revision count accepted as zero: %s", output)
			}
			reports, err := filepath.Glob(filepath.Join(env["FOSSA_OUTPUT_DIR"], "*", "report.json"))
			if err != nil || len(reports) != 0 || output.Len() != 0 {
				t.Fatalf("incomplete evidence published: reports=%v, output=%s, error=%v", reports, output, err)
			}
		})
	}
}

func TestMakeFindingsTargetKeepsOverridesLiteral(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "must-not-exist")
	malicious := "$(shell touch " + sentinel + ")"
	cmd := exec.Command("make", "-n", "fossa-findings", "PR="+malicious, "FOSSA_ENV_FILE="+malicious, "FOSSA_OUTPUT_DIR="+malicious, "FOSSA_REVISION="+malicious)
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Make target unavailable: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "go run ./scripts/fossafindings") {
		t.Fatalf("unexpected command: %s", output)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("Make override executed: %v", err)
	}
}

func TestFindingsRedactsReflectedCredentialFromArtifacts(t *testing.T) {
	env := map[string]string{"FOSSA_API_KEY": "test-secret", "FOSSA_REVISION": testRevision}
	c, output := findingsCommand(t, env, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/revisions/") {
			emptyFindings(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"issues":[],"note":"test-\u0073ecret"}`))
	})
	if err := c.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(env["FOSSA_OUTPUT_DIR"], "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		decoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(decoded)+output.String(), "test-secret") {
			t.Fatalf("credential leaked in %s", filepath.Base(path))
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatalf("artifact permissions not private: %v", err)
		}
	}
}
