package analysiscache_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/analysiscache"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMain(m *testing.M) { test.NewApp(); os.Exit(m.Run()) }

func TestAnalysisCacheManagementProgressCoalescesBeforeDelivery(t *testing.T) {
	q := &uitest.UIQueue{}
	ready, release := make(chan struct{}), make(chan struct{})
	provider := maintenance{inspect: func(_ context.Context, _ similarity.CacheRoots, progress func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
		for i := 1; i <= 20000; i++ {
			progress(similarity.CacheProgress{Records: i})
		}
		close(ready)
		<-release
		return similarity.CacheUsage{}, nil
	}}
	f := analysiscache.New(&cacheHost{}, analysiscache.Options{Provider: provider, Queue: q})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		f.Stop()
		f.Settle()
	})
	content := f.Content(true, 2048)
	<-ready
	if got := q.Len(); got != 1 {
		t.Fatalf("inspection queued %d transient updates, want one latest delivery", got)
	}
	q.Drain()
	if !strings.Contains(labels(content), "Processed 20000 records") {
		t.Fatal("coalescing lost the latest progress")
	}
	close(release)
	f.Settle()
	if f.Busy() {
		t.Fatal("coalescing lost final completion")
	}
}

type policy struct {
	enabled bool
	limit   int
}
type cacheHost struct {
	policies  []policy
	quiesced  int
	reasons   []analysiscache.QuiesceReason
	barriers  []<-chan struct{}
	onQuiesce func()
}

func (h *cacheHost) Quiesce(reason analysiscache.QuiesceReason) []<-chan struct{} {
	h.quiesced++
	h.reasons = append(h.reasons, reason)
	if h.onQuiesce != nil {
		h.onQuiesce()
	}
	return h.barriers
}
func (h *cacheHost) ApplyPolicy(enabled bool, limit int) {
	h.policies = append(h.policies, policy{enabled, limit})
}

type maintenance struct {
	inspect func(context.Context, similarity.CacheRoots, func(similarity.CacheProgress)) (similarity.CacheUsage, error)
	clean   func(context.Context, similarity.CacheCleanRequest, func(similarity.CacheProgress)) (similarity.CacheReport, error)
	retune  func(context.Context, similarity.CacheRetuneRequest, func(similarity.CacheProgress)) (similarity.CacheReport, error)
}

func (m maintenance) Inspect(ctx context.Context, roots similarity.CacheRoots, progress func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
	if m.inspect == nil {
		return similarity.CacheUsage{}, nil
	}
	return m.inspect(ctx, roots, progress)
}
func (m maintenance) Clean(ctx context.Context, request similarity.CacheCleanRequest, progress func(similarity.CacheProgress)) (similarity.CacheReport, error) {
	return m.clean(ctx, request, progress)
}
func (m maintenance) Retune(ctx context.Context, request similarity.CacheRetuneRequest, progress func(similarity.CacheProgress)) (similarity.CacheReport, error) {
	return m.retune(ctx, request, progress)
}

func descendants(root fyne.CanvasObject) []fyne.CanvasObject {
	objects := []fyne.CanvasObject{root}
	if c, ok := root.(*fyne.Container); ok {
		for _, child := range c.Objects {
			objects = append(objects, descendants(child)...)
		}
	}
	return objects
}
func labels(root fyne.CanvasObject) string {
	var text []string
	for _, object := range descendants(root) {
		if label, ok := object.(*widget.Label); ok {
			text = append(text, label.Text)
		}
	}
	return strings.Join(text, "\n")
}

func button(t *testing.T, root fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	for _, object := range descendants(root) {
		if b, ok := object.(*widget.Button); ok && b.Text == text {
			return b
		}
	}
	t.Fatalf("actual cache content missing button %q", text)
	return nil
}

