package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAVIFNoticesMatchReviewedPayload(t *testing.T) {
	if err := run(filepath.Join("..", ".."), false); err != nil {
		t.Fatal(err)
	}
}

func TestAVIFNoticesRejectUnreviewedInputs(t *testing.T) {
	root := t.TempDir()
	payload := []byte("reviewed WASM payload")
	license := []byte("Copyright Example\nAll conditions and disclaimers.\n")
	writeFixture(t, filepath.Join(root, "payload.wasm.gz"), payload)
	writeFixture(t, filepath.Join(root, "scripts/avifnotices/licenses/example.txt"), license)
	pin := modulePin{Path: "example.org/avif", Version: "v1.0.0", Files: []sourceFile{{Path: "payload.wasm.gz", SHA256: digest(payload)}}}
	entry := component{Name: "Example codec", Version: "v1", License: "MIT", Files: []sourceFile{{Path: "licenses/example.txt", SHA256: digest(license), Source: "https://example.org/revision/LICENSE"}}}
	modules := map[string]resolvedModule{pin.Path: {Path: pin.Path, Version: pin.Version, Dir: root}}
	for _, change := range []string{"valid", "missing pin", "duplicate pin", "version", "replacement", "payload", "missing payload", "missing component", "duplicate component", "missing license", "license bytes", "source provenance", "outside path"} {
		t.Run(change, func(t *testing.T) {
			candidate := manifest{Notes: "Reviewed fixture", Modules: []modulePin{pin}, Components: []component{entry}}
			candidate.Modules[0].Files = slices.Clone(pin.Files)
			candidate.Components[0].Files = slices.Clone(entry.Files)
			resolved := modules[pin.Path]
			switch change {
			case "missing pin":
				candidate.Modules = nil
			case "duplicate pin":
				candidate.Modules = append(candidate.Modules, pin)
			case "version":
				resolved.Version = "v2.0.0"
			case "replacement":
				resolved.Replace = &resolvedModule{Dir: root}
			case "payload":
				candidate.Modules[0].Files[0].SHA256 = digest([]byte("different WASM"))
			case "missing payload":
				candidate.Modules[0].Files[0].Path = "absent.wasm.gz"
			case "missing component":
				candidate.Components = nil
			case "duplicate component":
				candidate.Components = append(candidate.Components, entry)
			case "missing license":
				candidate.Components[0].Files = nil
			case "license bytes":
				candidate.Components[0].Files[0].SHA256 = digest([]byte("abbreviated terms"))
			case "source provenance":
				candidate.Components[0].Files[0].Source = ""
			case "outside path":
				candidate.Components[0].Files[0].Path = "../example.txt"
			}
			text, err := renderNotices(root, candidate, map[string]resolvedModule{pin.Path: resolved})
			if change != "valid" {
				if err == nil {
					t.Fatal("accepted unreviewed or incomplete source inventory")
				}
				return
			}
			if err != nil || !bytes.Contains(text, license) {
				t.Fatalf("complete original license text missing: %v", err)
			}
			encoded, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, filepath.Join(root, "scripts/avifnotices/manifest.json"), encoded)
			path := filepath.Join(root, "THIRD-PARTY-NOTICES.md")
			writeFixture(t, path, []byte("other notices before\n"+sectionStart+"\nstale\n"+sectionEnd+"\nother notices after\n"))
			if err := notices(root, false, modules); err == nil {
				t.Fatal("accepted a stale shipped notice")
			}
			if err := notices(root, true, modules); err != nil {
				t.Fatal(err)
			}
			if err := notices(root, false, modules); err != nil {
				t.Fatalf("regenerated notices fail their own check: %v", err)
			}
			updated, err := os.ReadFile(path)
			if err != nil || !bytes.HasPrefix(updated, []byte("other notices before\n")) || !bytes.HasSuffix(updated, []byte("other notices after\n")) || !bytes.Contains(updated, license) {
				t.Fatalf("generation lost surrounding notices or original license text: %v", err)
			}
		})
	}
}

func TestAVIFNoticesRejectMissingOrAmbiguousSection(t *testing.T) {
	for _, document := range []string{"", sectionStart, sectionEnd + sectionStart, sectionStart + sectionEnd + sectionStart} {
		if _, err := replaceSection([]byte(document), []byte("generated")); err == nil {
			t.Fatalf("accepted broken notice section %q", document)
		}
	}
}

func TestAVIFNoticesRunBeforeBuild(t *testing.T) {
	for _, check := range []struct{ path, want string }{
		{"Makefile", "verify-build: "},
		{".github/workflows/ci.yml", "run: make check-avif-notices"},
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", check.path))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), check.want) {
			t.Errorf("%s lacks %q", check.path, check.want)
		}
		if check.path == "Makefile" {
			for line := range strings.SplitSeq(string(data), "\n") {
				if strings.HasPrefix(line, check.want) && !strings.Contains(line, " check-avif-notices") {
					t.Error("verify-build must require check-avif-notices")
				}
			}
		}
	}
}

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func writeFixture(t *testing.T, path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
