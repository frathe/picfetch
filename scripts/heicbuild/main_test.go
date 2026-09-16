package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProvenanceRejectsChangedArtifactAndSources(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	records := map[string]string{"source.go": "0682c5f2076f099c34cfdd15a9e063849ed437a49677e6fcc5b4198c76575be5"}
	if err := checkDigests(root, records); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkDigests(root, records); err == nil {
		t.Fatal("changed source accepted")
	}
	if err := checkDigests(root, map[string]string{"missing.wasm": "00"}); err == nil {
		t.Fatal("missing artifact accepted")
	}
}

func TestImportGuardRejectsNativeCodec(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "sample.go")
	if err := os.WriteFile(file, []byte("package sample\nimport _ \"github.com/gen2brain/h265/heic\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkNativeSources(root); err == nil {
		t.Fatal("native codec accepted")
	}
	if err := os.WriteFile(file, []byte("package sample\nimport _ \"image/png\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkNativeSources(root); err != nil {
		t.Fatal(err)
	}
}

func TestImportGuardKeepsGuestRuntimeOutOfViewer(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "viewer.go")
	if err := os.WriteFile(file, []byte("package sample\nimport _ \"github.com/frathe/picfetch/internal/heicdecode/worker\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkNativeSources(root); err == nil {
		t.Fatal("viewer imported the embedded guest runtime")
	}
}

func TestMaintainedSourceIsInventoriedAndCannotBecomeNative(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"internal/heicdecode", guestDir, "docs/heic/notices", "scripts/heicbuild/testdata", "third_party/h265/heic"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	const source = "third_party/h265/heic/source.go"
	if err := os.WriteFile(filepath.Join(root, source), []byte("package heic\nimport _ \"github.com/gen2brain/h265/hevc\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	files, err := buildInputs(root)
	if err != nil || !slices.Contains(files, source) {
		t.Fatalf("maintained source missing from artifact inputs: %v, %v", files, err)
	}
	if err = checkNativeSources(root); err != nil {
		t.Fatalf("the separate decoder module is not a native application import: %v", err)
	}
	if err = os.WriteFile(filepath.Join(root, "native.go"), []byte("package sample\nimport _ \"github.com/gen2brain/h265/heic\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = checkNativeSources(root); err == nil {
		t.Fatal("maintained source was allowed into a native application")
	}
}

func TestMaintainedReplacementCannotBeRemovedOrRedirected(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, guestDir), 0700); err != nil {
		t.Fatal(err)
	}
	rootModule := "module example.test/app\n\ngo 1.27.1\n\nreplace github.com/gen2brain/h265 => ./third_party/h265\n"
	guestModule := "module example.test/guest\n\ngo 1.27.1\n\nrequire github.com/gen2brain/h265 v0.2.2\n\nreplace github.com/gen2brain/h265 => ../../third_party/h265\n"
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", rootModule)
	write(guestDir+"/go.mod", guestModule)
	if err := verifyReplacements(root); err != nil {
		t.Fatal(err)
	}
	write(guestDir+"/go.mod", "module example.test/guest\n")
	if err := verifyReplacements(root); err == nil {
		t.Fatal("removed guest replacement accepted")
	}
	write(guestDir+"/go.mod", guestModule)
	write("go.mod", "module example.test/app\nreplace github.com/gen2brain/h265 => ./unreviewed-source\n")
	if err := verifyReplacements(root); err == nil {
		t.Fatal("redirected root replacement accepted")
	}
}
