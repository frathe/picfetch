// Package explorer owns the content map and its camera. Analysis and transitions
// to Grid View belong to its host; the surface itself starts no background work.
package explorer

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image/color"
	"image/jpeg"
	"math"
	"slices"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/similarity"
)

// Host owns transitions while Map retains its view across those transitions.
type Host interface {
	OpenSimilarityCohort([]string)
	OpenSimilarityUnassigned([]string)
	LeaveSimilarityMap()
	UpdateSimilarityMap()
	SetSimilarityAutoUpdate(bool)
	ShowSimilarityPresets()
	Unfocus()
	Modifiers() fyne.KeyModifier
}

// Map is a pannable, zoomable collection of cohort piles.
type Map struct {
	widget.BaseWidget
	host               Host
	overlay            *fyne.Container
	status             *widget.Label
	unassigned         *widget.Button
	update             *widget.Button
	automatic          *widget.Check
	scene              *fyne.Container
	piles              []*Pile
	center             fyne.Position
	zoom               float32
	tagRows            *fyne.Container
	tagChecks          []*widget.Check
	tagChoices         map[string]bool
	unassignedTags     []string
	selectedSource     string
	granularity        *widget.Slider
	appliedGranularity float64
	items              []similarity.Item
	merges             []similarity.CohortMerge
	assignments        map[string]string
	cohortNames        map[string]string
	presetIDs          map[string]string
	unassignedSources  map[string]bool
}

// View is source-free camera geometry. Piles counts all current cohort piles,
// including those hidden by tag filters; that count determines the zoom floor.
type View struct {
	Piles             int
	Zoom, MinimumZoom float32
	Center            fyne.Position
	Size              fyne.Size
}

// View captures the map on UI, without retaining its images or memberships.
func (m *Map) View() View {
	return View{Piles: len(m.piles), Zoom: m.zoom, MinimumZoom: m.minimumZoom(), Center: m.center, Size: m.Size()}
}

func New(host Host) *Map {
	m := &Map{host: host, zoom: 1, scene: container.NewWithoutLayout()}
	m.ExtendBaseWidget(m)
	m.status = widget.NewLabel("")
	m.status.Truncation = fyne.TextTruncateEllipsis
	m.unassigned = widget.NewButton(lang.L("Unassigned"), nil)
	m.unassigned.Hide()
	toolbar := container.NewBorder(nil, nil, widget.NewButton(lang.L("Back to Viewer"), host.LeaveSimilarityMap), container.NewHBox(
		widget.NewButton(lang.L("Presets"), host.ShowSimilarityPresets), m.unassigned, widget.NewButton(lang.L("-"), func() { m.scale(1/1.2, fyne.NewPos(m.Size().Width/2, m.Size().Height/2)) }), widget.NewButton(lang.L("+"), func() { m.scale(1.2, fyne.NewPos(m.Size().Width/2, m.Size().Height/2)) }), widget.NewButton(lang.L("Fit map"), m.Fit)), m.status)
	m.update = widget.NewButton(lang.L("Update map"), host.UpdateSimilarityMap)
	m.update.Disable()
	m.automatic = widget.NewCheck(lang.L("Auto-update every 30 images"), host.SetSimilarityAutoUpdate)
	m.tagRows = container.NewVBox()
	tagScroll := container.NewVScroll(m.tagRows)
	tagScroll.SetMinSize(fyne.NewSize(160, 200))
	tagHeading := container.NewVBox(widget.NewLabel(lang.L("Tags")), container.NewHBox(
		widget.NewButton(lang.L("All tags"), func() { m.setAllTags(true) }),
		widget.NewButton(lang.L("Clear tags"), func() { m.setAllTags(false) })))
	tags := container.NewBorder(tagHeading, nil, nil, nil, tagScroll)
	var toggleTags *widget.Button
	toggleTags = widget.NewButton(lang.L("Hide tags"), func() {
		if tags.Visible() {
			tags.Hide()
			toggleTags.SetText(lang.L("Show tags"))
		} else {
			tags.Show()
			toggleTags.SetText(lang.L("Hide tags"))
		}
		m.host.Unfocus()
		m.overlay.Refresh()
	})
	toolbar = container.NewVBox(toolbar, container.NewHBox(toggleTags, m.update, m.automatic))
	m.granularity = widget.NewSlider(0, 100)
	m.granularity.Step = 1
	m.granularity.Value, m.appliedGranularity = 100, 100
	m.granularity.OnChangeEnded = func(value float64) {
		m.host.Unfocus()
		if value == m.appliedGranularity {
			return
		}
		m.appliedGranularity = value
		m.rebuildLayout(false)
	}
	granularity := container.NewVBox(widget.NewLabel(lang.L("Granularity")), container.NewBorder(nil, nil,
		widget.NewLabel(lang.L("Broader")), widget.NewLabel(lang.L("Finer")),
		container.NewGridWrap(fyne.NewSize(160, m.granularity.MinSize().Height), m.granularity)))
	toolbar = container.NewBorder(nil, nil, nil, granularity, toolbar)
	m.overlay = container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameBackground)), container.NewBorder(toolbar, nil, tags, nil, m))
	m.overlay.Hide()
	return m
}

