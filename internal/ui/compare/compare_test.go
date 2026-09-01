package compare

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

const waitTimeout = 5 * time.Second

func newReferenceFeature(loader Loader, callbacks Callbacks) *Feature {
	return newFeature(loader, callbacks, newCanvasPaneRenderer)
}

func walk(root fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(root)
	switch c := root.(type) {
	case *fyne.Container:
		for _, child := range c.Objects {
			walk(child, visit)
		}
	case *container.Clip:
		walk(c.Content, visit)
	}
}

func comparisonButton(t *testing.T, root fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	var found *widget.Button
	walk(root, func(object fyne.CanvasObject) {
		if button, ok := object.(*widget.Button); ok && button.Text == text {
			found = button
		}
	})
	if found == nil {
		t.Fatalf("comparison overlay has no %q button", text)
	}
	return found
}

func backButton(t *testing.T, root fyne.CanvasObject) *widget.Button {
	t.Helper()
	return comparisonButton(t, root, "Back to Grid")
}

func containsButton(root fyne.CanvasObject, text string) bool {
	found := false
	walk(root, func(object fyne.CanvasObject) {
		if button, ok := object.(*widget.Button); ok && button.Text == text {
			found = true
		}
	})
	return found
}

func buttonCard(t *testing.T, root fyne.CanvasObject, text string) (*fyne.Container, *canvas.Rectangle) {
	t.Helper()
	var card *fyne.Container
	var background *canvas.Rectangle
	walk(root, func(object fyne.CanvasObject) {
		candidate, ok := object.(*fyne.Container)
		if !ok || !containsButton(candidate, text) {
			return
		}
		for _, child := range candidate.Objects {
			if rectangle, ok := child.(*canvas.Rectangle); ok && rectangle.CornerRadius > 0 {
				card, background = candidate, rectangle
			}
		}
	})
	if card == nil {
		t.Fatalf("comparison %q button is not inside a rounded card", text)
	}
	return card, background
}

func waitForDone(t *testing.T, feature *Feature) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	if err := feature.Done().Wait(ctx); err != nil {
		t.Fatal("timed out waiting for comparison workers")
	}
}

func loadedImage(width, height int) *imaging.LoadedImage {
	return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, width, height))}}
}

type boundsCountingImage struct {
	image.Image
	calls atomic.Int64
}

func (i *boundsCountingImage) Bounds() image.Rectangle {
	i.calls.Add(1)
	return i.Image.Bounds()
}

func solidLoadedImage(width, height int, fill color.Color) *imaging.LoadedImage {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: fill}, image.Point{}, draw.Src)
	return &imaging.LoadedImage{Frames: []image.Image{img}}
}

func loadedVector(t *testing.T, width, height int) *imaging.LoadedImage {
	t.Helper()
	loaded, err := imaging.DecodeLoaded(context.Background(), uitest.SVGBytes(width, height), imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("decode SVG fixture: %v", err)
	}
	return loaded
}

func waitStarted(t *testing.T, started <-chan string) string {
	t.Helper()
	select {
	case name := <-started:
		return name
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for comparison loader to start")
		return ""
	}
}

type renderedPane struct {
	root   *fyne.Container
	image  *canvas.Image
	origin fyne.Position
}

func renderedPanes(root fyne.CanvasObject) []renderedPane {
	var panes []renderedPane
	var visit func(fyne.CanvasObject, fyne.Position)
	visit = func(object fyne.CanvasObject, parent fyne.Position) {
		origin := parent.Add(object.Position())
		candidate, ok := object.(*fyne.Container)
		if ok && len(candidate.Objects) == 2 {
			var images []*canvas.Image
			spinners := 0
			walk(candidate, func(descendant fyne.CanvasObject) {
				if img, ok := descendant.(*canvas.Image); ok {
					images = append(images, img)
				}
				if _, ok := descendant.(*widget.ProgressBarInfinite); ok {
					spinners++
				}
			})
			if len(images) == 1 && spinners == 1 {
				panes = append(panes, renderedPane{root: candidate, image: images[0], origin: origin})
			}
		}

		switch object := object.(type) {
		case *fyne.Container:
			for _, child := range object.Objects {
				visit(child, origin)
			}
		case *container.Clip:
			visit(object.Content, origin)
		}
	}
	visit(root, fyne.Position{})
	return panes
}

func paneScrollable(t *testing.T, pane renderedPane) fyne.Scrollable {
	t.Helper()
	var found fyne.Scrollable
	walk(pane.root, func(object fyne.CanvasObject) {
		if scrollable, ok := object.(fyne.Scrollable); ok {
			found = scrollable
		}
	})
	if found == nil {
		t.Fatal("comparison pane has no scrollable view")
	}
	return found
}

func paneDraggable(t *testing.T, pane renderedPane) fyne.Draggable {
	t.Helper()
	var found fyne.Draggable
	walk(pane.root, func(object fyne.CanvasObject) {
		if draggable, ok := object.(fyne.Draggable); ok {
			found = draggable
		}
	})
	if found == nil {
		t.Fatal("comparison pane has no draggable view")
	}
	return found
}

func paneCursorable(t *testing.T, pane renderedPane) desktop.Cursorable {
	t.Helper()
	var found desktop.Cursorable
	walk(pane.root, func(object fyne.CanvasObject) {
		if cursorable, ok := object.(desktop.Cursorable); ok {
			found = cursorable
		}
	})
	if found == nil {
		t.Fatal("comparison pane has no cursorable view")
	}
	return found
}

func paneHoverable(t *testing.T, pane renderedPane) desktop.Hoverable {
	t.Helper()
	var found desktop.Hoverable
	walk(pane.root, func(object fyne.CanvasObject) {
		if hoverable, ok := object.(desktop.Hoverable); ok {
			found = hoverable
		}
	})
	if found == nil {
		t.Fatal("comparison pane has no hoverable view")
	}
	return found
}

func horizontalResizeTarget(t *testing.T, root fyne.CanvasObject) (fyne.CanvasObject, fyne.Draggable) {
	t.Helper()
	var object fyne.CanvasObject
	var draggable fyne.Draggable
	walk(root, func(candidate fyne.CanvasObject) {
		cursorable, ok := candidate.(desktop.Cursorable)
		if !ok || cursorable.Cursor() != desktop.HResizeCursor {
			return
		}
		if drag, ok := candidate.(fyne.Draggable); ok {
			object, draggable = candidate, drag
		}
	})
	if object == nil {
		t.Fatal("swipe comparison has no horizontal divider drag target")
	}
	return object, draggable
}

func normalizedCenter(pane renderedPane) fyne.Position {
	return fyne.NewPos(
		(pane.root.Size().Width/2-pane.image.Position().X)/pane.image.Size().Width,
		(pane.root.Size().Height/2-pane.image.Position().Y)/pane.image.Size().Height,
	)
}

func normalizedPoint(pane renderedPane, point fyne.Position) fyne.Position {
	return fyne.NewPos(
		(point.X-pane.image.Position().X)/pane.image.Size().Width,
		(point.Y-pane.image.Position().Y)/pane.image.Size().Height,
	)
}

func approxPosition(a, b fyne.Position) bool {
	return uitest.ApproxEqual(a.X, b.X) && uitest.ApproxEqual(a.Y, b.Y)
}

func assertPaneOverlapsCenter(t *testing.T, pane renderedPane) {
	t.Helper()
	position, size, viewport := pane.image.Position(), pane.image.Size(), pane.root.Size()
	center := fyne.NewPos(viewport.Width/2, viewport.Height/2)
	if position.X > center.X+0.01 || position.X+size.Width < center.X-0.01 ||
		position.Y > center.Y+0.01 || position.Y+size.Height < center.Y-0.01 {
		t.Errorf("image span %v..%v does not overlap pane center %v", position,
			fyne.NewPos(position.X+size.Width, position.Y+size.Height), center)
	}
}

func labelTexts(root fyne.CanvasObject) []string {
	var texts []string
	walk(root, func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Visible() {
			texts = append(texts, label.Text)
		}
	})
	return texts
}

func containsLabel(root fyne.CanvasObject, text string) bool {
	found := false
	walk(root, func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Text == text {
			found = true
		}
	})
	return found
}

func labelCard(t *testing.T, root fyne.CanvasObject, text string) (*fyne.Container, *canvas.Rectangle) {
	t.Helper()
	var card *fyne.Container
	var background *canvas.Rectangle
	walk(root, func(object fyne.CanvasObject) {
		candidate, ok := object.(*fyne.Container)
		if !ok || !containsLabel(candidate, text) {
			return
		}
		for _, child := range candidate.Objects {
			if rectangle, ok := child.(*canvas.Rectangle); ok && rectangle.CornerRadius > 0 {
				card, background = candidate, rectangle
			}
		}
	})
	if card == nil {
		t.Fatalf("comparison identity %q is not inside a rounded card", text)
	}
	return card, background
}

func commonLabelRow(t *testing.T, root fyne.CanvasObject, left, right string) *fyne.Container {
	t.Helper()
	var row *fyne.Container
	walk(root, func(object fyne.CanvasObject) {
		candidate, ok := object.(*fyne.Container)
		if ok && containsLabel(candidate, left) && containsLabel(candidate, right) {
			row = candidate
		}
	})
	if row == nil {
		t.Fatal("comparison identities have no shared row")
	}
	return row
}

func TestCompareOverlay_ClosedSurfaceDoesNotChangeHostMinimumSize(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(nil, Callbacks{})
	if got := feature.Overlay().MinSize(); got != (fyne.Size{}) {
		t.Fatalf("closed comparison overlay MinSize = %v, want zero so it cannot resize the host stack", got)
	}
	spinners := 0
	walk(feature.Overlay(), func(object fyne.CanvasObject) {
		spinner, ok := object.(*widget.ProgressBarInfinite)
		if !ok {
			return
		}
		spinners++
		if spinner.Visible() || spinner.Running() {
			t.Errorf("closed comparison spinner = {Visible:%v Running:%v}, want inert", spinner.Visible(), spinner.Running())
		}
	})
	if spinners != 2 {
		t.Fatalf("comparison spinners = %d, want 2", spinners)
	}
}

func TestCompareOverlay_OpensImmediatelyAndBackExitsWhileLoading(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan struct{}, 2)
	loader := func(ctx context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		started <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	feature := newReferenceFeature(loader, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(640, 480))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})

	if !feature.Visible() || !feature.Overlay().Visible() {
		t.Fatal("Open must reveal the comparison overlay before either loader finishes")
	}
	var backdrop *canvas.Rectangle
	walk(feature.Overlay(), func(object fyne.CanvasObject) {
		if rectangle, ok := object.(*canvas.Rectangle); ok && rectangle.Size() == feature.Overlay().Size() {
			backdrop = rectangle
		}
	})
	if backdrop == nil || backdrop.Size() != feature.Overlay().Size() {
		t.Fatalf("comparison backdrop = %v, want one full-surface rectangle", backdrop)
	}
	if _, _, _, alpha := backdrop.FillColor.RGBA(); alpha != 0xffff {
		t.Errorf("comparison backdrop alpha = %#x, want opaque", alpha)
	}
	spinners := 0
	walk(feature.Overlay(), func(object fyne.CanvasObject) {
		if spinner, ok := object.(*widget.ProgressBarInfinite); ok && spinner.Visible() {
			spinners++
		}
	})
	if spinners != 2 {
		t.Errorf("visible loading spinners = %d, want one per pane", spinners)
	}

	test.Tap(backButton(t, feature.Overlay()))
	if feature.Visible() || feature.Overlay().Visible() {
		t.Fatal("Back to Grid must close only the comparison overlay")
	}
	waitForDone(t, feature)
}

