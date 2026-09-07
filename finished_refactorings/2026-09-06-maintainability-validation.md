# PicFetch audit validation — 2026-09-06

Companion evidence for [the canonical backlog](../needs_refactoring.md) and [implementation plan](2026-09-06-maintainability-plan.md). Production baseline: `2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f`. No audit probe was added to production or the maintained test suite.

## Completed checks

| Check | Result and meaning |
| --- | --- |
| Initial checkout | Clean isolated worktree, detached audited HEAD. Saved main checkout had unrelated XRandR work excluded from scope. |
| `make verify` | Exit 0. Format, offline TUF, Qodana exclusions, vet, build and Linux/amd64 Docker race suite passed. Shard manifest checked 617 internal/ui runnables. No golden regeneration. |
| Native focused package/script suite | Go 1.27.1 darwin/arm64: all 14 named packages and six script packages passed. Command below. No native desktop interaction. |
| UI/deletion overlays | Expected failure exit 1. Comparison cache lacks EXIF/size, queued animation schedules old delay, deletion removes a different list index after prompt-time reorder. These are baseline defect demonstrations, not failed implementation checks. |
| Imaging/mosaic/preview overlays | Fresh `-count=1` run completed. Some probes assert the defect and others log measured values; PASS means the observation completed, not that the production defect is fixed. |
| Clipboard failure subprocess | Exact extracted function under RLIMIT_FSIZE=1 returned empty path and nil error. Captured through a pipe so the process limit did not truncate stdout evidence. No clipboard command invoked. |
| Native Windows / actual GL / huge allocations | Not run. Platform consequences and allocation geometry are distinguished from runtime tests in each finding. |
| Payload and documentation checks | Links/anchors, stable-ID counts and plan coverage checked locally; only documentation staged. Exact commit and remote metadata are delivered separately so this document does not create a self-referential commit hash. |

Native command:

```sh
go test ./internal/clipboard ./internal/filepicker ./internal/filemanager ./internal/trash ./internal/wallpaper ./internal/displays ./internal/openwith ./internal/winpos ./internal/wincom ./internal/wingesture ./internal/appearance ./internal/launch ./internal/distribution ./internal/update ./scripts/...
```

Go cache permissions and local httptest port binding initially failed in the sandbox. Approved re-execution succeeded. These environment failures are not product findings. The native focused suite did not run main's macOS graft test; that omission remains explicit in MA-017. A duplicate-library linker warning appeared on macOS; it did not fail verification.

## Observed evidence

```text
MA-001: metadata, orientation, nextIFD each panic:
        slice bounds out of range [4294967295:1]
MA-002: 1920x1080 target, default settings, seed 42:
        aspect 100:   47642x485 layer,   92425480 bytes
        aspect 1000:  476339x485 layer,  924097660 bytes
        aspect 10000: 4763310x485 layer, 9240821400 bytes
        (geometry only; allocator never called)
MA-003: queued animation requests next delay=1s before callback advances to 2s frame
MA-004: prompt A at 0; reorder B,A; confirm: A removed on temp disk, list retains A
MA-006: GPS JPEG cached by comparison: HasEXIF=false, FileSize=0, expected size=739
MA-007: SaveRotated JPEG header=60x90; IFD0 width=90, height=60
MA-008: unrelated hashes, native non-race:
        10000: 35.309667ms; 20000: 140.824709ms; 50000: 857.155583ms
MA-011: writeTempPNG under RLIMIT_FSIZE=1 returned path="" err=<nil>
MA-018: different same-size bytes + different subsecond mtime share cache key=true
```

The UI animation probe deliberately holds the callback and observes the worker scheduling again. It is an audit demonstration of the current implementation, not a ready-to-land regression: a fixed implementation should instead be tested for correct sequence/acknowledgement without expecting that premature scheduling. The deletion probe uses the existing test helper that removes only files created under t.TempDir; it never calls real OS Trash.

## Reproduce with temporary Go overlays

The following fenced probes are a durable copy of the audit inputs. They depend on helpers in the pinned baseline. Run from that repository revision using a suitable Go/C/GL toolchain. This procedure writes only to a temporary directory and builds an overlay; no source file is changed. Appended snippets reuse the existing test file's imports/helpers. New virtual test files exist only through the overlay, so the maintained Qodana/shard manifests are unaffected.