func TestAnalysisCacheManagementUsageMeasuresDisabledStoresInActualContent(t *testing.T) {
	h := &cacheHost{}
	provider := maintenance{inspect: func(_ context.Context, _ similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
		return similarity.CacheUsage{General: similarity.CacheSize{Bytes: 3 * 1024 * 1024, Records: 4}, Favorite: similarity.CacheSize{Bytes: 2 * 1024 * 1024, Records: 2}, Incomplete: true}, nil
	}}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	content := f.Content(false, 2048)
	f.Settle()
	for _, want := range []string{"General: 3.0 MB, 4 records", "Favorites: 2.0 MB, 2 records", "Combined: 5.0 MB", "Usage is incomplete."} {
		if !strings.Contains(labels(content), want) {
			t.Fatalf("actual cache content missing %q: %s", want, labels(content))
		}
	}
	var entry *widget.Entry
	var check *widget.Check
	for _, object := range descendants(content) {
		switch object := object.(type) {
		case *widget.Entry:
			entry = object
		case *widget.Check:
			check = object
		}
	}
	if entry == nil || entry.Text != "2048" || check == nil || check.Checked {
		t.Fatal("standing settings not seeded in the actual content")
	}
	if h.quiesced != 0 || len(h.policies) != 0 {
		t.Fatal("read-only inspection retired producers or changed preferences")
	}
}

func TestAnalysisCacheManagementLimitValidatesAndPersistsOnlySuccessfulRetune(t *testing.T) {
	h := &cacheHost{}
	var requests []similarity.CacheRetuneRequest
	provider := maintenance{retune: func(_ context.Context, request similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
		requests = append(requests, request)
		if request.LimitBytes == 64*1024*1024 {
			return similarity.CacheReport{Failures: 1}, errors.New("disk is read only")
		}
		return similarity.CacheReport{AppliedLimit: request.LimitBytes}, nil
	}}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	content := f.Content(true, 2048)
	f.Settle()
	var entry *widget.Entry
	for _, object := range descendants(content) {
		if e, ok := object.(*widget.Entry); ok {
			entry = e
		}
	}
	if entry == nil {
		t.Fatal("cache limit entry is missing from the tab")
	}
	for _, invalid := range []string{"", "0", "-1", "1.5", "18446744073709551616", "17592186044416"} {
		entry.SetText(invalid)
		f.Settle()
		if len(h.policies) != 0 || len(requests) != 0 {
			t.Fatalf("invalid limit %q changed persisted policy", invalid)
		}
	}
	for _, prefix := range []string{"5", "50", "500"} {
		entry.SetText(prefix)
		f.Settle()
		if len(h.policies) != 0 || len(requests) != 0 {
			t.Fatal("typing a limit evicted records before the edit was submitted")
		}
	}
	entry.SetText("128")
	entry.OnSubmitted(entry.Text)
	f.Settle()
	if len(h.policies) != 1 || h.policies[0] != (policy{true, 128}) {
		t.Fatalf("valid limit did not persist after retune: %v", h.policies)
	}
	entry.SetText("64")
	entry.OnSubmitted(entry.Text)
	f.Settle()
	if len(h.policies) != 1 {
		t.Fatal("failed retune changed persisted policy")
	}
	if !strings.Contains(labels(content), "Maintenance is incomplete.") {
		t.Fatalf("failed shrink was presented as complete: %s", labels(content))
	}
	entry.SetText("256")
	test.Tap(button(t, content, "Apply cache limit"))
	f.Settle()
	if len(h.policies) != 2 || h.policies[1].limit != 256 {
		t.Fatalf("subsequent valid limit was not accepted: %v", h.policies)
	}
	if !requests[0].RetireWriters || !requests[1].RetireWriters || requests[2].RetireWriters {
		t.Fatalf("decreases did not retire old-budget writers independently of increases: %+v", requests)
	}
}

