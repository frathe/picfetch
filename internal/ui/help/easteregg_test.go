package help

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestManualView_SecretPhraseFiresCallbackOnce(t *testing.T) {
	var calls int
	v := newManualView(searchFixture, func() { calls++ })

	v.entry.SetText(secretPhrase)
	v.submit(v.entry.Text)

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestManualView_SecretPhraseCaseAndWhitespaceInsensitive(t *testing.T) {
	cases := []string{
		"PLEASE HYPNOTIZE ME",
		"Please Hypnotize Me",
		"  please hypnotize me  ",
	}

	for _, q := range cases {
		t.Run(q, func(t *testing.T) {
			var calls int
			v := newManualView(searchFixture, func() { calls++ })

			v.entry.SetText(q)
			v.submit(q)

			if calls != 1 {
				t.Errorf("calls = %d, want 1 for %q", calls, q)
			}
		})
	}
}

func TestManualView_SubstringOfSecretDoesNotFireIt(t *testing.T) {
	var calls int
	v := newManualView(searchFixture, func() { calls++ })

	q := "say " + secretPhrase + " now"
	v.entry.SetText(q)
	v.submit(q)

	if calls != 0 {
		t.Errorf("calls = %d, want 0 for a mere substring match", calls)
	}
}

func TestManualView_SecretPhraseClearsEntryAndSearchState(t *testing.T) {
	v := newManualView(searchFixture, func() {})

	v.entry.SetText(secretPhrase)
	v.submit(v.entry.Text)

	if v.entry.Text != "" {
		t.Errorf("entry text = %q, want empty", v.entry.Text)
	}
	if got := hitTexts(v.text.Segments); len(got) != 0 {
		t.Errorf("highlighted segments = %v, want none", got)
	}
	if v.current != nil {
		t.Errorf("current = %+v, want nil", v.current)
	}
	if v.state != (searchState{}) {
		t.Errorf("state = %+v, want zero value", v.state)
	}
}

func TestManualView_OrdinaryQueryDoesNotFireSecretAndStillHighlights(t *testing.T) {
	var calls int
	v := newManualView(searchFixture, func() { calls++ })

	v.entry.SetText("alpha")
	v.submit(v.entry.Text)

	if calls != 0 {
		t.Errorf("calls = %d, want 0 for an ordinary query", calls)
	}
	if got := hitTexts(v.text.Segments); len(got) != 2 {
		t.Errorf("highlighted = %v, want two alphas", got)
	}
}

func TestHelp_SecretPhraseInManualOpensSpiral(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	h := New(a, "PicFetch", nil)
	original := currentManual
	currentManual = func() string { return "just words" }
	t.Cleanup(func() { currentManual = original })
	h.ShowManual()
	calls := 0
	h.SetOnSpiral(func() { calls++ })
	h.manual.entry.OnSubmitted(secretPhrase)
	if calls != 1 {
		t.Fatalf("secret callback calls = %d, want 1", calls)
	}
}

func TestHelp_SecretCallbackCanChangeAfterOpening(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	h := New(a, "PicFetch", nil)
	original := currentManual
	currentManual = func() string { return "just words" }
	t.Cleanup(func() { currentManual = original })
	h.ShowManual()
	h.manual.entry.OnSubmitted(secretPhrase) // Missing callback is harmless.
	calls := 0
	h.SetOnSpiral(func() { calls++ })
	h.manual.entry.OnSubmitted(secretPhrase)
	h.SetOnSpiral(func() { calls += 10 })
	h.manual.entry.OnSubmitted(secretPhrase)
	if calls != 11 {
		t.Fatalf("callbacks produced %d, want 11", calls)
	}
}

func TestHelp_FinisSearchOpensCompanion(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	h := New(a, "PicFetch", nil)
	original := currentManual
	t.Cleanup(func() { currentManual = original })
	currentManual = func() string { return "Finis likes pictures." }
	h.ShowManual()
	for _, query := range []string{"finis", " FINIS "} {
		h.manual.entry.SetText(query)
		h.manual.entry.OnSubmitted(query)
		count := 0
		for _, window := range a.Driver().AllWindows() {
			if window.Title() == "Finis" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("companion windows = %d, want 1", count)
		}
		if h.manual.entry.Text != "" {
			t.Fatal("secret search was not cleared")
		}
	}
}