```sh
python3 - <<'PYCODE'
import json, pathlib, re, tempfile
root = pathlib.Path.cwd()
text = (root / 'plans/2026-09-06-maintainability-validation.md').read_text()
out = pathlib.Path(tempfile.mkdtemp(prefix='picfetch-audit-overlay-'))
replace = {}
pattern = r'<!-- probe: (\S+) (append|replace) -->\n```go\n(.*?)\n```'
for i, (target, mode, source) in enumerate(re.findall(pattern, text, re.S)):
    original = root / target
    merged = (original.read_text() + '\n' + source) if mode == 'append' else source
    output = out / ('probe_%d_test.go' % i)
    output.write_text(merged + '\n')
    replace[str(original)] = str(output)
assert len(replace) == 5, 'Expected all five probe blocks'
(out / 'overlay.json').write_text(json.dumps({'Replace': replace}))
print(out / 'overlay.json')
PYCODE
# Substitute the printed temporary overlay path below.
go test -overlay=/tmp/PRINTED-DIRECTORY/overlay.json ./internal/imaging ./internal/mosaic ./internal/favthumbs -run '^TestAudit' -count=1 -v
go test -overlay=/tmp/PRINTED-DIRECTORY/overlay.json ./internal/ui ./internal/ui/deletion -run '^TestAudit' -count=1
```

The second Go command is expected to fail on the audited baseline. Use the task's approved runtime/cache access if sandbox restrictions prevent compilation or local test fixtures. Do not convert those restrictions into product findings.

<!-- probe: internal/imaging/audit_repro_test.go replace -->
```go
package imaging

import (
	"encoding/binary"
	"fmt"
	"fyne.io/fyne/v2/storage"
	"os"
	"testing"
	"time"
)

func TestAuditOverflow(t *testing.T) {
	data := []byte{'I', 'I', 42, 0, 255, 255, 255, 255}
	cases := map[string]func(){"metadata": func() { ReadMetadata(data) }, "orientation": func() { parseExifOrientation(append([]byte("Exif\x00\x00"), data...)) }, "nextIFD": func() { nextIFDOffset(data, binary.LittleEndian, ^uint32(0)) }}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("REPRODUCED panic: %v", r)
				} else {
					t.Fatal("expected reproduced panic")
				}
			}()
			f()
		})
	}
}
func TestAuditDuplicateScale(t *testing.T) {
	for _, n := range []int{10000, 20000, 50000} {
		hs := make([]uint64, n)
		x := uint64(1)
		for i := range hs {
			x ^= x << 13
			x ^= x >> 7
			x ^= x << 17
			hs[i] = x
		}
		start := time.Now()
		g := DuplicateGroups(hs, 6)
		t.Log(fmt.Sprintf("n=%d groups=%d duration=%v", n, len(g), time.Since(start)))
	}
}

func TestAuditSaveRotatedDimensions(t *testing.T) {
	path := writeTempFile(t, "camera.jpg", dimensionTagJPEG(t, 90, 60))
	if err := SaveRotated(storage.NewFileURI(path), markedImage(60, 90)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ifds := readIFDs(t, data)
	w, h, ok := jpegFrameSize(data)
	t.Logf("header=%dx%d valid=%v IFD0 width=%d height=%d", w, h, ok, binary.LittleEndian.Uint32(ifds[tiffIFD0][0x0100]), binary.LittleEndian.Uint32(ifds[tiffIFD0][0x0101]))
}
```

<!-- probe: internal/mosaic/audit_repro_test.go replace -->
```go
package mosaic

import (
	"context"
	"errors"
	"image"
	"math"
	"testing"
)

func TestAuditAspectAllocation(t *testing.T) {
	stop := errors.New("stop before allocation")
	for _, aspect := range []float64{100, 1000, 10000} {
		_, err := walkLayout(context.Background(), image.Pt(1920, 1080), DefaultSettings(), 42, func() (candidate, error) { return candidate{id: 1, aspect: aspect}, nil }, func(p placement) error {
			w, h := int64(math.Ceil(p.imageRect.width*2))+8, int64(math.Ceil(p.imageRect.height*2))+8
			t.Logf("source aspect=%v first prepared layer=%dx%d bytes=%d", aspect, w, h, w*h*4)
			return stop
		})
		if !errors.Is(err, stop) {
			t.Fatal(err)
		}
	}
}
```

<!-- probe: internal/favthumbs/audit_repro_test.go replace -->
```go
package favthumbs

import (
	"fyne.io/fyne/v2/storage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuditSubsecondVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	a := time.Unix(1700000000, 100000000)
	if err := os.Chtimes(path, a, a); err != nil {
		t.Fatal(err)
	}
	before, ok := EntryName(storage.NewFileURI(path))
	if !ok {
		t.Fatal("stat")
	}
	if err := os.WriteFile(path, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	b := time.Unix(1700000000, 900000000)
	if err := os.Chtimes(path, b, b); err != nil {
		t.Fatal(err)
	}
	after, ok := EntryName(storage.NewFileURI(path))
	if !ok {
		t.Fatal("stat")
	}
	t.Logf("different bytes+nanosecond mtime share cache key=%v", before == after)
}
```