func TestAnalysisCacheManagementLimitJoinsProducersBeforeInspection(t *testing.T) {
	for _, tc := range []struct {
		limit            int
		cancel           bool
		cancelBeforeScan bool
	}{{1024, false, false}, {2048, false, false}, {4096, false, false}, {4096, true, false}, {4096, true, true}} {
		t.Run(fmt.Sprintf("limit=%d/cancel=%v/beforeScan=%v", tc.limit, tc.cancel, tc.cancelBeforeScan), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				first, second, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
				started := make(chan similarity.CacheRetuneRequest, 1)
				h := &cacheHost{barriers: []<-chan struct{}{first, second}}
				provider := maintenance{retune: func(ctx context.Context, request similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
					started <- request
					select {
					case <-release:
						return similarity.CacheReport{AppliedLimit: request.LimitBytes}, nil
					case <-ctx.Done():
						return similarity.CacheReport{}, ctx.Err()
					}
				}}
				f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
				t.Cleanup(func() {
					for _, done := range []chan struct{}{first, second, release} {
						select {
						case <-done:
						default:
							close(done)
						}
					}
					f.Stop()
					f.Settle()
				})
				f.Content(true, 2048)
				f.Settle()
				f.Retune(tc.limit)
				changed := tc.limit != 2048
				wantRetirements := 0
				if changed {
					wantRetirements = 1
				}
				if h.quiesced != wantRetirements {
					t.Fatalf("limit change did not retire captured producers before inspection: quiesced=%d", h.quiesced)
				}
				if tc.cancelBeforeScan {
					f.Close()
				}
				if changed {
					for _, done := range []chan struct{}{first, second} {
						synctest.Wait()
						if !f.Busy() {
							t.Fatal("maintenance stopped tracking producer retirement")
						}
						select {
						case <-started:
							t.Fatal("maintenance inspected before both producers stopped")
						default:
						}
						close(done)
					}
				}
				if tc.cancelBeforeScan {
					f.Settle()
					if f.Busy() || len(started) != 0 || len(h.policies) != 0 {
						t.Fatal("canceled retirement inspected or applied an unaccepted limit")
					}
					return
				}
				request := <-started
				if request.RetireWriters != (tc.limit < 2048) || len(h.policies) != 0 {
					t.Fatal("local retirement changed shared lease policy or committed an uninspected limit")
				}
				if tc.cancel {
					f.Close()
				} else {
					close(release)
				}
				f.Settle()
				wantPolicies := 1
				if tc.cancel {
					wantPolicies = 0
				}
				if f.Busy() || len(h.policies) != wantPolicies {
					t.Fatalf("retune completion lost policy/cancellation ownership: %+v", h.policies)
				}
				if !tc.cancel && h.policies[0] != (policy{true, tc.limit}) {
					t.Fatalf("accepted wrong policy: %+v", h.policies)
				}
			})
		})
	}
}

func TestAnalysisCacheManagementOptOutSurvivesInspectionFailureAndClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first, second := make(chan struct{}), make(chan struct{})
		h := &cacheHost{barriers: []<-chan struct{}{first, second}}
		provider := maintenance{
			inspect: func(_ context.Context, _ similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
				return similarity.CacheUsage{Incomplete: true}, errors.New("favorite is unreadable")
			},
			retune: func(_ context.Context, _ similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
				return similarity.CacheReport{Remaining: similarity.CacheUsage{Incomplete: true}}, errors.New("favorite is unreadable")
			},
		}
		q := &uitest.UIQueue{}
		f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: q})
		t.Cleanup(func() {
			for _, done := range []chan struct{}{first, second} {
				select {
				case <-done:
				default:
					close(done)
				}
			}
			f.Stop()
			f.Settle()
		})
		content := f.Content(true, 2048)
		f.Settle()
		var toggle *widget.Check
		for _, object := range descendants(content) {
			if check, ok := object.(*widget.Check); ok {
				toggle = check
			}
		}
		if toggle == nil {
			t.Fatal("loose persistence control is missing")
		}
		toggle.SetChecked(false)
		if len(h.policies) != 1 || h.policies[0] != (policy{false, 2048}) || h.quiesced != 1 {
			t.Fatalf("opt-out depended on successful inspection: policies=%v quiesced=%d", h.policies, h.quiesced)
		}
		f.Close()
		synctest.Wait()
		q.Drain()
		if !f.Busy() {
			t.Fatal("closing Settings stopped tracking producer retirement")
		}
		close(first)
		synctest.Wait()
		q.Drain()
		if !f.Busy() {
			t.Fatal("retirement completed before the second producer stopped")
		}
		close(second)
		f.Settle()
		if len(h.policies) != 1 || toggle.Checked {
			t.Fatal("closing Settings lost the committed opt-out")
		}
	})
}

