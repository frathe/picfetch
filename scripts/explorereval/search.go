package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type searchSource struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type searchQuery struct {
	ReferenceID string   `json:"reference_id"`
	Intent      string   `json:"intent"`
	RelevantIDs []string `json:"relevant_ids"`
}

type searchCorpus struct {
	Version int            `json:"version"`
	Sources []searchSource `json:"sources"`
	Queries []searchQuery  `json:"queries"`
}

func readSearchCorpus(library string) (searchCorpus, error) {
	var corpus searchCorpus
	file, err := os.Open(filepath.Join(library, "search-corpus.json"))
	if err != nil {
		return corpus, fmt.Errorf("search corpus: %w", err)
	}
	defer func() { _ = file.Close() }()
	const maximumBytes = 16 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil {
		return corpus, fmt.Errorf("search corpus: %w", err)
	}
	if len(data) > maximumBytes {
		return corpus, fmt.Errorf("search corpus exceeds 16 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		return corpus, fmt.Errorf("search corpus: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return corpus, fmt.Errorf("search corpus has trailing JSON after its manifest")
	}
	if corpus.Version != 1 {
		return corpus, fmt.Errorf("search corpus version must be 1")
	}
	if len(corpus.Sources) > (256*1024*1024)/(768*4) {
		return corpus, fmt.Errorf("search corpus exceeds the 256 MiB retained-vector budget")
	}
	sources, paths := make(map[string]bool), make(map[string]bool)
	for _, source := range corpus.Sources {
		if source.ID == "" || sources[source.ID] {
			return corpus, fmt.Errorf("search corpus has an empty or duplicate source identity %q", source.ID)
		}
		if !fs.ValidPath(source.Path) || source.Path == "." || strings.Contains(source.Path, `\`) || !filepath.IsLocal(filepath.FromSlash(source.Path)) {
			return corpus, fmt.Errorf("search corpus requires a relative source path: %q", source.Path)
		}
		if paths[source.Path] {
			return corpus, fmt.Errorf("search corpus has a duplicate source path %q", source.Path)
		}
		sources[source.ID], paths[source.Path] = true, true
	}
	content := make(map[string]bool)
	queries := make(map[[2]string]bool)
	for _, query := range corpus.Queries {
		if query.Intent != "content" && query.Intent != "appearance" {
			return corpus, fmt.Errorf("search corpus query intent must be content or appearance")
		}
		if !sources[query.ReferenceID] {
			return corpus, fmt.Errorf("search corpus has unknown reference %q", query.ReferenceID)
		}
		key := [2]string{query.ReferenceID, query.Intent}
		if queries[key] {
			return corpus, fmt.Errorf("search corpus has duplicate query %q/%s", query.ReferenceID, query.Intent)
		}
		queries[key] = true
		judgments := make(map[string]bool)
		for _, relevant := range query.RelevantIDs {
			if !sources[relevant] {
				return corpus, fmt.Errorf("search corpus has unknown relevant source %q", relevant)
			}
			if relevant == query.ReferenceID {
				return corpus, fmt.Errorf("search corpus reference cannot be relevant to itself")
			}
			if judgments[relevant] {
				return corpus, fmt.Errorf("search corpus has duplicate relevant source %q", relevant)
			}
			judgments[relevant] = true
		}
		if query.Intent == "content" {
			content[query.ReferenceID] = true
		}
	}
	if len(content) < 20 {
		return corpus, fmt.Errorf("search corpus requires at least 20 distinct content references")
	}
	return corpus, nil
}
