package update

import (
	"strings"
	"testing"
)

func TestWindowsApplyScript(t *testing.T) {
	got := windowsApplyScript(`C:\App\picfetch.exe`, `C:\cache\picfetch.exe`, 4242)
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
