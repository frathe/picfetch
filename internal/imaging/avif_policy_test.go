package imaging

import (
	"go/build"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAVIFBuildSelection(t *testing.T) {
	ctx := build.Default
	ctx.BuildTags = []string{"nodynamic"}
	imaging, err := ctx.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(imaging.Imports, "github.com/frathe/picfetch/internal/avifpolicy") {
		t.Fatal("imaging must depend on the AVIF build policy")
	}
	output, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/gen2brain/avif").Output()
	if err != nil {
		t.Fatal(err)
	}
	moduleDir := strings.TrimSpace(string(output))
	for _, os := range []string{"linux", "darwin", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			t.Run(os+"/"+arch, func(t *testing.T) {
				ctx := build.Default
				ctx.GOOS, ctx.GOARCH, ctx.CgoEnabled = os, arch, false
				ctx.BuildTags = []string{"nodynamic"}
				pkg, err := ctx.ImportDir(moduleDir, 0)
				if err != nil {
					t.Fatal(err)
				}
				if !slices.Contains(pkg.GoFiles, "avif_wazero.go") || !slices.Contains(pkg.GoFiles, "purego_other.go") || slices.Contains(pkg.GoFiles, "avif_dynamic.go") || slices.Contains(pkg.GoFiles, "avif_wasm2go.go") {
					t.Fatalf("unexpected AVIF implementation: %v", pkg.GoFiles)
				}
				for _, tags := range [][]string{nil, {"nodynamic"}, {"wasm2go"}, {"nodynamic", "wasm2go"}} {
					ctx.BuildTags = tags
					selected, err := ctx.MatchFile(filepath.Join("..", "avifpolicy"), "policy.go")
					if err != nil {
						t.Fatal(err)
					}
					want := slices.Contains(tags, "nodynamic") && !slices.Contains(tags, "wasm2go")
					if selected != want {
						t.Fatalf("policy selection with %v: %v", tags, selected)
					}
				}
			})
		}
	}
}
