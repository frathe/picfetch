# 01: Export prompt grows an options area, carrying an options value

**What to build:** The export prompt gains room for the controls tickets 02 and
03 will fill in, and an options value is threaded from the prompt through the
viewer's export runner into the imaging module's export entry point. The value
carries defaults only in this ticket, so the written file stays byte-identical to
what export produces today, and nothing the user can observe changes.

**The prompt stays a choice card. Do not port it to a Fyne dialog.** That was the
original plan here and it was wrong. The card's `Overlay()` is a member of the
window's own content stack, whose paint order is documented as load-bearing so
the prompt appears over an open grid; the app's key dispatcher routes to the card
through an explicit visibility check, entirely separately from the generic
"a Fyne dialog owns the keyboard" branch above it; and eight test files reach the
prompt through the card's own API — including one that asserts its **index within
the overlay stack**. A dialog port breaks all of that to gain nothing.

The objection that originally motivated the port — that the card cannot support
vertical navigation because the app dispatcher owns Up — is false. While the card
is visible the dispatcher hands it *every* key and returns; the card's key handler
simply ignores Up and Down today. They are free in exactly the state that needs
them.

So the work is additive: the shared choice card grows an optional slot for extra
rows above its button row, and optional Up/Down delegation for moving between
them. The delete confirmation shares this widget and must pass nothing, leaving
its own card byte-identical.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] The export prompt remains a choice card; its `Overlay()` stays in the window content stack at its current position
- [ ] The shared choice card accepts optional extra rows above its button row, and optional Up/Down delegation
- [ ] The delete confirmation passes no extra rows and renders and behaves exactly as before
- [ ] Format buttons still commit the export; Escape still cancels without opening the save panel
- [ ] The prompt is still refused during a comparison, while loading, and while the delete confirmation is visible
- [ ] Raising the delete confirmation is still refused while the export prompt is up
- [ ] Re-invoking the export shortcut while the prompt is up does not reset an already-moved selection
- [ ] An options value is carried from the prompt through the viewer's export runner into the imaging module's export entry point
- [ ] The imaging module's other export caller (the wallpaper path) passes defaults and is otherwise unchanged
- [ ] With defaults, exported files are byte-identical to what the previous implementation produced
- [ ] Every existing test that reaches the export prompt through the card's API still compiles and passes unmodified, across all eight files that do so
- [ ] Verified with the Docker test target, not a native `go test` run — see the verification note in the spec
