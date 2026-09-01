package compare_test

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"slices"
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
	"github.com/frathe/picfetch/internal/ui/compare"
	"github.com/frathe/picfetch/internal/uitest"
)

const waitTimeout = 5 * time.Second

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

func waitForDone(t *testing.T, feature *compare.Feature) {
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

func assertPaneCoversOrCenters(t *testing.T, pane renderedPane) {
	t.Helper()
	assertAxis := func(name string, position, imageSize, viewportSize float32) {
		t.Helper()
		if imageSize <= viewportSize+0.5 {
			want := (viewportSize - imageSize) / 2
			if !uitest.ApproxEqual(position, want) {
				t.Errorf("%s non-overflowing image position = %.3f, want centered %.3f", name, position, want)
			}
			return
		}
		if position > 0.01 || position+imageSize < viewportSize-0.01 {
			t.Errorf("%s overflowing image span = %.3f..%.3f, want to cover 0..%.3f",
				name, position, position+imageSize, viewportSize)
		}
	}
	assertAxis("horizontal", pane.image.Position().X, pane.image.Size().Width, pane.root.Size().Width)
	assertAxis("vertical", pane.image.Position().Y, pane.image.Size().Height, pane.root.Size().Height)
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

	feature := compare.New(nil, compare.Callbacks{})
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
	feature := compare.New(loader, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, compare.Callbacks{})
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

func TestCompareSwipeToggle_PermanentReadyGatedAndRelabels(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan string, 2)
	release := make(chan struct{})
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, compare.Callbacks{})
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

	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		index := 0
		if uri.Name() == "right.png" {
			index = 1
		}
		return &imaging.LoadedImage{Frames: []image.Image{frames[index]}}, nil
	}, compare.Callbacks{Repaint: func() { repaints.Add(1) }})
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

func TestCompareDividerKeys_StepClampAndNoopSideBySide(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	var modifiers fyne.KeyModifier
	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, compare.Callbacks{Modifiers: func() fyne.KeyModifier { return modifiers }})
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

func TestCompareLayoutToggle_PreservesLinkedTransformAndDivider(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(1600, 800), nil
	}, compare.Callbacks{})
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
	if center := normalizedCenter(inSwipe[0]); !approxPosition(center, wantCenter) {
		t.Errorf("normalized center after entering swipe = %v, want retained %v", center, wantCenter)
	}
	if multiplier := inSwipe[0].image.Size().Width / 800; !uitest.ApproxEqual(multiplier, wantMultiplier) {
		t.Errorf("fit-relative multiplier after entering swipe = %.4f, want retained %.4f", multiplier, wantMultiplier)
	}
	feature.HandleKey(fyne.KeyRight)

	test.Tap(comparisonButton(t, feature.Overlay(), "Side by side"))
	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 440) {
		t.Errorf("divider center after layout round trip = %.2f, want retained 440", center)
	}
	afterRoundTrip := renderedPanes(feature.Overlay())
	if center := normalizedCenter(afterRoundTrip[0]); !approxPosition(center, wantCenter) {
		t.Errorf("normalized center after layout round trip = %v, want retained %v", center, wantCenter)
	}
	if multiplier := afterRoundTrip[0].image.Size().Width / 800; !uitest.ApproxEqual(multiplier, wantMultiplier) {
		t.Errorf("fit-relative multiplier after layout round trip = %.4f, want retained %.4f", multiplier, wantMultiplier)
	}
}