func TestCompareToolbar_SwapIsPermanentAndReadyGated(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan string, 2)
	release := make(chan struct{})
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, Callbacks{})
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitStarted(t, started)
	waitStarted(t, started)

	swap := comparisonButton(t, feature.Overlay(), "Swap")
	back := backButton(t, feature.Overlay())
	if !swap.Visible() || !back.Visible() {
		t.Fatalf("loading toolbar buttons visible = Swap:%v Back:%v, want both permanent", swap.Visible(), back.Visible())
	}
	if !swap.Disabled() {
		t.Error("Swap is enabled before both images are ready")
	}
	if back.Disabled() {
		t.Error("Back to Grid is disabled while images are loading")
	}

	close(release)
	waitForDone(t, feature)
	if swap.Disabled() {
		t.Error("Swap stayed disabled after both images became ready")
	}
}

func TestCompareLinkControl_TopLeftCardAndReadyGate(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan string, 2)
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(640, 480))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitStarted(t, started)
	waitStarted(t, started)

	unlink := comparisonButton(t, feature.Overlay(), "Unlink")
	if !unlink.Visible() {
		t.Fatal("Unlink is not a permanent comparison control")
	}
	if !unlink.Disabled() {
		t.Error("Unlink is enabled before both images are ready")
	}

	linkCard, linkBackground := buttonCard(t, feature.Overlay(), "Unlink")
	swapCard, _ := buttonCard(t, feature.Overlay(), "Swap")
	if linkCard == swapCard {
		t.Fatal("Unlink and Swap share one toolbar card, want separate cards")
	}
	if _, _, _, alpha := linkBackground.FillColor.RGBA(); alpha == 0 || alpha == 0xffff {
		t.Errorf("link card background alpha = %#x, want partial translucency", alpha)
	}
	if linkCard.Size().Width >= feature.Overlay().Size().Width/2 || linkCard.Size().Height >= feature.Overlay().Size().Height/2 {
		t.Errorf("link card size = %v in %v overlay, want compact", linkCard.Size(), feature.Overlay().Size())
	}
	if left := linkCard.Position().X; left < 0 || left > theme.Padding() {
		t.Errorf("link card left gap = %v, want 0..%v", left, theme.Padding())
	}
	if top := linkCard.Position().Y; top < 0 || top > theme.Padding() {
		t.Errorf("link card top gap = %v, want 0..%v", top, theme.Padding())
	}

	if !containsButton(swapCard, "Back to Grid") {
		t.Fatal("Swap card no longer contains Back to Grid")
	}
	if containsButton(swapCard, "Unlink") {
		t.Error("Swap card contains Unlink, want it only in the top-left card")
	}
	if containsLabel(swapCard, "Unlinked") {
		t.Error("Swap card contains the Unlinked label, want it only in the top-left card")
	}
	if gap := feature.Overlay().Size().Width - swapCard.Position().X - swapCard.Size().Width; gap < 0 || gap > theme.Padding() {
		t.Errorf("Swap card right gap = %v, want 0..%v", gap, theme.Padding())
	}
	if top := swapCard.Position().Y; top < 0 || top > theme.Padding() {
		t.Errorf("Swap card top gap = %v, want 0..%v", top, theme.Padding())
	}

	close(release)
	waitForDone(t, feature)
	if got := comparisonButton(t, feature.Overlay(), "Unlink"); got != unlink {
		t.Fatal("Unlink control was replaced when comparison became ready")
	}
	if unlink.Disabled() {
		t.Error("Unlink stayed disabled after both images became ready")
	}
}

func TestCompareSwipeToggle_PermanentReadyGatedAndRelabels(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan string, 2)
	release := make(chan struct{})
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, Callbacks{})
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitStarted(t, started)
	waitStarted(t, started)

	swipe := comparisonButton(t, feature.Overlay(), "Swipe")
	if !swipe.Visible() {
		t.Fatal("Swipe is not a permanent comparison toolbar control")
	}
	if !swipe.Disabled() {
		t.Error("Swipe is enabled before both images are ready")
	}

	close(release)
	waitForDone(t, feature)
	if swipe.Disabled() {
		t.Fatal("Swipe stayed disabled after both images became ready")
	}

	test.Tap(swipe)
	if got := comparisonButton(t, feature.Overlay(), "Side by side"); got != swipe {
		t.Fatal("layout toggle was replaced instead of relabeled in place")
	}

	test.Tap(swipe)
	if got := comparisonButton(t, feature.Overlay(), "Swipe"); got != swipe {
		t.Fatal("layout toggle did not return to its side-by-side label")
	}
}

func TestCompareSwipeLayout_UsesAlignedFullViewportImagesAndKeepsChrome(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	left := solidLoadedImage(800, 400, color.RGBA{R: 255, A: 255})
	right := solidLoadedImage(600, 300, color.RGBA{B: 255, A: 255})
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))

	panes := renderedPanes(feature.Overlay())
	if len(panes) != 2 {
		t.Fatalf("rendered panes in swipe = %d, want 2", len(panes))
	}
	for i, pane := range panes {
		if pane.root.Size() != fyne.NewSize(800, 400) {
			t.Errorf("swipe pane %d viewport = %v, want full 800x400", i, pane.root.Size())
		}
		if pane.image.Size() != fyne.NewSize(800, 400) {
			t.Errorf("swipe pane %d image = %v, want fitted full-viewport 800x400", i, pane.image.Size())
		}
	}

	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	dividerCenter := divider.Position().X + divider.Size().Width/2
	if !uitest.ApproxEqual(dividerCenter, 400) || divider.Size().Height != 400 {
		t.Errorf("swipe divider = x %.2f height %.2f, want x 400 across full height", dividerCenter, divider.Size().Height)
	}

	captured := win.Canvas().Capture()
	if got := color.RGBAModel.Convert(captured.At(100, 200)); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("left reveal pixel = %v, want logical left red image", got)
	}
	if got := color.RGBAModel.Convert(captured.At(700, 200)); got != (color.RGBA{B: 255, A: 255}) {
		t.Errorf("right reveal pixel = %v, want logical right blue image", got)
	}

	card, _ := buttonCard(t, feature.Overlay(), "Side by side")
	if gap := feature.Overlay().Size().Width - card.Position().X - card.Size().Width; gap < 0 || gap > theme.Padding() {
		t.Errorf("swipe toolbar right gap = %v, want 0..%v", gap, theme.Padding())
	}
	row := commonLabelRow(t, feature.Overlay(), "left.png", "right.png")
	if row.Position().Y+row.Size().Height != feature.Overlay().Size().Height {
		t.Errorf("swipe badge row bottom = %v, want overlay bottom %v", row.Position().Y+row.Size().Height, feature.Overlay().Size().Height)
	}
}

func TestCompareSwipePointer_DividerDragChangesRevealWithoutPanning(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	for range 5 {
		feature.HandleKey(fyne.KeyPlus)
	}

	before := renderedPanes(feature.Overlay())
	if len(before) != 2 {
		t.Fatalf("rendered panes before divider drag = %d, want 2", len(before))
	}
	beforePositions := [2]fyne.Position{before[0].image.Position(), before[1].image.Position()}
	beforeCenters := [2]fyne.Position{normalizedCenter(before[0]), normalizedCenter(before[1])}
	divider, dividerDrag := horizontalResizeTarget(t, feature.Overlay())
	dividerDrag.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(80, 0)})
	dividerDrag.DragEnd()

	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 480) {
		t.Fatalf("divider center after +80 drag = %.2f, want 480", center)
	}
	afterDivider := renderedPanes(feature.Overlay())
	for i, pane := range afterDivider {
		if pane.image.Position() != beforePositions[i] || !approxPosition(normalizedCenter(pane), beforeCenters[i]) {
			t.Errorf("pane %d changed transform during divider drag: position %v center %v, want %v and %v",
				i, pane.image.Position(), normalizedCenter(pane), beforePositions[i], beforeCenters[i])
		}
	}
	afterDividerPositions := [2]fyne.Position{afterDivider[0].image.Position(), afterDivider[1].image.Position()}

	paneDraggable(t, afterDivider[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	afterPan := renderedPanes(feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 480) {
		t.Errorf("divider center after image pan = %.2f, want unchanged 480", center)
	}
	if afterPan[0].image.Position() == afterDividerPositions[0] {
		t.Fatal("dragging away from the divider did not pan the images")
	}
	if left, right := normalizedCenter(afterPan[0]), normalizedCenter(afterPan[1]); !approxPosition(left, right) {
		t.Errorf("linked centers after pane drag = %v and %v, want equal", left, right)
	}
}

func TestCompareSwipePointer_DividerDragDoesNotRefreshStaticContent(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	frames := [2]*boundsCountingImage{
		{Image: image.NewRGBA(image.Rect(0, 0, 1600, 800))},
		{Image: image.NewRGBA(image.Rect(0, 0, 1600, 800))},
	}
	var repaints atomic.Int64
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		index := 0
		if uri.Name() == "right.png" {
			index = 1
		}
		return &imaging.LoadedImage{Frames: []image.Image{frames[index]}}, nil
	}, Callbacks{Repaint: func() { repaints.Add(1) }})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))

	panes := renderedPanes(feature.Overlay())
	if len(panes) != 2 {
		t.Fatalf("rendered panes before divider drag = %d, want 2", len(panes))
	}
	beforePositions := [2]fyne.Position{panes[0].image.Position(), panes[1].image.Position()}
	divider, dividerDrag := horizontalResizeTarget(t, feature.Overlay())
	repaints.Store(0)
	for _, frame := range frames {
		frame.calls.Store(0)
	}

	for range 100 {
		dividerDrag.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(1, 0)})
	}
	dividerDrag.DragEnd()

	if got := repaints.Load(); got != 0 {
		t.Errorf("owner repaints during 100 divider events = %d, want 0", got)
	}
	for i, frame := range frames {
		if got := frame.calls.Load(); got != 0 {
			t.Errorf("pane %d image bounds probes during divider events = %d, want 0", i, got)
		}
	}
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 500) {
		t.Errorf("divider center after 100 one-pixel events = %.2f, want 500", center)
	}
	panes = renderedPanes(feature.Overlay())
	for i, pane := range panes {
		if pane.image.Position() != beforePositions[i] {
			t.Errorf("pane %d position during divider events = %v, want unchanged %v",
				i, pane.image.Position(), beforePositions[i])
		}
	}
}

func TestCompareInteraction_PanAndZoomDoNotRepaintOwner(t *testing.T) {
	for _, layout := range []struct {
		name  string
		swipe bool
	}{
		{name: "side by side"},
		{name: "swipe", swipe: true},
	} {
		t.Run(layout.name, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)

			var repaints atomic.Int64
			feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
				return loadedImage(2400, 1600), nil
			}, Callbacks{Repaint: func() { repaints.Add(1) }})
			win := test.NewWindow(feature.Overlay())
			win.SetPadded(false)
			win.Resize(fyne.NewSize(800, 400))
			t.Cleanup(win.Close)
			feature.Open([2]fyne.URI{
				storage.NewFileURI("left.png"),
				storage.NewFileURI("right.png"),
			})
			waitForDone(t, feature)
			if layout.swipe {
				test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
			}
			feature.HandleKey(fyne.KeyPlus)
			panes := renderedPanes(feature.Overlay())
			if len(panes) != 2 {
				t.Fatalf("rendered panes = %d, want 2", len(panes))
			}
			draggable := paneDraggable(t, panes[0])
			scrollable := paneScrollable(t, panes[0])
			repaints.Store(0)

			for range 50 {
				draggable.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(1, 1)})
				scrollable.Scrolled(&fyne.ScrollEvent{
					PointEvent: fyne.PointEvent{Position: fyne.NewPos(200, 200)},
					Scrolled:   fyne.NewDelta(0, 0.25),
				})
			}

			if got := repaints.Load(); got != 0 {
				t.Errorf("owner repaints during 100 pan/zoom events = %d, want 0", got)
			}
		})
	}
}

