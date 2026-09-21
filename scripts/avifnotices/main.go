// Command avifnotices checks the reviewed AVIF payload and its bundled notices.
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
)

const (
	sectionStart = "<!-- BEGIN GENERATED AVIF NOTICES -->"
	sectionEnd   = "<!-- END GENERATED AVIF NOTICES -->"
)

type sourceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Source string `json:"source,omitempty"`
}

type modulePin struct {
	Path    string       `json:"path"`
	Version string       `json:"version"`
	Files   []sourceFile `json:"files"`
}

type component struct {
	Name    string       `json:"name"`
	Version string       `json:"version"`
	License string       `json:"license"`
	Notes   string       `json:"notes,omitempty"`
	Files   []sourceFile `json:"files"`
}

type manifest struct {
	Notes      string      `json:"notes"`
	Modules    []modulePin `json:"modules"`
	Components []component `json:"components"`
}

type resolvedModule struct {
	Path, Version, Dir string
	Replace            *resolvedModule
}

func main() {
	root := flag.String("root", ".", "repository root")
	write := flag.Bool("write", false, "regenerate the AVIF notice section from reviewed inputs")
	flag.Parse()
	if err := run(*root, *write); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string, write bool) error {
	cmd := exec.Command("go", "list", "-mod=readonly", "-m", "-json", "github.com/gen2brain/avif", "github.com/tetratelabs/wazero")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("resolve AVIF modules: %w: %s", err, &stderr)
	}
	modules := make(map[string]resolvedModule)
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var module resolvedModule
		if err := decoder.Decode(&module); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		modules[module.Path] = module
	}
	return notices(root, write, modules)
}

func notices(root string, write bool, modules map[string]resolvedModule) error {
	data, err := os.ReadFile(filepath.Join(root, "scripts/avifnotices/manifest.json"))
	if err != nil {
		return err
	}
	var inventory manifest
	if err := json.Unmarshal(data, &inventory); err != nil {
		return err
	}
	generated, err := renderNotices(root, inventory, modules)
	if err != nil {
		return err
	}
	path := filepath.Join(root, "THIRD-PARTY-NOTICES.md")
	existing, err := os.ReadFile(path)
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
		return fmt.Errorf("AVIF notices are missing or stale; review scripts/avifnotices/manifest.json, then run make generate-avif-notices")
	}
	return os.WriteFile(path, updated, 0o644)
}

func renderNotices(root string, inventory manifest, modules map[string]resolvedModule) ([]byte, error) {
	if len(inventory.Modules) == 0 || len(inventory.Modules) != len(modules) || len(inventory.Components) == 0 || inventory.Notes == "" {
		return nil, fmt.Errorf("incomplete reviewed AVIF inventory")
	}
	var out bytes.Buffer
	out.WriteString(sectionStart + "\n\n## AVIF and embedded WASM licenses and notices\n\n")
	out.WriteString(inventory.Notes + "\n\n")
	seen := make(map[string]bool)
	for _, pin := range inventory.Modules {
		module, ok := modules[pin.Path]
		if !ok || seen[pin.Path] || module.Replace != nil || module.Version != pin.Version || len(pin.Files) == 0 {
			return nil, fmt.Errorf("AVIF module %s@%s changed, was replaced, or lacks reviewed inputs", pin.Path, pin.Version)
		}
		seen[pin.Path] = true
		_, _ = fmt.Fprintf(&out, "### Reviewed build input: %s %s\n\n", pin.Path, pin.Version)
		for _, file := range pin.Files {
			if _, err := readVerified(module.Dir, file); err != nil {
				return nil, err
			}
			_, _ = fmt.Fprintf(&out, "- `%s` SHA-256: `%s`\n", file.Path, file.SHA256)
		}
		out.WriteByte('\n')
	}
	seen = make(map[string]bool)
	for _, entry := range inventory.Components {
		if entry.Name == "" || seen[entry.Name] || entry.Version == "" || entry.License == "" || len(entry.Files) == 0 {
			return nil, fmt.Errorf("incomplete or duplicate AVIF component %q", entry.Name)
		}
		seen[entry.Name] = true
		_, _ = fmt.Fprintf(&out, "### %s\n\nVersion/source revision: %s\n\nLicense: %s\n\n", entry.Name, entry.Version, entry.License)
		if entry.Notes != "" {
			out.WriteString(entry.Notes + "\n\n")
		}
		for _, file := range entry.Files {
			if file.Source == "" {
				return nil, fmt.Errorf("notice %s lacks source provenance", file.Path)
			}
			text, err := readVerified(filepath.Join(root, "scripts/avifnotices"), file)
			if err != nil {
				return nil, err
			}
			_, _ = fmt.Fprintf(&out, "Source: %s\n\n````text\n", file.Source)
			out.Write(text)
			if !bytes.HasSuffix(text, []byte("\n")) {
				out.WriteByte('\n')
			}
			out.WriteString("````\n\n")
		}
	}
	out.WriteString(sectionEnd)
	return out.Bytes(), nil
}

func readVerified(dir string, file sourceFile) ([]byte, error) {
	if !filepath.IsLocal(file.Path) {
		return nil, fmt.Errorf("non-local source path %q", file.Path)
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file.Path)))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || fmt.Sprintf("%x", sha256.Sum256(data)) != file.SHA256 {
		return nil, fmt.Errorf("%s changed; review the payload/source license before updating notices", file.Path)
	}
	return data, nil
}

func replaceSection(existing, generated []byte) ([]byte, error) {
	start, end := []byte(sectionStart), []byte(sectionEnd)
	first, last := bytes.Index(existing, start), bytes.Index(existing, end)
	if bytes.Count(existing, start) != 1 || bytes.Count(existing, end) != 1 || last < first {
		return nil, fmt.Errorf("missing or duplicate AVIF notice section markers")
	}
	updated := append(bytes.Clone(existing[:first]), generated...)
	return append(updated, existing[last+len(end):]...), nil
}