func TestCompareLayoutTransition_PreservesFitStateAndDivider(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(1600, 800), nil
		}
		return loadedImage(600, 1200), nil
	}, compare.Callbacks{})
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
	wantCenter := normalizedCenter(before[0])
	wantMultiplier := before[0].image.Size().Width / 400
	if approxPosition(wantCenter, fyne.NewPos(0.5, 0.5)) || wantMultiplier <= 1 {
		t.Fatalf("transition setup = {center:%v multiplier:%.4f}, want a panned zoomed view", wantCenter, wantMultiplier)
	}

	assertState := func(name string, fitted [2]fyne.Size) {
		t.Helper()
		panes := renderedPanes(feature.Overlay())
		if len(panes) != 2 {
			t.Fatalf("%s rendered panes = %d, want 2", name, len(panes))
		}
		for i, pane := range panes {
			if center := normalizedCenter(pane); !approxPosition(center, wantCenter) {
				t.Errorf("%s pane %d center = %v, want retained %v", name, i, center, wantCenter)
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
	assertState("swipe", [2]fyne.Size{fyne.NewSize(800, 400), fyne.NewSize(200, 400)})
	feature.HandleKey(fyne.KeyRight)

	test.Tap(comparisonButton(t, feature.Overlay(), "Side by side"))
	assertState("side by side", [2]fyne.Size{fyne.NewSize(400, 200), fyne.NewSize(200, 400)})

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	assertState("swipe round trip", [2]fyne.Size{fyne.NewSize(800, 400), fyne.NewSize(200, 400)})
	divider, _ := horizontalResizeTarget(t, feature.Overlay())
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, 440) {
		t.Errorf("divider after layout round trip = %.2f, want retained 440", center)
	}
}

func TestCompareResize_PreservesFitStateAndReclampsCenter(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		swipe                   bool
		proportionalSize        fyne.Size
		proportionalFitted      fyne.Size
		clampingSize            fyne.Size
		clampingFitted          fyne.Size
		wantClampedCenterX      float32
		wantProportionalDivider float32
		wantClampedDivider      float32
	}{
		{
			name:               "side by side",
			proportionalSize:   fyne.NewSize(1000, 500),
			proportionalFitted: fyne.NewSize(500, 250),
			clampingSize:       fyne.NewSize(2400, 400),
			clampingFitted:     fyne.NewSize(800, 400),
			wantClampedCenterX: 0.48,
		},
		{
			name:                    "swipe",
			swipe:                   true,
			proportionalSize:        fyne.NewSize(1000, 500),
			proportionalFitted:      fyne.NewSize(1000, 500),
			clampingSize:            fyne.NewSize(2000, 400),
			clampingFitted:          fyne.NewSize(800, 400),
			wantClampedCenterX:      0.5,
			wantProportionalDivider: 550,
			wantClampedDivider:      1100,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)

			feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
				if uri.Name() == "large.png" {
					return loadedImage(1600, 800), nil
				}
				return loadedImage(1200, 600), nil
			}, compare.Callbacks{})
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
			paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(10000, 0)})
			panes = renderedPanes(feature.Overlay())
			wantCenter := fyne.NewPos(0.32, 0.5)
			if center := normalizedCenter(panes[0]); !approxPosition(center, wantCenter) {
				t.Fatalf("initial clamped center = %v, want %v", center, wantCenter)
			}

			assertResizeState := func(name string, fitted fyne.Size, wantCenter fyne.Position) {
				t.Helper()
				panes := renderedPanes(feature.Overlay())
				if len(panes) != 2 {
					t.Fatalf("%s rendered panes = %d, want 2", name, len(panes))
				}
				for i, pane := range panes {
					if center := normalizedCenter(pane); !approxPosition(center, wantCenter) {
						t.Errorf("%s pane %d center = %v, want %v", name, i, center, wantCenter)
					}
					wantSize := fyne.NewSize(fitted.Width*1.5625, fitted.Height*1.5625)
					if !uitest.ApproxEqual(pane.image.Size().Width, wantSize.Width) ||
						!uitest.ApproxEqual(pane.image.Size().Height, wantSize.Height) {
						t.Errorf("%s pane %d image size = %v, want fit-relative %v", name, i, pane.image.Size(), wantSize)
					}
					assertPaneCoversOrCenters(t, pane)
				}
			}

			win.Resize(tc.proportionalSize)
			assertResizeState("proportional resize", tc.proportionalFitted, wantCenter)
			if tc.swipe {
				divider, _ := horizontalResizeTarget(t, feature.Overlay())
				if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, tc.wantProportionalDivider) {
					t.Errorf("proportional resize divider = %.2f, want %.2f", center, tc.wantProportionalDivider)
				}
			}

			win.Resize(tc.clampingSize)
			assertResizeState("clamping resize", tc.clampingFitted, fyne.NewPos(tc.wantClampedCenterX, 0.5))
			if tc.swipe {
				divider, _ := horizontalResizeTarget(t, feature.Overlay())
				if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, tc.wantClampedDivider) {
					t.Errorf("clamping resize divider = %.2f, want %.2f", center, tc.wantClampedDivider)
				}
			}
		})
	}
}