func TestCompareDividerKeys_StepClampAndNoopSideBySide(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	var modifiers fyne.KeyModifier
	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{Modifiers: func() fyne.KeyModifier { return modifiers }})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.HandleKey(fyne.KeyRight)
	feature.HandleKey(fyne.KeyHome)
	feature.HandleKey(fyne.KeyEnd)
	feature.HandleKey(fyne.KeyLeft)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	assertDividerCenter := func(want float32) {
		t.Helper()
		got := divider.Position().X + divider.Size().Width/2
		if !uitest.ApproxEqual(got, want) {
			t.Fatalf("divider center = %.2f, want %.2f", got, want)
		}
	}
	assertDividerCenter(400)

	feature.HandleKey(fyne.KeyRight)
	assertDividerCenter(440)
	modifiers = fyne.KeyModifierShift
	feature.HandleKey(fyne.KeyRight)
	assertDividerCenter(448)
	feature.HandleKey(fyne.KeyLeft)
	assertDividerCenter(440)
	modifiers = 0
	feature.HandleKey(fyne.KeyHome)
	assertDividerCenter(0)
	feature.HandleKey(fyne.KeyLeft)
	assertDividerCenter(0)
	feature.HandleKey(fyne.KeyEnd)
	assertDividerCenter(800)
	feature.HandleKey(fyne.KeyRight)
	assertDividerCenter(800)
}

func TestCompareLinkToggle_DividerDragDoesNotRelinkPanes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))

	_, divider := horizontalResizeTarget(t, feature.Overlay())
	divider.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 0)})
	if got := labelTexts(feature.Overlay()); !slices.Contains(got, "Unlinked") {
		t.Errorf("labels after unlinked divider drag = %v, want Unlinked", got)
	}
}

func TestCompareLayoutToggle_PreservesCameraAcrossRoundTripsAndDivider(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	for range 4 {
		feature.HandleKey(fyne.KeyPlus)
	}
	before := renderedPanes(feature.Overlay())
	paneDraggable(t, before[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	before = renderedPanes(feature.Overlay())
	wantCenter := normalizedCenter(before[0])
	wantMultiplier := before[0].image.Size().Width / 400

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	inSwipe := renderedPanes(feature.Overlay())
	wantSwipeCenter := normalizedCenter(inSwipe[0])
	if multiplier := inSwipe[0].image.Size().Width / 800; !uitest.ApproxEqual(multiplier, wantMultiplier) {
		t.Errorf("fit-relative multiplier after entering swipe = %.4f, want retained %.4f", multiplier, wantMultiplier)
	}
	feature.HandleKey(fyne.KeyRight)

	test.Tap(comparisonButton(t, feature.Overlay(), "Side by side"))
	backSideBySide := renderedPanes(feature.Overlay())
	if center := normalizedCenter(backSideBySide[0]); !approxPosition(center, wantCenter) {
		t.Errorf("side-by-side center after layout round trip = %v, want %v", center, wantCenter)
	}
	if multiplier := backSideBySide[0].image.Size().Width / 400; !uitest.ApproxEqual(multiplier, wantMultiplier) {
		t.Errorf("side-by-side multiplier after layout round trip = %.4f, want %.4f", multiplier, wantMultiplier)
	}
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 440) {
		t.Errorf("divider center after layout round trip = %.2f, want retained 440", center)
	}
	afterRoundTrip := renderedPanes(feature.Overlay())
	if center := normalizedCenter(afterRoundTrip[0]); !approxPosition(center, wantSwipeCenter) {
		t.Errorf("swipe center after layout round trip = %v, want %v", center, wantSwipeCenter)
	}
	if multiplier := afterRoundTrip[0].image.Size().Width / 800; !uitest.ApproxEqual(multiplier, wantMultiplier) {
		t.Errorf("fit-relative multiplier after layout round trip = %.4f, want retained %.4f", multiplier, wantMultiplier)
	}
}

func TestCompareLayoutTransition_PreservesPhotoAndCameraStateOnRoundTrip(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(1600, 800), nil
		}
		return loadedImage(600, 1200), nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}

	before := renderedPanes(feature.Overlay())
	paneDraggable(t, before[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	before = renderedPanes(feature.Overlay())
	wantSideCenters := [2]fyne.Position{normalizedCenter(before[0]), normalizedCenter(before[1])}
	wantMultiplier := before[0].image.Size().Width / 400
	if approxPosition(wantSideCenters[0], fyne.NewPos(0.5, 0.5)) || wantMultiplier <= 1 {
		t.Fatalf("transition setup = {center:%v multiplier:%.4f}, want a panned zoomed view", wantSideCenters[0], wantMultiplier)
	}

	assertState := func(name string, fitted [2]fyne.Size, wantCenters [2]fyne.Position) {
		t.Helper()
		panes := renderedPanes(feature.Overlay())
		if len(panes) != 2 {
			t.Fatalf("%s rendered panes = %d, want 2", name, len(panes))
		}
		for i, pane := range panes {
			if center := normalizedCenter(pane); !approxPosition(center, wantCenters[i]) {
				t.Errorf("%s pane %d center = %v, want retained %v", name, i, center, wantCenters[i])
			}
			widthMultiplier := pane.image.Size().Width / fitted[i].Width
			heightMultiplier := pane.image.Size().Height / fitted[i].Height
			if !uitest.ApproxEqual(widthMultiplier, wantMultiplier) || !uitest.ApproxEqual(heightMultiplier, wantMultiplier) {
				t.Errorf("%s pane %d fit multipliers = %.4f x %.4f, want retained %.4f",
					name, i, widthMultiplier, heightMultiplier, wantMultiplier)
			}
		}
	}

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	inSwipe := renderedPanes(feature.Overlay())
	wantSwipeCenters := [2]fyne.Position{normalizedCenter(inSwipe[0]), normalizedCenter(inSwipe[1])}
	assertState("swipe", [2]fyne.Size{fyne.NewSize(800, 400), fyne.NewSize(200, 400)}, wantSwipeCenters)
	feature.HandleKey(fyne.KeyRight)

	test.Tap(comparisonButton(t, feature.Overlay(), "Side by side"))
	assertState("side by side", [2]fyne.Size{fyne.NewSize(400, 200), fyne.NewSize(200, 400)}, wantSideCenters)

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	assertState("swipe round trip", [2]fyne.Size{fyne.NewSize(800, 400), fyne.NewSize(200, 400)}, wantSwipeCenters)
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 440) {
		t.Errorf("divider after layout round trip = %.2f, want retained 440", center)
	}
}

func TestCompareResize_PreservesPhotoAndCameraStateOnRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		swipe                   bool
		proportionalFitted      fyne.Size
		wideSize                fyne.Size
		wantProportionalDivider float32
		wantWideDivider         float32
	}{
		{
			name:               "side by side",
			proportionalFitted: fyne.NewSize(500, 250),
			wideSize:           fyne.NewSize(2400, 400),
		},
		{
			name:                    "swipe",
			swipe:                   true,
			proportionalFitted:      fyne.NewSize(1000, 500),
			wideSize:                fyne.NewSize(2000, 400),
			wantProportionalDivider: 550,
			wantWideDivider:         1100,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)

			feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
				if uri.Name() == "large.png" {
					return loadedImage(1600, 800), nil
				}
				return loadedImage(1200, 600), nil
			}, Callbacks{})
			win := test.NewWindow(feature.Overlay())
			win.SetPadded(false)
			win.Resize(fyne.NewSize(800, 400))
			t.Cleanup(win.Close)
			feature.Open([2]fyne.URI{
				storage.NewFileURI("large.png"),
				storage.NewFileURI("small.png"),
			})
			waitForDone(t, feature)
			if tc.swipe {
				test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
				feature.HandleKey(fyne.KeyRight)
			}
			feature.HandleKey(fyne.KeyPlus)
			feature.HandleKey(fyne.KeyPlus)

			panes := renderedPanes(feature.Overlay())
			paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
			panes = renderedPanes(feature.Overlay())
			wantInitial := [2]struct {
				size     fyne.Size
				position fyne.Position
			}{
				{size: panes[0].image.Size(), position: panes[0].image.Position()},
				{size: panes[1].image.Size(), position: panes[1].image.Position()},
			}

			assertRoundTrip := func(name string) {
				t.Helper()
				panes := renderedPanes(feature.Overlay())
				for i, pane := range panes {
					if !uitest.ApproxEqual(pane.image.Size().Width, wantInitial[i].size.Width) ||
						!uitest.ApproxEqual(pane.image.Size().Height, wantInitial[i].size.Height) ||
						!approxPosition(pane.image.Position(), wantInitial[i].position) {
						t.Errorf("%s pane %d geometry = {%v %v}, want {%v %v}", name, i,
							pane.image.Size(), pane.image.Position(), wantInitial[i].size, wantInitial[i].position)
					}
				}
			}

			win.Resize(fyne.NewSize(1000, 500))
			for i, pane := range renderedPanes(feature.Overlay()) {
				wantSize := fyne.NewSize(tc.proportionalFitted.Width*1.5625, tc.proportionalFitted.Height*1.5625)
				if !uitest.ApproxEqual(pane.image.Size().Width, wantSize.Width) ||
					!uitest.ApproxEqual(pane.image.Size().Height, wantSize.Height) {
					t.Errorf("proportional resize pane %d size = %v, want %v", i, pane.image.Size(), wantSize)
				}
			}
			if tc.swipe {
				divider, _ := horizontalResizeTarget(t, feature.Overlay())
				if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, tc.wantProportionalDivider) {
					t.Errorf("proportional resize divider = %.2f, want %.2f", center, tc.wantProportionalDivider)
				}
			}

			win.Resize(fyne.NewSize(800, 400))
			assertRoundTrip("proportional round trip")

			win.Resize(tc.wideSize)
			if tc.swipe {
				divider, _ := horizontalResizeTarget(t, feature.Overlay())
				if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, tc.wantWideDivider) {
					t.Errorf("wide resize divider = %.2f, want %.2f", center, tc.wantWideDivider)
				}
			}
			win.Resize(fyne.NewSize(800, 400))
			assertRoundTrip("wide round trip")
		})
	}
}

func TestCompareCameraHome_PreservesDivergentPhotoPosesAcrossTransitions(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "large.png" {
			return loadedImage(1600, 800), nil
		}
		return loadedImage(1200, 600), nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("large.png"),
		storage.NewFileURI("small.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	paneHoverable(t, renderedPanes(feature.Overlay())[1]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyMinus)
	feature.ToggleLink()

	type geometry struct {
		size     fyne.Size
		position fyne.Position
	}
	capture := func() [2]geometry {
		panes := renderedPanes(feature.Overlay())
		return [2]geometry{
			{size: panes[0].image.Size(), position: panes[0].image.Position()},
			{size: panes[1].image.Size(), position: panes[1].image.Position()},
		}
	}
	assertGeometry := func(name string, want [2]geometry) {
		t.Helper()
		panes := renderedPanes(feature.Overlay())
		for i, pane := range panes {
			if !uitest.ApproxEqual(pane.image.Size().Width, want[i].size.Width) ||
				!uitest.ApproxEqual(pane.image.Size().Height, want[i].size.Height) ||
				!approxPosition(pane.image.Position(), want[i].position) {
				t.Errorf("%s pane %d geometry = {%v %v}, want {%v %v}", name, i,
					pane.image.Size(), pane.image.Position(), want[i].size, want[i].position)
			}
		}
	}

	homeSideBySide := capture()
	feature.HandleKey(fyne.KeyPlus)
	paneDraggable(t, renderedPanes(feature.Overlay())[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(30, 15)})
	feature.HandleKey(fyne.Key1)
	assertGeometry("side-by-side camera home", homeSideBySide)

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	homeSwipe := capture()
	feature.HandleKey(fyne.KeyPlus)
	paneDraggable(t, renderedPanes(feature.Overlay())[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-20, 10)})
	feature.HandleKey(fyne.Key1)
	assertGeometry("swipe camera home", homeSwipe)

	win.Resize(fyne.NewSize(1000, 500))
	homeResized := capture()
	feature.HandleKey(fyne.KeyMinus)
	paneDraggable(t, renderedPanes(feature.Overlay())[1]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(15, -10)})
	feature.HandleKey(fyne.Key1)
	assertGeometry("resized camera home", homeResized)
}

