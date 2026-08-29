package update

import (
	"strings"
	"testing"
)

func TestWindowsApplyScript(t *testing.T) {
	got := windowsApplyScript(`C:\App\picfetch.exe`, `C:\cache\picfetch.exe`, 4242, ApplyOptions{})
	for _, want := range []string{
		`tasklist /FI "PID eq 4242"`,
		`copy /Y`,
		`C:\cache\picfetch.exe`,
		`C:\App\picfetch.exe`,
		`del "%~f0"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("script missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, `del "C:\cache\picfetch.exe"`) && !strings.Contains(got, `del C:\cache\picfetch.exe`) {
		t.Errorf("script does not delete staged file:\n%s", got)
	}
	if strings.Contains(strings.ToLower(got), "start ") {
		t.Errorf("script must not relaunch picfetch:\n%s", got)
	}
}

func TestWindowsApplyScript_RelaunchesAfterSuccessfulCopyAndStageDelete(t *testing.T) {
	got := windowsApplyScript(`C:\App & Tools\picfetch.exe`, `C:\cache\100% ready\picfetch.exe`, 4242, ApplyOptions{Relaunch: true})
	wantsInOrder := []string{
		`copy /Y "C:\cache\100%% ready\picfetch.exe" "C:\App & Tools\picfetch.exe.new" >NUL`,
		`if errorlevel 1 (`,
		`>>"%TEMP%\picfetch-update.log" echo PicFetch update copy failed.`,
		`goto cleanup`,
		`move /Y "C:\App & Tools\picfetch.exe" "C:\App & Tools\picfetch.exe.old" >NUL`,
		`move /Y "C:\App & Tools\picfetch.exe.new" "C:\App & Tools\picfetch.exe" >NUL`,
		`move /Y "C:\App & Tools\picfetch.exe.old" "C:\App & Tools\picfetch.exe" >NUL`,
		`>>"%TEMP%\picfetch-update.log" echo PicFetch update install failed.`,
		`del "C:\App & Tools\picfetch.exe.old"`,
		`del "C:\cache\100%% ready\picfetch.exe"`,
		`start "" "C:\App & Tools\picfetch.exe"`,
		`if errorlevel 1 >>"%TEMP%\picfetch-update.log" echo PicFetch update relaunch failed.`,
		`:cleanup`,
		`del "C:\App & Tools\picfetch.exe.new" 2>NUL`,
		`del "%~f0"`,
	}
	last := -1
	for _, want := range wantsInOrder {
		at := strings.Index(got, want)
		if at < 0 {
			t.Fatalf("script missing %q:\n%s", want, got)
		}
		if at <= last {
			t.Fatalf("%q appears out of order:\n%s", want, got)
		}
		last = at
	}
	if strings.Contains(got, `copy /Y "C:\cache\100%% ready\picfetch.exe" "C:\App & Tools\picfetch.exe"`) {
		t.Fatalf("script overwrites the installed executable directly:\n%s", got)
	}
}
