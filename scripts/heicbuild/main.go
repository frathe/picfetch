// Command heicbuild verifies and reproduces the development-only WASI guest.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	decoderModule   = "github.com/gen2brain/h265"
	decoderVersion  = "v0.2.3"
	decoderRevision = "b2d46ba787d8f0a2025bd106443ab1b1c7cd010f"
	decoderSum      = "h1:+fEP2Xf1CoZ21SxA2YpqnPZb6Y/hAEkcgjU1gMsOhrk="
	decoderZipSHA   = "2838bcb83b8da357a19788ad5d8a9d55c16bd4a12961fa81e3ab282974f1ed7a"
	compilerVersion = "go1.27.1"
	guestDir        = "scripts/heicguest"
	guestArtifact   = "internal/heicdecode/worker/decoder.wasm"
	manifestPath    = guestDir + "/decoder.json"
)

type manifest struct {
	Version                                                int
	Module, Revision, ModuleSum, ModuleZipSHA256, Compiler string
	Files                                                  map[string]string
}

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: heicbuild build|check|imports|fixture|photo-fixture")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(mode string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if mode == "imports" {
		return checkImports(root)
	}
	if mode != "build" && mode != "check" && mode != "fixture" && mode != "photo-fixture" {
		return errors.New("unknown heicbuild mode")
	}
	if err = verifySource(root); err != nil {
		return err
	}
	if mode == "fixture" {
		return generateFixture(root)
	}
	if mode == "photo-fixture" {
		return generatePhotoFixture(root)
	}
	files, err := buildInputs(root)
	if err != nil {
		return err
	}
	if mode == "check" {
		data, err := os.ReadFile(filepath.Join(root, manifestPath))
		if err != nil {
			return err
		}
		var m manifest
		if err = json.Unmarshal(data, &m); err != nil {
			return err
		}
		if m.Version != 1 || m.Module != decoderModule+"@"+decoderVersion || m.Revision != decoderRevision || m.ModuleSum != decoderSum || m.ModuleZipSHA256 != decoderZipSHA || m.Compiler != compilerVersion {
			return errors.New("guest manifest disagrees with reviewed source/toolchain pins")
		}
		expected := append(append([]string(nil), files...), guestArtifact)
		if len(m.Files) != len(expected) {
			return errors.New("guest input inventory changed")
		}
		for _, name := range expected {
			if _, ok := m.Files[name]; !ok {
				return fmt.Errorf("missing guest input %s", name)
			}
		}
		if err = checkDigests(root, m.Files); err != nil {
			return err
		}
	}
	dir, err := os.MkdirTemp("", "picfetch-heic-build-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	artifact := filepath.Join(dir, "decoder.wasm")
	if _, err = goCommand(root, true, "build", "-mod=readonly", "-tags=noasm", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", artifact, "."); err != nil {
		return err
	}
	built, err := os.ReadFile(artifact)
	if err != nil {
		return err
	}
	if mode == "check" {
		original, err := os.ReadFile(filepath.Join(root, guestArtifact))
		if err != nil {
			return err
		}
		if !bytes.Equal(original, built) {
			return errors.New("guest rebuild differs from checked artifact")
		}
		fmt.Println("HEIC development guest: source, notices, input inventory and reproducible artifact verified")
		return nil
	}
	if err = os.WriteFile(filepath.Join(root, guestArtifact), built, 0644); err != nil {
		return err
	}
	m := manifest{Version: 1, Module: decoderModule + "@" + decoderVersion, Revision: decoderRevision, ModuleSum: decoderSum, ModuleZipSHA256: decoderZipSHA, Compiler: compilerVersion, Files: make(map[string]string)}
	for _, name := range append(files, guestArtifact) {
		sum, err := fileDigest(filepath.Join(root, name))
		if err != nil {
			return err
		}
		m.Files[name] = sum
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(root, manifestPath), append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Println("Built development guest and refreshed its provenance manifest; production HEIC remains disabled")
	return nil
}

func goCommand(root string, guest bool, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	overrides := map[string]string{"GOWORK": "off", "GOTOOLCHAIN": "local", "GOEXPERIMENT": "", "GOFLAGS": "", "GOWASM": "", "GOAMD64": "v1", "GOARM64": "v8.0", "CGO_ENABLED": "0"}
	if guest {
		cmd.Dir = filepath.Join(root, guestDir)
		overrides["GOOS"] = "wasip1"
		overrides["GOARCH"] = "wasm"
	}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, overridden := overrides[key]; !overridden {
			cmd.Env = append(cmd.Env, item)
		}
	}
	for key, value := range overrides {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, output)
	}
	return output, nil
}

