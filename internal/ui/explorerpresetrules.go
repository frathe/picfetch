package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/imaging"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
)

func presetRuleFields(rule explorerpresets.Rule, changed func()) (*widget.Form, func() explorerpresets.Rule) {
	makeName := widget.NewEntry()
	makeName.SetPlaceHolder(lang.L("Camera make"))
	makeName.SetText(rule.Make)
	formats := []string{lang.L("Any file type")}
	for _, ext := range imaging.SupportedExtensions() {
		ext = strings.TrimPrefix(ext, ".")
		if ext == "jpeg" {
			ext = "jpg"
		}
		if ext == "tif" {
			ext = "tiff"
		}
		if !slices.Contains(formats, ext) {
			formats = append(formats, ext)
		}
	}
	if rule.Format != "" && !slices.Contains(formats, rule.Format) {
		formats = append(formats, rule.Format)
	}
	fileType := widget.NewSelect(formats, nil)
	fileType.PlaceHolder = lang.L("File type")
	if rule.Format == "" {
		fileType.SetSelectedIndex(0)
	} else {
		fileType.SetSelected(rule.Format)
	}
	orientationLabels := []string{lang.L("Any orientation"), lang.L("Portrait"), lang.L("Landscape"), lang.L("Square")}
	orientations := []string{"", "portrait", "landscape", "square"}
	if !slices.Contains(orientations, rule.Orientation) {
		orientations = append(orientations, rule.Orientation)
		orientationLabels = append(orientationLabels, rule.Orientation)
	}
	orientation := widget.NewSelect(orientationLabels, nil)
	orientation.PlaceHolder = lang.L("Image orientation")
	orientation.SetSelectedIndex(max(0, slices.Index(orientations, rule.Orientation)))
	numbers := make([]*widget.Entry, 4)
	labels := []string{lang.L("Minimum width"), lang.L("Maximum width"), lang.L("Minimum height"), lang.L("Maximum height")}
	form := widget.NewForm(widget.NewFormItem(lang.L("Camera make"), makeName), widget.NewFormItem(lang.L("File type"), fileType), widget.NewFormItem(lang.L("Image orientation"), orientation))
	model, from, through := widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	model.SetPlaceHolder(lang.L("Camera model"))
	model.SetText(rule.Model)
	from.SetPlaceHolder(lang.L("Capture date from (YYYY-MM-DD)"))
	from.SetText(rule.CaptureFrom)
	through.SetPlaceHolder(lang.L("Capture date through (YYYY-MM-DD)"))
	through.SetText(rule.CaptureThrough)
	form.Append(lang.L("Camera model"), model)
	form.Append(lang.L("Capture date from (YYYY-MM-DD)"), from)
	form.Append(lang.L("Capture date through (YYYY-MM-DD)"), through)
	traits := explorerui.Traits()
	for _, tag := range rule.Tags {
		if !slices.ContainsFunc(traits, func(t explorerui.Trait) bool { return t.ID == tag }) {
			traits = append(traits, explorerui.Trait{ID: tag, Label: fmt.Sprintf(lang.L("Unavailable tag: %s"), tag)})
		}
	}
	var checks []*widget.Check
	tagRows := container.NewVBox()
	for _, trait := range traits {
		check := widget.NewCheck(trait.Label, func(_ bool) { changed() })
		check.Checked = slices.Contains(rule.Tags, trait.ID)
		checks = append(checks, check)
		tagRows.Add(check)
	}
	form.Append(lang.L("Match every selected tag"), container.NewGridWrap(fyne.NewSize(240, 140), container.NewVScroll(tagRows)))
	for i, value := range []int{rule.MinWidth, rule.MaxWidth, rule.MinHeight, rule.MaxHeight} {
		numbers[i] = widget.NewEntry()
		numbers[i].SetPlaceHolder(labels[i])
		if value != 0 {
			numbers[i].SetText(strconv.Itoa(value))
		}
		form.Append(labels[i], numbers[i])
		numbers[i].OnChanged = func(_ string) { changed() }
	}
	makeName.OnChanged = func(_ string) { changed() }
	for _, entry := range []*widget.Entry{model, from, through} {
		entry.OnChanged = func(_ string) { changed() }
	}
	fileType.OnChanged = func(_ string) { changed() }
	orientation.OnChanged = func(_ string) { changed() }
	read := func() explorerpresets.Rule {
		r := rule
		r.Make = strings.TrimSpace(makeName.Text)
		r.Model, r.CaptureFrom, r.CaptureThrough = strings.TrimSpace(model.Text), strings.TrimSpace(from.Text), strings.TrimSpace(through.Text)
		r.Tags = nil
		for i, check := range checks {
			if check.Checked {
				r.Tags = append(r.Tags, traits[i].ID)
			}
		}
		r.Format = fileType.Selected
		if fileType.SelectedIndex() <= 0 {
			r.Format = ""
		}
		r.Orientation = orientations[max(0, orientation.SelectedIndex())]
		values := []*int{&r.MinWidth, &r.MaxWidth, &r.MinHeight, &r.MaxHeight}
		for i, entry := range numbers {
			*values[i] = 0
			if entry.Text != "" {
				n, err := strconv.Atoi(entry.Text)
				if err != nil || n <= 0 {
					n = -1
				}
				*values[i] = n
			}
		}
		return r
	}
	return form, read
}