func (m *Map) Overlay() fyne.CanvasObject { return m.overlay }
func (m *Map) Show()                      { m.overlay.Show() }
func (m *Map) Hide()                      { m.overlay.Hide() }
func (m *Map) Visible() bool              { return m.overlay.Visible() }
func (m *Map) Status(text string)         { m.status.SetText(text) }

// SetAutomatic synchronizes the toolbar without emitting a setting change.
func (m *Map) SetAutomatic(on bool) { m.automatic.Checked = on; m.automatic.Refresh() }

// UpdateState reflects whether new representations can be added to the map.
func (m *Map) UpdateState(available, busy bool) {
	text := lang.L("Update map")
	if busy {
		text = lang.L("Updating map...")
	}
	m.update.SetText(text)
	if available && !busy {
		m.update.Enable()
	} else {
		m.update.Disable()
	}
}

// SetResult replaces cohort membership without moving the user's camera.
func (m *Map) SetResult(items []similarity.Item, merges []similarity.CohortMerge) {
	if items == nil {
		m.assignments, m.cohortNames = nil, nil
		m.presetIDs = nil
		m.unassignedSources = nil
		m.tagChoices = nil
		m.selectedSource = ""
		m.granularity.Value, m.appliedGranularity = 100, 100
		m.granularity.Refresh()
	}
	m.items, m.merges = items, merges
	m.rebuild()
}

func (m *Map) rebuild() { m.rebuildLayout(true) }