func TestAnalysisCacheManagementClearConfirmationProgressCancellationAndStaleReport(t *testing.T) {
	h := &cacheHost{}
	q := &uitest.UIQueue{}
	var confirm func(bool)
	started := make(chan struct{})
	provider := maintenance{clean: func(ctx context.Context, request similarity.CacheCleanRequest, progress func(similarity.CacheProgress)) (similarity.CacheReport, error) {
		if request.Mode == similarity.RemoveStale {
			return similarity.CacheReport{Skipped: 2, Unavailable: 1, Remaining: similarity.CacheUsage{General: similarity.CacheSize{Bytes: 1024 * 1024, Records: 2}}}, nil
		}
		progress(similarity.CacheProgress{Records: 1, Bytes: 1024 * 1024})
		close(started)
		<-ctx.Done()
		return similarity.CacheReport{Canceled: true, RemovedRecords: 1, RemovedBytes: 1024 * 1024, Failures: 1, Skipped: 2, Remaining: similarity.CacheUsage{Incomplete: true, General: similarity.CacheSize{Bytes: 2 * 1024 * 1024, Records: 3}}}, ctx.Err()
	}}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: q, ConfirmClear: func(callback func(bool)) { confirm = callback }})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	content := f.Content(true, 2048)
	f.Settle()
	clearButton := button(t, content, "Clear analysis cache")
	clearButton.OnTapped()
	if confirm == nil || f.Busy() {
		t.Fatal("clear started before explicit confirmation")
	}
	confirm(false)
	if f.Busy() {
		t.Fatal("rejected clear was admitted")
	}
	clearButton.OnTapped()
	confirm(true)
	<-started
	q.Drain()
	if !strings.Contains(labels(content), "Processed 1 records (1.0 MB).") {
		t.Fatalf("worker progress absent: %s", labels(content))
	}
	button(t, content, "Cancel").OnTapped()
	f.Settle()
	for _, want := range []string{"Removed 1 records (1.0 MB); remaining 3 records (2.0 MB).", "failures: 1", "Maintenance was canceled.", "Maintenance is incomplete."} {
		if !strings.Contains(labels(content), want) {
			t.Fatalf("partial cleanup missing %q: %s", want, labels(content))
		}
	}
	button(t, content, "Remove stale entries").OnTapped()
	f.Settle()
	if !strings.Contains(labels(content), "Skipped: 2; unavailable: 1; failures: 0.") {
		t.Fatalf("stale cleanup availability report missing: %s", labels(content))
	}
}