func TestCompareDividerReset_NewOpenStartsSideBySideAtHalf(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	open := func(left, right string) {
		t.Helper()
		feature.Open([2]fyne.URI{storage.NewFileURI(left), storage.NewFileURI(right)})
		waitForDone(t, feature)
	}
	open("first-left.png", "first-right.png")
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	feature.HandleKey(fyne.KeyRight)
	feature.HandleKey(fyne.KeyRight)
	feature.Close()

	open("next-left.png", "next-right.png")
	if !containsButton(feature.Overlay(), "Swipe") || containsButton(feature.Overlay(), "Side by side") {
		t.Fatal("new comparison did not reset to side-by-side layout")
	}
	feature.HandleKey(fyne.KeyRight)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 400) {
		t.Errorf("new comparison divider center = %.2f, want reset 400", center)
	}
}

func TestCompareSessionReset_StartsCanonicalState(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	images := map[string]*imaging.LoadedImage{
		"first-left.png":  loadedImage(1600, 800),
		"first-right.png": loadedImage(1200, 600),
		"next-left.png":   loadedImage(800, 400),
		"next-right.png":  loadedImage(200, 800),
	}
	var orders [][2]string
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		return images[uri.Name()], nil
	}, Callbacks{OrderChanged: func(left, right string) {
		orders = append(orders, [2]string{left, right})
	}})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	open := func(left, right string) {
		t.Helper()
		feature.Open([2]fyne.URI{storage.NewFileURI(left), storage.NewFileURI(right)})
		waitForDone(t, feature)
	}

	open("first-left.png", "first-right.png")
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	feature.HandleKey(fyne.KeyRight)
	feature.HandleKey(fyne.KeyRight)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes := renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	test.Tap(comparisonButton(t, feature.Overlay(), "Swap"))
	feature.ToggleLink()
	if got := labelTexts(feature.Overlay()); !slices.ContainsFunc(got, func(text string) bool {
		return strings.HasPrefix(text, "Unlinked")
	}) {
		t.Fatalf("first session labels before close = %v, want Unlinked", got)
	}
	feature.Close()

	open("next-left.png", "next-right.png")
	for _, text := range labelTexts(feature.Overlay()) {
		if strings.HasPrefix(text, "Unlinked") {
			t.Fatalf("new session labels = %v, want linked reset", labelTexts(feature.Overlay()))
		}
	}
	if !containsButton(feature.Overlay(), "Swipe") || containsButton(feature.Overlay(), "Side by side") {
		t.Fatal("new comparison did not reset to side-by-side layout")
	}
	panes = renderedPanes(feature.Overlay())
	wantSizes := [2]fyne.Size{fyne.NewSize(400, 200), fyne.NewSize(100, 400)}
	nextNames := [2]string{"next-left.png", "next-right.png"}
	for i, pane := range panes {
		if pane.image.Image != images[nextNames[i]].Frames[0] {
			t.Errorf("new session pane %d did not restore source order", i)
		}
		if pane.image.Size() != wantSizes[i] || normalizedCenter(pane) != fyne.NewPos(0.5, 0.5) {
			t.Errorf("new session pane %d = {size:%v center:%v}, want fitted %v at center",
				i, pane.image.Size(), normalizedCenter(pane), wantSizes[i])
		}
	}
	if got, want := orders, [][2]string{
		{"first-left.png", "first-right.png"},
		{"first-right.png", "first-left.png"},
		{"next-left.png", "next-right.png"},
	}; !slices.Equal(got, want) {
		t.Errorf("session owner orders = %q, want %q", got, want)
	}

	feature.HandleKey(fyne.KeyRight)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 400) {
		t.Errorf("new session divider = %.2f, want reset 400", center)
	}
}

func TestCompareToolbar_CompactTranslucentCardIsAtTopRight(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(32, 24), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(640, 480))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	card, background := buttonCard(t, feature.Overlay(), "Swap")
	if !containsButton(card, "Back to Grid") {
		t.Fatal("Swap and Back to Grid do not share one toolbar card")
	}
	if _, _, _, alpha := background.FillColor.RGBA(); alpha == 0 || alpha == 0xffff {
		t.Errorf("toolbar background alpha = %#x, want partial translucency", alpha)
	}
	if card.Size().Width >= feature.Overlay().Size().Width/2 || card.Size().Height >= feature.Overlay().Size().Height/2 {
		t.Errorf("toolbar card size = %v in %v overlay, want compact", card.Size(), feature.Overlay().Size())
	}
	if gap := feature.Overlay().Size().Width - card.Position().X - card.Size().Width; gap < 0 || gap > theme.Padding() {
		t.Errorf("toolbar right gap = %v, want 0..%v", gap, theme.Padding())
	}
	if top := card.Position().Y; top < 0 || top > theme.Padding() {
		t.Errorf("toolbar top gap = %v, want 0..%v", top, theme.Padding())
	}
}

func TestCompareIdentity_BadgesUseBasenamesAndShortestDistinguishingSuffix(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	for _, tc := range []struct {
		name    string
		paths   [2]string
		wantIDs []string
	}{
		{
			name:    "different basenames",
			paths:   [2]string{"/library/left.jpg", "/archive/right.png"},
			wantIDs: []string{"left.jpg", "right.png"},
		},
		{
			name:    "shared basename and nearest folder",
			paths:   [2]string{"/library/one/trip/shared.jpg", "/archive/two/trip/shared.jpg"},
			wantIDs: []string{"one/trip/shared.jpg", "two/trip/shared.jpg"},
		},
		{
			name:    "filesystem root is a directory component",
			paths:   [2]string{"/shared.jpg", "/folder/shared.jpg"},
			wantIDs: []string{"/shared.jpg", "folder/shared.jpg"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
				return loadedImage(32, 24), nil
			}, Callbacks{})
			feature.Open([2]fyne.URI{
				storage.NewFileURI(tc.paths[0]),
				storage.NewFileURI(tc.paths[1]),
			})
			waitForDone(t, feature)

			if got := labelTexts(feature.Overlay()); !slices.Equal(got, tc.wantIDs) {
				t.Errorf("badge identities = %q, want %q", got, tc.wantIDs)
			}
		})
	}
}

func TestCompareIdentity_TranslucentBadgesStayAtBottomCorners(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(32, 24), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(640, 480))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("/library/left.jpg"),
		storage.NewFileURI("/archive/right.png"),
	})
	waitForDone(t, feature)

	left, leftBG := labelCard(t, feature.Overlay(), "left.jpg")
	right, rightBG := labelCard(t, feature.Overlay(), "right.png")
	row := commonLabelRow(t, feature.Overlay(), "left.jpg", "right.png")
	for name, background := range map[string]*canvas.Rectangle{"left": leftBG, "right": rightBG} {
		if _, _, _, alpha := background.FillColor.RGBA(); alpha == 0 || alpha == 0xffff {
			t.Errorf("%s badge background alpha = %#x, want partial translucency", name, alpha)
		}
	}
	if row.Position().Y+row.Size().Height != feature.Overlay().Size().Height {
		t.Errorf("badge row bottom = %v, want overlay bottom %v", row.Position().Y+row.Size().Height, feature.Overlay().Size().Height)
	}
	if left.Position().X != 0 {
		t.Errorf("left badge x = %v, want left corner 0", left.Position().X)
	}
	if edge := right.Position().X + right.Size().Width; edge != row.Size().Width {
		t.Errorf("right badge edge = %v, want row edge %v", edge, row.Size().Width)
	}
}

func TestCompareSwap_ExchangesReadyRolesWithoutReload(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	left := loadedImage(101, 51)
	right := loadedImage(202, 52)
	var loads atomic.Int32
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		loads.Add(1)
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(801, 500))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("/library/left.png"),
		storage.NewFileURI("/archive/right.png"),
	})
	waitForDone(t, feature)

	before := renderedPanes(feature.Overlay())
	if len(before) != 2 {
		t.Fatalf("rendered panes before Swap = %d, want 2", len(before))
	}
	beforeGeometry := [2]struct {
		position fyne.Position
		size     fyne.Size
	}{
		{before[0].origin, before[0].root.Size()},
		{before[1].origin, before[1].root.Size()},
	}

	test.Tap(comparisonButton(t, feature.Overlay(), "Swap"))

	after := renderedPanes(feature.Overlay())
	if len(after) != 2 || after[0].image.Image != right.Frames[0] || after[1].image.Image != left.Frames[0] {
		t.Error("Swap did not exchange the displayed left and right frames")
	}
	if got, want := labelTexts(feature.Overlay()), []string{"right.png", "left.png"}; !slices.Equal(got, want) {
		t.Errorf("badge identities after Swap = %q, want %q", got, want)
	}
	if got := loads.Load(); got != 2 {
		t.Errorf("loader calls after Swap = %d, want the original 2", got)
	}
	if !feature.Ready() {
		t.Error("Swap took a ready comparison out of ready state")
	}
	for i, pane := range after {
		if pane.origin != beforeGeometry[i].position || pane.root.Size() != beforeGeometry[i].size || pane.image.FillMode != canvas.ImageFillContain {
			t.Errorf("pane %d after Swap = {%v %v fill:%v}, want fitted geometry {%v %v}",
				i, pane.origin, pane.root.Size(), pane.image.FillMode, beforeGeometry[i].position, beforeGeometry[i].size)
		}
	}
}

func TestCompareSwapState_PreservesSessionAndExchangesSwipeRoles(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	left := solidLoadedImage(1600, 800, color.RGBA{R: 255, A: 255})
	right := solidLoadedImage(600, 1200, color.RGBA{B: 255, A: 255})
	var loads atomic.Int32
	var orders [][2]string
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		loads.Add(1)
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, Callbacks{OrderChanged: func(left, right string) {
		orders = append(orders, [2]string{left, right})
	}})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	feature.HandleKey(fyne.KeyRight)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes := renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	before := renderedPanes(feature.Overlay())
	if len(before) != 2 {
		t.Fatalf("rendered panes before Swap = %d, want 2", len(before))
	}
	beforeCenters := [2]fyne.Position{normalizedCenter(before[0]), normalizedCenter(before[1])}
	beforePositions := [2]fyne.Position{before[0].image.Position(), before[1].image.Position()}
	beforeSizes := [2]fyne.Size{before[0].image.Size(), before[1].image.Size()}
	beforeCapture := win.Canvas().Capture()
	if got := color.RGBAModel.Convert(beforeCapture.At(100, 200)); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("left reveal before Swap = %v, want red left source", got)
	}
	if got := color.RGBAModel.Convert(beforeCapture.At(700, 200)); got != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("right reveal before Swap = %v, want blue right source", got)
	}

	test.Tap(comparisonButton(t, feature.Overlay(), "Swap"))

	after := renderedPanes(feature.Overlay())
	if len(after) != 2 || after[0].image.Image != right.Frames[0] || after[1].image.Image != left.Frames[0] {
		t.Fatal("Swap did not exchange the displayed source roles")
	}
	for i, pane := range after {
		from := 1 - i
		if pane.image.Position() != beforePositions[from] || pane.image.Size() != beforeSizes[from] {
			t.Errorf("pane %d transformed geometry after Swap = {%v %v}, want prior source geometry {%v %v}",
				i, pane.image.Position(), pane.image.Size(), beforePositions[from], beforeSizes[from])
		}
		if center := normalizedCenter(pane); !approxPosition(center, beforeCenters[from]) {
			t.Errorf("pane %d center after Swap = %v, want prior source center %v", i, center, beforeCenters[from])
		}
	}
	if !containsButton(feature.Overlay(), "Side by side") {
		t.Error("Swap changed the active swipe layout")
	}
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 440) {
		t.Errorf("divider after Swap = %.2f, want retained 440", center)
	}
	if got, want := labelTexts(feature.Overlay()), []string{"right.png", "left.png"}; !slices.Equal(got, want) {
		t.Errorf("badge order after Swap = %q, want %q", got, want)
	}
	if got, want := orders, [][2]string{{"left.png", "right.png"}, {"right.png", "left.png"}}; !slices.Equal(got, want) {
		t.Errorf("owner orders = %q, want %q", got, want)
	}
	if got := loads.Load(); got != 2 {
		t.Errorf("loader calls after transformed Swap = %d, want original 2", got)
	}
	afterCapture := win.Canvas().Capture()
	if got := color.RGBAModel.Convert(afterCapture.At(100, 200)); got != (color.RGBA{B: 255, A: 255}) {
		t.Errorf("left reveal after Swap = %v, want blue former right source", got)
	}
	if got := color.RGBAModel.Convert(afterCapture.At(700, 200)); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("right reveal after Swap = %v, want red former left source", got)
	}
}