func (m *Map) rebuildLayout(preservePositions bool) {
	roots := map[string]string{}
	var root func(string) string
	root = func(key string) string {
		parent, ok := roots[key]
		if !ok || parent == key {
			return key
		}
		roots[key] = root(parent)
		return roots[key]
	}
	limit := int(float64(len(m.merges)) * (100 - m.appliedGranularity) / 100)
	for _, merge := range m.merges[:limit] {
		left, right := root(merge.Left), root(merge.Right)
		if right < left {
			left, right = right, left
		}
		roots[right] = left
	}
	groups := map[string][]similarity.Item{}
	var unassigned []string
	assignedSeen := map[string]bool{}
	m.unassignedTags = nil
	for _, item := range m.items {
		if item.Error != "" {
			continue
		}
		if key := m.assignments[item.Path]; key != "" {
			if assignedSeen[item.Path] {
				continue
			}
			assignedSeen[item.Path] = true
			item.Cohort = key
		} else if m.unassignedSources[item.Path] {
			item.Cohort = "unassigned"
		}
		if item.Cohort == "unassigned" {
			unassigned = append(unassigned, item.Path)
			m.unassignedTags = append(m.unassignedTags, itemTags(item)...)
			continue
		}
		if item.Cohort != "" {
			key := root(item.Cohort)
			groups[key] = append(groups[key], item)
		}
	}
	m.unassigned.SetText(fmt.Sprintf(lang.L("Unassigned (%d)"), len(unassigned)))
	m.unassigned.OnTapped = func() { m.host.OpenSimilarityUnassigned(unassigned) }
	if len(unassigned) > 0 {
		m.unassigned.Show()
	} else {
		m.unassigned.Hide()
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	previous := m.piles
	m.piles = nil
	m.scene.RemoveAll()
	for _, key := range keys {
		members := groups[key]
		sort.Slice(members, func(i, j int) bool { return members[i].Path < members[j].Path })
		p := newPile(m, members)
		name := m.cohortNames[key]
		if name == "" {
			name = subjectTitle(members)
		}
		if name != "" {
			label := fmt.Sprintf(lang.L("%s (%d)"), name, len(members))
			runes := []rune(name)
			for len(runes) > 1 && fyne.MeasureText(label, 16, fyne.TextStyle{}).Width > 280 {
				runes = runes[:len(runes)-1]
				label = fmt.Sprintf(lang.L("%s (%d)"), string(runes)+"...", len(members))
			}
			p.label.Text = label
		}
		m.piles = append(m.piles, p)
		m.scene.Add(p)
	}
	orientPiles(m.piles, m.Size())
	m.normalizeSpacing()
	if preservePositions {
		anchorPiles(m.piles, previous)
	} else {
		placePiles(m.piles)
		m.centerLayout()
	}
	m.zoom = max(m.minimumZoom(), m.zoom)
	m.setTags(m.items)
	m.filterTags()
	// Fyne can retain removed widgets in its renderer caches. Release their
	// encoded/decoded pixels explicitly and invalidate each image texture.
	for _, pile := range previous {
		for _, picture := range pile.pictures {
			picture.File, picture.Resource, picture.Image = "", nil, nil
			picture.Refresh()
		}
		pile.members, pile.pictures, pile.aspects = nil, nil, nil
		pile.tags = nil
	}
	m.Refresh()
}

// Explicit granularity changes start a fresh arrangement at the current zoom.
// Keep the selected source's group in view, or center the new arrangement.
func (m *Map) centerLayout() {
	m.center = fyne.Position{}
	if len(m.piles) == 0 {
		return
	}
	lo, hi := m.piles[0].world, m.piles[0].world
	for _, pile := range m.piles {
		if m.selectedSource != "" && slices.Contains(pile.members, m.selectedSource) {
			m.center = pile.world
			return
		}
		lo.X, lo.Y = min(lo.X, pile.world.X), min(lo.Y, pile.world.Y)
		hi.X, hi.Y = max(hi.X, pile.world.X), max(hi.Y, pile.world.Y)
	}
	m.center = fyne.NewPos((lo.X+hi.X)/2, (lo.Y+hi.Y)/2)
}

func (m *Map) Fit() {
	m.zoom = 1
	m.center = fyne.Position{}
	if len(m.piles) > 0 {
		lo, hi := m.piles[0].world, m.piles[0].world
		for _, p := range m.piles {
			lo.X = min(lo.X, p.world.X)
			lo.Y = min(lo.Y, p.world.Y)
			hi.X = max(hi.X, p.world.X)
			hi.Y = max(hi.Y, p.world.Y)
		}
		m.center = fyne.NewPos((lo.X+hi.X)/2, (lo.Y+hi.Y)/2)
		m.zoom = max(m.minimumZoom(), min(m.Size().Width/(hi.X-lo.X+pileWidth+20), m.Size().Height/(hi.Y-lo.Y+pileHeight+30)))
	}
	m.Refresh()
}

// Maps with more than 100 cohort piles need a tighter zoom floor.
// Small maps retain their original zoom range; every pile keeps all its samples.
func (m *Map) minimumZoom() float32 {
	if len(m.piles) > 100 {
		return .5
	}
	return .03
}

// ExpandToFit adds room for discovered piles down to the map's zoom floor.
// At that floor, discovery preserves the camera instead of chasing new piles.
func (m *Map) ExpandToFit() {
	size := m.Size()
	half := fyne.NewPos(size.Width/(2*m.zoom), size.Height/(2*m.zoom))
	lo, hi := m.center.Subtract(half), m.center.Add(half)
	beforeLo, beforeHi := lo, hi
	for _, p := range m.piles {
		lo.X, lo.Y = min(lo.X, p.world.X-pileWidth/2-10), min(lo.Y, p.world.Y-pileHeight/2-15)
		hi.X, hi.Y = max(hi.X, p.world.X+pileWidth/2+10), max(hi.Y, p.world.Y+pileHeight/2+15)
	}
	if lo == beforeLo && hi == beforeHi {
		return
	}
	next := max(m.minimumZoom(), min(m.zoom, size.Width/(hi.X-lo.X), size.Height/(hi.Y-lo.Y)))
	if next == m.zoom {
		return
	}
	m.zoom = next
	m.center = fyne.NewPos((lo.X+hi.X)/2, (lo.Y+hi.Y)/2)
	m.Refresh()
}

func (m *Map) Dragged(e *fyne.DragEvent) {
	m.center.X -= e.Dragged.DX / m.zoom
	m.center.Y -= e.Dragged.DY / m.zoom
	m.Refresh()
}
func (m *Map) DragEnd() {}
func (m *Map) Scrolled(e *fyne.ScrollEvent) {
	if m.host.Modifiers()&fyne.KeyModifierShift != 0 {
		m.center.X -= e.Scrolled.DX / m.zoom
		m.center.Y -= e.Scrolled.DY / m.zoom
		m.Refresh()
		return
	}
	m.scale(float32(math.Exp(float64(e.Scrolled.DY)/180)), e.Position)
}
func (m *Map) scale(factor float32, at fyne.Position) {
	next := max(m.minimumZoom(), min(8, m.zoom*factor))
	if next == m.zoom {
		return
	}
	delta := at.Subtract(fyne.NewPos(m.Size().Width/2, m.Size().Height/2))
	m.center.X += delta.X * (1/m.zoom - 1/next)
	m.center.Y += delta.Y * (1/m.zoom - 1/next)
	m.zoom = next
	m.Refresh()
}
func (m *Map) CreateRenderer() fyne.WidgetRenderer {
	return &mapRenderer{m: m, bg: canvas.NewRectangle(theme.Color(theme.ColorNameBackground)), clip: container.NewClip(m.scene)}
}

type mapRenderer struct {
	m    *Map
	bg   *canvas.Rectangle
	clip *container.Clip
}

func (r *mapRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.clip.Resize(size)
	for _, p := range r.m.piles {
		pileSize := fyne.NewSize(pileWidth*r.m.zoom, pileHeight*r.m.zoom)
		pos := fyne.NewPos(size.Width/2+(p.world.X-r.m.center.X-pileWidth/2)*r.m.zoom, size.Height/2+(p.world.Y-r.m.center.Y-pileHeight/2)*r.m.zoom)
		// Decode just beyond the viewport; retain a wider margin so small
		// back-and-forth pans do not repeatedly discard and decode the edge.
		margin := pileSize.Width
		if p.nearViewport {
			margin *= 2
		}
		near := p.Visible() && pos.X+pileSize.Width >= -margin && pos.Y+pileSize.Height >= -margin && pos.X <= size.Width+margin && pos.Y <= size.Height+margin
		changed := p.nearViewport != near
		p.nearViewport = near
		if changed && !near {
			for _, picture := range p.pictures {
				// Keep the encoded source and its identity, but release decoded
				// pixels and invalidate any texture without re-reading the JPEG.
				picture.Image = nil
				canvas.Refresh(picture)
			}
		}
		p.Move(pos)
		p.Resize(pileSize)
		if changed {
			p.Refresh()
		}
	}
}
func (r *mapRenderer) MinSize() fyne.Size { return fyne.NewSize(400, 300) }
func (r *mapRenderer) Refresh() {
	r.bg.FillColor = theme.Color(theme.ColorNameBackground)
	r.bg.Refresh()
	r.Layout(r.m.Size())
	canvas.Refresh(r.m)
}
func (r *mapRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.clip} }
func (r *mapRenderer) Destroy()                     {}