func TestAnalysisCacheManagementWritersSettleDrainsQuiescenceAndWaitsBothWriters(t *testing.T) {
	roots := similarity.CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()}
	records := filepath.Join(roots.GeneralDir, "v1")
	if err := os.Mkdir(records, 0700); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(records, strings.Repeat("a", 64)+".json")
	if err := os.WriteFile(record, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	first, second, requested := make(chan struct{}), make(chan struct{}), make(chan struct{})
	h := &cacheHost{barriers: []<-chan struct{}{first, second}, onQuiesce: func() { close(requested) }}
	f := analysiscache.New(h, analysiscache.Options{Roots: roots, Queue: &uitest.UIQueue{}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	f.Content(true, 2048)
	f.Settle()
	f.Retune(2048)
	f.Settle()
	if h.quiesced != 0 {
		t.Fatal("inspection or unchanged budget retired a producer")
	}
	checks := make(chan error, 1)
	go func() {
		<-requested
		_, before := os.Stat(record)
		close(first)
		_, afterFirst := os.Stat(record)
		close(second)
		checks <- errors.Join(before, afterFirst)
	}()
	f.Clean(similarity.ClearAll)
	f.Settle()
	if err := <-checks; err != nil {
		t.Fatalf("cleanup removed data before both writers stopped: %v", err)
	}
	if _, err := os.Stat(record); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup did not remove managed record after retirement: %v", err)
	}
	if h.quiesced != 1 || f.Busy() {
		t.Fatalf("maintenance did not finish observed retirement: quiesced=%d busy=%v", h.quiesced, f.Busy())
	}
}

func TestAnalysisCacheManagementQuiescenceReasons(t *testing.T) {
	roots := similarity.CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()}
	records := filepath.Join(roots.GeneralDir, "v1")
	if err := os.Mkdir(records, 0700); err != nil {
		t.Fatal(err)
	}
	h := &cacheHost{}
	f := analysiscache.New(h, analysiscache.Options{Roots: roots, Queue: &uitest.UIQueue{}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	content := f.Content(true, 1)
	f.Settle()
	if len(h.reasons) != 0 {
		t.Fatal("initial usage inspection retired producers")
	}
	for _, tc := range []struct {
		name string
		run  func()
		want []analysiscache.QuiesceReason
	}{
		{"inspect", f.Inspect, nil},
		{"unchanged-limit", func() { f.Retune(1) }, nil},
		{"increase-limit", func() { f.Retune(2) }, []analysiscache.QuiesceReason{analysiscache.PolicyChange}},
		{"decrease-limit", func() { f.Retune(1) }, []analysiscache.QuiesceReason{analysiscache.PolicyChange, analysiscache.RecordRemoval}},
		{"clear", func() { f.Clean(similarity.ClearAll) }, []analysiscache.QuiesceReason{analysiscache.RecordRemoval}},
		{"stale", func() { f.Clean(similarity.RemoveStale) }, []analysiscache.QuiesceReason{analysiscache.RecordRemoval}},
		{"automatic-eviction", func() {
			if err := os.WriteFile(filepath.Join(records, strings.Repeat("a", 64)+".json"), []byte("12345"), 0600); err != nil {
				t.Fatal(err)
			}
			f.MakeRoom(1024 * 1024)
		}, []analysiscache.QuiesceReason{analysiscache.AutomaticEviction}},
		{"disable-persistence", func() {
			for _, object := range descendants(content) {
				if check, ok := object.(*widget.Check); ok {
					check.SetChecked(false)
					return
				}
			}
			t.Fatal("persistence toggle is absent")
		}, []analysiscache.QuiesceReason{analysiscache.PolicyChange}},
	} {
		h.reasons = nil
		tc.run()
		f.Settle()
		if !slices.Equal(h.reasons, tc.want) {
			t.Fatalf("%s maintenance phases: got %v, want %v", tc.name, h.reasons, tc.want)
		}
	}
}

func TestAnalysisCacheManagementLifecycleClosedViewRejectsQueuedProgressAndCompletion(t *testing.T) {
	h := &cacheHost{}
	q := &uitest.UIQueue{}
	entered, release := make(chan struct{}), make(chan struct{})
	reads := 0
	provider := maintenance{inspect: func(_ context.Context, _ similarity.CacheRoots, progress func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
		reads++
		if reads == 1 {
			progress(similarity.CacheProgress{Records: 99})
			close(entered)
			<-release
			return similarity.CacheUsage{General: similarity.CacheSize{Bytes: 99 * 1024 * 1024}}, nil
		}
		return similarity.CacheUsage{General: similarity.CacheSize{Bytes: 2 * 1024 * 1024}}, nil
	}}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: q})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		f.Stop()
		f.Settle()
	})
	old := f.Content(true, 2048)
	<-entered
	f.Close()
	current := f.Content(true, 2048)
	q.Drain()
	if strings.Contains(labels(old), "99") || strings.Contains(labels(current), "99") {
		t.Fatal("queued progress was delivered into a closed/reopened view")
	}
	close(release)
	f.Settle()
	if !strings.Contains(labels(current), "General: 2.0 MB") || strings.Contains(labels(current), "99") {
		t.Fatalf("old completion replaced reopened measurement: %s", labels(current))
	}
	f.Stop()
	f.Inspect()
	f.Retune(256)
	f.MakeRoom(123)
	f.Settle()
	if reads != 2 || f.Busy() {
		t.Fatal("terminal stop admitted work")
	}
}

