package ui

import (
	"errors"
	"fmt"
	"image/color"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/explorerpresets"
)

func (v *viewer) analyzeSimilaritySelection() {
	if v.comparisonActive() || v.explorer.cohortSaving || !v.grid.Visible() || !v.explorer.unassignedCohort || v.grid.SelectionCount() < 2 {
		return
	}
	if v.explorer.cohortDialog != nil {
		v.explorer.cohortDialog.Hide()
	}
	revision := v.explorer.lifecycle.currentRevision()
	var paths []string
	for _, i := range v.grid.Selection() {
		if i >= 0 && i < v.FileCount() {
			paths = append(paths, v.FileAt(i).Path())
		}
	}
	traits := v.explorer.surface.SharedTraits(paths)
	name := widget.NewEntry()
	name.SetPlaceHolder(lang.L("Cohort name"))
	name.Validator = func(value string) error {
		if !v.explorer.surface.CohortNameAvailable(value) {
			return errors.New(lang.L("Enter a unique cohort name (1-80 characters)."))
		}
		return nil
	}
	checks := container.NewVBox()
	var choices []*widget.Check
	include := widget.NewCheck(lang.L("Include other matching Unassigned images"), nil)
	include.Checked = true
	var matches, tags []string
	selectionCurrent := func() bool {
		current := v.explorer.surface.SharedTraits(paths)
		if len(current) == 0 || len(tags) == 0 {
			return false
		}
		for _, tag := range tags {
			found := false
			for _, trait := range current {
				found = found || trait.ID == tag
			}
			if !found {
				return false
			}
		}
		return true
	}
	count := widget.NewLabel("")
	errorLabel := widget.NewLabel("")
	errorLabel.Wrapping = fyne.TextWrapWord
	list := widget.NewList(func() int { return len(matches) }, func() fyne.CanvasObject {
		label := widget.NewLabel("")
		label.Truncation = fyne.TextTruncateEllipsis
		return label
	}, func(i widget.ListItemID, o fyne.CanvasObject) {
		path := matches[i]
		o.(*widget.Label).SetText(fmt.Sprintf(lang.L("%s - %s"), filepath.Base(path), filepath.Dir(path)))
	})
	var review dialog.Dialog
	pending := false
	var create *widget.Button
	create = widget.NewButton(lang.L("Create cohort"), func() {
		if pending {
			return
		}
		if v.stopping || v.comparisonActive() || revision != v.explorer.lifecycle.currentRevision() || v.explorer.cohortDialog != review || !v.grid.Visible() || !v.explorer.unassignedCohort || !selectionCurrent() {
			errorLabel.SetText(lang.L("The images or cohort name changed. Analyze the selection again."))
			return
		}
		if v.explorer.cohortLoadErr != nil {
			errorLabel.SetText(lang.L("Could not save the cohort. Reopen the favorite and try again."))
			return
		}
		before := v.explorer.surface.Cohorts()
		if !v.explorer.surface.CreateCohort(name.Text, matches, tags) {
			errorLabel.SetText(lang.L("The images or cohort name changed. Analyze the selection again."))
			return
		}
		finish := func() {
			review.Hide()
			v.grid.Close()
			v.explorer.cohort = nil
			v.explorer.unassignedCohort = false
			v.backToSimilarityMap()
		}
		store := v.explorer.cohortStore
		if store == nil {
			finish()
			return
		}
		groups := v.explorer.surface.Cohorts()
		ctx := v.explorer.token.context()
		pending = true
		v.explorer.cohortSaving = true
		create.Disable()
		v.explorer.workers.Go(func() {
			err := store.Save(ctx, groups)
			v.explorer.ui.Do(func() {
				if revision != v.explorer.lifecycle.currentRevision() || v.stopping {
					return
				}
				pending = false
				v.explorer.cohortSaving = false
				if err != nil {
					fyne.LogError("save favorite cohort", err)
					v.explorer.surface.RestoreCohorts(before)
					errorLabel.SetText(lang.L("Could not save the cohort. Reopen the favorite and try again."))
					if v.explorer.cohortDialog != review {
						v.ShowToast(lang.L("Could not save the cohort. Reopen the favorite and try again."))
					}
					create.Enable()
					return
				}
				if v.explorer.cohortDialog == review {
					finish()
				}
			})
		})
	})
	refresh := func() {
		tags = nil
		for i, check := range choices {
			if check.Checked {
				tags = append(tags, traits[i].ID)
			}
		}
		matches = v.explorer.surface.MatchUnassigned(tags)
		if !include.Checked {
			matches = slices.DeleteFunc(matches, func(path string) bool { return !slices.Contains(paths, path) })
		}
		count.SetText(fmt.Sprintf(lang.L("%d images will move from Unassigned"), len(matches)))
		list.Refresh()
		if !pending && len(matches) > 1 && selectionCurrent() && name.Validate() == nil {
			create.Enable()
		} else {
			create.Disable()
		}
	}
	for _, trait := range traits {
		check := widget.NewCheck(trait.Label, func(_ bool) { refresh() })
		check.Checked = true
		choices = append(choices, check)
		checks.Add(check)
	}
	if len(traits) == 0 {
		checks.Add(widget.NewLabel(lang.L("No shared visual tags found. Select different images.")))
	}
	name.OnChanged = func(_ string) { refresh() }
	include.OnChanged = func(_ bool) { refresh() }
	name.OnSubmitted = func(_ string) {
		if !create.Disabled() {
			create.OnTapped()
		}
	}
	refresh()
	cancel := widget.NewButton(lang.L("Cancel"), func() { review.Hide() })
	savePreset := widget.NewButton(lang.L("Save as preset"), func() {
		if pending || v.stopping || revision != v.explorer.lifecycle.currentRevision() {
			return
		}
		v.editSimilarityPreset(explorerpresets.Preset{Name: name.Text, Rule: explorerpresets.Rule{Tags: slices.Clone(tags)}})
	})
	matchList := container.NewGridWrap(fyne.NewSize(520, 150), list)
	if len(traits) == 0 {
		include.Hide()
		count.Hide()
		matchList.Hide()
		name.Hide()
		errorLabel.Hide()
	}
	body := container.NewVBox(widget.NewLabel(fmt.Sprintf(lang.L("Shared visual tags in %d selected images"), len(paths))), checks, include, count,
		matchList, name, errorLabel,
		container.NewHBox(cancel, create, savePreset))
	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(440, 0))
	review = dialog.NewCustomWithoutButtons(lang.L("Create cohort from Unassigned"), container.NewStack(width, body), v.win)
	review.SetOnClosed(func() {
		if v.explorer.cohortDialog == review {
			v.explorer.cohortDialog = nil
		}
		v.win.Canvas().Unfocus()
	})
	v.explorer.cohortDialog = review
	review.Show()
	if len(traits) > 0 {
		v.win.Canvas().Focus(name)
	} else {
		v.win.Canvas().Focus(cancel)
	}
}
