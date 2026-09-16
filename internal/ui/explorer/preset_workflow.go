package explorer

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
)

func (f *Feature) showPresetDialog(title string, body fyne.CanvasObject) dialog.Dialog {
	if f.cohortDialog != nil {
		f.cohortDialog.Hide()
	}
	d := dialog.NewCustomWithoutButtons(title, body, f.win)
	d.SetOnClosed(func() {
		if f.cohortDialog == d {
			f.presetOp.invalidate()
			f.cohortDialog = nil
		}
		f.win.Canvas().Unfocus()
	})
	f.cohortDialog = d
	d.Show()
	return d
}

// ShowSimilarityPresets opens the one local library without starting analysis.
func (f *Feature) ShowSimilarityPresets() {
	if f.stopping || f.host.Presentation().ComparisonActive || len(f.sources) == 0 || f.cohortSaving {
		return
	}
	status := widget.NewLabel(lang.L("Loading presets..."))
	var d dialog.Dialog
	body := container.NewVBox(status, widget.NewButton(lang.L("Close"), func() { d.Hide() }))
	d = f.showPresetDialog(lang.L("Presets"), body)
	token := f.presetOp.begin()
	store := f.presets
	f.presetWorkers.Go(func() {
		all, err := store.Load(token.context())
		f.ui.Do(func() {
			if !token.current() || f.stopping {
				return
			}
			if err != nil {
				fyne.LogError("load similarity presets", err)
				status.SetText(lang.L("Could not load presets. The saved library was left unchanged."))
				return
			}
			f.buildPresetBrowser(all)
		})
	})
}

func (f *Feature) buildPresetBrowser(all []explorerpresets.Preset) {
	slices.SortFunc(all, func(a, b explorerpresets.Preset) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	filtered := slices.Clone(all)
	search := widget.NewEntry()
	search.SetPlaceHolder(lang.L("Search presets"))
	list := widget.NewList(func() int { return len(filtered) }, func() fyne.CanvasObject { return widget.NewButton("", nil) }, func(i widget.ListItemID, o fyne.CanvasObject) {
		p := filtered[i]
		b := o.(*widget.Button)
		b.SetText(p.Name)
		b.OnTapped = func() { f.editSimilarityPreset(p) }
	})
	search.OnChanged = func(query string) {
		filtered = nil
		for _, p := range all {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(query)) {
				filtered = append(filtered, p)
			}
		}
		list.Refresh()
	}
	var d dialog.Dialog
	body := container.NewVBox(search, container.NewGridWrap(fyne.NewSize(560, 280), list), container.NewHBox(
		widget.NewButton(lang.L("New preset"), func() { f.editSimilarityPreset(explorerpresets.Preset{}) }),
		widget.NewButton(lang.L("Close"), func() { d.Hide() })))
	d = f.showPresetDialog(lang.L("Presets"), body)
}

func (f *Feature) editSimilarityPreset(p explorerpresets.Preset) {
	name := widget.NewEntry()
	name.SetPlaceHolder(lang.L("Preset name"))
	name.SetText(p.Name)
	var refresh func(string)
	rules, readRule := presetRuleFields(f.images, p.Rule, func() {
		if refresh != nil {
			refresh("")
		}
	})
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	if p.ID != "" && !p.Compatible() {
		status.SetText(lang.L("This preset uses an older analysis version. Review its rules and save it before applying."))
	}
	draft := func() explorerpresets.Preset { q := p; q.Name = name.Text; q.Rule = readRule(); return q }
	var save, preview *widget.Button
	save = widget.NewButton(lang.L("Save preset"), func() {
		q := draft()
		token := f.presetOp.begin()
		save.Disable()
		preview.Disable()
		store := f.presets
		f.presetWorkers.Go(func() {
			saved, err := store.Save(token.context(), q)
			f.ui.Do(func() {
				if !token.current() || f.stopping {
					return
				}
				if err != nil {
					fyne.LogError("save similarity preset", err)
					status.SetText(lang.L("Could not save the preset. Check its name and rules and try again."))
					save.Enable()
					return
				}
				if f.surface.HasPreset(saved.ID) {
					f.previewSimilarityPreset(saved)
				} else {
					f.ShowSimilarityPresets()
				}
			})
		})
	})
	preview = widget.NewButton(lang.L("Preview preset"), func() { f.previewSimilarityPreset(p) })
	if p.ID == "" || !p.Compatible() {
		preview.Disable()
	}
	refresh = func(_ string) {
		q := draft()
		if explorerpresets.ValidName(q.Name) && q.Rule.Validate() == nil {
			save.Enable()
		} else {
			save.Disable()
		}
		preview.Disable()
	}
	name.OnChanged = refresh
	if !explorerpresets.ValidName(p.Name) || p.Rule.Validate() != nil {
		save.Disable()
	}
	var d dialog.Dialog
	remove := widget.NewButton(lang.L("Delete preset"), func() { f.deleteSimilarityPreset(p) })
	if p.ID == "" {
		remove.Disable()
	}
	body := container.NewVBox(widget.NewForm(widget.NewFormItem(lang.L("Preset name"), name)), container.NewGridWrap(fyne.NewSize(560, 380), container.NewVScroll(rules)), status,
		container.NewHBox(widget.NewButton(lang.L("Cancel"), func() { d.Hide() }), save, preview, remove))
	d = f.showPresetDialog(lang.L("Edit preset"), body)
}

