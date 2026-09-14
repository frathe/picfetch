// Package analysiscache owns the analysis cache settings and maintenance UI.
package analysiscache

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/similarity"
)

type Host interface {
	// Quiesce may retain fully prepared producers with no pending writes during automatic eviction.
	Quiesce(preservePrepared bool) []<-chan struct{}
	ApplyPolicy(bool, int)
}

type UIQueue interface {
	Do(func())
	Drain() bool
}
type Options struct {
	Roots        similarity.CacheRoots
	Provider     similarity.CacheMaintenanceProvider
	Queue        UIQueue
	ConfirmClear func(func(bool))
	Changed      func()
}

const mebibyte = 1024 * 1024

// Feature and all its presentation state belong to the UI goroutine.
type Feature struct {
	host                                                         Host
	options                                                      Options
	ui                                                           UIQueue
	syncing, pendingRoom, pendingInspect, open, stopped, enabled bool
	pendingReserve                                               uint64
	limitMiB                                                     int
	view                                                         uint64
	current                                                      *work
	workers                                                      []*work
	notice                                                       chan struct{}
	general, favorite, combined, status, progress                *widget.Label
	limit                                                        *widget.Entry
	loose                                                        *widget.Check
	refresh, clearButton, stale, cancel                          *widget.Button
}

func New(host Host, options Options) *Feature {
	f := &Feature{host: host, enabled: true, limitMiB: 2048, notice: make(chan struct{}, 1)}
	f.Configure(options)
	return f
}

// Configure replaces adapters before work starts; drain retired callbacks before
// changing the queue. Each worker captures its provider, roots and UI queue.
func (f *Feature) Configure(options Options) {
	f.options = options
	f.ui = options.Queue
	if f.ui == nil {
		f.ui = featureQueue{}
	}
}

func (f *Feature) SetRoots(roots similarity.CacheRoots) { f.options.Roots = roots }

// SetPolicy seeds standing settings without writing them back to the host.
func (f *Feature) SetPolicy(enabled bool, limitMiB int) {
	f.enabled = enabled
	if limitMiB > 0 && uint64(limitMiB) <= ^uint64(0)/mebibyte {
		f.limitMiB = limitMiB
	} else {
		f.limitMiB = 2048
	}
}

func (f *Feature) Content(enabled bool, limitMiB int) fyne.CanvasObject {
	f.Close()
	f.SetPolicy(enabled, limitMiB)
	f.open = !f.stopped
	f.loose = widget.NewCheck(lang.L("Cache analysis for loose images"), nil)
	f.loose.Checked = enabled
	f.limit = widget.NewEntry()
	f.limit.Text = strconv.Itoa(f.limitMiB)
	view := f.view
	current := func() bool { return f.open && f.view == view && !f.syncing }
	f.limit.Validator = func(text string) error {
		if _, ok := parseLimit(text); !ok {
			return errors.New(lang.L("Enter a positive whole number of MB."))
		}
		return nil
	}
	f.limit.OnSubmitted = func(text string) {
		if limit, ok := parseLimit(text); current() && ok {
			f.Retune(limit)
		}
	}
	f.loose.OnChanged = func(enabled bool) {
		if !current() {
			return
		}
		f.setEnabled(enabled)
	}
	f.general = widget.NewLabel(lang.L("General: not measured"))
	f.favorite = widget.NewLabel(lang.L("Favorites: not measured"))
	f.combined = widget.NewLabel(lang.L("Combined: not measured"))
	f.status = widget.NewLabel("")
	f.status.Wrapping = fyne.TextWrapWord
	f.progress = widget.NewLabel("")
	f.refresh = widget.NewButton(lang.L("Refresh usage"), func() {
		if current() && !f.Busy() {
			f.Inspect()
		}
	})
	f.clearButton = widget.NewButton(lang.L("Clear analysis cache"), func() {
		if !current() || f.Busy() || f.options.ConfirmClear == nil {
			return
		}
		answered := false
		f.options.ConfirmClear(func(confirmed bool) {
			if answered {
				return
			}
			answered = true
			if confirmed && current() && !f.Busy() {
				f.Clean(similarity.ClearAll)
			}
		})
	})
	f.stale = widget.NewButton(lang.L("Remove stale entries"), func() {
		if current() && !f.Busy() {
			f.Clean(similarity.RemoveStale)
		}
	})
	f.cancel = widget.NewButton(lang.L("Cancel"), func() {
		if current() && f.current != nil {
			f.current.cancel()
			f.status.SetText(lang.L("Canceling maintenance..."))
		}
	})
	applyLimit := widget.NewButton(lang.L("Apply cache limit"), func() { f.limit.OnSubmitted(f.limit.Text) })
	content := container.NewVBox(f.loose, widget.NewLabel(lang.L("General analysis cache limit (MB)")), container.NewBorder(nil, nil, nil, applyLimit, f.limit),
		f.general, f.favorite, f.combined, container.NewGridWithColumns(2, f.refresh, f.cancel), f.clearButton, f.stale, f.status, f.progress)
	f.Inspect()
	return content
}

