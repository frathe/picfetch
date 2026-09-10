package explorer

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/frathe/picfetch/internal/favstore"
)

// Trait is a recognized, localized visual tag shared by selected sources.
type Trait struct{ ID, Label string }

// Cohorts returns an independent snapshot, including members not yet present
// in a partial analysis. Saving another group must retain those members.
func (m *Map) Cohorts() []favstore.Cohort {
	groups := make([]favstore.Cohort, 0, len(m.cohortNames))
	for key, name := range m.cohortNames {
		group := favstore.Cohort{Name: name}
		for path, assigned := range m.assignments {
			if assigned == key {
				group.Paths = append(group.Paths, path)
			}
		}
		sort.Strings(group.Paths)
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups
}

// RestoreCohorts overlays saved explicit membership before analysis delivery.
// Invalid names and overlapping members in edited metadata are ignored.
func (m *Map) RestoreCohorts(groups []favstore.Cohort) {
	m.assignments, m.cohortNames = map[string]string{}, map[string]string{}
	for _, group := range groups {
		if !m.CohortNameAvailable(group.Name) || len(group.Paths) == 0 {
			continue
		}
		key := cohortKey(group.Name)
		for _, path := range group.Paths {
			if path != "" && m.assignments[path] == "" {
				m.assignments[path] = key
				m.cohortNames[key] = group.Name
			}
		}
	}
	m.rebuild()
}

func cohortKey(name string) string {
	return fmt.Sprintf("manual-%x", sha256.Sum256([]byte(strings.TrimSpace(name))))
}

// SharedTraits accepts only distinct sources currently in Unassigned. A frozen
// grid may outlive its map, so a stale source invalidates the whole selection.
func (m *Map) SharedTraits(paths []string) []Trait {
	selected := make(map[string]bool, len(paths))
	for _, path := range paths {
		selected[path] = true
	}
	if len(selected) < 2 {
		return nil
	}
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, item := range m.items {
		if !selected[item.Path] || seen[item.Path] {
			continue
		}
		if item.Error != "" || item.Cohort != "unassigned" || m.assignments[item.Path] != "" {
			return nil
		}
		seen[item.Path] = true
		tags := slices.Clone(item.Tags)
		slices.Sort(tags)
		for _, tag := range slices.Compact(tags) {
			if tag != "" && tagLabel(tag) != "" {
				counts[tag]++
			}
		}
	}
	if len(seen) != len(selected) {
		return nil
	}
	var traits []Trait
	for tag, count := range counts {
		if count == len(selected) {
			traits = append(traits, Trait{ID: tag, Label: tagLabel(tag)})
		}
	}
	sort.Slice(traits, func(i, j int) bool { return traits[i].Label < traits[j].Label })
	return traits
}

// MatchUnassigned returns unique current sources carrying every chosen trait.
func (m *Map) MatchUnassigned(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	for _, tag := range tags {
		if tag == "" || tagLabel(tag) == "" {
			return nil
		}
	}
	var paths []string
	seen := map[string]bool{}
	for _, item := range m.items {
		if item.Error != "" || item.Cohort != "unassigned" || m.assignments[item.Path] != "" || seen[item.Path] {
			continue
		}
		matches := true
		for _, tag := range tags {
			matches = matches && slices.Contains(item.Tags, tag)
		}
		if matches {
			paths = append(paths, item.Path)
			seen[item.Path] = true
		}
	}
	sort.Strings(paths)
	return paths
}

// CohortNameAvailable applies the same name contract used at creation.
func (m *Map) CohortNameAvailable(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 || strings.ContainsAny(name, "\r\n\t") {
		return false
	}
	for _, existing := range m.cohortNames {
		if strings.EqualFold(existing, name) {
			return false
		}
	}
	return true
}

// CreateCohort captures exactly the reviewed sources. New publications cannot
// silently expand this set or return its surviving members to automatic groups.
// A source moved out of Unassigned since review makes the whole request stale.
func (m *Map) CreateCohort(name string, paths, tags []string) bool {
	if !m.CohortNameAvailable(name) {
		return false
	}
	paths = slices.Clone(paths)
	sort.Strings(paths)
	paths = slices.Compact(paths)
	if len(paths) < 2 {
		return false
	}
	available := m.MatchUnassigned(tags)
	availableSet := make(map[string]bool, len(available))
	for _, path := range available {
		availableSet[path] = true
	}
	for _, path := range paths {
		if !availableSet[path] {
			return false
		}
	}
	name = strings.TrimSpace(name)
	key := cohortKey(name)
	if m.assignments == nil {
		m.assignments = map[string]string{}
		m.cohortNames = map[string]string{}
	}
	for _, path := range paths {
		m.assignments[path] = key
	}
	m.cohortNames[key] = name
	for _, tag := range tags {
		m.tagChoices[tag] = true
	}
	m.selectedSource = paths[0]
	m.rebuild()
	for _, pile := range m.piles {
		if slices.Contains(pile.members, paths[0]) {
			m.center = pile.world
			break
		}
	}
	m.Refresh()
	return true
}
