package widgets

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// threeChoices gives clamping something real to hit at both ends - a
// two-choice card (deletion's own) can't tell "clamped" apart from "always
// jumps to the other end", since with two positions they're the same thing.
func threeChoices(chosen *[]int) []Choice {
	return []Choice{
		{Label: "A", OnChosen: func() { *chosen = append(*chosen, 0) }},
		{Label: "B", OnChosen: func() { *chosen = append(*chosen, 1) }},
		{Label: "C", OnChosen: func() { *chosen = append(*chosen, 2) }},
	}
}

func TestChoiceCard_SelectClampsAtTheLowEnd(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	c.Select(1)
	c.Select(-5)

	if got := c.Selected(); got != 0 {
		t.Errorf("Selected() = %d, want clamped to 0", got)
	}
	if !c.Ring(0).Visible() || c.Ring(1).Visible() || c.Ring(2).Visible() {
		t.Error("only index 0's ring should be visible after clamping low")
	}
}

func TestChoiceCard_SelectClampsAtTheHighEnd(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	c.Select(50)

	if got := c.Selected(); got != 2 {
		t.Errorf("Selected() = %d, want clamped to the last index (2)", got)
	}
	if c.Ring(0).Visible() || c.Ring(1).Visible() || !c.Ring(2).Visible() {
		t.Error("only the last index's ring should be visible after clamping high")
	}
}

func TestChoiceCard_HandleKeyLeftRightClampRatherThanWrap(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if got := c.Selected(); got != 0 {
		t.Errorf("Selected() = %d after Left at index 0, want 0 (no wrap)", got)
	}

	c.Select(2)
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got := c.Selected(); got != 2 {
		t.Errorf("Selected() = %d after Right at the last index, want 2 (no wrap)", got)
	}
}

func TestChoiceCard_ShowResetsSelectionToIndexZero(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	c.Select(2)
	c.Show("pick one")

	if got := c.Selected(); got != 0 {
		t.Errorf("Selected() after Show = %d, want reset to 0", got)
	}
	if !c.Ring(0).Visible() || c.Ring(1).Visible() || c.Ring(2).Visible() {
		t.Error("only index 0's ring should be visible after Show resets the selection")
	}
	if got := c.Message().Text; got != "pick one" {
		t.Errorf("Message().Text = %q, want %q", got, "pick one")
	}
}

func TestChoiceCard_ReturnRunsTheSelectedChoice(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("pick one")

	c.Select(1)
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if want := []int{1}; len(chosen) != 1 || chosen[0] != want[0] {
		t.Errorf("chosen = %v, want %v (index 1's OnChosen)", chosen, want)
	}
	if c.Visible() {
		t.Error("the card should hide once a choice is confirmed")
	}
}

// TestChoiceCard_ReturnHidesBeforeRunningTheChoice guards the ordering
// performDelete's move to Trash depends on: whatever the chosen action does
// (deletion's own trash.Move, a future export dialog) must not find the
// card still reporting itself visible.
func TestChoiceCard_ReturnHidesBeforeRunningTheChoice(t *testing.T) {
	var sawVisible bool
	var c *ChoiceCard
	c = NewChoiceCard(nil, Choice{
		Label:    "go",
		OnChosen: func() { sawVisible = c.Visible() },
	})
	c.Show("go?")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if sawVisible {
		t.Error("OnChosen ran while the card still reported itself visible")
	}
}

func TestChoiceCard_EnterAlsoConfirms(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("pick one")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEnter})

	if want := []int{0}; len(chosen) != 1 || chosen[0] != want[0] {
		t.Errorf("chosen = %v, want %v (the default index 0)", chosen, want)
	}
}

func TestChoiceCard_EscapeCancelsWithoutRunningAnyChoice(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("pick one")

	c.Select(2)
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if c.Visible() {
		t.Error("Escape should hide the card")
	}
	if len(chosen) != 0 {
		t.Errorf("chosen = %v, want none - Escape must not run any choice", chosen)
	}
}

func TestChoiceCard_EscapeRunsTheRegisteredOnCancel(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("pick one")

	cancelled := 0
	c.SetOnCancel(func() { cancelled++ })

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if cancelled != 1 {
		t.Errorf("onCancel ran %d times, want exactly 1", cancelled)
	}
}

func TestChoiceCard_EscapeWithoutOnCancelJustHides(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("pick one")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape}) // no SetOnCancel call - must not panic

	if c.Visible() {
		t.Error("Escape should still hide the card with no onCancel registered")
	}
}