func (f *Feature) Inspect() { f.submit(operation{intent: inspectUsage}) }

func (f *Feature) showUsage(usage similarity.CacheUsage) {
	if !f.open {
		return
	}
	f.general.SetText(fmt.Sprintf(lang.L("General: %.1f MB, %d records"), float64(usage.General.Bytes)/mebibyte, usage.General.Records))
	f.favorite.SetText(fmt.Sprintf(lang.L("Favorites: %.1f MB, %d records"), float64(usage.Favorite.Bytes)/mebibyte, usage.Favorite.Records))
	f.combined.SetText(fmt.Sprintf(lang.L("Combined: %.1f MB"), (float64(usage.General.Bytes)+float64(usage.Favorite.Bytes))/mebibyte))
	if usage.Incomplete {
		f.status.SetText(lang.L("Usage is incomplete."))
	} else {
		f.status.SetText("")
	}
}

func parseLimit(text string) (int, bool) {
	value, err := strconv.ParseUint(strings.TrimSpace(text), 10, 64)
	if err != nil || value == 0 || value > ^uint64(0)/mebibyte || value > uint64(^uint(0)>>1) {
		return 0, false
	}
	return int(value), true
}

// Persistence is a standing preference, independent of readable cache contents.
// Retire the captured producers on UI, then track their completion off UI even
// if Settings closes before their last native operation returns.
func (f *Feature) setEnabled(enabled bool) {
	f.submit(operation{intent: retirePolicy, enabled: enabled})
}

// Retune commits an explicit limit only after successful general maintenance
// while its Settings view remains current.
func (f *Feature) Retune(limitMiB int) {
	f.submit(operation{intent: applyLimit, limitMiB: limitMiB})
}

// MakeRoom coalesces automatic eviction without changing standing policy.
// Its operation survives Settings closure.
func (f *Feature) MakeRoom(needBytes uint64) {
	f.submit(operation{intent: evictRecords, reserve: needBytes})
}

func (f *Feature) showReport(r result) {
	f.showUsage(r.report.Remaining)
	remaining := r.report.Remaining
	text := fmt.Sprintf(lang.L("Removed %d records (%.1f MB); remaining %d records (%.1f MB)."), r.report.RemovedRecords, float64(r.report.RemovedBytes)/mebibyte, remaining.General.Records+remaining.Favorite.Records, (float64(remaining.General.Bytes)+float64(remaining.Favorite.Bytes))/mebibyte)
	text += "\n" + fmt.Sprintf(lang.L("Skipped: %d; unavailable: %d; failures: %d."), r.report.Skipped, r.report.Unavailable, r.report.Failures)
	canceled := r.report.Canceled
	if canceled {
		text += "\n" + lang.L("Maintenance was canceled.")
	}
	if r.incomplete {
		text += "\n" + lang.L("Maintenance is incomplete.")
	} else if !canceled {
		text += "\n" + lang.L("Maintenance complete.")
	}
	f.status.SetText(text)
}

// Clean is admitted after the Clear confirmation or the stale-cleanup command.
func (f *Feature) Clean(mode similarity.CacheCleanMode) {
	f.submit(operation{intent: cleanRecords, mode: mode})
}

func (f *Feature) changed() {
	if f.open && f.refresh != nil {
		busy := f.Busy()
		for _, button := range []*widget.Button{f.refresh, f.clearButton, f.stale} {
			if busy {
				button.Disable()
			} else {
				button.Enable()
			}
		}
		if f.options.ConfirmClear == nil {
			f.clearButton.Disable()
		}
		if busy && f.current != nil {
			f.cancel.Enable()
		} else {
			f.cancel.Disable()
		}
	}
	if f.options.Changed != nil {
		f.options.Changed()
	}
}
