package ui

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
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
)

func (v *viewer) showPresetDialog(title string, body fyne.CanvasObject) dialog.Dialog {
	if v.explorer.cohortDialog != nil {
		v.explorer.cohortDialog.Hide()
	}
	d := dialog.NewCustomWithoutButtons(title, body, v.win)
	d.SetOnClosed(func() {
		if v.explorer.cohortDialog == d {
			v.explorer.presetOp.invalidate()
			v.explorer.cohortDialog = nil
		}
		v.win.Canvas().Unfocus()
	})
	v.explorer.cohortDialog = d
	d.Show()
	return d
}

// ShowSimilarityPresets opens the one local library without starting analysis.
func (v *viewer) ShowSimilarityPresets() {
	if v.stopping || v.comparisonActive() || len(v.explorer.sources) == 0 || v.explorer.cohortSaving {
		return
	}
	status := widget.NewLabel(lang.L("Loading presets..."))
	var d dialog.Dialog
	body := container.NewVBox(status, widget.NewButton(lang.L("Close"), func() { d.Hide() }))
	d = v.showPresetDialog(lang.L("Presets"), body)
	token := v.explorer.presetOp.begin()
	store := v.explorer.presets
	v.explorer.presetWorkers.Go(func() {
		all, err := store.Load(token.context())
		v.explorer.ui.Do(func() {
			if !token.current() || v.stopping {
				return
			}
			if err != nil {
				fyne.LogError("load similarity presets", err)
				status.SetText(lang.L("Could not load presets. The saved library was left unchanged."))
				return
			}
			v.buildPresetBrowser(all)
		})
	})
}

func (v *viewer) buildPresetBrowser(all []explorerpresets.Preset) {
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
		b.OnTapped = func() { v.editSimilarityPreset(p) }
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
		widget.NewButton(lang.L("New preset"), func() { v.editSimilarityPreset(explorerpresets.Preset{}) }),
		widget.NewButton(lang.L("Close"), func() { d.Hide() })))
	d = v.showPresetDialog(lang.L("Presets"), body)
}

func (v *viewer) editSimilarityPreset(p explorerpresets.Preset) {
	name := widget.NewEntry()
	name.SetPlaceHolder(lang.L("Preset name"))
	name.SetText(p.Name)
	var refresh func(string)
	rules, readRule := presetRuleFields(p.Rule, func() {
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
		token := v.explorer.presetOp.begin()
		save.Disable()
		preview.Disable()
		store := v.explorer.presets
		v.explorer.presetWorkers.Go(func() {
			saved, err := store.Save(token.context(), q)
			v.explorer.ui.Do(func() {
				if !token.current() || v.stopping {
					return
				}
				if err != nil {
					fyne.LogError("save similarity preset", err)
					status.SetText(lang.L("Could not save the preset. Check its name and rules and try again."))
					save.Enable()
					return
				}
				if v.explorer.surface.HasPreset(saved.ID) {
					v.previewSimilarityPreset(saved)
				} else {
					v.ShowSimilarityPresets()
				}
			})
		})
	})
	preview = widget.NewButton(lang.L("Preview preset"), func() { v.previewSimilarityPreset(p) })
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
	remove := widget.NewButton(lang.L("Delete preset"), func() { v.deleteSimilarityPreset(p) })
	if p.ID == "" {
		remove.Disable()
	}
	body := container.NewVBox(widget.NewForm(widget.NewFormItem(lang.L("Preset name"), name)), container.NewGridWrap(fyne.NewSize(560, 380), container.NewVScroll(rules)), status,
		container.NewHBox(widget.NewButton(lang.L("Cancel"), func() { d.Hide() }), save, preview, remove))
	d = v.showPresetDialog(lang.L("Edit preset"), body)
}

