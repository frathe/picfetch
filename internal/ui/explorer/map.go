// Package explorer owns the content map and its camera. Analysis and transitions
// to Grid View belong to its host; the surface itself starts no background work.
package explorer

import (
	"crypto/sha256"
	"fmt"
	"image/color"
	"math"
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
	LeaveSimilarityMap()
}

// Map is a pannable, zoomable collection of cohort piles.
type Map struct {
	widget.BaseWidget
	host       Host
	overlay    *fyne.Container
	status     *widget.Label
	unassigned *widget.Button
	scene      *fyne.Container
	piles      []*Pile
	center     fyne.Position
	zoom       float32
}

func New(host Host) *Map {
	m := &Map{host: host, zoom: 1, scene: container.NewWithoutLayout()}
	m.ExtendBaseWidget(m)
	m.status = widget.NewLabel("")
	m.status.Truncation = fyne.TextTruncateEllipsis
	m.unassigned = widget.NewButton(lang.L("Unassigned"), nil)
	m.unassigned.Hide()
	toolbar := container.NewBorder(nil, nil, widget.NewButton(lang.L("Back to Viewer"), host.LeaveSimilarityMap), container.NewHBox(
		m.unassigned, widget.NewButton(lang.L("-"), func() { m.scale(1/1.2, fyne.NewPos(m.Size().Width/2, m.Size().Height/2)) }), widget.NewButton(lang.L("+"), func() { m.scale(1.2, fyne.NewPos(m.Size().Width/2, m.Size().Height/2)) }), widget.NewButton(lang.L("Fit map"), m.Fit)), m.status)
	m.overlay = container.NewStack(canvas.NewRectangle(theme.BackgroundColor()), container.NewBorder(toolbar, nil, nil, nil, m))
	m.overlay.Hide()
	return m
}

func (m *Map) Overlay() fyne.CanvasObject { return m.overlay }
func (m *Map) Show()                      { m.overlay.Show() }
func (m *Map) Hide()                      { m.overlay.Hide() }
func (m *Map) Visible() bool              { return m.overlay.Visible() }
func (m *Map) Status(text string)         { m.status.SetText(text) }

// SetResult replaces cohort membership without moving the user's camera.
func (m *Map) SetResult(items []similarity.Item) {
	groups := map[string][]similarity.Item{}
	var unassigned []string
	for _, item := range items {
		if item.Error != "" {
			continue
		}
		if item.Cohort == "unassigned" {
			unassigned = append(unassigned, item.Path)
			continue
		}
		if item.Cohort != "" {
			groups[item.Cohort] = append(groups[item.Cohort], item)
		}
	}
	m.unassigned.SetText(fmt.Sprintf(lang.L("Unassigned (%d)"), len(unassigned)))
	m.unassigned.OnTapped = func() { m.host.OpenSimilarityCohort(unassigned) }
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
	m.piles = nil
	m.scene.RemoveAll()
	for _, key := range keys {
		members := groups[key]
		sort.Slice(members, func(i, j int) bool { return members[i].Path < members[j].Path })
		p := newPile(m, members)
		m.piles = append(m.piles, p)
		m.scene.Add(p)
	}
	m.normalizeSpacing()
	m.Refresh()
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
		m.zoom = max(.03, min(m.Size().Width/(hi.X-lo.X+260), m.Size().Height/(hi.Y-lo.Y+220)))
	}
	m.Refresh()
}