func TestCompareSideBySide_FitsFirstFramesInFixedEqualPanes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	left := loadedImage(640, 320)
	right := loadedImage(300, 900)
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(801, 500))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	if !feature.Ready() {
		t.Fatal("comparison did not become ready after both successful loads")
	}
	panes := renderedPanes(feature.Overlay())
	if len(panes) != 2 {
		t.Fatalf("rendered panes = %d, want 2", len(panes))
	}
	if panes[0].origin != fyne.NewPos(0, 0) || panes[0].root.Size() != fyne.NewSize(400.5, 500) {
		t.Errorf("left pane geometry = %v %v, want (0,0) 400.5x500", panes[0].origin, panes[0].root.Size())
	}
	if panes[1].origin != fyne.NewPos(400.5, 0) || panes[1].root.Size() != fyne.NewSize(400.5, 500) {
		t.Errorf("right pane geometry = %v %v, want (400.5,0) 400.5x500", panes[1].origin, panes[1].root.Size())
	}
	if panes[0].image.Image != left.Frames[0] || panes[1].image.Image != right.Frames[0] {
		t.Error("pane images do not follow the Open source order")
	}
	for i, pane := range panes {
		if !pane.image.Visible() || pane.image.FillMode != canvas.ImageFillContain {
			t.Errorf("pane %d image = {Visible:%v FillMode:%v}, want visible fitted content", i, pane.image.Visible(), pane.image.FillMode)
		}
	}
}

func TestCompareVector_RasterizesAtCurrentDisplaySize(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	vector := loadedVector(t, 40, 20)
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return loadedImage(40, 20), nil
	}, Callbacks{})
	feature.SetUIQueue(&uitest.UIQueue{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatal("timed out waiting for comparison vector raster")
	}

	panes := renderedPanes(feature.Overlay())
	if len(panes) != 2 {
		t.Fatalf("rendered panes = %d, want 2", len(panes))
	}
	if got := panes[0].image.Image.Bounds().Size(); got != image.Pt(400, 200) {
		t.Errorf("SVG raster = %v, want 400x200 for its current fitted display size", got)
	}
}

func TestCompareVector_UnlinkedZoomRerasterizesOnlyTheTargetPane(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	vectors := [2]*imaging.LoadedImage{loadedVector(t, 40, 20), loadedVector(t, 40, 20)}
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vectors[0], nil
		}
		return vectors[1], nil
	}, Callbacks{})
	feature.SetUIQueue(&uitest.UIQueue{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.svg"),
	})
	settle := func() []renderedPane {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()
		if err := feature.Settle(ctx); err != nil {
			t.Fatal("timed out waiting for comparison vector raster")
		}
		return renderedPanes(feature.Overlay())
	}
	panes := settle()
	feature.ToggleLink()
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	panes = settle()

	if got := panes[0].image.Image.Bounds().Size(); got != image.Pt(500, 250) {
		t.Errorf("target SVG raster after unlinked + = %v, want 500x250", got)
	}
	if got := panes[1].image.Image.Bounds().Size(); got != image.Pt(400, 200) {
		t.Errorf("other SVG raster after unlinked + = %v, want unchanged 400x200", got)
	}
}

func TestCompareVector_RerasterizesAcrossZoomLayoutResizeAndSwap(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	vector := loadedVector(t, 40, 20)
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return loadedImage(40, 20), nil
	}, Callbacks{})
	feature.SetUIQueue(&uitest.UIQueue{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	settle := func() []renderedPane {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()
		if err := feature.Settle(ctx); err != nil {
			t.Fatal("timed out waiting for comparison vector raster")
		}
		panes := renderedPanes(feature.Overlay())
		if len(panes) != 2 {
			t.Fatalf("rendered panes = %d, want 2", len(panes))
		}
		return panes
	}
	assertRaster := func(side int, want image.Point) {
		t.Helper()
		if got := settle()[side].image.Image.Bounds().Size(); got != want {
			t.Errorf("side %d SVG raster = %v, want %v", side, got, want)
		}
	}

	assertRaster(0, image.Pt(400, 200))
	feature.HandleKey(fyne.KeyPlus)
	assertRaster(0, image.Pt(500, 250))
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	assertRaster(0, image.Pt(1000, 500))
	feature.Overlay().Resize(fyne.NewSize(960, 600))
	assertRaster(0, image.Pt(1200, 600))
	test.Tap(comparisonButton(t, feature.Overlay(), "Swap"))
	assertRaster(1, image.Pt(1200, 600))
	test.Tap(comparisonButton(t, feature.Overlay(), "Side by side"))
	assertRaster(1, image.Pt(600, 300))
}

func TestCompareRasterFidelity_PreservesFullDecodedFramesAcrossTransforms(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	left := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	right := image.NewRGBA(image.Rect(0, 0, 800, 1200))
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.png" {
			return &imaging.LoadedImage{Frames: []image.Image{left}}, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{right}}, nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 600))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	assertSources := func(wantLeft, wantRight image.Image) {
		t.Helper()
		panes := renderedPanes(feature.Overlay())
		if len(panes) != 2 {
			t.Fatalf("rendered panes = %d, want 2", len(panes))
		}
		if panes[0].image.Image != wantLeft || panes[1].image.Image != wantRight {
			t.Errorf("pane sources = (%p, %p), want original decoded frames (%p, %p)",
				panes[0].image.Image, panes[1].image.Image, wantLeft, wantRight)
		}
	}

	assertSources(left, right)
	feature.HandleKey(fyne.KeyPlus)
	panes := renderedPanes(feature.Overlay())
	paneScrollable(t, panes[0]).Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(200, 250)},
		Scrolled:   fyne.NewDelta(0, 10),
	})
	paneDraggable(t, panes[1]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(30, -20)})
	feature.Overlay().Resize(fyne.NewSize(960, 640))
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	assertSources(left, right)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swap"))
	assertSources(right, left)
}

func TestCompareLinkedTransform_DifferentImagesShareCenterAndFitMultiplier(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)

	feature.HandleKey(fyne.KeyPlus)

	panes := renderedPanes(feature.Overlay())
	if len(panes) != 2 {
		t.Fatalf("rendered panes = %d, want 2", len(panes))
	}
	wantSizes := [2]fyne.Size{
		fyne.NewSize(500, 250),
		fyne.NewSize(125, 500),
	}
	for i, pane := range panes {
		if got := pane.image.Size(); got != wantSizes[i] {
			t.Errorf("pane %d image size after + = %v, want %v", i, got, wantSizes[i])
		}
		center := fyne.NewPos(
			(pane.root.Size().Width/2-pane.image.Position().X)/pane.image.Size().Width,
			(pane.root.Size().Height/2-pane.image.Position().Y)/pane.image.Size().Height,
		)
		if center != fyne.NewPos(0.5, 0.5) {
			t.Errorf("pane %d normalized center after + = %v, want (0.5,0.5)", i, center)
		}
	}
}

func TestCompareCameraHome_ReturnsToStoredPhotoArrangement(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.Key1)
	paneHoverable(t, renderedPanes(feature.Overlay())[1]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	feature.ToggleLink()
	home := renderedPanes(feature.Overlay())
	wantSizes := [2]fyne.Size{home[0].image.Size(), home[1].image.Size()}
	wantPositions := [2]fyne.Position{home[0].image.Position(), home[1].image.Position()}

	feature.HandleKey(fyne.KeyPlus)
	paneDraggable(t, renderedPanes(feature.Overlay())[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(30, 20)})
	feature.HandleKey(fyne.Key1)
	panes = renderedPanes(feature.Overlay())
	for i, pane := range panes {
		if pane.image.Size() != wantSizes[i] || !approxPosition(pane.image.Position(), wantPositions[i]) {
			t.Errorf("pane %d at camera home = {%v %v}, want stored {%v %v}",
				i, pane.image.Position(), pane.image.Size(), wantPositions[i], wantSizes[i])
		}
	}

	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	for i, pane := range panes {
		want := fyne.NewSize(wantSizes[i].Width*1.25, wantSizes[i].Height*1.25)
		if pane.image.Size() != want {
			t.Errorf("pane %d after camera home then + = %v, want common 1.25x step %v", i, pane.image.Size(), want)
		}
	}
}

func TestCompareFit_ReturnsBothImagesToCanonicalCenteredFit(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	feature.HandleKey(fyne.Key1)
	feature.HandleKey(fyne.KeyPlus)

	feature.HandleKey(fyne.Key0)

	panes := renderedPanes(feature.Overlay())
	wantSizes := [2]fyne.Size{fyne.NewSize(400, 200), fyne.NewSize(100, 400)}
	wantPositions := [2]fyne.Position{fyne.NewPos(0, 100), fyne.NewPos(150, 0)}
	for i, pane := range panes {
		if pane.image.Size() != wantSizes[i] || pane.image.Position() != wantPositions[i] {
			t.Errorf("pane %d after 0 = {%v %v}, want fitted {%v %v}",
				i, pane.image.Position(), pane.image.Size(), wantPositions[i], wantSizes[i])
		}
	}
}

func TestCompareZoom_MinusAndEqualScaleBothFromTheSharedView(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)

	feature.HandleKey(fyne.KeyMinus)
	panes := renderedPanes(feature.Overlay())
	wantSmaller := [2]fyne.Size{fyne.NewSize(320, 160), fyne.NewSize(80, 320)}
	for i, pane := range panes {
		if pane.image.Size() != wantSmaller[i] {
			t.Errorf("pane %d after - = %v, want %v", i, pane.image.Size(), wantSmaller[i])
		}
	}

	feature.HandleKey(fyne.KeyEqual)
	wantFit := [2]fyne.Size{fyne.NewSize(400, 200), fyne.NewSize(100, 400)}
	for i, pane := range panes {
		if pane.image.Size() != wantFit[i] {
			t.Errorf("pane %d after - then = = %v, want fitted %v", i, pane.image.Size(), wantFit[i])
		}
	}
}

func TestCompareCameraZoom_WheelAnchorsCorrespondingPointsInBothPanes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}

	before := renderedPanes(feature.Overlay())
	cursor := fyne.NewPos(220, 180)
	anchored := [2]fyne.Position{normalizedPoint(before[0], cursor), normalizedPoint(before[1], cursor)}
	leftWidth := before[0].image.Size().Width
	rightHeight := before[1].image.Size().Height
	paneScrollable(t, before[0]).Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: cursor},
		Scrolled:   fyne.NewDelta(0, 10),
	})

	after := renderedPanes(feature.Overlay())
	if after[0].image.Size().Width <= leftWidth || after[1].image.Size().Height <= rightHeight {
		t.Fatal("wheel zoom did not enlarge both comparison images")
	}
	leftFactor := after[0].image.Size().Width / leftWidth
	rightFactor := after[1].image.Size().Height / rightHeight
	if !uitest.ApproxEqual(leftFactor, rightFactor) {
		t.Errorf("wheel multipliers = left %.4f right %.4f, want one shared multiplier", leftFactor, rightFactor)
	}
	for i, pane := range after {
		if got := normalizedPoint(pane, cursor); !approxPosition(got, anchored[i]) {
			t.Errorf("pane %d normalized point under cursor after camera wheel = %v, want anchored %v", i, got, anchored[i])
		}
	}
}

func TestCompareCameraClamp_WheelKeepsBothPhotosOverTheirPaneCenters(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)

	panes := renderedPanes(feature.Overlay())
	paneScrollable(t, panes[0]).Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(0, 0)},
		Scrolled:   fyne.NewDelta(0, 100),
	})

	panes = renderedPanes(feature.Overlay())
	for _, pane := range panes {
		assertPaneOverlapsCenter(t, pane)
	}
}