func (v *viewer) deleteSimilarityPreset(p explorerpresets.Preset) {
	status := widget.NewLabel(lang.L("Delete this preset? Existing cohort members will be kept."))
	status.Wrapping = fyne.TextWrapWord
	var d dialog.Dialog
	var remove *widget.Button
	remove = widget.NewButton(lang.L("Delete preset"), func() {
		if v.explorer.cohortSaving {
			return
		}
		token := v.explorer.presetOp.begin()
		remove.Disable()
		store := v.explorer.presets
		v.explorer.presetWorkers.Go(func() {
			err := store.Delete(token.context(), p.ID)
			v.explorer.ui.Do(func() {
				if !token.current() || v.stopping {
					return
				}
				if err != nil {
					fyne.LogError("delete similarity preset", err)
					status.SetText(lang.L("Could not delete the preset. Try again."))
					remove.Enable()
					return
				}
				before := v.explorer.surface.Cohorts()
				v.explorer.surface.DetachPreset(p.ID)
				v.savePresetCohorts(before, d, status, v.ShowSimilarityPresets)
			})
		})
	})
	d = v.showPresetDialog(lang.L("Delete preset"), container.NewVBox(status, container.NewHBox(widget.NewButton(lang.L("Cancel"), func() { d.Hide() }), remove)))
}

func (v *viewer) previewSimilarityPreset(p explorerpresets.Preset) {
	status := widget.NewLabel(lang.L("Finding matching images..."))
	var rows []string
	list := widget.NewList(func() int { return len(rows) }, func() fyne.CanvasObject {
		label := widget.NewLabel("")
		label.Truncation = fyne.TextTruncateEllipsis
		return label
	}, func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(rows[i]) })
	var reviewed explorerui.PresetReview
	var d dialog.Dialog
	apply := widget.NewButton(lang.L("Apply preset"), func() {
		if v.explorer.cohortSaving {
			return
		}
		if v.explorer.cohortLoadErr != nil {
			status.SetText(lang.L("Could not save the cohort. Reopen the favorite and try again."))
			return
		}
		before := v.explorer.surface.Cohorts()
		if !v.explorer.surface.ApplyPreset(reviewed) {
			status.SetText(lang.L("The images or cohort name changed. Preview the preset again."))
			return
		}
		v.savePresetCohorts(before, d, status, func() {
			d.Hide()
			v.grid.Close()
			v.explorer.cohort = nil
			v.explorer.unassignedCohort = false
			v.backToSimilarityMap()
		})
	})
	apply.Disable()
	body := container.NewVBox(status, container.NewGridWrap(fyne.NewSize(560, 250), list), container.NewHBox(widget.NewButton(lang.L("Cancel"), func() { d.Hide() }), apply))
	d = v.showPresetDialog(lang.L("Preview preset"), body)
	token := v.explorer.presetOp.begin()
	review := v.explorer.surface.PreparePreset(p)
	v.explorer.presetWorkers.Go(func() {
		var matches []similarity.Item
		for _, item := range review.Candidates {
			if !token.current() {
				return
			}
			if p.Rule.Matches(item) {
				matches = append(matches, item)
			}
		}
		v.explorer.ui.Do(func() {
			if !token.current() || v.stopping {
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

func (v *viewer) savePresetCohorts(before favstore.CohortState, d dialog.Dialog, status *widget.Label, finish func()) {
	store := v.explorer.cohortStore
	if store == nil {
		finish()
		return
	}
	groups := v.explorer.surface.Cohorts()
	token := v.explorer.token
	v.explorer.cohortSaving = true
	v.explorer.presetWorkers.Go(func() {
		err := store.Save(token.context(), groups)
		v.explorer.ui.Do(func() {
			if !token.current() || v.stopping {
				return
			}
			v.explorer.cohortSaving = false
			if err != nil {
				fyne.LogError("save preset cohort", err)
				v.explorer.surface.RestoreCohorts(before)
				message := lang.L("Could not save the cohort. Reopen the favorite and try again.")
				status.SetText(message)
				if v.explorer.cohortDialog != d {
					v.ShowToast(message)
				}
				return
			}
			if v.explorer.cohortDialog == d {
				finish()
			}
		})
	})
}
