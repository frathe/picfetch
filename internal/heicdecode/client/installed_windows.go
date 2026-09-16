//go:build windows && (amd64 || arm64)

package client

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/heicdecode/winisolation"
)

func prepareInstalled(ctx context.Context, source string, digest [32]byte, root string) (string, func(), error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	if err := privateCacheDirectory(root); err != nil {
		return "", nil, err
	}
	rootHandle, err := pinCachePath(root, true)
	if err != nil {
		return "", nil, err
	}
	handles := []windows.Handle{rootHandle}
	var once sync.Once
	release := func() {
		once.Do(func() {
			for i := len(handles) - 1; i >= 0; i-- {
				_ = windows.CloseHandle(handles[i])
			}
		})
	}
	success := false
	defer func() {
		if !success {
			release()
		}
	}()
	locked, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	lock, err := lockCache(locked, filepath.Join(root, "publication.lock"))
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = windows.CloseHandle(lock) }()
	path, err := stageHelper(ctx, source, digest, root, func(path string) error {
		// Reject reparse points and hard links before changing a copy's permissions.
		file, err := pinCachePath(path, false)
		if err != nil {
			return err
		}
		defer func() { _ = windows.CloseHandle(file) }()
		return winisolation.PrepareExecutable(path)
	})
	if err != nil {
		return "", nil, err
	}
	for _, entry := range []struct {
		path      string
		directory bool
	}{{filepath.Dir(path), true}, {path, false}} {
		handle, pinErr := pinCachePath(entry.path, entry.directory)
		if pinErr != nil {
			return "", nil, pinErr
		}
		handles = append(handles, handle)
	}
	// Verify the published bytes while write/delete-denying leases are held.
	if err = verifyStaged(ctx, path, digest); err != nil {
		return "", nil, err
	}
	cleanObsoleteHelpers(root, filepath.Dir(path))
	success = true
	return path, release, nil
}

// Only this user and SYSTEM receive inherited access to the private cache.
// The helper receives its separate, noninherited read/execute grant later.
func privateCacheDirectory(root string) error {
	if !filepath.IsAbs(root) || filepath.Dir(root) == root {
		return ErrUnavailable
	}
	if err := os.MkdirAll(filepath.Dir(root), 0700); err != nil {
		return err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sid := user.User.Sid.String()
	descriptor, err := windows.SecurityDescriptorFromString("O:" + sid + "D:P(A;OICI;FA;;;" + sid + ")(A;OICI;FA;;;SY)")
	if err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return err
	}
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: descriptor}
	if err = windows.CreateDirectory(name, &attributes); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	handle, err := pinCachePath(root, true)
	if err != nil {
		return err
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	existing, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := existing.Owner()
	if err != nil {
		return err
	}
	if owner == nil || !windows.EqualSid(owner, user.User.Sid) {
		return fmt.Errorf("%w: private cache owner", ErrUnavailable)
	}
	acl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}

func pinCachePath(path string, directory bool) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	flags := uint32(windows.FILE_FLAG_OPEN_REPARSE_POINT)
	access, share := uint32(windows.GENERIC_READ), uint32(windows.FILE_SHARE_READ)
	if directory {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
		access = windows.FILE_READ_ATTRIBUTES | windows.READ_CONTROL | windows.WRITE_DAC
		share |= windows.FILE_SHARE_WRITE
	}
	handle, err := windows.CreateFile(name, access, share, nil, windows.OPEN_EXISTING, flags, 0)
	if err != nil {
		return 0, err
	}
	var info windows.ByHandleFileInformation
	err = windows.GetFileInformationByHandle(handle, &info)
	if err == nil && (info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || (info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0) != directory || (!directory && info.NumberOfLinks != 1)) {
		err = fmt.Errorf("%w: private cache entry identity", ErrUnavailable)
	}
	if err != nil {
		_ = windows.CloseHandle(handle)
		return 0, err
	}
	return handle, nil
}

func lockCache(ctx context.Context, path string) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	for {
		if err = ctx.Err(); err != nil {
			return 0, err
		}
		handle, openErr := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if openErr == nil {
			var info windows.ByHandleFileInformation
			if err = windows.GetFileInformationByHandle(handle, &info); err == nil && (info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 || info.NumberOfLinks != 1) {
				err = ErrUnavailable
			}
			if err != nil {
				_ = windows.CloseHandle(handle)
				return 0, err
			}
			return handle, nil
		}
		if !errors.Is(openErr, windows.ERROR_SHARING_VIOLATION) {
			return 0, openErr
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
}

func cleanObsoleteHelpers(root, current string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		if path == current || !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		arch, digest, ok := strings.Cut(entry.Name(), "-")
		decoded, err := hex.DecodeString(digest)
		if !ok || (arch != "amd64" && arch != "arm64") || err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != digest {
			continue
		}
		// An active client holds a write/delete-denying file lease. A failed removal
		// is safe to defer until a later startup; never recursively force cleanup.
		_ = removeStaged(path)
	}
}