func TestCompareCameraPan_DragAndShiftWheelMoveBothPhotosByTheSamePoints(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	var modifiers fyne.KeyModifier
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{Modifiers: func() fyne.KeyModifier { return modifiers }})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}

	before := renderedPanes(feature.Overlay())
	wantAfterDrag := [2]fyne.Position{
		before[0].image.Position().Add(fyne.NewPos(40, 20)),
		before[1].image.Position().Add(fyne.NewPos(40, 20)),
	}
	draggable := paneDraggable(t, before[0])
	draggable.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	draggable.DragEnd()

	afterDrag := renderedPanes(feature.Overlay())
	for i, pane := range afterDrag {
		if got := pane.image.Position(); !approxPosition(got, wantAfterDrag[i]) {
			t.Errorf("camera-dragged pane %d position = %v, want %v", i, got, wantAfterDrag[i])
		}
	}

	modifiers = fyne.KeyModifierShift
	wantAfterShiftWheel := [2]fyne.Position{
		afterDrag[0].image.Position().Add(fyne.NewPos(-15, 25)),
		afterDrag[1].image.Position().Add(fyne.NewPos(-15, 25)),
	}
	paneScrollable(t, afterDrag[1]).Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(-15, 25)})
	afterShiftWheel := renderedPanes(feature.Overlay())
	for i, pane := range afterShiftWheel {
		if got := pane.image.Position(); !approxPosition(got, wantAfterShiftWheel[i]) {
			t.Errorf("camera Shift+wheel pane %d position = %v, want %v", i, got, wantAfterShiftWheel[i])
		}
	}
}

func TestCompareLinkToggle_DragMovesOnlyGesturePaneWithoutHeldModifier(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()

	before := renderedPanes(feature.Overlay())
	wantLeft := before[0].image.Position().Add(fyne.NewPos(40, 20))
	wantRight := before[1].image.Position()
	paneDraggable(t, before[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})

	after := renderedPanes(feature.Overlay())
	if got := after[0].image.Position(); !approxPosition(got, wantLeft) {
		t.Errorf("unlinked dragged pane position = %v, want %v", got, wantLeft)
	}
	if got := after[1].image.Position(); !approxPosition(got, wantRight) {
		t.Errorf("other pane position after unlinked drag = %v, want unchanged %v", got, wantRight)
	}
}

func TestCompareLinkToggle_WheelZoomsOnlyTheGesturePane(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()

	before := renderedPanes(feature.Overlay())
	leftWidth := before[0].image.Size().Width
	wantRight := before[1].image.Size()
	paneScrollable(t, before[0]).Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(200, 200)},
		Scrolled:   fyne.NewDelta(0, 10),
	})

	after := renderedPanes(feature.Overlay())
	if after[0].image.Size().Width <= leftWidth {
		t.Errorf("unlinked wheel pane width = %.2f, want greater than %.2f", after[0].image.Size().Width, leftWidth)
	}
	if got := after[1].image.Size(); got != wantRight {
		t.Errorf("other pane size after unlinked wheel = %v, want unchanged %v", got, wantRight)
	}
}

func TestCompareLinkToggle_ShowsUnlinkedStatusAndTracksLastHoveredPane(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	unlink := comparisonButton(t, feature.Overlay(), "Unlink")
	feature.ToggleLink()
	if got := labelTexts(feature.Overlay()); !slices.Contains(got, "Unlinked") {
		t.Fatalf("labels after first toggle = %v, want Unlinked", got)
	}
	link := comparisonButton(t, feature.Overlay(), "Link")
	if link != unlink {
		t.Fatal("link toggle was replaced instead of relabeled in place")
	}
	linkCard, _ := buttonCard(t, feature.Overlay(), "Link")
	statusCard, _ := labelCard(t, feature.Overlay(), "Unlinked")
	if statusCard != linkCard {
		t.Fatal("Unlinked status is not beside Link in the top-left card")
	}
	var status *widget.Label
	walk(linkCard, func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Text == "Unlinked" {
			status = label
		}
	})
	if status == nil {
		t.Fatal("top-left link card has no visible Unlinked status")
	}
	if status.Position().X < link.Position().X+link.Size().Width {
		t.Errorf("Unlinked status x = %v, want it after Link ending at %v", status.Position().X, link.Position().X+link.Size().Width)
	}

	panes := renderedPanes(feature.Overlay())
	left := paneHoverable(t, panes[0])
	left.MouseIn(&desktop.MouseEvent{})
	if got := labelTexts(feature.Overlay()); !slices.Contains(got, "Unlinked: Left") {
		t.Errorf("labels while left pane is targeted = %v, want Unlinked: Left", got)
	}
	left.MouseOut()
	if got := labelTexts(feature.Overlay()); !slices.Contains(got, "Unlinked: Left") {
		t.Errorf("labels after leaving left pane = %v, want last target retained", got)
	}

	paneHoverable(t, panes[1]).MouseMoved(&desktop.MouseEvent{})
	if got := labelTexts(feature.Overlay()); !slices.Contains(got, "Unlinked: Right") {
		t.Errorf("labels while right pane is targeted = %v, want Unlinked: Right", got)
	}

	feature.ToggleLink()
	if got := comparisonButton(t, feature.Overlay(), "Unlink"); got != unlink {
		t.Fatal("relink replaced the comparison link control")
	}
	for _, text := range labelTexts(feature.Overlay()) {
		if strings.HasPrefix(text, "Unlinked") {
			t.Fatalf("labels after relink toggle = %v, want no unlink status", labelTexts(feature.Overlay()))
		}
	}
}

func TestCompareLinkToggle_TransformKeysRequireAndOnlyAffectLastHoveredPane(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()

	before := renderedPanes(feature.Overlay())
	wantLeft := before[0].image.Size()
	wantRight := before[1].image.Size()
	feature.HandleKey(fyne.KeyPlus)
	panes := renderedPanes(feature.Overlay())
	if panes[0].image.Size() != wantLeft || panes[1].image.Size() != wantRight {
		t.Fatalf("unlinked + without a pane target changed sizes to %v and %v", panes[0].image.Size(), panes[1].image.Size())
	}

	paneHoverable(t, panes[1]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	if panes[0].image.Size() != wantLeft {
		t.Errorf("left pane after right-targeted + = %v, want unchanged %v", panes[0].image.Size(), wantLeft)
	}
	if panes[1].image.Size().Width <= wantRight.Width {
		t.Errorf("right pane after right-targeted + = %v, want larger than %v", panes[1].image.Size(), wantRight)
	}

	feature.HandleKey(fyne.Key1)
	panes = renderedPanes(feature.Overlay())
	if got := panes[1].image.Size(); got != fyne.NewSize(800, 400) {
		t.Errorf("right pane after right-targeted 1 = %v, want actual size", got)
	}
	if got := normalizedCenter(panes[1]); !approxPosition(got, fyne.NewPos(0.5, 0.5)) {
		t.Errorf("right pane center after right-targeted 1 = %v, want centered", got)
	}
	if panes[0].image.Size() != wantLeft {
		t.Errorf("left pane after right-targeted 1 = %v, want unchanged %v", panes[0].image.Size(), wantLeft)
	}

	feature.HandleKey(fyne.Key0)
	panes = renderedPanes(feature.Overlay())
	if panes[1].image.Size() != wantRight || !approxPosition(normalizedCenter(panes[1]), fyne.NewPos(0.5, 0.5)) {
		t.Errorf("right pane after right-targeted 0 = {%v %v}, want fitted and centered", panes[1].image.Size(), normalizedCenter(panes[1]))
	}
}

func TestCompareLinkToggle_UnlockKeepsCurrentCameraView(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	panes := renderedPanes(feature.Overlay())

	feature.ToggleLink()
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	paneHoverable(t, panes[1]).MouseIn(&desktop.MouseEvent{})
	feature.ToggleLink()

	feature.HandleKey(fyne.KeyPlus)
	linked := renderedPanes(feature.Overlay())
	if linked[0].image.Size().Width != 625 || linked[1].image.Size().Width != 500 {
		t.Fatalf("linked camera widths after + = %.2f and %.2f, want 625 and 500", linked[0].image.Size().Width, linked[1].image.Size().Width)
	}
	want := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: linked[0].image.Size(), position: linked[0].image.Position()},
		{size: linked[1].image.Size(), position: linked[1].image.Position()},
	}

	feature.ToggleLink()
	unlocked := renderedPanes(feature.Overlay())
	for i, pane := range unlocked {
		if pane.image.Size() != want[i].size || pane.image.Position() != want[i].position {
			t.Errorf("pane %d moved while unlocking: got {%v %v}, want {%v %v}",
				i, pane.image.Size(), pane.image.Position(), want[i].size, want[i].position)
		}
	}
}

func TestCompareLinkToggle_RelockKeepsDivergentPhotoPoses(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	before := renderedPanes(feature.Overlay())
	want := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: before[0].image.Size(), position: before[0].image.Position()},
		{size: before[1].image.Size(), position: before[1].image.Position()},
	}

	feature.ToggleLink()
	locked := renderedPanes(feature.Overlay())
	for i, pane := range locked {
		if pane.image.Size() != want[i].size || pane.image.Position() != want[i].position {
			t.Errorf("pane %d moved while locking: got {%v %v}, want {%v %v}",
				i, pane.image.Size(), pane.image.Position(), want[i].size, want[i].position)
		}
	}
}

func TestCompareLockedPan_MovesDivergentPhotoPosesAsOneCamera(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	for range 3 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes = renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	paneHoverable(t, renderedPanes(feature.Overlay())[1]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	feature.ToggleLink()

	before := renderedPanes(feature.Overlay())
	want := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: before[0].image.Size(), position: before[0].image.Position()},
		{size: before[1].image.Size(), position: before[1].image.Position()},
	}
	delta := fyne.NewDelta(20, 10)
	paneDraggable(t, before[0]).Dragged(&fyne.DragEvent{Dragged: delta})
	after := renderedPanes(feature.Overlay())
	for i := range after {
		if after[i].image.Size() != want[i].size {
			t.Errorf("pane %d size after camera pan = %v, want unchanged %v", i, after[i].image.Size(), want[i].size)
		}
		if got, wantPosition := after[i].image.Position(), want[i].position.Add(delta); got != wantPosition {
			t.Errorf("pane %d position after camera pan = %v, want %v", i, got, wantPosition)
		}
	}
}

func TestCompareLockedFit_FramesDivergentPhotoPosesWithoutChangingTheirRatio(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	for range 3 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes = renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(80, 40)})
	paneHoverable(t, renderedPanes(feature.Overlay())[1]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	paneDraggable(t, panes[1]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-40, -20)})
	feature.ToggleLink()

	before := renderedPanes(feature.Overlay())
	wantRatio := before[0].image.Size().Width / before[1].image.Size().Width
	feature.HandleKey(fyne.Key0)
	after := renderedPanes(feature.Overlay())
	if got := after[0].image.Size().Width / after[1].image.Size().Width; !uitest.ApproxEqual(got, wantRatio) {
		t.Errorf("photo width ratio after camera fit = %.4f, want retained %.4f", got, wantRatio)
	}
	for i, pane := range after {
		position, size, viewport := pane.image.Position(), pane.image.Size(), pane.root.Size()
		if position.X < -0.01 || position.Y < -0.01 ||
			position.X+size.Width > viewport.Width+0.01 || position.Y+size.Height > viewport.Height+0.01 {
			t.Errorf("pane %d after camera fit spans %v..%v in %v, want fully visible",
				i, position, fyne.NewPos(position.X+size.Width, position.Y+size.Height), viewport)
		}
	}
}

