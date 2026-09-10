package similarity

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimePlatforms(t *testing.T) {
	for _, tc := range []struct {
		os, arch, library string
	}{
		{"linux", "amd64", "lib/libonnxruntime.so.1.29.0"},
		{"darwin", "arm64", "lib/libonnxruntime.1.29.0.dylib"},
		{"linux", "arm64", ""},
		{"windows", "amd64", ""},
	} {
		asset, supported := platformRuntime(tc.os, tc.arch)
		if supported != (tc.library != "") || asset.library != tc.library {
			t.Errorf("%s/%s: %+v, supported=%v", tc.os, tc.arch, asset, supported)
		}
		if supported {
			matches := 0
			for line := range strings.SplitSeq(strings.TrimSpace(assetChecksums), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[1] == asset.directory+"/"+asset.library && len(fields[0]) == 64 {
					matches++
				}
			}
			if matches != 1 || len(asset.digest) != 64 || asset.size <= 0 {
				t.Errorf("%s/%s runtime lacks complete checksum/size pins", tc.os, tc.arch)
			}
		}
	}
}

func TestUnpackRuntime(t *testing.T) {
	asset, err := currentRuntime()
	if err != nil {
		t.Skip("no native runtime on this platform")
	}
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
			files, err := unpackRuntime(ctx, root)
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
