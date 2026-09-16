//go:build heicnative && windows && (amd64 || arm64)

package ui

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/distribution"
)

func verifyNativeHEICIdentity(t *testing.T) {
	t.Helper()
	token := windows.GetCurrentProcessToken()
	user, err := token.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	groups, err := token.GetTokenGroups()
	if err != nil {
		t.Fatal(err)
	}
	administrators, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range groups.AllGroups() {
		if windows.EqualSid(group.Sid, administrators) {
			t.Fatal("native HEIC qualification requires a standard user, including no filtered administrator membership")
		}
	}
	t.Logf("standard-user sid=%s", user.User.Sid.String())
	if os.Getenv("PICFETCH_HEIC_ACTIVATION_MSIX") != "1" {
		return
	}
	if !distribution.StoreManaged {
		t.Fatal("installed test-MSIX is not Store-managed")
	}
	procedure := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentPackageFullName")
	if err = procedure.Find(); err != nil {
		t.Fatal(err)
	}
	var count uint32
	result, _, _ := procedure.Call(uintptr(unsafe.Pointer(&count)), 0)
	if result != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) || count == 0 || count > 32768 {
		t.Fatalf("installed test-MSIX has no package identity: %d", result)
	}
	name := make([]uint16, count)
	result, _, _ = procedure.Call(uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&name[0])))
	if result != 0 {
		t.Fatalf("read package identity: %d", result)
	}
	t.Logf("installed MSIX identity=%s", windows.UTF16ToString(name))
}

func preserveNativeHEICInstallation(t *testing.T, helper string) func() {
	t.Helper()
	read := func() string {
		t.Helper()
		descriptor, err := windows.GetNamedSecurityInfo(helper, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		return descriptor.String()
	}
	before := read()
	return func() {
		if read() != before {
			t.Error("activation modified immutable installed helper permissions")
		}
	}
}

// The installed test package declares this probe as a separate application.
// Windows launches its installed executable with the package's real identity.
func TestNativeInstalledHEICActivation(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(executable)
	configBytes, err := os.ReadFile(filepath.Join(root, "heic-activation.json"))
	if err != nil {
		t.Fatal("installed test-MSIX configuration is required: ", err)
	}
	var config struct {
		Evidence, Commit, UserSID string
		SessionID                 uint32
	}
	if err = json.Unmarshal(configBytes, &config); err != nil || !filepath.IsAbs(config.Evidence) || config.Commit == "" || config.UserSID == "" || config.SessionID == 0 {
		t.Fatal("invalid installed test-MSIX evidence configuration")
	}
	output, err := os.OpenFile(config.Evidence+".log", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	// This process runs only this native fixture. Keep its log open through the
	// testing package's final PASS/FAIL output; process exit closes the handle.
	os.Stdout, os.Stderr = output, output
	log.SetOutput(output)
	t.Cleanup(func() {
		record := struct {
			Test, OS, Arch, Commit string
			Passed                 bool
		}{
			"TestNativeInstalledHEICActivation", runtime.GOOS, runtime.GOARCH, config.Commit, !t.Failed(),
		}
		data, encodeErr := json.MarshalIndent(record, "", "  ")
		if encodeErr != nil {
			t.Error(encodeErr)
			return
		}
		temporary := config.Evidence + ".pending"
		if writeErr := os.WriteFile(temporary, data, 0600); writeErr != nil {
			t.Error(writeErr)
			return
		}
		if renameErr := os.Rename(temporary, config.Evidence+".json"); renameErr != nil {
			t.Error(renameErr)
		}
	})
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || user.User.Sid.String() != config.UserSID {
		t.Fatal("installed activation did not use the provisioned standard account")
	}
	var sessionID uint32
	if err = windows.ProcessIdToSessionId(uint32(os.Getpid()), &sessionID); err != nil || sessionID != config.SessionID {
		t.Fatal("installed activation did not use the provisioned desktop session")
	}
	t.Logf("installed MSIX session=%d", sessionID)
	t.Setenv("PICFETCH_HEIC_ACTIVATION_CHILD", "1")
	t.Setenv("PICFETCH_HEIC_ACTIVATION_MSIX", "1")
	t.Setenv("PICFETCH_HEIC_ACTIVATION_FIXTURE", filepath.Join(root, "heic-activation-fixture.heic"))
	TestNativePackagedHEICActivation(t)
}
