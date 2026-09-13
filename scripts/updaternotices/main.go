// Command updaternotices checks the reviewed updater license inventory and
// notice delivery in release archives. It uses only the standard library.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const sectionStart = "<!-- BEGIN GENERATED UPDATER NOTICES -->"
const sectionEnd = "<!-- END GENERATED UPDATER NOTICES -->"

type noticeFile struct {
	Path       string   `json:"path"`
	SHA256     string   `json:"sha256"`
	Start      int      `json:"start,omitempty"`
	End        int      `json:"end,omitempty"`
	AppliesTo  []string `json:"applies_to,omitempty"`
	Repository bool     `json:"repository,omitempty"`
	Source     string   `json:"source,omitempty"`
}

type noticeModule struct {
	Module   string       `json:"module"`
	Version  string       `json:"version"`
	Source   string       `json:"source"`
	License  string       `json:"license"`
	Notes    string       `json:"notes,omitempty"`
	Packages []string     `json:"packages"`
	Files    []noticeFile `json:"files"`
}

type goModule struct {
	Path, Version, Dir string
	Main               bool
	Replace            *goModule
	packages           []string
}

func main() {
	root := flag.String("root", ".", "repository root")
	write := flag.Bool("write", false, "regenerate the updater section after reviewing the manifest")
	var artifacts artifactPaths
	flag.Var(&artifacts, "artifact", "finished ZIP, tar.gz, MSIX or MSIX bundle to check (repeatable; skips source regeneration)")
	flag.Parse()
	if err := run(*root, *write, artifacts); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string, write bool, artifacts []string) error {
	if len(artifacts) == 0 {
		return notices(root, write)
	}
	if write {
		return fmt.Errorf("-write cannot be combined with -artifact")
	}
	for _, artifact := range artifacts {
		if err := inspectArtifact(root, artifact); err != nil {
			return fmt.Errorf("%s: %w", artifact, err)
		}
		fmt.Printf("notices verified: %s\n", artifact)
	}
	return nil
}

type artifactPaths []string

func (p *artifactPaths) String() string { return strings.Join(*p, ", ") }
func (p *artifactPaths) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func notices(root string, write bool) error {
	manifest, err := os.ReadFile(filepath.Join(root, "scripts/updaternotices/manifest.json"))
	if err != nil {
		return err
	}
	var inventory []noticeModule
	if err := json.Unmarshal(manifest, &inventory); err != nil {
		return err
	}
	modules, err := updaterModules(root)
	if err != nil {
		return err
	}
	generated, err := renderNotices(root, inventory, modules)
	if err != nil {
		return err
	}
	out := filepath.Join(root, "THIRD-PARTY-NOTICES.md")
	existing, err := os.ReadFile(out)
	if err != nil {
		return err
	}
	updated, err := replaceSection(existing, generated)
	if err != nil {
		return err
	}
	if bytes.Equal(existing, updated) {
		return nil
	}
	if !write {
		return fmt.Errorf("updater notices are missing or stale; review manifest.json, then run make generate-updater-notices")
	}
	return os.WriteFile(out, updated, 0o644)
}

func updaterModules(root string) (map[string]goModule, error) {
	modules := make(map[string]goModule)
	for _, targetOS := range []string{"darwin", "linux", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			cmd := exec.Command("go", "list", "-mod=readonly", "-tags=no_emoji", "-deps", "-json", "./internal/update")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "GOOS="+targetOS, "GOARCH="+arch, "CGO_ENABLED=1", "GOWORK=off")
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			output, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("inventory %s/%s: %w: %s", targetOS, arch, err, &stderr)
			}
			decoder := json.NewDecoder(bytes.NewReader(output))
			for {
				var pkg struct {
					ImportPath string
					Module     *goModule
				}
				if err := decoder.Decode(&pkg); err == io.EOF {
					break
				} else if err != nil {
					return nil, err
				}
				if pkg.Module == nil || pkg.Module.Main {
					continue
				}
				module := *pkg.Module
				if module.Replace != nil {
					return nil, fmt.Errorf("updater module %s has an unreviewed replacement", module.Path)
				}
				module.packages = modules[module.Path].packages
				if !slices.Contains(module.packages, pkg.ImportPath) {
					module.packages = append(module.packages, pkg.ImportPath)
					slices.Sort(module.packages)
				}
				modules[module.Path] = module
			}
		}
	}
	return modules, nil
}