func (f *Feature) deleteSimilarityPreset(p explorerpresets.Preset) {
	status := widget.NewLabel(lang.L("Delete this preset? Existing cohort members will be kept."))
	status.Wrapping = fyne.TextWrapWord
	var d dialog.Dialog
	var remove *widget.Button
	remove = widget.NewButton(lang.L("Delete preset"), func() {
		if f.cohortSaving {
			return
		}
		token := f.presetOp.begin()
		remove.Disable()
		store := f.presets
		f.presetWorkers.Go(func() {
			err := store.Delete(token.context(), p.ID)
			f.ui.Do(func() {
				if !token.current() || f.stopping {
					return
				}
				if err != nil {
					fyne.LogError("delete similarity preset", err)
					status.SetText(lang.L("Could not delete the preset. Try again."))
					remove.Enable()
					return
				}
				before := f.surface.Cohorts()
				f.surface.DetachPreset(p.ID)
				f.savePresetCohorts(before, d, status, f.ShowSimilarityPresets)
			})
		})
	})
	d = f.showPresetDialog(lang.L("Delete preset"), container.NewVBox(status, container.NewHBox(widget.NewButton(lang.L("Cancel"), func() { d.Hide() }), remove)))
}

func (f *Feature) previewSimilarityPreset(p explorerpresets.Preset) {
	status := widget.NewLabel(lang.L("Finding matching images..."))
	var rows []string
	list := widget.NewList(func() int { return len(rows) }, func() fyne.CanvasObject {
		label := widget.NewLabel("")
		label.Truncation = fyne.TextTruncateEllipsis
		return label
	}, func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(rows[i]) })
	var reviewed PresetReview
	var d dialog.Dialog
	apply := widget.NewButton(lang.L("Apply preset"), func() {
		if f.cohortSaving {
			return
		}
		if f.cohortLoadErr != nil {
			status.SetText(lang.L("Could not save the cohort. Reopen the favorite and try again."))
			return
		}
		before := f.surface.Cohorts()
		if !f.surface.ApplyPreset(reviewed) {
			status.SetText(lang.L("The images or cohort name changed. Preview the preset again."))
			return
		}
		f.savePresetCohorts(before, d, status, func() {
			d.Hide()

			f.cohort = nil
			f.unassignedCohort = false
			f.ReturnToMap()
		})
	})
	apply.Disable()
	body := container.NewVBox(status, container.NewGridWrap(fyne.NewSize(560, 250), list), container.NewHBox(widget.NewButton(lang.L("Cancel"), func() { d.Hide() }), apply))
	d = f.showPresetDialog(lang.L("Preview preset"), body)
	token := f.presetOp.begin()
	review := f.surface.PreparePreset(p)
	f.presetWorkers.Go(func() {
		var matches []similarity.Item
		for _, item := range review.Candidates {
			if !token.current() {
				return
			}
			if p.Rule.Matches(item) {
				matches = append(matches, item)
			}
		}
		f.ui.Do(func() {
			if !token.current() || f.stopping {
				return
			}
			review.Matches = matches
			reviewed = review
			members := map[string]bool{}
			for _, path := range review.Members {
				members[path] = true
			}
			matched := map[string]bool{}
			for _, item := range matches {
				matched[item.Path] = true
			}
			added, removed, retained, pending := 0, 0, 0, len(members)
			for _, item := range review.Candidates {
				if members[item.Path] {
					pending--
					if matched[item.Path] {
						retained++
					} else {
						removed++
						rows = append(rows, fmt.Sprintf(lang.L("Remove: %s"), filepath.Base(item.Path)))
					}
				} else if matched[item.Path] {
					added++
				}
				if matched[item.Path] {
					rows = append(rows, fmt.Sprintf(lang.L("Keep or add: %s"), filepath.Base(item.Path)))
				}
			}
			list.Refresh()
			status.SetText(fmt.Sprintf(lang.L("%d added, %d removed, %d retained, %d pending"), added, removed, retained, pending))
			if (len(matches) >= 2 || len(review.Members) > 0) && p.Compatible() && p.Rule.Validate() == nil {
				apply.Enable()
			}
		})
	})
}

func (f *Feature) savePresetCohorts(before favstore.CohortState, d dialog.Dialog, status *widget.Label, finish func()) {
	store := f.cohortStore
	if store == nil {
		finish()
		return
	}
	groups := f.surface.Cohorts()
	token := f.token
	f.cohortSaving = true
	f.presetWorkers.Go(func() {
		err := store.Save(token.context(), groups)
		f.ui.Do(func() {
			if !token.current() || f.stopping {
				return
			}
			f.cohortSaving = false
			if err != nil {
				fyne.LogError("save preset cohort", err)
				f.surface.RestoreCohorts(before)
				message := lang.L("Could not save the cohort. Reopen the favorite and try again.")
				status.SetText(message)
				if f.cohortDialog != d {
					f.host.ShowToast(message)
				}
				return
			}
			if f.cohortDialog == d {
				finish()
			}
		})
	})
}