func TestChoiceCard_VisibleTransitions(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	if c.Visible() {
		t.Error("a fresh card should start hidden")
	}

	c.Show("pick one")
	if !c.Visible() {
		t.Error("Visible() should be true after Show")
	}

	c.Hide()
	if c.Visible() {
		t.Error("Visible() should be false after Hide")
	}
}

func TestChoiceCard_RepaintFiresOnShowSelectAndHide(t *testing.T) {
	var chosen []int
	repaints := 0
	c := NewChoiceCard(func() { repaints++ }, threeChoices(&chosen)...)

	c.Show("pick one")
	if repaints == 0 {
		t.Error("Show should trigger at least one repaint")
	}

	before := repaints
	c.Select(1)
	if repaints <= before {
		t.Error("Select should trigger a repaint")
	}

	before = repaints
	c.Hide()
	if repaints <= before {
		t.Error("Hide should trigger a repaint")
	}
}

// TestChoiceCard_RepaintCallbackIsOptional: nil is what a caller with no
// repaint hook passes (this package's own tests do, above and below), and
// every mutating method has to tolerate it.
func TestChoiceCard_RepaintCallbackIsOptional(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	c.Show("pick one")
	c.Select(1)
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
}

func TestChoiceCard_ClickRunsThatButtonRegardlessOfSelection(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("pick one")

	c.Select(2) // keyboard selection points at index 2

	c.runChoice(0)() // a click on button 0 always runs button 0's action

	if want := []int{0}; len(chosen) != 1 || chosen[0] != want[0] {
		t.Errorf("chosen = %v, want %v - a click ignores the keyboard selection", chosen, want)
	}
}

// TestChoiceCard_ConfirmingAChoiceWithNoActionJustHides covers the shape
// deletion actually ships: its Cancel choice carries no OnChosen at all,
// because hiding is everything Cancel has ever had to do. Every fixture
// above hands each choice a function, so without this the nil branch in
// runChoice would only ever be exercised from another package's suite - and
// a regression there is a nil-func panic on the safest button on the card.
func TestChoiceCard_ConfirmingAChoiceWithNoActionJustHides(t *testing.T) {
	c := NewChoiceCard(nil, Choice{Label: "cancel"})
	c.Show("cancel?")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if c.Visible() {
		t.Error("confirming an action-less choice should still hide the card")
	}
}

// TestChoiceCard_NoChoicesIsInertRatherThanPanicking pins what a card built
// with no choices does, now that this is an exported widget any caller can
// reach: Select's clamp lands on 0 rather than on -1, and runChoice's range
// check keeps Return from indexing an empty slice. A choiceless card is
// still a caller's mistake - it just surfaces as a card that does nothing
// instead of a panic on the next key press.
func TestChoiceCard_NoChoicesIsInertRatherThanPanicking(t *testing.T) {
	c := NewChoiceCard(nil)

	c.Show("nothing to pick")
	if got := c.Selected(); got < 0 {
		t.Errorf("Selected() = %d, want a non-negative index", got)
	}

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if c.Visible() {
		t.Error("Return should still hide the card with nothing to run")
	}
}

func TestChoiceCard_RingReturnsNilOutOfRange(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)

	if c.Ring(-1) != nil {
		t.Error("Ring(-1) should be nil")
	}
	if c.Ring(3) != nil {
		t.Error("Ring(len(choices)) should be nil")
	}
}

// --- extra rows ------------------------------------------------------------
//
// The optional block a card can draw above its button row (the export
// prompt's size limit and metadata checkbox). Everything below is the card's
// side of that contract - which stop the keys go to, and which keys never
// leave the button row whichever stop is focused. What the rows themselves
// draw is their own package's business.

// fakeRows is a two-stop ExtraRows recording what the card hands it, so the
// tests below can assert on delegation without pulling in a real options
// widget. handles is the set of keys it claims to have used, so a test can
// play both a row with something to activate and a row without.
type fakeRows struct {
	content fyne.CanvasObject
	focused int
	keys    []fyne.KeyName
	handles map[fyne.KeyName]bool
	resets  int
}

func newFakeRows() *fakeRows {
	return &fakeRows{content: widget.NewLabel("rows"), focused: -1}
}

func (f *fakeRows) Content() fyne.CanvasObject { return f.content }

func (f *fakeRows) Rows() int { return 2 }

func (f *fakeRows) Focus(row int) { f.focused = row }

func (f *fakeRows) HandleKey(ev *fyne.KeyEvent) bool {
	f.keys = append(f.keys, ev.Name)

	return f.handles[ev.Name]
}

func (f *fakeRows) Reset() { f.resets++ }

