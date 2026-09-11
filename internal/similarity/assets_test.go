package similarity

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"hash/crc32"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/distribution"
)

func TestStoreDownloadPolicy(t *testing.T) {
	if !distribution.StoreManaged {
		t.Skip("Store build policy")
	}
	if got := AssetDownloadBytes(); got != 371807752+394 {
		t.Fatalf("Store must download only model data: got %d bytes", got)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	t.Setenv("PICFETCH_SIMILARITY_ASSETS", cache)
	root, err := runtimeDirectory(cache)
	if err != nil || root != filepath.Dir(executable) {
		t.Fatalf("Store runtime followed a model/cache override: %s, %v", root, err)
	}
	requests := 0
	client := Client{Assets: cache, HTTPClient: &http.Client{Transport: similarityAssetTransport(func(_ *http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("unexpected download")
	})}}
	if _, err := client.InstallAssets(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "Microsoft Store") || requests != 0 {
		t.Fatalf("missing bundled runtime triggered a download: requests=%d err=%v", requests, err)
	}
}

type similarityAssetTransport func(*http.Request) (*http.Response, error)

func (f similarityAssetTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRuntimeDownloadSelection(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		asset, ok := platformRuntime("windows", arch)
		if !ok {
			t.Fatalf("missing Windows runtime: %s", arch)
		}
		for _, bundled := range []bool{false, true} {
			downloads := assetDownloads(asset, bundled)
			want := 3
			if bundled {
				want = 2
			}
			if len(downloads) != want {
				t.Fatalf("%s bundled=%v: %d downloads", arch, bundled, len(downloads))
			}
			for _, download := range downloads {
				if bundled && !strings.HasPrefix(download.address, "https://huggingface.co/") {
					t.Fatalf("Store setup downloads native code: %+v", download)
				}
			}
		}
	}
}

func TestStageWindowsRuntimeRejectsInvalidArchive(t *testing.T) {
	t.Run("checksum", func(t *testing.T) {
		root := t.TempDir()
		archive := filepath.Join(root, "wrong.zip")
		file, err := os.Create(archive)
		if err != nil {
			t.Fatal(err)
		}
		asset, _ := platformRuntime("windows", "amd64")
		sizeErr := file.Truncate(asset.size)
		closeErr := file.Close()
		if sizeErr != nil || closeErr != nil {
			t.Fatalf("archive fixture: %v, %v", sizeErr, closeErr)
		}
		out := filepath.Join(root, "package")
		if err := StageWindowsRuntime(context.Background(), "amd64", archive, out); err == nil || !strings.Contains(err.Error(), "checksum") {
			t.Fatalf("runtime archive hash was not checked: %v", err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("rejected archive changed package output: %v", err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := StageWindowsRuntime(ctx, "amd64", "missing.zip", t.TempDir()); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled package staging: %v", err)
		}
	})
	for _, arch := range []string{"amd64", "arm64", "386"} {
		t.Run(arch, func(t *testing.T) {
			root := t.TempDir()
			archive := filepath.Join(root, "runtime.zip")
			if err := os.WriteFile(archive, []byte("not a release archive"), 0600); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(root, "package")
			if err := StageWindowsRuntime(context.Background(), arch, archive, out); err == nil {
				t.Fatal("unverified runtime entered Store staging")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("rejected archive changed package output: %v", err)
			}
		})
	}
}

func TestRuntimePlatforms(t *testing.T) {
	for _, tc := range []struct {
		os, arch, library string
	}{
		{"linux", "amd64", "lib/libonnxruntime.so.1.29.0"},
		{"darwin", "arm64", "lib/libonnxruntime.1.29.0.dylib"},
		{"linux", "arm64", ""},
		{"windows", "amd64", "lib/onnxruntime.dll"},
		{"windows", "arm64", "lib/onnxruntime.dll"},
	} {
		asset, supported := platformRuntime(tc.os, tc.arch)
		if supported != (tc.library != "") || asset.library != tc.library {
			t.Errorf("%s/%s: %+v, supported=%v", tc.os, tc.arch, asset, supported)
		}
		if supported {
			for _, name := range append([]string{asset.library}, asset.supportLibraries...) {
				matches := 0
				for line := range strings.SplitSeq(strings.TrimSpace(assetChecksums), "\n") {
					fields := strings.Fields(line)
					if len(fields) == 2 && fields[1] == asset.directory+"/"+name && len(fields[0]) == 64 {
						matches++
					}
				}
				if matches != 1 {
					t.Errorf("%s/%s library %s lacks an unambiguous checksum", tc.os, tc.arch, name)
				}
			}
			if len(asset.digest) != 64 || asset.size <= 0 || asset.archiveExtension == "" {
				t.Errorf("%s/%s runtime lacks complete checksum/size pins", tc.os, tc.arch)
			}
		}
	}
}

func TestUnpackRuntime(t *testing.T) {
	asset, _ := platformRuntime("linux", "amd64")
	for _, mode := range []string{"valid", "missing", "duplicate", "symlink", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			archive, err := os.Create(filepath.Join(root, "runtime.tgz"))
			if err != nil {
				t.Fatal(err)
			}
			compressed := gzip.NewWriter(archive)
			writer := tar.NewWriter(compressed)
			names := []string{asset.directory + "/" + asset.library, asset.directory + "/LICENSE", asset.directory + "/ThirdPartyNotices.txt"}
			if mode == "missing" {
				names = names[:2]
			}
			if mode == "duplicate" {
				names = append(names, names[0])
			}
			// Untrusted paths are ignored without filesystem effects.
			names = append(names, "../escaped")
			for i, name := range names {
				header := &tar.Header{Name: "./" + name, Typeflag: tar.TypeReg, Mode: 0600, Size: 1}
				if mode == "symlink" && i == 0 {
					header.Typeflag, header.Size, header.Linkname = tar.TypeSymlink, 0, "../escaped"
				}
				if err := writer.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				if header.Size != 0 {
					if _, err := writer.Write([]byte("x")); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, closer := range []interface{ Close() error }{writer, compressed, archive} {
				if err := closer.Close(); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			files, err := unpackRuntimeAsset(ctx, root, asset)
			if mode != "valid" {
				if err == nil {
					t.Fatal("invalid/cancelled archive accepted")
				}
				return
			}
			if err != nil || len(files) != 3 {
				t.Fatalf("files=%v, err=%v", files, err)
			}
			for _, name := range files {
				if data, err := os.ReadFile(filepath.Join(root, name)); err != nil || string(data) != "x" {
					t.Fatalf("extracted file %s: %q, %v", name, data, err)
				}
			}
			if _, err := os.Stat(filepath.Join(root, "..", "escaped")); !os.IsNotExist(err) {
				t.Fatalf("archive escaped staging: %v", err)
			}
		})
	}
}

func TestUnpackWindowsRuntime(t *testing.T) {
	asset, _ := platformRuntime("windows", "amd64")
	for _, mode := range []string{"valid", "missing", "duplicate", "symlink", "directory", "oversized", "truncated", "checksum", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			archive, err := os.Create(filepath.Join(root, "runtime.zip"))
			if err != nil {
				t.Fatal(err)
			}
			writer := zip.NewWriter(archive)
			names := asset.files()
			if mode == "missing" {
				names = names[:len(names)-1]
			}
			if mode == "duplicate" {
				names = append(names, names[0])
			}
			names = append(names, "../escaped", `..\escaped`, asset.directory+"/lib/onnxruntime.dll:stream", asset.directory+"/lib/onnxruntime.pdb")
			for i, name := range names {
				header := &zip.FileHeader{Name: name, Method: zip.Store, CRC32: crc32.ChecksumIEEE([]byte("x")), CompressedSize64: 1, UncompressedSize64: 1}
				header.SetMode(0600)
				if i == 0 {
					switch mode {
					case "symlink":
						header.SetMode(os.ModeSymlink | 0600)
					case "directory":
						header.SetMode(os.ModeDir | 0700)
					case "oversized":
						header.UncompressedSize64 = 100<<20 + 1
					case "truncated":
						header.UncompressedSize64 = 2
					case "checksum":
						header.CRC32++
					}
				}
				entry, err := writer.CreateRaw(header)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := entry.Write([]byte("x")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			files, err := unpackRuntimeAsset(ctx, root, asset)
			if mode != "valid" {
				if err == nil {
					t.Fatal("invalid/cancelled ZIP accepted")
				}
				return
			}
			if err != nil || len(files) != 5 {
				t.Fatalf("files=%v, err=%v", files, err)
			}
			for _, name := range asset.files() {
				if data, err := os.ReadFile(filepath.Join(root, name)); err != nil || string(data) != "x" {
					t.Fatalf("extracted file %s: %q, %v", name, data, err)
				}
			}
			for _, path := range []string{filepath.Join(root, "..", "escaped"), filepath.Join(root, asset.directory, "lib", "onnxruntime.pdb"), filepath.Join(root, asset.directory, "lib", "onnxruntime.dll:stream")} {
				if _, err := os.Stat(path); err == nil {
					t.Fatalf("untrusted ZIP entry extracted: %s", path)
				}
			}
		})
	}
}

// Cancel after the initial admission check, when the file checksum is read.
type checksumCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *checksumCancelContext) Err() error {
	c.checks++
	if c.checks == 2 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestVerifyAssetsCancellationDuringChecksum(t *testing.T) {
	if _, err := currentRuntime(); err != nil {
		t.Skip("requires supported runtime")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "vision_model.onnx"), make([]byte, 128*1024), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checking := &checksumCancelContext{Context: ctx, cancel: cancel}
	if err := VerifyAssets(checking, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("checksum did not observe in-file cancellation: %v", err)
	}
}