// Pile is a tappable sample of a cohort. Its full membership is opened on tap.
type Pile struct {
	widget.BaseWidget
	owner        *Map
	members      []string
	pictures     []*canvas.Image
	aspects      []float32
	world        fyne.Position
	label        *canvas.Text
	tags         []string
	selected     bool
	nearViewport bool
}

func newPile(m *Map, items []similarity.Item) *Pile {
	p := &Pile{owner: m}
	p.ExtendBaseWidget(p)
	for _, item := range items {
		p.members = append(p.members, item.Path)
		p.tags = append(p.tags, itemTags(item)...)
		if len(item.Position) == 2 {
			p.world.X += item.Position[0]
			p.world.Y += item.Position[1]
		}
	}
	p.world.X /= float32(len(items))
	p.world.Y /= float32(len(items))
	// Stable seeded sampling without replacement. Membership changes are the
	// only input to sample selection; camera changes never resample.
	sampled := append([]similarity.Item(nil), items...)
	sort.Slice(sampled, func(i, j int) bool {
		a := sha256.Sum256([]byte(sampled[i].Path))
		b := sha256.Sum256([]byte(sampled[j].Path))
		return string(a[:]) < string(b[:])
	})
	seen := make(map[string]bool, min(15, len(sampled)))
	for _, item := range sampled {
		if seen[item.Path] {
			continue
		}
		seen[item.Path] = true
		// Previews are JPEG even for SVG sources. Fyne also uses the resource
		// name to select its decoder, so it must describe the preview's format.
		img := canvas.NewImageFromResource(fyne.NewStaticResource(item.Path+".preview.jpg", item.Preview))
		img.FillMode = canvas.ImageFillContain
		p.pictures = append(p.pictures, img)
		aspect := float32(1)
		if config, err := jpeg.DecodeConfig(bytes.NewReader(item.Preview)); err == nil && config.Height > 0 {
			aspect = float32(config.Width) / float32(config.Height)
		}
		p.aspects = append(p.aspects, aspect)
		if len(p.pictures) == 15 {
			break
		}
	}
	label := fmt.Sprintf(lang.L("%d images"), len(items))
	if len(items) == 1 {
		label = lang.L("1 image")
	}
	p.label = canvas.NewText(label, color.White)
	p.label.Alignment = fyne.TextAlignCenter
	return p
}
func (p *Pile) Tapped(_ *fyne.PointEvent) {
	if !p.selected {
		p.owner.selectedSource = p.members[0]
		p.owner.syncSelection()
	}
	p.owner.host.OpenSimilarityCohort(append([]string(nil), p.members...))
}
func (p *Pile) Dragged(e *fyne.DragEvent) { p.owner.Dragged(e) }
func (p *Pile) DragEnd()                  {}
func (p *Pile) Scrolled(e *fyne.ScrollEvent) {
	scroll := *e
	scroll.Position = scroll.Position.Add(p.Position())
	p.owner.Scrolled(&scroll)
}
func (p *Pile) CreateRenderer() fyne.WidgetRenderer {
	objects := make([]fyne.CanvasObject, 0, 2*len(p.pictures)+2)
	for _, img := range p.pictures {
		objects = append(objects, canvas.NewRectangle(color.NRGBA{R: 230, G: 232, B: 237, A: 255}), img)
	}
	objects = append(objects, canvas.NewRectangle(color.NRGBA{R: 35, G: 39, B: 48, A: 240}), p.label)
	highlight := canvas.NewRectangle(color.Transparent)
	highlight.StrokeWidth = 3
	highlight.StrokeColor = theme.Color(theme.ColorNamePrimary)
	return &pileRenderer{p: p, objects: objects, highlight: highlight}
}