func TestCompareUnlinkedFit_ResetsOnlyTargetInCurrentCameraView(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	for range 2 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes := renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(30, 20)})
	feature.ToggleLink()
	panes = renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	wantRightSize := panes[1].image.Size()
	wantRightPosition := panes[1].image.Position()

	feature.HandleKey(fyne.Key0)
	panes = renderedPanes(feature.Overlay())
	if got, want := panes[0].image.Size(), fyne.NewSize(400, 200); got != want {
		t.Errorf("target size after unlinked fit = %v, want %v", got, want)
	}
	if got, want := panes[0].image.Position(), fyne.NewPos(0, 100); !approxPosition(got, want) {
		t.Errorf("target position after unlinked fit = %v, want %v", got, want)
	}
	if panes[1].image.Size() != wantRightSize || panes[1].image.Position() != wantRightPosition {
		t.Errorf("other pane changed during unlinked fit: got {%v %v}, want {%v %v}",
			panes[1].image.Size(), panes[1].image.Position(), wantRightSize, wantRightPosition)
	}
}

func TestCompareCameraPan_SurvivesUnlockWithoutChangingGeometry(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	for range 4 {
		feature.HandleKey(fyne.KeyPlus)
	}
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(70, 0)})
	paneHoverable(t, panes[1]).MouseIn(&desktop.MouseEvent{})
	for range 2 {
		feature.HandleKey(fyne.KeyPlus)
	}
	feature.ToggleLink()
	linkedBefore := renderedPanes(feature.Overlay())
	beforePositions := [2]fyne.Position{linkedBefore[0].image.Position(), linkedBefore[1].image.Position()}
	paneDraggable(t, renderedPanes(feature.Overlay())[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(30, 0)})
	linkedAfter := renderedPanes(feature.Overlay())
	want := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: linkedAfter[0].image.Size(), position: linkedAfter[0].image.Position()},
		{size: linkedAfter[1].image.Size(), position: linkedAfter[1].image.Position()},
	}
	for i := range linkedAfter {
		if got, expected := linkedAfter[i].image.Position(), beforePositions[i].Add(fyne.NewPos(30, 0)); !approxPosition(got, expected) {
			t.Errorf("pane %d camera-pan position = %v, want %v", i, got, expected)
		}
	}

	feature.ToggleLink()
	unlocked := renderedPanes(feature.Overlay())
	for i := range unlocked {
		if unlocked[i].image.Size() != want[i].size || !approxPosition(unlocked[i].image.Position(), want[i].position) {
			t.Errorf("pane %d changed while unlocking after camera pan: got {%v %v}, want {%v %v}",
				i, unlocked[i].image.Size(), unlocked[i].image.Position(), want[i].size, want[i].position)
		}
	}
}

func TestCompareCameraWheel_PreservesDivergenceAndSurvivesUnlock(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	for range 2 {
		feature.HandleKey(fyne.KeyPlus)
	}
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 0)})
	paneHoverable(t, panes[1]).MouseIn(&desktop.MouseEvent{})
	feature.ToggleLink()
	linkedBefore := renderedPanes(feature.Overlay())
	cursor := fyne.NewPos(300, 200)
	beforeSizes := [2]fyne.Size{linkedBefore[0].image.Size(), linkedBefore[1].image.Size()}
	anchored := [2]fyne.Position{normalizedPoint(linkedBefore[0], cursor), normalizedPoint(linkedBefore[1], cursor)}
	paneScrollable(t, linkedBefore[0]).Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: cursor},
		Scrolled:   fyne.NewDelta(0, 10),
	})
	linkedAfter := renderedPanes(feature.Overlay())
	ratio := linkedAfter[0].image.Size().Width / beforeSizes[0].Width
	if got := linkedAfter[1].image.Size().Width / beforeSizes[1].Width; !uitest.ApproxEqual(got, ratio) {
		t.Errorf("camera wheel ratios = %.4f and %.4f, want equal", ratio, got)
	}
	for i, pane := range linkedAfter {
		if got := normalizedPoint(pane, cursor); !approxPosition(got, anchored[i]) {
			t.Errorf("pane %d point under camera-wheel cursor = %v, want %v", i, got, anchored[i])
		}
	}
	want := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: linkedAfter[0].image.Size(), position: linkedAfter[0].image.Position()},
		{size: linkedAfter[1].image.Size(), position: linkedAfter[1].image.Position()},
	}

	feature.ToggleLink()
	unlocked := renderedPanes(feature.Overlay())
	for i := range unlocked {
		if unlocked[i].image.Size() != want[i].size || !approxPosition(unlocked[i].image.Position(), want[i].position) {
			t.Errorf("pane %d changed while unlocking after camera wheel: got {%v %v}, want {%v %v}",
				i, unlocked[i].image.Size(), unlocked[i].image.Position(), want[i].size, want[i].position)
		}
	}
}

func TestCompareSwapWhileUnlinked_LocksAndClearsDivergentPhotoPoses(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	feature.ToggleLink()
	link := comparisonButton(t, feature.Overlay(), "Link")
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	if panes = renderedPanes(feature.Overlay()); panes[0].image.Size().Width != 500 || panes[1].image.Size().Width != 400 {
		t.Fatalf("unlinked setup widths = %.2f and %.2f, want 500 and 400", panes[0].image.Size().Width, panes[1].image.Size().Width)
	}

	test.Tap(comparisonButton(t, feature.Overlay(), "Swap"))
	if got := comparisonButton(t, feature.Overlay(), "Unlink"); got != link {
		t.Fatal("Swap replaced the link control instead of restoring its Unlink state")
	}
	panes = renderedPanes(feature.Overlay())
	if panes[0].image.Size().Width != 500 || panes[1].image.Size().Width != 500 {
		t.Errorf("widths after Swap relink = %.2f and %.2f, want winning 500 shared", panes[0].image.Size().Width, panes[1].image.Size().Width)
	}
	for _, text := range labelTexts(feature.Overlay()) {
		if strings.HasPrefix(text, "Unlinked") {
			t.Errorf("labels after Swap while unlinked = %v, want no unlink status", labelTexts(feature.Overlay()))
			break
		}
	}

	beforeWidths := [2]float32{panes[0].image.Size().Width, panes[1].image.Size().Width}
	paneScrollable(t, panes[0]).Scrolled(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(200, 200)},
		Scrolled:   fyne.NewDelta(0, 10),
	})
	panes = renderedPanes(feature.Overlay())
	leftRatio := panes[0].image.Size().Width / beforeWidths[0]
	rightRatio := panes[1].image.Size().Width / beforeWidths[1]
	if leftRatio <= 1 || !uitest.ApproxEqual(leftRatio, rightRatio) {
		t.Errorf("wheel ratios after Swap = %.4f and %.4f, want one linked increase", leftRatio, rightRatio)
	}

	feature.ToggleLink()
	panes = renderedPanes(feature.Overlay())
	if panes[0].image.Size() != panes[1].image.Size() || !approxPosition(normalizedCenter(panes[0]), normalizedCenter(panes[1])) {
		t.Errorf("photo poses after Swap and fresh unlink = {%v %v} and {%v %v}, want reset together",
			panes[0].image.Size(), normalizedCenter(panes[0]), panes[1].image.Size(), normalizedCenter(panes[1]))
	}
}

func TestCompareOpenStartsLinkedAndFreshToggleUnlinks(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	modifiers := fyne.KeyModifierControl
	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{Modifiers: func() fyne.KeyModifier { return modifiers }})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	unlink := comparisonButton(t, feature.Overlay(), "Unlink")
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	if got := labelTexts(feature.Overlay()); slices.Contains(got, "Unlinked: Left") {
		t.Fatalf("labels after opening with Control held = %v, want linked start", got)
	}

	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	if panes[0].image.Size().Width != 500 || panes[1].image.Size().Width != 500 {
		t.Errorf("first linked + widths = %.2f and %.2f, want 500 and 500", panes[0].image.Size().Width, panes[1].image.Size().Width)
	}

	feature.ToggleLink()
	if got := comparisonButton(t, feature.Overlay(), "Link"); got != unlink {
		t.Fatal("fresh link toggle replaced the Unlink control instead of relabeling it")
	}
	if got := labelTexts(feature.Overlay()); !slices.Contains(got, "Unlinked: Left") {
		t.Fatalf("labels after fresh toggle = %v, want Unlinked: Left", got)
	}
	beforeRight := renderedPanes(feature.Overlay())[1].image.Size()
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	if panes[0].image.Size().Width <= 500 || panes[1].image.Size() != beforeRight {
		t.Errorf("unlinked + sizes = %v and %v, want only left enlarged from 500", panes[0].image.Size(), panes[1].image.Size())
	}
}

func TestCompareUnlinkedShiftWheel_AllowsOnlyTheTargetToOverscrollToImageEdgeAtPaneCenter(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	var modifiers fyne.KeyModifier
	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{Modifiers: func() fyne.KeyModifier { return modifiers }})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	modifiers = fyne.KeyModifierShift
	feature.ToggleLink()

	panes := renderedPanes(feature.Overlay())
	if got := paneCursorable(t, panes[0]).Cursor(); got != desktop.PointerCursor {
		t.Errorf("fitted pane cursor while unlinked = %v, want pointer because local overscroll is available", got)
	}
	paneScrollable(t, panes[0]).Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(10000, -10000)})
	panes = renderedPanes(feature.Overlay())
	if got := normalizedCenter(panes[0]); !approxPosition(got, fyne.NewPos(0, 1)) {
		t.Errorf("target local center after extreme unlinked Shift-wheel = %v, want edge clamp {0 1}", got)
	}
	if got := normalizedCenter(panes[1]); !approxPosition(got, fyne.NewPos(0.5, 0.5)) {
		t.Errorf("other local center after extreme unlinked Shift-wheel = %v, want unchanged", got)
	}
	if !uitest.ApproxEqual(panes[0].image.Position().X, panes[0].root.Size().Width/2) ||
		!uitest.ApproxEqual(panes[0].image.Position().Y+panes[0].image.Size().Height, panes[0].root.Size().Height/2) {
		t.Errorf("overscrolled target span = {%v %v} in %v, want selected image edges at pane center",
			panes[0].image.Position(), panes[0].image.Size(), panes[0].root.Size())
	}
}

func TestCompareUnlinkedPan_AfterCameraPanStopsAtPaneCenter(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	panes := renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(80, 40)})
	feature.ToggleLink()
	panes = renderedPanes(feature.Overlay())
	wantOtherSize := panes[1].image.Size()
	wantOtherPosition := panes[1].image.Position()

	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(10000, -10000)})
	panes = renderedPanes(feature.Overlay())
	left := panes[0]
	if !uitest.ApproxEqual(left.image.Position().X, left.root.Size().Width/2) ||
		!uitest.ApproxEqual(left.image.Position().Y+left.image.Size().Height, left.root.Size().Height/2) {
		t.Errorf("camera-offset target span = {%v %v} in %v, want selected image edges at pane center",
			left.image.Position(), left.image.Size(), left.root.Size())
	}
	assertPaneOverlapsCenter(t, left)
	if panes[1].image.Size() != wantOtherSize || !approxPosition(panes[1].image.Position(), wantOtherPosition) {
		t.Errorf("other pane changed during camera-offset local pan: got {%v %v}, want {%v %v}",
			panes[1].image.Size(), panes[1].image.Position(), wantOtherSize, wantOtherPosition)
	}
}

func TestCompareLinkToggle_RepeatedTogglesNeverChangeDivergentGeometry(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))

	feature.ToggleLink()
	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(10000, 0)})
	paneHoverable(t, renderedPanes(feature.Overlay())[1]).MouseIn(&desktop.MouseEvent{})
	before := renderedPanes(feature.Overlay())
	want := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: before[0].image.Size(), position: before[0].image.Position()},
		{size: before[1].image.Size(), position: before[1].image.Position()},
	}

	for toggle := range 4 {
		feature.ToggleLink()
		panes := renderedPanes(feature.Overlay())
		for i, pane := range panes {
			if pane.image.Size() != want[i].size || !approxPosition(pane.image.Position(), want[i].position) {
				t.Errorf("toggle %d pane %d geometry = {%v %v}, want unchanged {%v %v}",
					toggle+1, i, pane.image.Size(), pane.image.Position(), want[i].size, want[i].position)
			}
		}
	}
}