func verifySource(root string) error {
	version, err := goCommand(root, true, "env", "GOVERSION")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(version)) != compilerVersion {
		return fmt.Errorf("guest requires reviewed compiler %s", compilerVersion)
	}
	data, err := goCommand(root, true, "mod", "download", "-json", decoderModule+"@"+decoderVersion)
	if err != nil {
		return err
	}
	var module struct {
		Path, Version, Sum, Zip, Dir string
		Origin                       struct{ Hash string }
	}
	if err = json.Unmarshal(data, &module); err != nil {
		return err
	}
	if module.Path != decoderModule || module.Version != decoderVersion || module.Sum != decoderSum || module.Origin.Hash != decoderRevision {
		return errors.New("upstream module provenance differs from pin")
	}
	sum, err := fileDigest(module.Zip)
	if err != nil {
		return err
	}
	if sum != decoderZipSHA {
		return errors.New("upstream module archive digest mismatch")
	}
	if _, err = goCommand(root, true, "mod", "verify"); err != nil {
		return err
	}
	shipped, err := os.ReadFile(filepath.Join(root, "docs/heic/notices/h265-LICENSE"))
	if err != nil {
		return err
	}
	original, err := os.ReadFile(filepath.Join(module.Dir, "LICENSE"))
	if err != nil {
		return err
	}
	if !bytes.Equal(shipped, original) {
		return errors.New("h265 notice differs from pinned upstream")
	}
	return nil
}

func buildInputs(root string) ([]string, error) {
	var files []string
	for _, dir := range []string{"internal/heicdecode", guestDir, "docs/heic/notices", "scripts/heicbuild/testdata"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// The native host is outside the guest's compiled dependency graph.
				// Its embedded artifact is inventoried explicitly below.
				if dir == "internal/heicdecode" && path != filepath.Join(root, dir) {
					return filepath.SkipDir
				}
				return nil
			}
			name := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
			if strings.HasSuffix(name, "_test.go") || name == guestArtifact || name == manifestPath {
				return nil
			}
			files = append(files, name)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	files = append(files, "scripts/heicbuild/main.go", "scripts/heicbuild/fixture.go")
	sort.Strings(files)
	return files, nil
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func checkDigests(root string, records map[string]string) error {
	for name, want := range records {
		if !filepath.IsLocal(name) {
			return errors.New("nonlocal provenance path")
		}
		got, err := fileDigest(filepath.Join(root, name))
		if err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("digest mismatch: %s", name)
		}
	}
	return nil
}

func forbiddenImport(path string) bool {
	for _, module := range []string{decoderModule, "github.com/gen2brain/heic", "github.com/frathe/heic"} {
		if path == module || strings.HasPrefix(path, module+"/") {
			return true
		}
	}
	return false
}

func checkNativeSources(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == guestDir || (rel != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "bin" || d.Name() == "fyne-cross")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, item := range file.Imports {
			imported, err := strconv.Unquote(item.Path.Value)
			if err != nil {
				return err
			}
			if forbiddenImport(imported) {
				return fmt.Errorf("forbidden native codec import %s in %s", imported, rel)
			}
			if imported == "github.com/frathe/picfetch/internal/heicdecode/worker" && rel != "cmd/picfetch-heic-worker/main.go" {
				return fmt.Errorf("embedded HEIC runtime is helper-only: %s", rel)
			}
		}
		return nil
	})
}

func checkImports(root string) error {
	if err := checkNativeSources(root); err != nil {
		return err
	}
	data, err := goCommand(root, true, "list", "-mod=readonly", "-deps", "-f", "{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{end}}", ".")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && line != "github.com/frathe/picfetch v0.0.0" && line != decoderModule+" "+decoderVersion && line != "github.com/frathe/picfetch/scripts/heicguest" {
			return fmt.Errorf("unreviewed guest dependency: %s", line)
		}
	}
	data, err = goCommand(root, false, "list", "-mod=readonly", "-tags=no_emoji,nodynamic", "-deps", "./internal/heicdecode", "./internal/imaging", "./internal/similarity")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if forbiddenImport(line) {
			return fmt.Errorf("native graph contains %s", line)
		}
	}
	if err = filepath.WalkDir(filepath.Join(root, guestDir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(source, []byte("//go:build wasip1 && wasm\n")) {
			return fmt.Errorf("guest source must remain WASI-only: %s", path)
		}
		return nil
	}); err != nil {
		return err
	}

	fmt.Println("HEIC imports: codec only in reviewed WASI guest; native source and imaging/analysis graph clear")
	return nil
}