func renderNotices(root string, inventory []noticeModule, modules map[string]goModule) ([]byte, error) {
	if len(inventory) == 0 || len(inventory) != len(modules) {
		return nil, fmt.Errorf("reviewed updater inventory has %d modules; production closure has %d", len(inventory), len(modules))
	}
	var out bytes.Buffer
	out.WriteString(sectionStart + "\n\n## Updater dependency licenses and notices\n\n")
	out.WriteString("This section covers the production updater dependency union for macOS, Linux\nand Windows (amd64 and arm64). Versions and source files are recorded in\n`scripts/updaternotices/manifest.json`. Identical source texts share a single\ncopy below; each module lists every applicable text. Source comments are\nretained verbatim where a file carries its own license.\n\n")
	out.WriteString("Sources that reference Apache-2.0 use the complete [Apache license text](#updater-text-cfc7749b96f6) below.\n\n")
	texts := make(map[string][]byte)
	seen := make(map[string]bool)
	for _, entry := range inventory {
		module, ok := modules[entry.Module]
		if !ok || seen[entry.Module] || module.Version != entry.Version || !slices.Equal(module.packages, entry.Packages) {
			return nil, fmt.Errorf("module %s@%s is duplicated or differs from the production closure; review licenses", entry.Module, entry.Version)
		}
		seen[entry.Module] = true
		if entry.Source == "" || entry.License == "" || len(entry.Files) == 0 {
			return nil, fmt.Errorf("module %s lacks source/license files", entry.Module)
		}
		_, _ = fmt.Fprintf(&out, "### %s %s\n\nLicense: %s\n\nSource: %s\n\n", entry.Module, entry.Version, entry.License, entry.Source)
		if entry.Notes != "" {
			out.WriteString(entry.Notes + "\n\n")
		}
		for _, file := range entry.Files {
			if !filepath.IsLocal(file.Path) {
				return nil, fmt.Errorf("non-local notice path %q", file.Path)
			}
			dir := module.Dir
			if file.Repository {
				if file.Source == "" {
					return nil, fmt.Errorf("supplement %s lacks pinned source provenance", file.Path)
				}
				dir = root
			}
			data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file.Path)))
			if err != nil {
				return nil, err
			}
			if fmt.Sprintf("%x", sha256.Sum256(data)) != file.SHA256 {
				return nil, fmt.Errorf("%s/%s changed; review its license text", entry.Module, file.Path)
			}
			label := file.Path
			if file.Start != 0 || file.End != 0 {
				lines := bytes.SplitAfter(data, []byte("\n"))
				if file.Start < 1 || file.End < file.Start || file.End > len(lines) {
					return nil, fmt.Errorf("invalid notice line range for %s/%s", entry.Module, file.Path)
				}
				data = bytes.Join(lines[file.Start-1:file.End], nil)
				label += fmt.Sprintf(" (lines %d-%d)", file.Start, file.End)
			}
			id := fmt.Sprintf("%x", sha256.Sum256(data))
			texts[id] = data
			_, _ = fmt.Fprintf(&out, "- `%s`: [license text %s](#updater-text-%s)\n", label, id[:12], id[:12])
			if len(file.AppliesTo) > 0 {
				_, _ = fmt.Fprintf(&out, "  Applies to: `%s`.\n", strings.Join(file.AppliesTo, "`, `"))
			}
			if file.Source != "" {
				_, _ = fmt.Fprintf(&out, "  Original source: %s\n", file.Source)
			}
		}
		out.WriteByte('\n')
	}
	ids := make([]string, 0, len(texts))
	for id := range texts {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		_, _ = fmt.Fprintf(&out, "### Updater text %s\n\n````text\n", id[:12])
		out.Write(texts[id])
		if !bytes.HasSuffix(texts[id], []byte("\n")) {
			out.WriteByte('\n')
		}
		out.WriteString("````\n\n")
	}
	out.WriteString(sectionEnd + "\n")
	return out.Bytes(), nil
}

func replaceSection(existing, generated []byte) ([]byte, error) {
	text := string(existing)
	start := strings.Index(text, sectionStart)
	end := strings.Index(text, sectionEnd)
	if start == -1 && end == -1 {
		return append(append(bytes.Clone(existing), []byte("\n---\n\n")...), generated...), nil
	}
	if start < 0 || end < start || strings.Count(text, sectionStart) != 1 || strings.Count(text, sectionEnd) != 1 {
		return nil, fmt.Errorf("updater notice section markers are malformed")
	}
	end += len(sectionEnd)
	if end < len(existing) && existing[end] == '\n' {
		end++
	}
	updated := append(bytes.Clone(existing[:start]), generated...)
	return append(updated, existing[end:]...), nil
}