type pileRenderer struct {
	p         *Pile
	objects   []fyne.CanvasObject
	highlight *canvas.Rectangle
}

func (r *pileRenderer) Layout(size fyne.Size) {
	if !r.p.nearViewport {
		return
	}
	r.highlight.Move(fyne.NewPos(2, 2))
	r.highlight.Resize(fyne.NewSize(max(0, size.Width-4), max(0, size.Height-4)))
	if r.p.selected {
		r.highlight.Show()
	} else {
		r.highlight.Hide()
	}
	scale := size.Width / pileWidth
	for i, img := range r.p.pictures {
		// A sunflower pattern spreads samples without a rigid thumbnail grid.
		// Membership supplies the stable sample order, so panning never reshuffles.
		angle := float64(i) * 2.399963229728653
		radius := math.Sqrt((float64(i) + .5) / float64(len(r.p.pictures)))
		x := pileWidth/2 + float32(math.Cos(angle)*radius)*130
		y := float32(136) + float32(math.Sin(angle)*radius)*86
		w := min(float32(110), 90*r.p.aspects[i])
		h := w / r.p.aspects[i]
		pos := fyne.NewPos((x-w/2)*scale, (y-h/2)*scale)
		sz := fyne.NewSize(w*scale, h*scale)
		frame := scale
		r.objects[i*2].Move(pos.Subtract(fyne.NewPos(frame, frame)))
		r.objects[i*2].Resize(sz.Add(fyne.NewSize(2*frame, 2*frame)))
		img.Move(pos)
		img.Resize(sz)
		if img.Image == nil {
			img.Refresh()
		}
	}
	for _, o := range r.objects[len(r.objects)-2:] {
		o.Move(fyne.NewPos(40*scale, 278*scale))
		o.Resize(fyne.NewSize(300*scale, 24*scale))
	}
	r.p.label.TextSize = 16 * scale
}
func (r *pileRenderer) MinSize() fyne.Size { return fyne.NewSize(24, 19) }
func (r *pileRenderer) Refresh() {
	if !r.p.nearViewport {
		canvas.Refresh(r.p)
		return
	}
	r.highlight.StrokeColor = theme.Color(theme.ColorNamePrimary)
	r.Layout(r.p.Size())
	r.highlight.Refresh()
	for _, o := range r.objects {
		// Selection and theme refreshes do not change preview sources. Redraw
		// their existing pixels without asking canvas.Image to decode again.
		canvas.Refresh(o)
	}
}
func (r *pileRenderer) Objects() []fyne.CanvasObject {
	if !r.p.nearViewport {
		return nil
	}
	return append(r.objects, r.highlight)
}
func (r *pileRenderer) Destroy() {}

// Projection coordinates have no fixed scale. Normalize cohort centers before
// applying the camera so one fitted map cannot shrink its piles to a few pixels
// merely because a fresh UMAP fit used a larger numeric range.
func (m *Map) normalizeSpacing() {
	if len(m.piles) == 0 {
		return
	}
	lo, hi := m.piles[0].world, m.piles[0].world
	for _, p := range m.piles {
		lo.X = min(lo.X, p.world.X)
		lo.Y = min(lo.Y, p.world.Y)
		hi.X = max(hi.X, p.world.X)
		hi.Y = max(hi.Y, p.world.Y)
	}
	span := max(hi.X-lo.X, hi.Y-lo.Y)
	extent := float32(math.Sqrt(float64(len(m.piles)))) * (pileWidth + pileGap)
	for _, p := range m.piles {
		if span == 0 {
			p.world = fyne.Position{}
			continue
		}
		p.world.X = (p.world.X - (lo.X+hi.X)/2) / span * extent
		p.world.Y = (p.world.Y - (lo.Y+hi.Y)/2) / span * extent
	}
}
