// Package macbundle owns the Apple Store application's installed native layout.
package macbundle

import (
	"context"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RuntimeDirectory resolves the shared Frameworks directory from a known app
// executable. Cache paths and arbitrary ancestor searches are never considered.
func RuntimeDirectory(executable string) (string, error) {
	if !filepath.IsAbs(executable) || filepath.Clean(executable) != executable {
		return "", errors.New("unrecognized app executable")
	}
	directory := filepath.Dir(executable)
	if filepath.Base(directory) != "MacOS" || filepath.Base(filepath.Dir(directory)) != "Contents" {
		return "", errors.New("unrecognized app executable")
	}
	container := filepath.Dir(filepath.Dir(directory))
	if filepath.Base(executable) == "picfetch-image-worker" && filepath.Base(container) == "io.github.frathe.picfetch.worker.xpc" {
		services := filepath.Dir(container)
		if filepath.Base(services) != "XPCServices" || filepath.Base(filepath.Dir(services)) != "Contents" {
			return "", errors.New("unrecognized XPC app layout")
		}
		container = filepath.Dir(filepath.Dir(services))
	} else if filepath.Base(executable) != "PicFetch" && filepath.Base(executable) != "picfetch" {
		return "", errors.New("unrecognized app executable")
	}
	if !strings.HasSuffix(container, ".app") {
		return "", errors.New("runtime requires an application bundle")
	}
	return filepath.Join(container, "Contents", "Frameworks"), nil
}

// CurrentRuntimeDirectory uses the native executable identity, independent of
// argv spelling and the sandbox's replacement working directory.
func CurrentRuntimeDirectory() (string, error) {
	executable, err := executablePath()
	if err != nil {
		return "", err
	}
	return RuntimeDirectory(executable)
}

// VerifyLibrary checks the selected architecture and both native code/resource
// seals. Ad-hoc signatures are usable for local qualification; distribution trust
// and provisioning are separately enforced by the packaging/submission route.
func VerifyLibrary(ctx context.Context, root, library, arch string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(root) || filepath.Base(root) != "Frameworks" || filepath.Base(filepath.Dir(root)) != "Contents" || !strings.HasSuffix(filepath.Dir(filepath.Dir(root)), ".app") || filepath.Dir(library) != root {
		return errors.New("runtime must be directly inside app Frameworks")
	}
	canonical, err := filepath.EvalSymlinks(library)
	if err != nil {
		return err
	}
	expected, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	app, err := filepath.EvalSymlinks(filepath.Dir(filepath.Dir(root)))
	if err != nil {
		return err
	}
	if expected != filepath.Join(app, "Contents", "Frameworks") {
		return errors.New("runtime directory escapes its bundle")
	}
	info, err := os.Lstat(library)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || filepath.Dir(canonical) != expected {
		return errors.New("runtime library escapes its bundle")
	}
	cpu, ok := map[string]macho.Cpu{"arm64": macho.CpuArm64, "amd64": macho.CpuAmd64}[arch]
	if !ok {
		return fmt.Errorf("unsupported runtime architecture %q", arch)
	}
	file, err := macho.Open(library)
	if err != nil {
		return err
	}
	valid := file.Cpu == cpu && file.Type == macho.TypeDylib
	_ = file.Close()
	if !valid {
		return errors.New("runtime library has the wrong architecture or Mach-O type")
	}
	if err := verifyNative(filepath.Dir(filepath.Dir(root)), library); err != nil {
		return err
	}
	return ctx.Err()
}
