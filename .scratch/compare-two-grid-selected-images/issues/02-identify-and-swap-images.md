# 02: Identify and swap compared images

**What to build:** Make each side unambiguous and let the user exchange the
left and right roles without altering the grid selection or its order.

**Blocked by:** 01: Open and close a fitted side-by-side comparison

**Status:** ready-for-agent

## Acceptance criteria

- [ ] A compact translucent toolbar remains visible at the top right of the
  comparison content. It includes working **Swap** and **Back to Grid**
  controls; **Swap** stays disabled until both images are ready while **Back to
  Grid** stays enabled.
  Verify: `go test ./internal/ui/... -run 'CompareToolbar' -count=1`
- [ ] Translucent badges identify the left and right images at the respective
  bottom corners. Each normally shows its base filename. When the base names
  match, both show the shortest distinguishing suffix containing directory and
  filename components.
  Verify: `go test ./internal/ui/... -run 'CompareIdentity' -count=1`
- [ ] While comparison is active, the title is exactly `Compare: left.jpg |
  right.jpg - PicFetch`, using the displayed badge identities in left-to-right
  order. It updates immediately after Swap and the highlighted grid title is
  restored on exit.
  Verify: `go test ./internal/ui/... -run 'CompareTitle' -count=1`
- [ ] **Swap** exchanges the displayed images, badges, title order, and logical
  left/right roles. In the currently available side-by-side layout it keeps
  the comparison fitted and does not restart decoding.
  Verify: `go test ./internal/ui/... -run 'CompareSwap' -count=1`
- [ ] Swap never mutates selection membership, selection gesture order, file
  order, filter state, highlight, or scroll position in the covered grid.
  Verify: `go test ./internal/ui/... -run 'CompareSwapPreservesGrid' -count=1`
- [ ] New toolbar and identity strings are localized in every catalogue and
  documented in both manuals.
  Verify: `go test ./... -run 'Translations|Manual' -count=1`

## Comments

