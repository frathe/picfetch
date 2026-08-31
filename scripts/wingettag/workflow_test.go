package wingettag

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkflowGatesUntrustedReleaseTag(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", ".github", "workflows", "winget.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	yml := string(raw)

	if strings.Contains(yml, "release-tag: ${{ github.event.workflow_run.head_branch }}") {
		t.Error("release-tag must not be taken directly from workflow_run.head_branch")
	}
	if n := strings.Count(yml, "github.event.workflow_run.head_branch"); n != 1 {
		t.Errorf("head_branch must appear exactly once (as CANDIDATE env), found %d", n)
	}
	if !strings.Contains(yml, "CANDIDATE: ${{ github.event.workflow_run.head_branch }}") {
		t.Error("head_branch must enter the job only as the CANDIDATE env var")
	}
	if !strings.Contains(yml, "github.event.workflow_run.path == '.github/workflows/release.yml'") {
		t.Error("job must require the triggering workflow file to be release.yml, not any workflow named Release")
	}
	if !strings.Contains(yml, "github.event.workflow_run.event == 'push'") {
		t.Error("job must require the triggering workflow's event to be push")
	}
	if !strings.Contains(yml, Pattern) {
		t.Errorf("job must allowlist tags with %s", Pattern)
	}
	if !strings.Contains(yml, "release-tag: ${{ steps.tag.outputs.tag }}") {
		t.Error("winget-releaser must receive the allowlisted tag output, not the raw event field")
	}
	if !strings.Contains(yml, "GH_REPO: ${{ github.repository }}") {
		t.Error("gh must set GH_REPO; the job has no checkout, so gh cannot infer the repo from git")
	}
	if !strings.Contains(yml, "gh release view") {
		t.Error("job must require a published GitHub release before calling winget-releaser")
	}
	if !strings.Contains(yml, "isDraft") {
		t.Error("job must reject draft GitHub releases, not only that gh release view succeeds")
	}
	if !strings.Contains(yml, `[ "$draft" != "false" ]`) {
		t.Error("job must require isDraft to be false")
	}
}
