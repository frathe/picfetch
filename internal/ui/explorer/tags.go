package explorer

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2/container"
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
	case "costume":
		return lang.L("Costume")
	case "traditional_clothing":
		return lang.L("Traditional dress")
	case "clothing":
		return lang.L("Clothing")
	case "jewelry":
		return lang.L("Jewelry")
	case "train":
		return lang.L("Train")
	case "bus":
		return lang.L("Bus")
	case "truck":
		return lang.L("Truck")
	case "tram":
		return lang.L("Tram")
	case "fish":
		return lang.L("Fish")
	case "reptile":
		return lang.L("Reptile")
	case "castle":
		return lang.L("Castle")
	case "church":
		return lang.L("Church")
	case "temple":
		return lang.L("Temple")
	case "tower":
		return lang.L("Tower")
	case "ruins":
		return lang.L("Ruins")
	case "street":
		return lang.L("Street")
	case "park":
		return lang.L("Park")
	case "garden":
		return lang.L("Garden")
	case "waterfall":
		return lang.L("Waterfall")
	case "desert":
		return lang.L("Desert")
	case "countryside":
		return lang.L("Countryside")
	case "cave":
		return lang.L("Cave")
	case "furniture":
		return lang.L("Furniture")
	case "musical_instrument":
		return lang.L("Musical instrument")
	case "toy":
		return lang.L("Toy")
	case "sculpture":
		return lang.L("Sculpture")
	case "painting":
		return lang.L("Painting")
	case "drawing":
		return lang.L("Drawing")
	case "camera":
		return lang.L("Camera")
	case "book":
		return lang.L("Book")
	case "sign":
		return lang.L("Sign")
	case "computer":
		return lang.L("Computer")
	case "sports":
		return lang.L("Sports")
	case "concert":
		return lang.L("Concert")
	case "festival":
		return lang.L("Festival")
	case "wedding":
		return lang.L("Wedding")
	case "dance":
		return lang.L("Dance")
	case "hiking":
		return lang.L("Hiking")
	case "camping":
		return lang.L("Camping")
	case "swimming":
		return lang.L("Swimming")
	case "skiing":
		return lang.L("Skiing")
	case "drink":
		return lang.L("Drink")
	case "fruit":
		return lang.L("Fruit")
	case "dessert":
		return lang.L("Dessert")
	case "":
		return lang.L("Untagged")
	}
	return ""
}

// subjectTitle describes common content using existing tags, without another
// inference pass. A tag must cover at least half of the distinct sources.
func subjectTitle(items []similarity.Item) string {
	sources := map[string]bool{}
	counts := map[string]map[string]bool{}
	for _, item := range items {
		sources[item.Path] = true
		for _, tag := range item.Tags {
			if tag == "" || tagLabel(tag) == "" {
				continue
			}
			if counts[tag] == nil {
				counts[tag] = map[string]bool{}
			}
			counts[tag][item.Path] = true
		}
	}
	var common []string
	for tag, members := range counts {
		if len(members)*2 >= len(sources) {
			common = append(common, tag)
		}
	}
	sort.Slice(common, func(i, j int) bool {
		if left, right := len(counts[common[i]]), len(counts[common[j]]); left != right {
			return left > right
		}
		return tagLabel(common[i]) < tagLabel(common[j])
	})
	labels := make([]string, 0, min(2, len(common)))
	for _, tag := range common[:min(2, len(common))] {
		labels = append(labels, tagLabel(tag))
	}
	return strings.Join(labels, " / ")
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
	m.tagChecks = nil
	for _, key := range keys {
		if _, known := m.tagChoices[key]; !known {
			m.tagChoices[key] = true
		}
		check := widget.NewCheck(tagLabel(key), func(on bool) {
			m.tagChoices[key] = on
			m.filterTags()
			m.host.Unfocus()
		})
		check.Checked = m.tagChoices[key]
		paths := make([]string, 0, len(counts[key]))
		for path := range counts[key] {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		count := widget.NewHyperlink(fmt.Sprintf(lang.L("(%d)"), len(paths)), nil)
		count.OnTapped = func() {
			m.host.Unfocus()
			if len(paths) > 0 {
				m.host.OpenSimilarityCohort(append([]string(nil), paths...))
			}
		}
		m.tagChecks = append(m.tagChecks, check)
		m.tagRows.Add(container.NewBorder(nil, nil, nil, count, check))
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
	m.syncSelection()
	m.Refresh()
}

func (m *Map) setAllTags(on bool) {
	m.host.Unfocus()
	for key := range m.tagChoices {
		m.tagChoices[key] = on
	}
	for _, check := range m.tagChecks {
		check.Checked = on
		check.Refresh()
	}
	m.filterTags()
}