func (m *Map) Dragged(e *fyne.DragEvent) {
	m.center.X -= e.Dragged.DX / m.zoom
	m.center.Y -= e.Dragged.DY / m.zoom
	m.Refresh()
}
func (m *Map) DragEnd() {}
func (m *Map) Scrolled(e *fyne.ScrollEvent) {
	m.scale(float32(math.Exp(float64(e.Scrolled.DY)/180)), e.Position)
}
func (m *Map) scale(factor float32, at fyne.Position) {
	next := max(.03, min(8, m.zoom*factor))
	delta := at.Subtract(fyne.NewPos(m.Size().Width/2, m.Size().Height/2))
	m.center.X += delta.X * (1/m.zoom - 1/next)
	m.center.Y += delta.Y * (1/m.zoom - 1/next)
	m.zoom = next
	m.Refresh()
}
func (m *Map) CreateRenderer() fyne.WidgetRenderer {
	return &mapRenderer{m: m, bg: canvas.NewRectangle(theme.BackgroundColor()), clip: container.NewClip(m.scene)}
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
		p.Resize(fyne.NewSize(240*r.m.zoom, 190*r.m.zoom))
		p.Move(fyne.NewPos(size.Width/2+(p.world.X-r.m.center.X-120)*r.m.zoom, size.Height/2+(p.world.Y-r.m.center.Y-95)*r.m.zoom))
	}
}
func (r *mapRenderer) MinSize() fyne.Size { return fyne.NewSize(400, 300) }
func (r *mapRenderer) Refresh() {
	r.bg.FillColor = theme.BackgroundColor()
	r.bg.Refresh()
	r.Layout(r.m.Size())
	canvas.Refresh(r.m)
}
func (r *mapRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.clip} }
func (r *mapRenderer) Destroy()                     {}

// Pile is a tappable sample of a cohort. Its full membership is opened on tap.
type Pile struct {
	widget.BaseWidget
	owner    *Map
	members  []string
	pictures []*canvas.Image
	world    fyne.Position
	label    *canvas.Text
}

func newPile(m *Map, items []similarity.Item) *Pile {
	p := &Pile{owner: m}
	p.ExtendBaseWidget(p)
	for _, item := range items {
		p.members = append(p.members, item.Path)
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
		img := canvas.NewImageFromResource(fyne.NewStaticResource(item.Path, item.Preview))
		img.FillMode = canvas.ImageFillContain
		p.pictures = append(p.pictures, img)
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
	p.owner.host.OpenSimilarityCohort(append([]string(nil), p.members...))
}
func (p *Pile) Dragged(e *fyne.DragEvent) { p.owner.Dragged(e) }
func (p *Pile) DragEnd()                  {}
func (p *Pile) Scrolled(e *fyne.ScrollEvent) {
	copy := *e
	copy.Position = copy.Position.Add(p.Position())
	p.owner.Scrolled(&copy)
}
func (p *Pile) CreateRenderer() fyne.WidgetRenderer {
	objects := make([]fyne.CanvasObject, 0, 2*len(p.pictures)+2)
	for _, img := range p.pictures {
		objects = append(objects, canvas.NewRectangle(color.NRGBA{R: 230, G: 232, B: 237, A: 255}), img)
	}
	objects = append(objects, canvas.NewRectangle(color.NRGBA{R: 35, G: 39, B: 48, A: 240}), p.label)
	return &pileRenderer{p: p, objects: objects}
}

type pileRenderer struct {
	p       *Pile
	objects []fyne.CanvasObject
}

func (r *pileRenderer) Layout(size fyne.Size) {
	scale := size.Width / 240
	for i, img := range r.p.pictures {
		hash := sha256.Sum256([]byte(img.Resource.Name()))
		x := float32(hash[0] % 110)
		y := float32(hash[1] % 76)
		pos := fyne.NewPos(x*scale, y*scale)
		sz := fyne.NewSize(124*scale, 90*scale)
		r.objects[i*2].Move(pos)
		r.objects[i*2].Resize(sz)
		img.Move(pos.Add(fyne.NewPos(3*scale, 3*scale)))
		img.Resize(sz.Subtract(fyne.NewSize(6*scale, 6*scale)))
	}
	for _, o := range r.objects[len(r.objects)-2:] {
		o.Move(fyne.NewPos(30*scale, 165*scale))
		o.Resize(fyne.NewSize(180*scale, 24*scale))
	}
	r.p.label.TextSize = max(9, 14*scale)
}
func (r *pileRenderer) MinSize() fyne.Size { return fyne.NewSize(24, 19) }
func (r *pileRenderer) Refresh() {
	r.Layout(r.p.Size())
	for _, o := range r.objects {
		o.Refresh()
	}
}
func (r *pileRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *pileRenderer) Destroy()                     {}

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
	extent := float32(math.Sqrt(float64(len(m.piles)))) * 280
	for _, p := range m.piles {
		if span == 0 {
			p.world = fyne.Position{}
			continue
		}
		p.world.X = (p.world.X - (lo.X+hi.X)/2) / span * extent
		p.world.Y = (p.world.Y - (lo.Y+hi.Y)/2) / span * extent
	}
}