<!-- probe: internal/ui/compare_fidelity_test.go append -->
```go
func TestAuditCompareCacheMetadata(t *testing.T) {
	v := newTestViewer(t)
	uri := uitest.TempGPSJPEGURI(t, "metadata.jpg", 20, 15, 50.0, 4.0)
	data, _, err := imaging.ReadAndProbe(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := v.loadComparedImage(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	cached, ok := v.imgCache.Get(uri.String())
	if !ok || cached != loaded {
		t.Fatal("comparison did not seed shared cache")
	}
	if !loaded.HasEXIF {
		t.Error("comparison cache HasEXIF=false for GPS JPEG")
	}
	if loaded.FileSize != int64(len(data)) {
		t.Errorf("comparison cache FileSize=%d, want %d", loaded.FileSize, len(data))
	}
}

type auditQueueDriver struct {
	fyne.Driver
	calls chan func()
}

func (d *auditQueueDriver) DoFromGoroutine(f func(), wait bool) {
	if wait {
		f()
		return
	}
	d.calls <- f
}

type auditQueueApp struct {
	fyne.App
	driver fyne.Driver
}

func (a auditQueueApp) Driver() fyne.Driver { return a.driver }
func TestAuditAnimationQueuesBeforeFrameApplied(t *testing.T) {
	v := newTestViewer(t)
	app := fyne.CurrentApp()
	q := &auditQueueDriver{Driver: app.Driver(), calls: make(chan func(), 4)}
	fyne.SetCurrentApp(auditQueueApp{App: app, driver: q})
	defer fyne.SetCurrentApp(app)
	requested := make(chan time.Duration, 4)
	ticks := make(chan time.Time)
	v.frameAfter = func(d time.Duration) <-chan time.Time { requested <- d; return ticks }
	token := v.loadLifecycle.begin()
	stopped := make(chan struct{})
	go v.animate(token, []image.Image{image.NewRGBA(image.Rect(0, 0, 1, 1)), image.NewRGBA(image.Rect(0, 0, 1, 1))}, []time.Duration{time.Second, 2 * time.Second}, func() { close(stopped) })
	<-requested
	ticks <- time.Time{}
	apply := <-q.calls
	var d time.Duration
	select {
	case d = <-requested:
	case <-time.After(testTimeout):
		t.Fatal("expected current implementation to advance before queued callback")
	}
	token.cancelContext()
	<-stopped
	apply()
	t.Fatalf("next delay requested before frame application: got %v (old frame delay), want to wait for queued frame", d)
}
```

<!-- probe: internal/ui/deletion/batch_test.go append -->
```go
func TestAuditDeleteReorderWhilePromptOpen(t *testing.T) {
	stubTrashMove(t)
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg"), gen: 1}
	a, b := host.files[0], host.files[1]
	c := New(host)
	c.RequestFiles(targetsFor(host, 0))
	host.files = []fyne.URI{b, a}
	host.gen++
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()
	if _, err := os.Stat(a.Path()); !os.IsNotExist(err) {
		t.Fatal("expected selected A moved")
	}
	if len(host.files) != 1 || host.files[0].String() != b.String() {
		t.Errorf("remaining files=%v, want only existing B %s", host.files, b)
	}
}
```

## Clipboard fault probe

Save the following as a temporary Go file and run it on a POSIX host supporting RLIMIT_FSIZE. It reproduces the exact private helper rather than reaching a real clipboard adapter. The maintained fix should test the package function via an appropriate narrow seam or isolated package subprocess.

Pipe stdout to an unrestricted parent before writing a log, for example `go run /tmp/clipboard-probe.go | cat > /tmp/clipboard-probe.log`. Redirecting the limited process straight to a regular log file truncates the evidence at one byte. The process limit affects only the probe process; its temp file is removed by the function.

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func writeTempPNG(data []byte) (string, error) {
	f, err := os.CreateTemp("", "picfetch_clip_*.png")
	if err != nil {
		return "", err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		err := os.Remove(f.Name())
		if err != nil {
			return "", err
		}
		return "", err
	}
	if err := f.Close(); err != nil {
		err := os.Remove(f.Name())
		if err != nil {
			return "", err
		}
		return "", err
	}
	return f.Name(), nil
}

func main() {
	signal.Ignore(syscall.SIGXFSZ)
	err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 1, Max: 1})
	if err != nil {
		panic(err)
	}
	path, err := writeTempPNG([]byte("longer than one byte"))
	fmt.Printf("writeTempPNG under RLIMIT_FSIZE=1 returned path=%q err=%v\n", path, err)
}
```
