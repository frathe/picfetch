package main

import (
	"os"
	"path/filepath"
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