func TestAnalysisCacheManagementLifecycleLateConfirmationAndSuccessfulRetuneAreDiscarded(t *testing.T) {
	h := &cacheHost{}
	q := &uitest.UIQueue{}
	var confirm func(bool)
	entered, release := make(chan struct{}), make(chan struct{})
	provider := maintenance{retune: func(_ context.Context, request similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
		close(entered)
		<-release
		return similarity.CacheReport{AppliedLimit: request.LimitBytes}, nil
	}}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: q, ConfirmClear: func(callback func(bool)) { confirm = callback }})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		f.Stop()
		f.Settle()
	})
	content := f.Content(true, 2048)
	f.Settle()
	button(t, content, "Clear analysis cache").OnTapped()
	f.Close()
	confirm(true)
	if f.Busy() {
		t.Fatal("confirmation from closed Settings admitted cleanup")
	}
	f.Content(true, 2048)
	f.Settle()
	f.Retune(128)
	<-entered
	f.Close()
	close(release)
	f.Settle()
	if len(h.policies) != 0 {
		t.Fatal("retune completion after Settings closed persisted stale preferences")
	}
}

func TestAnalysisCacheManagementLimitAutomaticReserveSurvivesViewClose(t *testing.T) {
	h := &cacheHost{}
	started, release := make(chan struct{}), make(chan struct{})
	var request similarity.CacheRetuneRequest
	var canceled error
	provider := maintenance{retune: func(ctx context.Context, r similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
		request = r
		close(started)
		<-release
		canceled = ctx.Err()
		return similarity.CacheReport{AppliedLimit: r.LimitBytes}, ctx.Err()
	}}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}, Roots: similarity.CacheRoots{GeneralDir: "/isolated-cache"}})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		f.Stop()
		f.Settle()
	})
	f.SetPolicy(true, 128)
	f.MakeRoom(512)
	<-started
	f.Close()
	close(release)
	f.Settle()
	if canceled != nil || request.ReserveBytes != 512 || request.LimitBytes != 128*1024*1024 || request.Roots.GeneralDir != "/isolated-cache" {
		t.Fatalf("automatic reserve lost lifetime/policy: %+v, %v", request, canceled)
	}
	if len(h.policies) != 0 {
		t.Fatal("automatic eviction rewrote standing preferences")
	}
}

func TestAnalysisCacheManagementLimitAutomaticReserveDefersToUserCleanup(t *testing.T) {
	h := &cacheHost{}
	started, release := make(chan struct{}), make(chan struct{})
	var cleanupCanceled error
	var reserves []uint64
	provider := maintenance{
		clean: func(ctx context.Context, _ similarity.CacheCleanRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
			close(started)
			<-release
			cleanupCanceled = ctx.Err()
			return similarity.CacheReport{}, ctx.Err()
		},
		retune: func(_ context.Context, request similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
			reserves = append(reserves, request.ReserveBytes)
			return similarity.CacheReport{AppliedLimit: request.LimitBytes}, nil
		},
	}
	f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		f.Stop()
		f.Settle()
	})
	f.Content(true, 2048)
	f.Settle()
	f.Clean(similarity.ClearAll)
	<-started
	f.MakeRoom(512)
	f.MakeRoom(1024)
	f.MakeRoom(256)
	close(release)
	f.Settle()
	if cleanupCanceled != nil {
		t.Fatalf("automatic cache admission canceled explicit user cleanup: %v", cleanupCanceled)
	}
	if len(reserves) != 1 || reserves[0] != 1024 {
		t.Fatalf("pending automatic reserves were not coalesced: %v", reserves)
	}
}

