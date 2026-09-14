package explorer

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

func (f *Feature) AnalyzeSelection(selected []string) {
	if f.host.Presentation().ComparisonActive || f.cohortSaving || !f.host.Presentation().GridVisible || !f.unassignedCohort || len(selected) < 2 {
		return
	}
	if f.cohortDialog != nil {
		f.cohortDialog.Hide()
	}
	revision := f.lifecycle.currentRevision()
	paths := slices.Clone(selected)
	traits := f.surface.SharedTraits(paths)
	name := widget.NewEntry()
	name.SetPlaceHolder(lang.L("Cohort name"))
	name.Validator = func(value string) error {
		if !f.surface.CohortNameAvailable(value) {
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
		current := f.surface.SharedTraits(paths)
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
		if f.stopping || f.host.Presentation().ComparisonActive || revision != f.lifecycle.currentRevision() || f.cohortDialog != review || !f.host.Presentation().GridVisible || !f.unassignedCohort || !selectionCurrent() {
			errorLabel.SetText(lang.L("The images or cohort name changed. Analyze the selection again."))
			return
		}
		if f.cohortLoadErr != nil {
			errorLabel.SetText(lang.L("Could not save the cohort. Reopen the favorite and try again."))
			return
		}
		before := f.surface.Cohorts()
		if !f.surface.CreateCohort(name.Text, matches, tags) {
			errorLabel.SetText(lang.L("The images or cohort name changed. Analyze the selection again."))
			return
		}
		finish := func() {
			review.Hide()

			f.cohort = nil
			f.unassignedCohort = false
			f.ReturnToMap()
		}
		store := f.cohortStore
		if store == nil {
			finish()
			return
		}
		groups := f.surface.Cohorts()
		ctx := f.token.context()
		pending = true
		f.cohortSaving = true
		create.Disable()
		f.workers.Go(func() {
			err := store.Save(ctx, groups)
			f.ui.Do(func() {
				if revision != f.lifecycle.currentRevision() || f.stopping {
					return
				}
				pending = false
				f.cohortSaving = false
				if err != nil {
					fyne.LogError("save favorite cohort", err)
					f.surface.RestoreCohorts(before)
					errorLabel.SetText(lang.L("Could not save the cohort. Reopen the favorite and try again."))
					if f.cohortDialog != review {
						f.host.ShowToast(lang.L("Could not save the cohort. Reopen the favorite and try again."))
					}
					create.Enable()
					return
				}
				if f.cohortDialog == review {
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
		matches = f.surface.MatchUnassigned(tags)
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
		if pending || f.stopping || revision != f.lifecycle.currentRevision() {
			return
		}
		f.editSimilarityPreset(explorerpresets.Preset{Name: name.Text, Rule: explorerpresets.Rule{Tags: slices.Clone(tags)}})
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
	review = dialog.NewCustomWithoutButtons(lang.L("Create cohort from Unassigned"), container.NewStack(width, body), f.win)
	review.SetOnClosed(func() {
		if f.cohortDialog == review {
			f.cohortDialog = nil
		}
		f.win.Canvas().Unfocus()
	})
	f.cohortDialog = review
	review.Show()
	if len(traits) > 0 {
		f.win.Canvas().Focus(name)
	} else {
		f.win.Canvas().Focus(cancel)
	}
}
