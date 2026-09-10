package explorer

import (
	"slices"
	"sort"
	"strings"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/similarity"
)

// PresetReview freezes the facts and membership a user is asked to approve.
// Matching fills Matches on a worker; applying never admits later arrivals.
type PresetReview struct {
	Preset              explorerpresets.Preset
	Candidates, Matches []similarity.Item
	Members             []string
}

func (m *Map) isUnassigned(item similarity.Item) bool {
	return m.assignments[item.Path] == "" && (item.Cohort == "unassigned" || m.unassignedSources[item.Path])
}

func (m *Map) presetKey(id string) string {
	if id != "" {
		for key, linked := range m.presetIDs {
			if linked == id {
				return key
			}
		}
	}
	return ""
}

func (m *Map) HasPreset(id string) bool { return m.presetKey(id) != "" }

// DetachPreset keeps exact current membership after its reusable rule is deleted.
func (m *Map) DetachPreset(id string) { delete(m.presetIDs, m.presetKey(id)) }

func (m *Map) presetMembers(id string) []string {
	key := m.presetKey(id)
	var members []string
	if key != "" {
		for path, assigned := range m.assignments {
			if assigned == key {
				members = append(members, path)
			}
		}
	}
	sort.Strings(members)
	return members
}

// PreparePreset captures facts without retaining any preview pixels.
func (m *Map) PreparePreset(p explorerpresets.Preset) PresetReview {
	review := PresetReview{Preset: p, Members: m.presetMembers(p.ID)}
	seen := map[string]bool{}
	key := m.presetKey(p.ID)
	for _, item := range m.items {
		if item.Error != "" || seen[item.Path] || (!m.isUnassigned(item) && (key == "" || m.assignments[item.Path] != key)) {
			continue
		}
		seen[item.Path] = true
		item.Preview, item.Embedding, item.Position = nil, nil, nil
		item.Tags = slices.Clone(item.Tags)
		review.Candidates = append(review.Candidates, item)
	}
	return review
}

// ApplyPreset installs the reviewed change, preserving not-yet-analyzed members.
func (m *Map) ApplyPreset(review PresetReview) bool {
	p := review.Preset
	key := m.presetKey(p.ID)
	if !p.Compatible() || p.Rule.Validate() != nil || !explorerpresets.ValidName(p.Name) || (key == "" && len(review.Matches) < 2) || !slices.Equal(review.Members, m.presetMembers(p.ID)) {
		return false
	}
	for other, name := range m.cohortNames {
		if other != key && strings.EqualFold(name, p.Name) {
			return false
		}
	}
	current := map[string]similarity.Item{}
	for _, item := range m.PreparePreset(p).Candidates {
		current[item.Path] = item
	}
	matched := map[string]bool{}
	for _, item := range review.Matches {
		if matched[item.Path] {
			return false
		}
		matched[item.Path] = true
	}
	members := map[string]bool{}
	for _, path := range review.Members {
		members[path] = true
	}
	evaluated := map[string]bool{}
	for _, item := range review.Candidates {
		evaluated[item.Path] = true
		if !members[item.Path] && !matched[item.Path] {
			continue
		}
		now, ok := current[item.Path]
		if !ok || now.Size != item.Size || now.ModifiedNS != item.ModifiedNS || now.SHA256 != item.SHA256 || now.Facts != item.Facts || p.Rule.Matches(now) != matched[item.Path] {
			return false
		}
	}
	for path := range matched {
		if !evaluated[path] {
			return false
		}
	}
	if key == "" {
		key = "preset-" + p.ID
	}
	if m.assignments == nil {
		m.assignments = map[string]string{}
		m.cohortNames = map[string]string{}
	}
	if m.presetIDs == nil {
		m.presetIDs = map[string]string{}
	}
	if m.unassignedSources == nil {
		m.unassignedSources = map[string]bool{}
	}
	retained := len(review.Matches)
	for _, path := range review.Members {
		if !evaluated[path] {
			retained++
			continue
		}
		if !matched[path] {
			delete(m.assignments, path)
			m.unassignedSources[path] = true
		}
	}
	for path := range matched {
		m.assignments[path] = key
		delete(m.unassignedSources, path)
	}
	if retained == 0 {
		delete(m.cohortNames, key)
		delete(m.presetIDs, key)
	} else {
		m.presetIDs[key], m.cohortNames[key] = p.ID, p.Name
	}
	for _, tag := range p.Rule.Tags {
		m.tagChoices[tag] = true
	}
	if len(review.Matches) > 0 {
		m.selectedSource = review.Matches[0].Path
	}
	m.rebuild()
	for _, pile := range m.piles {
		if slices.Contains(pile.members, m.selectedSource) {
			m.center = pile.world
			break
		}
	}
	m.Refresh()
	return true
}