// TestChoiceCard_ShowStartsOnTheButtonRow is what keeps Cmd/Ctrl+E followed
// by Return a two-keystroke export: however many rows the card carries, the
// selection starts on the buttons, not in the options above them.
func TestChoiceCard_ShowStartsOnTheButtonRow(t *testing.T) {
	var chosen []int
	rows := newFakeRows()
	c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)

	c.Show("pick one")

	if rows.focused != -1 {
		t.Errorf("rows focused = %d after Show, want -1 (the button row holds the selection)", rows.focused)
	}

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if len(chosen) != 1 || chosen[0] != 0 {
		t.Errorf("chosen = %v, want the first choice run straight from Show", chosen)
	}
}

// TestChoiceCard_ShowResetsTheRows pins the rule that the prompt always
// states the whole truth about what it is about to write: options never
// carry over from a previous open.
func TestChoiceCard_ShowResetsTheRows(t *testing.T) {
	var chosen []int
	rows := newFakeRows()
	c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)

	c.Show("first")
	c.Hide()
	c.Show("second")

	if rows.resets != 2 {
		t.Errorf("rows reset %d times, want one per Show (2)", rows.resets)
	}
}

func TestChoiceCard_UpAndDownMoveBetweenRowsAndButtonsAndClamp(t *testing.T) {
	var chosen []int
	rows := newFakeRows()
	c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)
	c.Show("pick one")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if rows.focused != 1 {
		t.Errorf("rows focused = %d after Up from the buttons, want the last row (1)", rows.focused)
	}
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if rows.focused != 0 {
		t.Errorf("rows focused = %d after a second Up, want the first row (0)", rows.focused)
	}
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if rows.focused != 0 {
		t.Errorf("rows focused = %d after Up at the top, want it clamped to 0", rows.focused)
	}

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if rows.focused != 1 {
		t.Errorf("rows focused = %d after Down, want row 1", rows.focused)
	}
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if rows.focused != -1 {
		t.Errorf("rows focused = %d after Down from the last row, want -1 (the buttons)", rows.focused)
	}
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if rows.focused != -1 {
		t.Errorf("rows focused = %d after Down at the bottom, want it clamped to the buttons", rows.focused)
	}
}

// TestChoiceCard_KeysGoToWhicheverStopHoldsTheSelection covers the split
// that makes one keyboard serve both blocks: Left/Right steer the buttons
// while the buttons hold the selection, and the rows once it has moved up
// into them.
func TestChoiceCard_KeysGoToWhicheverStopHoldsTheSelection(t *testing.T) {
	var chosen []int
	rows := newFakeRows()
	c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)
	c.Show("pick one")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got := c.Selected(); got != 1 {
		t.Errorf("Selected() = %d after Right on the button row, want 1", got)
	}
	if len(rows.keys) != 0 {
		t.Errorf("rows received %v, want nothing while the buttons hold the selection", rows.keys)
	}

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})

	if want := []fyne.KeyName{fyne.KeyRight, fyne.KeySpace}; len(rows.keys) != 2 ||
		rows.keys[0] != want[0] || rows.keys[1] != want[1] {
		t.Errorf("rows received %v, want %v once the selection moved up into them", rows.keys, want)
	}
	if got := c.Selected(); got != 1 {
		t.Errorf("Selected() = %d, want the button selection left where it was (1)", got)
	}
}

// TestChoiceCard_ReturnAndEscapeAlwaysReachTheButtons is the other half of
// that split: the two committing keys belong to the prompt as a whole, not
// to whichever row the selection happens to be sitting on.
func TestChoiceCard_ReturnAndEscapeAlwaysReachTheButtons(t *testing.T) {
	t.Run("return commits from a row with nothing to activate", func(t *testing.T) {
		var chosen []int
		rows := newFakeRows()
		c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)
		c.Show("pick one")

		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight}) // buttons: A -> B
		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})    // up into the rows
		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

		if len(chosen) != 1 || chosen[0] != 1 {
			t.Errorf("chosen = %v, want the selected button (1) run once the row declined Return", chosen)
		}
		if c.Visible() {
			t.Error("the card should be hidden after Return")
		}
	})

	// The other half of that rule, and the one a user pressing Return on a
	// highlighted checkbox is relying on: a row that says it used the key
	// keeps it, and the prompt stays up.
	t.Run("return is the row's when the row activates something", func(t *testing.T) {
		var chosen []int
		rows := newFakeRows()
		rows.handles = map[fyne.KeyName]bool{fyne.KeyReturn: true}
		c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)
		c.Show("pick one")

		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

		if len(chosen) != 0 {
			t.Errorf("chosen = %v, want nothing committed while the row was using Return", chosen)
		}
		if !c.Visible() {
			t.Error("the card should still be up: the row took Return, it did not commit")
		}
		if len(rows.keys) != 1 || rows.keys[0] != fyne.KeyReturn {
			t.Errorf("rows received %v, want the Return handed down to them", rows.keys)
		}
	})

	t.Run("escape cancels from a row", func(t *testing.T) {
		var chosen []int
		cancelled := 0
		rows := newFakeRows()
		c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)
		c.SetOnCancel(func() { cancelled++ })
		c.Show("pick one")

		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
		c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

		if cancelled != 1 {
			t.Errorf("onCancel ran %d times, want 1", cancelled)
		}
		if len(chosen) != 0 {
			t.Errorf("chosen = %v, want nothing run by Escape", chosen)
		}
		if c.Visible() {
			t.Error("Escape should hide the card from a focused row too")
		}
		if len(rows.keys) != 0 {
			t.Errorf("rows received %v, want Escape kept by the card", rows.keys)
		}
	})
}

