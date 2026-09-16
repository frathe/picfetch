// Command heicpackage stages the isolated helper and its notices. Staging does
// not enable HEIC viewing or claim native/platform qualification.
package main

import (
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frathe/picfetch/internal/heicdecode/client"
)

type options struct{ mode, root, out, system, architecture, identity string }

func main() {
	var opts options
	flag.StringVar(&opts.mode, "mode", "build", "build the helper or finalize its manifest after signing")
	flag.StringVar(&opts.root, "root", ".", "repository root")
	flag.StringVar(&opts.out, "out", "", "installation directory, or .app bundle root on macOS")
	flag.StringVar(&opts.system, "os", runtime.GOOS, "target OS: darwin, linux, windows")
	flag.StringVar(&opts.architecture, "arch", runtime.GOARCH, "target Go architecture: amd64 or arm64")
	flag.StringVar(&opts.identity, "identity", "-", "macOS codesign identity; default is local ad-hoc signing")
	flag.Parse()
	if err := stage(opts); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func stage(opts options) error {
	if opts.out == "" || (opts.architecture != "amd64" && opts.architecture != "arm64") || (opts.mode != "build" && opts.mode != "finalize") {
		return errors.New("invalid HEIC package target/output/mode")
	}
	root, err := filepath.Abs(opts.root)
	if err != nil {
		return err
	}
	destination, err := filepath.Abs(opts.out)
	if err != nil {
		return err
	}
	executable, manifest, err := client.PackagePaths(destination, opts.system)
	if err != nil {
		return err
	}
	if opts.mode == "build" {
		if opts.system == "darwin" && runtime.GOOS != "darwin" {
			return errors.New("macOS HEIC packaging requires a native macOS host")
		}
		if err = os.MkdirAll(filepath.Dir(executable), 0755); err != nil {
			return err
		}
		command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-tags", "no_emoji,nodynamic", "-ldflags=-s -w -buildid=", "-o", executable, "./cmd/picfetch-heic-worker")
		command.Dir = root
		cgo := "0"
		if opts.system == "darwin" {
			cgo = "1"
		}
		command.Env = buildEnvironment(map[string]string{"GOOS": opts.system, "GOARCH": opts.architecture, "CGO_ENABLED": cgo, "GOTOOLCHAIN": "local", "GOWORK": "off", "GOFLAGS": "", "GOEXPERIMENT": "", "GOAMD64": "v1", "GOARM64": "v8.0"})
		if output, buildErr := command.CombinedOutput(); buildErr != nil {
			return fmt.Errorf("build HEIC helper: %w: %s", buildErr, output)
		}
		if opts.system == "darwin" {
			bundle := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
			if err = copyFile(filepath.Join(root, "packaging", "heic", "macos.Info.plist"), filepath.Join(bundle, "Contents", "Info.plist")); err != nil {
				return err
			}
			command = exec.Command("/usr/bin/codesign", "--force", "--sign", opts.identity, "--options", "runtime", "--entitlements", filepath.Join(root, "packaging", "heic", "macos.entitlements.plist"), bundle)
			if output, signErr := command.CombinedOutput(); signErr != nil {
				return fmt.Errorf("sign HEIC helper: %w: %s", signErr, output)
			}
			command = exec.Command("/usr/bin/codesign", "--verify", "--strict", "--verbose=2", bundle)
			if output, verifyErr := command.CombinedOutput(); verifyErr != nil {
				return fmt.Errorf("verify HEIC helper signature: %w: %s", verifyErr, output)
			}
		}
	}
	if err = verifyTarget(executable, opts.system, opts.architecture); err != nil {
		return err
	}
	notices := filepath.Join(filepath.Dir(manifest), "notices")
	if err = os.MkdirAll(notices, 0755); err != nil {
		return err
	}
	for _, name := range []string{"h265-LICENSE", "Go-LICENSE", "Go-PATENTS", "wazero-LICENSE", "wazero-NOTICE"} {
		if err = copyFile(filepath.Join(root, "docs", "heic", "notices", name), filepath.Join(notices, name)); err != nil {
			return err
		}
	}
	executableDigest, err := fileDigest(executable)
	if err != nil {
		return err
	}
	guestDigest, err := fileDigest(filepath.Join(root, "internal", "heicdecode", "worker", "decoder.wasm"))
	if err != nil {
		return err
	}
	record := client.PackageManifest{Version: 1, GOOS: opts.system, GOARCH: opts.architecture, ExecutableSHA256: executableDigest, GuestSHA256: guestDigest}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(manifest, append(data, '\n'), 0644); err != nil {
		return err
	}
	_, _, err = client.LoadPackage(destination, opts.system, opts.architecture)
	return err
}

func buildEnvironment(overrides map[string]string) []string {
	var environment []string
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if _, overridden := overrides[strings.ToUpper(key)]; !overridden {
			environment = append(environment, value)
		}
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}

func copyFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0644)
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(file, 64*1024*1024+1))
	if err != nil {
		return "", err
	}
	if n == 0 || n > 64*1024*1024 {
		return "", errors.New("HEIC package input size is invalid")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func verifyTarget(path, system, architecture string) error {
	switch system {
	case "darwin":
		file, err := macho.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		expected := macho.CpuAmd64
		if architecture == "arm64" {
			expected = macho.CpuArm64
		}
		if file.Cpu == expected && file.Type == macho.TypeExec {
			return nil
		}
	case "linux":
		file, err := elf.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		expected := elf.EM_X86_64
		if architecture == "arm64" {
			expected = elf.EM_AARCH64
		}
		if file.Machine == expected && file.Class == elf.ELFCLASS64 && file.Type == elf.ET_EXEC {
			return nil
		}
	case "windows":
		file, err := pe.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		expected := uint16(pe.IMAGE_FILE_MACHINE_AMD64)
		if architecture == "arm64" {
			expected = pe.IMAGE_FILE_MACHINE_ARM64
		}
		if file.Machine == expected && file.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE != 0 && file.Characteristics&pe.IMAGE_FILE_DLL == 0 {
			return nil
		}
	}
	return errors.New("HEIC helper binary target does not match its package")
}