func TestCompareActualSizeTransition_PreservesOneToOneScale(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	nativeSizes := [2]fyne.Size{fyne.NewSize(1600, 800), fyne.NewSize(1200, 600)}
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "large.png" {
			return loadedImage(1600, 800), nil
		}
		return loadedImage(1200, 600), nil
	}, compare.Callbacks{})
	win := test.NewWindow(feature.Overlay())
	win.SetPadded(false)
	win.Resize(fyne.NewSize(800, 400))
	t.Cleanup(win.Close)
	feature.Open([2]fyne.URI{
		storage.NewFileURI("large.png"),
		storage.NewFileURI("small.png"),
	})
	waitForDone(t, feature)
	feature.HandleKey(fyne.Key1)

	panes := renderedPanes(feature.Overlay())
	paneDraggable(t, panes[0]).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(100, 50)})
	panes = renderedPanes(feature.Overlay())
	wantCenter := fyne.NewPos(0.4375, 0.4375)
	if center := normalizedCenter(panes[0]); !approxPosition(center, wantCenter) {
		t.Fatalf("actual-size setup center = %v, want %v", center, wantCenter)
	}

	assertActual := func(name string, wantCenter fyne.Position) {
		t.Helper()
		panes := renderedPanes(feature.Overlay())
		if len(panes) != 2 {
			t.Fatalf("%s rendered panes = %d, want 2", name, len(panes))
		}
		for i, pane := range panes {
			if pane.image.Size() != nativeSizes[i] {
				t.Errorf("%s pane %d size = %v, want native 1:1 size %v", name, i, pane.image.Size(), nativeSizes[i])
			}
			if center := normalizedCenter(pane); !approxPosition(center, wantCenter) {
				t.Errorf("%s pane %d center = %v, want closest valid %v", name, i, center, wantCenter)
			}
			assertPaneCoversOrCenters(t, pane)
		}
	}

	test.Tap(comparisonButton(t, feature.Overlay(), "Swipe"))
	assertActual("swipe", wantCenter)

	win.Resize(fyne.NewSize(1000, 500))
	assertActual("resized swipe", wantCenter)

	win.Resize(fyne.NewSize(2000, 500))
	clampedCenter := fyne.NewPos(0.5, wantCenter.Y)
	assertActual("wide swipe", clampedCenter)

	test.Tap(comparisonButton(t, feature.Overlay(), "Side by side"))
	assertActual("wide side by side", clampedCenter)
}

func TestCompareDividerReset_NewOpenStartsSideBySideAtHalf(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(800, 400), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		return images[uri.Name()], nil
	}, compare.Callbacks{OrderChanged: func(left, right string) {
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
	feature.Close()

	open("next-left.png", "next-right.png")
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

	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(32, 24), nil
	}, compare.Callbacks{})
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
			feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
				return loadedImage(32, 24), nil
			}, compare.Callbacks{})
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

	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return loadedImage(32, 24), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		loads.Add(1)
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		loads.Add(1)
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, compare.Callbacks{OrderChanged: func(left, right string) {
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
	wantCenter := normalizedCenter(before[0])
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
		if center := normalizedCenter(pane); !approxPosition(center, wantCenter) {
			t.Errorf("pane %d center after Swap = %v, want retained %v", i, center, wantCenter)
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.png" {
			return left, nil
		}
		return right, nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return loadedImage(40, 20), nil
	}, compare.Callbacks{})
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

func TestCompareVector_RerasterizesAcrossZoomLayoutResizeAndSwap(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	vector := loadedVector(t, 40, 20)
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return loadedImage(40, 20), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.png" {
			return &imaging.LoadedImage{Frames: []image.Image{left}}, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{right}}, nil
	}, compare.Callbacks{})
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

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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

func TestCompareActualSize_UsesEachImagesOwnPixelDimensionsAndRecenters(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)
	feature.HandleKey(fyne.KeyPlus)

	feature.HandleKey(fyne.Key1)

	panes := renderedPanes(feature.Overlay())
	wantSizes := [2]fyne.Size{fyne.NewSize(800, 400), fyne.NewSize(200, 800)}
	wantPositions := [2]fyne.Position{fyne.NewPos(-200, 0), fyne.NewPos(100, -200)}
	for i, pane := range panes {
		if pane.image.Size() != wantSizes[i] || pane.image.Position() != wantPositions[i] {
			t.Errorf("pane %d at actual size = {%v %v}, want {%v %v}",
				i, pane.image.Position(), pane.image.Size(), wantPositions[i], wantSizes[i])
		}
	}
	feature.HandleKey(fyne.KeyPlus)
	panes = renderedPanes(feature.Overlay())
	wantZoomed := [2]fyne.Size{fyne.NewSize(1000, 500), fyne.NewSize(250, 1000)}
	for i, pane := range panes {
		if pane.image.Size() != wantZoomed[i] {
			t.Errorf("pane %d after actual size then + = %v, want absolute-step %v", i, pane.image.Size(), wantZoomed[i])
		}
	}
}