// TestChoiceCard_RowsContentIsInTheCard walks the overlay rather than
// trusting Visible(): a widget built and then left out of its container
// still reports true, so only finding it under the overlay's root proves it
// reaches the screen.
func TestChoiceCard_RowsContentIsInTheCard(t *testing.T) {
	var chosen []int
	rows := newFakeRows()
	c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)

	if !inTree(c.Overlay(), rows.content) {
		t.Error("the rows' content should be part of the card's overlay")
	}
}

// TestChoiceCard_WithoutRowsIgnoresUpAndDown is the delete confirmation's
// card: the same widget with nothing passed for the rows slot, where Up and
// Down stay as inert as they have always been.
func TestChoiceCard_WithoutRowsIgnoresUpAndDown(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("delete?")

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyDown})

	if got := c.Selected(); got != 0 {
		t.Errorf("Selected() = %d after Up/Down on a card with no rows, want 0", got)
	}
	if !c.Visible() {
		t.Error("Up/Down must not dismiss a card with no rows")
	}
	if len(chosen) != 0 {
		t.Errorf("chosen = %v, want Up/Down to run nothing", chosen)
	}
}

// inTree reports whether want is somewhere under root, walking containers
// the way infoview's own inCard helper does.
func inTree(root, want fyne.CanvasObject) bool {
	if root == want {
		return true
	}
	if c, ok := root.(*fyne.Container); ok {
		for _, o := range c.Objects {
			if inTree(o, want) {
				return true
			}
		}
	}
	return false
}

// TestChoiceCard_OnlyOneRingIsAtFullStrength is the readability rule the
// extra rows made necessary: with a ring on the buttons and a mark on the
// focused row, two highlights at once leave the eye with no answer to
// "where am I?". The buttons keep showing which choice is selected, muted,
// and brighten again when the selection comes back down to them.
func TestChoiceCard_OnlyOneRingIsAtFullStrength(t *testing.T) {
	var chosen []int
	rows := newFakeRows()
	c := NewChoiceCardWithRows(nil, rows, threeChoices(&chosen)...)
	c.Show("pick one")

	full := strokeAlpha(t, c.Ring(0))

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	muted := strokeAlpha(t, c.Ring(0))
	if muted >= full {
		t.Errorf("the button ring is drawn at alpha %d with the selection up in the rows, want it muted below %d", muted, full)
	}
	if !c.Ring(0).Visible() {
		t.Error("the button ring should stay visible while muted - it is still the choice Return would run")
	}

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if back := strokeAlpha(t, c.Ring(0)); back != full {
		t.Errorf("the button ring is drawn at alpha %d back on the button row, want the full %d", back, full)
	}
}

// TestChoiceCard_WithoutRowsKeepsItsRingAtFullStrength is the delete
// confirmation: one block on the card, so nothing ever mutes.
func TestChoiceCard_WithoutRowsKeepsItsRingAtFullStrength(t *testing.T) {
	var chosen []int
	c := NewChoiceCard(nil, threeChoices(&chosen)...)
	c.Show("delete?")

	full := strokeAlpha(t, c.Ring(0))

	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})

	if got := strokeAlpha(t, c.Ring(1)); got != full {
		t.Errorf("ring alpha = %d on a card with no rows, want the full %d", got, full)
	}
}

// strokeAlpha is a ring's stroke opacity, the difference between "you are
// here" and "this is still the choice".
func strokeAlpha(t *testing.T, ring *canvas.Rectangle) uint8 {
	t.Helper()

	if ring == nil {
		t.Fatal("no ring to read a stroke from")
	}
	_, _, _, a := ring.StrokeColor.RGBA()

	return uint8(a >> 8)
}
