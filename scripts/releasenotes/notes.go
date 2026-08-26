package main

import (
	"fmt"
	"regexp"
	"strings"
)

const changelogFmt = "**Full Changelog**: https://github.com/frathe/picfetch/compare/%s...%s"

const doneSkeleton = `## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

`

var (
	h2Done      = regexp.MustCompile(`(?m)^## Done[ \t]*\r?$`)
	h2Any       = regexp.MustCompile(`(?m)^## `)
	extraBlanks = regexp.MustCompile(`\n{3,}`)
)

// Build turns the ## Done section of todos.md into GitHub release notes:
// empty #### categories are dropped, remaining ATX headings are promoted one
// level (so they match previous releases once un-nested from ## Done), and
// the compare changelog line is appended.
func Build(todos, prev, next string) (string, error) {
	_, body, _, err := splitDone(todos)
	if err != nil {
		return "", err
	}
	body = dropEmptyH4(body)
	body = collapseBlanks(body)
	if !hasNoteItems(body) {
		return "", fmt.Errorf("todos.md ## Done has no release-note items")
	}
	body = promoteHeadings(body)
	return body + "\n\n" + fmt.Sprintf(changelogFmt, versionTag(prev), versionTag(next)) + "\n", nil
}

// ClearDone replaces the ## Done body with the empty category skeleton,
// leaving the rest of the file unchanged.
func ClearDone(todos string) (string, error) {
	before, _, after, err := splitDone(todos)
	if err != nil {
		return "", err
	}
	return before + doneSkeleton + after, nil
}

func versionTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func splitDone(md string) (before, body, after string, err error) {
	loc := h2Done.FindStringIndex(md)
	if loc == nil {
		return "", "", "", fmt.Errorf("todos.md has no ## Done section")
	}
	before = md[:loc[0]]
	rest := md[loc[1]:]
	switch {
	case strings.HasPrefix(rest, "\r\n"):
		rest = rest[2:]
	case strings.HasPrefix(rest, "\n"):
		rest = rest[1:]
	}
	next := h2Any.FindStringIndex(rest)
	if next == nil {
		return before, rest, "", nil
	}
	return before, rest[:next[0]], rest[next[0]:], nil
}

func dropEmptyH4(body string) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	var cat []string
	flush := func() {
		if cat == nil {
			return
		}
		if h4HasContent(cat) {
			out = append(out, cat...)
		}
		cat = nil
	}
	for _, line := range lines {
		if isH4(line) {
			flush()
			cat = []string{line}
			continue
		}
		if cat != nil {
			cat = append(cat, line)
			continue
		}
		out = append(out, line)
	}
	flush()
	return strings.Join(out, "\n")
}

func isH4(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#### ")
}

func h4HasContent(cat []string) bool {
	for i, line := range cat {
		if i == 0 {
			continue
		}
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		return true
	}
	return false
}

func hasNoteItems(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			return true
		}
	}
	return false
}

func promoteHeadings(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		n, rest, ok := atx(strings.TrimRight(line, "\r"))
		if ok && n >= 2 {
			lines[i] = strings.Repeat("#", n-1) + rest
		}
	}
	return strings.Join(lines, "\n")
}

func atx(line string) (level int, rest string, ok bool) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 {
		return 0, "", false
	}
	if i == len(line) {
		return i, "", true
	}
	if line[i] != ' ' {
		return 0, "", false
	}
	return i, line[i:], true
}

func collapseBlanks(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = extraBlanks.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
