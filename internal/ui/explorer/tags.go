package explorer

import (
	"fmt"
	"sort"

	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/similarity"
)

func tagLabel(id string) string {
	switch id {
	case "animal":
		return lang.L("Animal")
	case "bird":
		return lang.L("Bird")
	case "cat":
		return lang.L("Cat")
	case "dog":
		return lang.L("Dog")
	case "horse":
		return lang.L("Horse")
	case "insect":
		return lang.L("Insect")
	case "person":
		return lang.L("Person")
	case "portrait":
		return lang.L("Portrait")
	case "flower":
		return lang.L("Flower")
	case "tree":
		return lang.L("Tree")
	case "forest":
		return lang.L("Forest")
	case "mountain":
		return lang.L("Mountain")
	case "beach":
		return lang.L("Beach")
	case "ocean":
		return lang.L("Ocean")
	case "lake":
		return lang.L("Lake")
	case "river":
		return lang.L("River")
	case "snow":
		return lang.L("Snow")
	case "sunset":
		return lang.L("Sunset")
	case "building":
		return lang.L("Building")
	case "city":
		return lang.L("City")
	case "bridge":
		return lang.L("Bridge")
	case "car":
		return lang.L("Car")
	case "bicycle":
		return lang.L("Bicycle")
	case "motorcycle":
		return lang.L("Motorcycle")
	case "boat":
		return lang.L("Boat")
	case "aircraft":
		return lang.L("Aircraft")
	case "food":
		return lang.L("Food")
	case "indoor":
		return lang.L("Indoors")
	case "document":
		return lang.L("Document")
	case "screenshot":
		return lang.L("Screenshot")
	case "artwork":
		return lang.L("Artwork")
	case "":
		return lang.L("Untagged")
	}
	return ""
}

func itemTags(item similarity.Item) []string {
	var tags []string
	for _, tag := range item.Tags {
		if tag != "" && tagLabel(tag) != "" {
			tags = append(tags, tag)
		}
	}
	if len(tags) == 0 {
		return []string{""}
	}
	return tags
}

func (m *Map) setTags(items []similarity.Item) {
	counts := map[string]map[string]bool{"": {}}
	for _, item := range items {
		if item.Error != "" || item.Cohort == "" {
			continue
		}
		for _, tag := range itemTags(item) {
			if counts[tag] == nil {
				counts[tag] = map[string]bool{}
			}
			counts[tag][item.Path] = true
		}
	}
	if m.tagChoices == nil {
		m.tagChoices = map[string]bool{}
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return tagLabel(keys[i]) < tagLabel(keys[j]) })
	m.tagRows.RemoveAll()
	for _, key := range keys {
		if _, known := m.tagChoices[key]; !known {
			m.tagChoices[key] = true
		}
		check := widget.NewCheck(fmt.Sprintf(lang.L("%s (%d)"), tagLabel(key), len(counts[key])), func(on bool) {
			m.tagChoices[key] = on
			m.filterTags()
			m.host.Unfocus()
		})
		check.Checked = m.tagChoices[key]
		m.tagRows.Add(check)
	}
}

func (m *Map) filterTags() {
	active := func(tags []string) bool {
		for _, tag := range tags {
			if m.tagChoices[tag] {
				return true
			}
		}
		return false
	}
	for _, pile := range m.piles {
		if active(pile.tags) {
			pile.Show()
		} else {
			pile.Hide()
		}
	}
	if active(m.unassignedTags) {
		m.unassigned.Show()
	} else {
		m.unassigned.Hide()
	}
}

func (m *Map) setAllTags(on bool) {
	m.host.Unfocus()
	for key := range m.tagChoices {
		m.tagChoices[key] = on
	}
	for _, object := range m.tagRows.Objects {
		check := object.(*widget.Check)
		check.Checked = on
		check.Refresh()
	}
	m.filterTags()
}