func TestAnalysisCacheManagementInspectionDefersToMutation(t *testing.T) {
	for _, retune := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup", true: "retune"}[retune], func(t *testing.T) {
			started, release := make(chan context.Context, 1), make(chan struct{})
			inspections := 0
			mutation := func(ctx context.Context) (similarity.CacheReport, error) {
				started <- ctx
				<-release
				return similarity.CacheReport{AppliedLimit: 128 * 1024 * 1024}, ctx.Err()
			}
			provider := maintenance{
				inspect: func(_ context.Context, _ similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
					inspections++
					return similarity.CacheUsage{}, nil
				},
				clean: func(ctx context.Context, _ similarity.CacheCleanRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
					return mutation(ctx)
				},
				retune: func(ctx context.Context, _ similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
					return mutation(ctx)
				},
			}
			h := &cacheHost{}
			f := analysiscache.New(h, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
				f.Stop()
				f.Settle()
			})
			f.Content(true, 2048)
			f.Settle()
			if retune {
				f.Retune(128)
			} else {
				f.Clean(similarity.ClearAll)
			}
			ctx := <-started
			f.Inspect()
			f.Inspect()
			if ctx.Err() != nil {
				t.Fatal("usage refresh canceled a committed user operation")
			}
			close(release)
			f.Settle()
			if inspections != 2 || f.Busy() {
				t.Fatalf("refreshes did not coalesce after mutation: %d", inspections)
			}
			if retune && (len(h.policies) != 1 || h.policies[0].limit != 128) {
				t.Fatalf("refresh discarded applied limit: %v", h.policies)
			}
		})
	}
}

func TestAnalysisCacheManagementLimitAutomaticReserveSurvivesViewOpen(t *testing.T) {
	for _, closeView := range []bool{false, true} {
		t.Run(map[bool]string{false: "inspect-after-eviction", true: "closed-view-discards-inspection"}[closeView], func(t *testing.T) {
			started := make(chan context.Context, 1)
			release := make(chan struct{})
			inspections := 0
			provider := maintenance{
				retune: func(ctx context.Context, request similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
					started <- ctx
					<-release
					return similarity.CacheReport{AppliedLimit: request.LimitBytes}, ctx.Err()
				},
				inspect: func(_ context.Context, _ similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
					inspections++
					return similarity.CacheUsage{General: similarity.CacheSize{Bytes: 2 * 1024 * 1024}}, nil
				},
			}
			f := analysiscache.New(&cacheHost{}, analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
				f.Stop()
				f.Settle()
			})
			f.MakeRoom(512)
			ctx := <-started
			f.Content(true, 2048)
			content := f.Content(true, 2048)
			if err := ctx.Err(); err != nil {
				t.Fatalf("opening Settings canceled automatic eviction: %v", err)
			}
			if closeView {
				f.Close()
			}
			close(release)
			f.Settle()
			wantInspections := 1
			if closeView {
				wantInspections = 0
			}
			if inspections != wantInspections || f.Busy() {
				t.Fatalf("inspection after eviction: calls=%d want=%d busy=%v", inspections, wantInspections, f.Busy())
			}
			if !closeView && !strings.Contains(labels(content), "General: 2.0 MB") {
				t.Fatalf("latest Settings view missed post-eviction usage: %s", labels(content))
			}
		})
	}
}

func TestAnalysisCacheManagementLimitIgnoresFavoriteInspectionFailure(t *testing.T) {
	roots := similarity.CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()}
	blocked := filepath.Join(roots.FavoritesDir, "Blocked")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("file-list.json", filepath.Join(blocked, "file-list.json")); err != nil {
		t.Fatal(err)
	}
	records := filepath.Join(roots.GeneralDir, "v1")
	if err := os.Mkdir(records, 0700); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(records, strings.Repeat("a", 64)+".json")
	if err := os.WriteFile(record, make([]byte, 2*1024*1024), 0600); err != nil {
		t.Fatal(err)
	}
	h := &cacheHost{}
	f := analysiscache.New(h, analysiscache.Options{Provider: similarity.CacheManager{}, Queue: &uitest.UIQueue{}, Roots: roots})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	content := f.Content(true, 2048)
	f.Settle()
	f.Retune(1)
	f.Settle()
	if len(h.policies) != 1 || h.policies[0] != (policy{true, 1}) {
		t.Fatalf("healthy general-cache limit did not persist: %v", h.policies)
	}
	if _, err := os.Stat(record); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("general cache was not evicted: %v", err)
	}
	if !strings.Contains(labels(content), "Maintenance is incomplete.") {
		t.Fatal("Favorite failure was hidden")
	}
}