func TestCompareUnlinkedLayoutAndResize_PreserveEachLocalCenterModeAndFactor(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()

	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.Key1)
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(80, 40)})
	paneHoverable(t, panes[1]).MouseIn(&desktop.MouseEvent{})
	feature.HandleKey(fyne.KeyPlus)
	before := renderedPanes(feature.Overlay())
	wantCenters := [2]fyne.Position{normalizedCenter(before[0]), normalizedCenter(before[1])}

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	inSwipe := renderedPanes(feature.Overlay())
	if got := inSwipe[0].image.Size(); got != fyne.NewSize(800, 400) {
		t.Errorf("absolute local size in swipe = %v, want retained 800x400", got)
	}
	if got := inSwipe[1].image.Size(); got != fyne.NewSize(1000, 500) {
		t.Errorf("fit-relative local size in swipe = %v, want recomputed 1000x500", got)
	}
	for i, pane := range inSwipe {
		if got := normalizedCenter(pane); !approxPosition(got, wantCenters[i]) {
			t.Errorf("swipe local pane %d center = %v, want retained %v", i, got, wantCenters[i])
		}
	}

	feature.Overlay().Resize(fyne.NewSize(1000, 600))
	afterResize := renderedPanes(feature.Overlay())
	if got := afterResize[0].image.Size(); got != fyne.NewSize(800, 400) {
		t.Errorf("absolute local size after resize = %v, want retained 800x400", got)
	}
	if got := afterResize[1].image.Size(); got != fyne.NewSize(1250, 625) {
		t.Errorf("fit-relative local size after resize = %v, want recomputed 1250x625", got)
	}
	for i, pane := range afterResize {
		if got := normalizedCenter(pane); !approxPosition(got, wantCenters[i]) {
			t.Errorf("resized local pane %d center = %v, want retained %v", i, got, wantCenters[i])
		}
	}
}

func TestCompareCameraFitAndHome_DoNotRewritePhotoPoses(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	feature.ToggleLink()

	panes := renderedPanes(feature.Overlay())
	paneHoverable(t, panes[0]).MouseIn(&desktop.MouseEvent{})
	for range 3 {
		feature.HandleKey(fyne.KeyPlus)
	}
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 20)})
	paneHoverable(t, panes[1]).MouseIn(&desktop.MouseEvent{})
	for range 2 {
		feature.HandleKey(fyne.KeyPlus)
	}
	paneDraggable(t, panes[1]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-40, -20)})
	home := renderedPanes(feature.Overlay())
	wantHome := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: home[0].image.Size(), position: home[0].image.Position()},
		{size: home[1].image.Size(), position: home[1].image.Position()},
	}
	wantRatio := wantHome[0].size.Width / wantHome[1].size.Width

	feature.ToggleLink()
	feature.HandleKey(fyne.Key0)
	afterFit := renderedPanes(feature.Overlay())
	if got := afterFit[0].image.Size().Width / afterFit[1].image.Size().Width; !uitest.ApproxEqual(got, wantRatio) {
		t.Errorf("photo ratio after camera fit = %.4f, want retained %.4f", got, wantRatio)
	}
	wantFit := [2]struct {
		size     fyne.Size
		position fyne.Position
	}{
		{size: afterFit[0].image.Size(), position: afterFit[0].image.Position()},
		{size: afterFit[1].image.Size(), position: afterFit[1].image.Position()},
	}

	feature.ToggleLink()
	feature.ToggleLink()
	stillFit := renderedPanes(feature.Overlay())
	for i, pane := range stillFit {
		if pane.image.Size() != wantFit[i].size || !approxPosition(pane.image.Position(), wantFit[i].position) {
			t.Errorf("pane %d camera-fit geometry changed across toggles: got {%v %v}, want {%v %v}",
				i, pane.image.Size(), pane.image.Position(), wantFit[i].size, wantFit[i].position)
		}
	}

	feature.HandleKey(fyne.Key1)
	afterHome := renderedPanes(feature.Overlay())
	for i, pane := range afterHome {
		if pane.image.Size() != wantHome[i].size || !approxPosition(pane.image.Position(), wantHome[i].position) {
			t.Errorf("pane %d after camera home = {%v %v}, want stored photo pose {%v %v}",
				i, pane.image.Size(), pane.image.Position(), wantHome[i].size, wantHome[i].position)
		}
	}
}

func TestCompareCameraPan_CursorReportsTableMovementAvailability(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)

	panes := renderedPanes(feature.Overlay())
	if got := paneCursorable(t, panes[0]).Cursor(); got != desktop.PointerCursor {
		t.Errorf("fitted comparison cursor = %v, want pointer for camera movement", got)
	}
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes = renderedPanes(feature.Overlay())
	if got := paneCursorable(t, panes[0]).Cursor(); got != desktop.PointerCursor {
		t.Errorf("pannable comparison cursor = %v, want pointer", got)
	}
}

func TestCompareCameraPan_RepeatedExtremeInputKeepsBothPhotosOverPaneCenters(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}

	for i := range 20 {
		panes := renderedPanes(feature.Overlay())
		delta := fyne.NewDelta(10000, 10000)
		if i%2 != 0 {
			delta = fyne.NewDelta(-20000, -20000)
		}
		paneDraggable(t, panes[i%2]).Dragged(&fyne.DragEvent{Dragged: delta})
		panes = renderedPanes(feature.Overlay())
		for _, pane := range panes {
			assertPaneOverlapsCenter(t, pane)
		}
	}
}

func TestCompareCameraPan_RepeatedInputOnEitherPaneKeepsOnePointDelta(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}

	for i := range 100 {
		panes := renderedPanes(feature.Overlay())
		delta := fyne.NewDelta(3, -2)
		if i%2 != 0 {
			delta = fyne.NewDelta(-3, 2)
		}
		wantPositions := [2]fyne.Position{
			panes[0].image.Position().Add(delta),
			panes[1].image.Position().Add(delta),
		}
		paneDraggable(t, panes[i%2]).Dragged(&fyne.DragEvent{Dragged: delta})
		panes = renderedPanes(feature.Overlay())
		for pane, want := range wantPositions {
			if got := panes[pane].image.Position(); !approxPosition(got, want) {
				t.Fatalf("iteration %d pane %d position = %v, want camera delta at %v", i, pane, got, want)
			}
		}
	}
}

func TestCompareFitReset_PanThenZeroRestoresCanonicalTransform(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes := renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})

	feature.HandleKey(fyne.Key0)
	panes = renderedPanes(feature.Overlay())
	wantSizes := [2]fyne.Size{fyne.NewSize(400, 200), fyne.NewSize(100, 400)}
	for i, pane := range panes {
		if pane.image.Size() != wantSizes[i] || normalizedCenter(pane) != fyne.NewPos(0.5, 0.5) {
			t.Errorf("pane %d after fit reset = size %v center %v, want %v and (0.5,0.5)",
				i, pane.image.Size(), normalizedCenter(pane), wantSizes[i])
		}
	}
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	for i, pane := range panes {
		if normalizedCenter(pane) != fyne.NewPos(0.5, 0.5) {
			t.Errorf("pane %d center after fit reset then + = %v, want canonical (0.5,0.5)", i, normalizedCenter(pane))
		}
	}
}

func TestCompareLoading_StartsBothSourcesConcurrentlyAndCompletesReady(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan string, 2)
	release := make(chan struct{})
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, Callbacks{})
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})

	names := []string{waitStarted(t, started), waitStarted(t, started)}
	slices.Sort(names)
	if !slices.Equal(names, []string{"left.png", "right.png"}) {
		t.Fatalf("started sources = %v, want both sides", names)
	}
	if feature.Ready() {
		t.Fatal("comparison became ready while both loaders were blocked")
	}
	close(release)
	waitForDone(t, feature)
	if !feature.Ready() {
		t.Fatal("completion signal fired before both panes became ready")
	}
}

func TestCompareOpen_NotifiesOwnerBeforeLoaderStarts(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	var opened atomic.Bool
	var loaderStartedEarly atomic.Bool
	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		if !opened.Load() {
			loaderStartedEarly.Store(true)
		}
		return loadedImage(32, 24), nil
	}, Callbacks{Opened: func() {
		opened.Store(true)
	}})
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitForDone(t, feature)

	if !opened.Load() {
		t.Fatal("Open did not notify its owner")
	}
	if loaderStartedEarly.Load() {
		t.Fatal("comparison loader started before the owner was notified")
	}
}

func TestCompareCancel_BackCancelsBothWorkersWithoutFailure(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan string, 2)
	cancelled := make(chan string, 2)
	failures := 0
	feature := newReferenceFeature(func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-ctx.Done()
		cancelled <- uri.Name()
		return nil, ctx.Err()
	}, Callbacks{Failed: func(fyne.URI, error) { failures++ }})
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	waitStarted(t, started)
	waitStarted(t, started)

	test.Tap(backButton(t, feature.Overlay()))
	waitForDone(t, feature)
	cancelledNames := []string{waitStarted(t, cancelled), waitStarted(t, cancelled)}
	slices.Sort(cancelledNames)
	if !slices.Equal(cancelledNames, []string{"left.png", "right.png"}) {
		t.Errorf("cancelled sources = %v, want both sides", cancelledNames)
	}
	if failures != 0 {
		t.Errorf("failure callbacks after user cancellation = %d, want 0", failures)
	}
}

func TestCompareFailure_ClosesAndCancelsTheOtherSide(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	for _, failingName := range []string{"left.png", "right.png"} {
		t.Run(failingName, func(t *testing.T) {
			wantErr := errors.New("broken image")
			cancelled := make(chan string, 1)
			var failedURI fyne.URI
			var failedErr error
			feature := newReferenceFeature(func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
				if uri.Name() == failingName {
					return nil, wantErr
				}
				<-ctx.Done()
				cancelled <- uri.Name()
				return nil, ctx.Err()
			}, Callbacks{Failed: func(uri fyne.URI, err error) {
				failedURI, failedErr = uri, err
			}})
			feature.Open([2]fyne.URI{
				storage.NewFileURI("left.png"),
				storage.NewFileURI("right.png"),
			})
			waitForDone(t, feature)

			if feature.Visible() || feature.Ready() {
				t.Fatal("a failed comparison must close instead of leaving partial content")
			}
			if failedURI == nil || failedURI.Name() != failingName || !errors.Is(failedErr, wantErr) {
				t.Errorf("failure callback = (%v, %v), want (%s, %v)", failedURI, failedErr, failingName, wantErr)
			}
			otherName := "right.png"
			if failingName == otherName {
				otherName = "left.png"
			}
			select {
			case got := <-cancelled:
				if got != otherName {
					t.Errorf("cancelled source = %q, want %q", got, otherName)
				}
			default:
				t.Errorf("%s failure did not cancel %s", failingName, otherName)
			}
		})
	}
}

func TestCompareStale_OlderCompletionCannotRepaintANewerSession(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	oldRelease := make(chan struct{})
	oldStarted := make(chan string, 2)
	newLeft := loadedImage(101, 51)
	newRight := loadedImage(202, 52)
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		switch uri.Name() {
		case "old-left.png", "old-right.png":
			oldStarted <- uri.Name()
			<-oldRelease // deliberately ignores cancellation to exercise staleness
			return loadedImage(7, 7), nil
		case "new-left.png":
			return newLeft, nil
		default:
			return newRight, nil
		}
	}, Callbacks{})
	feature.Open([2]fyne.URI{
		storage.NewFileURI("old-left.png"),
		storage.NewFileURI("old-right.png"),
	})
	oldHandle := feature.Done().Current()
	waitStarted(t, oldStarted)
	waitStarted(t, oldStarted)

	feature.Open([2]fyne.URI{
		storage.NewFileURI("new-left.png"),
		storage.NewFileURI("new-right.png"),
	})
	waitForDone(t, feature)
	close(oldRelease)
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	if err := oldHandle.Wait(ctx); err != nil {
		t.Fatal("timed out waiting for stale comparison workers")
	}

	panes := renderedPanes(feature.Overlay())
	if len(panes) != 2 || panes[0].image.Image != newLeft.Frames[0] || panes[1].image.Image != newRight.Frames[0] {
		t.Error("stale completion replaced the newer comparison images")
	}
}
