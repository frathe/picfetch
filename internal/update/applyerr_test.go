package update

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func TestApplyError_UnwrapsAndFormats(t *testing.T) {
	inner := errors.New("boom")
	err := &ApplyError{Op: "copy", Path: `C:\App\picfetch.exe`, Err: inner}
	if !errors.Is(err, inner) {
		t.Errorf("ApplyError does not unwrap to its cause")
	}
	for _, want := range []string{"copy", `C:\App\picfetch.exe`, "boom"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Error() = %q, missing %q", err.Error(), want)
		}
	}
}

func TestClassifyApplyError_PermissionIsAccessDenied(t *testing.T) {
	err := &ApplyError{Op: "copy", Path: "p", Err: &fs.PathError{Op: "open", Path: "p", Err: fs.ErrPermission}}
	if got := ClassifyApplyError(err); got != ReasonAccessDenied {
		t.Errorf("ClassifyApplyError = %q, want %q", got, ReasonAccessDenied)
	}
}

func TestClassifyApplyError_NilAndUnknown(t *testing.T) {
	if got := ClassifyApplyError(nil); got != ReasonUnknown {
		t.Errorf("nil = %q, want %q", got, ReasonUnknown)
	}
	if got := ClassifyApplyError(errors.New("plain")); got != ReasonUnknown {
		t.Errorf("plain = %q, want %q", got, ReasonUnknown)
	}
}
