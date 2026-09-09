package explorer

import (
	"math"
	"slices"

	"fyne.io/fyne/v2"
)

// HandleKey keeps zoom and directional stack navigation on the map surface.
func (m *Map) HandleKey(key fyne.KeyName) {
	switch key {
	case fyne.KeyPlus, fyne.KeyEqual:
		m.scale(1.2, fyne.NewPos(m.Size().Width/2, m.Size().Height/2))
	case fyne.KeyMinus:
		m.scale(1/1.2, fyne.NewPos(m.Size().Width/2, m.Size().Height/2))
	case fyne.KeyReturn, fyne.KeyEnter:
		for _, pile := range m.piles {
			if pile.Visible() && pile.selected {
				pile.Tapped(nil)
				return
			}
		}
	case fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeyDown:
		m.selectDirection(key)
	}
}

func (m *Map) syncSelection() {
	found := false
	for _, pile := range m.piles {
		selected := pile.Visible() && m.selectedSource != "" && slices.Contains(pile.members, m.selectedSource)
		found = found || selected
		if pile.selected != selected {
			pile.selected = selected
			pile.Refresh()
		}
	}
	if !found {
		m.selectedSource = ""
	}
}

func (m *Map) selectDirection(key fyne.KeyName) {
	origin, active := m.center, false
	for _, pile := range m.piles {
		if pile.Visible() && pile.selected {
			origin, active = pile.world, true
			break
		}
	}
	var next *Pile
	best := float32(math.Inf(1))
	for _, pile := range m.piles {
		if !pile.Visible() || pile.selected {
			continue
		}
		delta := pile.world.Subtract(origin)
		score := delta.X*delta.X + delta.Y*delta.Y
		if active {
			forward := delta.X
			switch key {
			case fyne.KeyLeft:
				forward = -delta.X
			case fyne.KeyUp:
				forward = -delta.Y
			case fyne.KeyDown:
				forward = delta.Y
			}
			if forward <= 0 {
				continue
			}
			score /= forward
		}
		if score < best {
			next, best = pile, score
		}
	}
	if next == nil {
		return
	}
	m.selectedSource = next.members[0]
	m.syncSelection()
	// Move only far enough to expose the active stack, retaining the zoom.
	half := fyne.NewPos(m.Size().Width/(2*m.zoom), m.Size().Height/(2*m.zoom))
	margin := fyne.NewPos(min(pileWidth/2+8, half.X), min(pileHeight/2+8, half.Y))
	m.center.X = max(next.world.X-half.X+margin.X, min(m.center.X, next.world.X+half.X-margin.X))
	m.center.Y = max(next.world.Y-half.Y+margin.Y, min(m.center.Y, next.world.Y+half.Y-margin.Y))
	m.Refresh()
}
