package main

import (
	"strings"
	"testing"
)

const sampleTodos = `# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- possibility to drag a selection rectangle in grid view to select multiple images.

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT

## TODO

- leftover work
`

func TestBuild_DropsEmptyCategoriesAndAppendsChangelog(t *testing.T) {
	got, err := Build(sampleTodos, "0.2.7", "0.2.8")
	if err != nil {
		t.Fatal(err)
	}
	want := `## What's Changed

### New Features

- possibility to drag a selection rectangle in grid view to select multiple images.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.7...v0.2.8
`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuild_KeepsOnlyCategoriesWithItems(t *testing.T) {
	md := `# x

## Done

### What's Changed

#### New Features

#### Bugfix

- Fix fyne install by declaring the app icon in FyneApp.toml

#### Internal

- Split the test monolith

## TODO
`
	got, err := Build(md, "v0.2.4", "v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "New Features") {
		t.Fatalf("empty New Features should be dropped:\n%s", got)
	}
	if !strings.Contains(got, "### Bugfix") || !strings.Contains(got, "### Internal") {
		t.Fatalf("kept categories missing:\n%s", got)
	}
	if !strings.Contains(got, "compare/v0.2.4...v0.2.5") {
		t.Fatalf("changelog tags: %s", got)
	}
}

func TestBuild_RejectsEmptyDone(t *testing.T) {
	md := `# x

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## TODO
`
	_, err := Build(md, "0.1.0", "0.1.1")
	if err == nil || !strings.Contains(err.Error(), "no release-note items") {
		t.Fatalf("got %v, want no release-note items", err)
	}
}

func TestBuild_MissingDone(t *testing.T) {
	_, err := Build("# x\n\n## TODO\n", "0.1.0", "0.1.1")
	if err == nil || !strings.Contains(err.Error(), "## Done") {
		t.Fatalf("got %v, want missing ## Done", err)
	}
}

func TestClearDone_KeepsSkeletonAndRestOfFile(t *testing.T) {
	got, err := ClearDone(sampleTodos)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "selection rectangle") {
		t.Fatalf("Done items should be cleared:\n%s", got)
	}
	if !strings.Contains(got, "## TODO") || !strings.Contains(got, "- leftover work") {
		t.Fatalf("rest of file lost:\n%s", got)
	}
	wantHead := `# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT
`
	if !strings.HasPrefix(got, wantHead) {
		t.Fatalf("got prefix:\n%s", got)
	}
}