func TestCompareFit_ReturnsBothImagesToCanonicalCenteredFit(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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

func TestCompareZoom_WheelAnchorsOnePaneAndUpdatesTheSharedView(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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
	anchored := normalizedPoint(before[0], cursor)
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
	if got := normalizedPoint(after[0], cursor); !approxPosition(got, anchored) {
		t.Errorf("normalized point under cursor after wheel = %v, want anchored %v", got, anchored)
	}
	if left, right := normalizedCenter(after[0]), normalizedCenter(after[1]); !approxPosition(left, right) {
		t.Errorf("normalized centers after wheel = left %v right %v, want linked", left, right)
	}
}

func TestCompareClamp_WheelUsesBothImagesValidPanRanges(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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
		assertPaneCoversOrCenters(t, pane)
	}
	if left, right := normalizedCenter(panes[0]), normalizedCenter(panes[1]); !approxPosition(left, right) {
		t.Errorf("normalized centers after clamp = left %v right %v, want linked", left, right)
	}
}

func TestComparePanInputs_DragAndShiftWheelMoveTheSharedCenter(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	var modifiers fyne.KeyModifier
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{Modifiers: func() fyne.KeyModifier { return modifiers }})
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
	leftPosition := before[0].image.Position()
	draggable := paneDraggable(t, before[0])
	draggable.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 20)})
	draggable.DragEnd()

	afterDrag := renderedPanes(feature.Overlay())
	if got := afterDrag[0].image.Position(); !approxPosition(got, leftPosition.Add(fyne.NewPos(40, 20))) {
		t.Errorf("dragged pane position = %v, want %v", got, leftPosition.Add(fyne.NewPos(40, 20)))
	}
	if left, right := normalizedCenter(afterDrag[0]), normalizedCenter(afterDrag[1]); !approxPosition(left, right) {
		t.Errorf("normalized centers after drag = left %v right %v, want linked", left, right)
	}

	modifiers = fyne.KeyModifierShift
	rightPosition := afterDrag[1].image.Position()
	paneScrollable(t, afterDrag[1]).Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(-15, 25)})
	afterShiftWheel := renderedPanes(feature.Overlay())
	if got := afterShiftWheel[1].image.Position(); !approxPosition(got, rightPosition.Add(fyne.NewPos(-15, 25))) {
		t.Errorf("Shift+wheel pane position = %v, want %v", got, rightPosition.Add(fyne.NewPos(-15, 25)))
	}
	if left, right := normalizedCenter(afterShiftWheel[0]), normalizedCenter(afterShiftWheel[1]); !approxPosition(left, right) {
		t.Errorf("normalized centers after Shift+wheel = left %v right %v, want linked", left, right)
	}
}

func TestComparePanInputs_CursorReportsSharedPanAvailability(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("wide.png"),
		storage.NewFileURI("tall.png"),
	})
	waitForDone(t, feature)

	panes := renderedPanes(feature.Overlay())
	if got := paneCursorable(t, panes[0]).Cursor(); got != desktop.DefaultCursor {
		t.Errorf("fitted comparison cursor = %v, want default", got)
	}
	for range 7 {
		feature.HandleKey(fyne.KeyPlus)
	}
	panes = renderedPanes(feature.Overlay())
	if got := paneCursorable(t, panes[0]).Cursor(); got != desktop.PointerCursor {
		t.Errorf("pannable comparison cursor = %v, want pointer", got)
	}
}

func TestCompareNoOverscroll_RepeatedExtremePanStopsAtSharedBoundaries(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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
			assertPaneCoversOrCenters(t, pane)
		}
	}
}

func TestCompareNoDrift_RepeatedInputOnEitherPaneKeepsOneCenter(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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
		paneDraggable(t, panes[i%2]).Dragged(&fyne.DragEvent{Dragged: delta})
		panes = renderedPanes(feature.Overlay())
		if left, right := normalizedCenter(panes[0]), normalizedCenter(panes[1]); !approxPosition(left, right) {
			t.Fatalf("iteration %d normalized centers = left %v right %v, want linked", i, left, right)
		}
	}
}

func TestCompareFitReset_PanThenZeroRestoresCanonicalTransform(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "wide.png" {
			return loadedImage(800, 400), nil
		}
		return loadedImage(200, 800), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-release
		return loadedImage(32, 24), nil
	}, compare.Callbacks{})
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
	feature := compare.New(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		if !opened.Load() {
			loaderStartedEarly.Store(true)
		}
		return loadedImage(32, 24), nil
	}, compare.Callbacks{Opened: func() {
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
	feature := compare.New(func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-ctx.Done()
		cancelled <- uri.Name()
		return nil, ctx.Err()
	}, compare.Callbacks{Failed: func(fyne.URI, error) { failures++ }})
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
			feature := compare.New(func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
				if uri.Name() == failingName {
					return nil, wantErr
				}
				<-ctx.Done()
				cancelled <- uri.Name()
				return nil, ctx.Err()
			}, compare.Callbacks{Failed: func(uri fyne.URI, err error) {
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
	feature := compare.New(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
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
	}, compare.Callbacks{})
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
