package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

type receipt struct {
	Schema             int     `json:"schema"`
	Release            release `json:"release"`
	Notes              string  `json:"notes"`
	NotesSHA           string  `json:"notes_sha256"`
	BaseVersion        string  `json:"base_version"`
	BasePackageVersion string  `json:"base_package_version"`
	BaseSubmission     string  `json:"base_submission"`
	SubmissionID       string  `json:"submission_id,omitempty"`
	Phase              string  `json:"phase"`
	MetadataSHA        string  `json:"metadata_sha256,omitempty"`
}

type gitRunner func(context.Context, string, ...string) ([]byte, error)

func gitCommand(ctx context.Context, root string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = root
	b, err := c.Output()
	if err != nil {
		return nil, fmt.Errorf("git could not verify release provenance")
	}
	return b, nil
}
func waitContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type githubAPI struct{ rt runtime }

func encode(v any) ([]byte, error) { return json.Marshal(v) }

func normalizeRuntime(rt runtime) runtime {
	if rt.HTTP == nil {
		rt.HTTP = &http.Client{Timeout: 2 * time.Minute}
	}
	if rt.Now == nil {
		rt.Now = time.Now
	}
	if rt.Wait == nil {
		rt.Wait = waitContext
	}
	if rt.Git == nil {
		rt.Git = gitCommand
	}
	if rt.GitHubURL == "" {
		rt.GitHubURL = "https://api.github.com"
	}
	if rt.StoreURL == "" {
		rt.StoreURL = "https://manage.devcenter.microsoft.com/v1.0/my"
	}
	return rt
}
