# PicFetch — TODOs

## Done

## Grid completions are marshalled through a drainable queue

`go test -race ./internal/ui/grid/` was failing on three tests, and the
cause was never the grid: Fyne's test driver implements `DoFromGoroutine`
as a bare `f()`, so every `fyne.Do` body ran on the decode-pool worker,
racing the test goroutine on `canvas.Image`, `widget.Label`, `g.matches`
and `g.groupSizes`. `Overview` now marshals through a per-instance
`uiQueue`: `fyneQueue` (a straight `fyne.Do`) in the app, `uitest.UIQueue`
under the package's own tests, drained by the looping `Settle`. A
`parkDecodes` harness helper fills the decode pool so a test can open a
cold grid with nothing decoding under it, which is what makes a
"hashes still pending" window deterministic. All three tests still open
the grid on a cold cache with hashes pending, so that flow keeps its
coverage.

## Hide-duplicates applies immediately and progressively

`SetHideDuplicates` now shows "Hiding duplicates" chrome right away and
rebuilds the hide filter after each `hashRemaining` job without yanking
the highlight; `Shift+D` browse still waits for the last hash job.
`newTestUI` installs `uitest.UIQueue` via `SetUIQueue` so the app test
suite drains grid completions the same way the grid package already did.
Mid-window coverage is
`TestSetHideDuplicates_PendingShowsChromeAndLeavesUnhashedVisible`.

## Duplicate finder (perceptual hash) in the grid — L

A grid mode that computes a perceptual hash (dHash/aHash — ~30 lines of
pure Go over the already-decoded thumbnails) per file in the background
via the existing bounded thumbnail worker pool, then groups visually
identical shots and badges duplicates. Combined with #1/#2, this answers
"which of these 400 photos are the same shot twice?" — a job no quick
viewer does well today. Hash work rides the thumbnail pipeline's
claim/semaphore machinery, so it inherits cancellation and memory bounds
for free.

## browse duplicates
- when highlighting an image that had duplicated and then pushing shift+d shows all duplicates for that item in a grid view

### bug in duplication detection
Tester bugreport:
- I am scanning a big amount of images of mixed formats
- if I go do g grid view
- and then select d to filter duplicates
- the first imgae in list (unsorted) is assigned thousands of duplicates
- when I highlight it and press shift d
- I see many different images moth of them are not duplicates of the inital image

Note (2026-08-24): root cause found and fixed — see
`.superpowers/sdd/opus5-dupes-report.md`. It was never the clustering.
`DifferenceHash` reduced the thumbnail with `draw.ApproxBiLinear`, which
samples a fixed 4 pixels per output cell no matter how far it is
downscaling, so any picture with a thin subject on a flat background
(line art, sketches, screenshots, letterboxed shots) lost its subject
entirely and hashed to a handful of set bits. Two hashes can never differ
by more than popcount(a)+popcount(b), so every such picture was within
distance 10 of every other one, regardless of subject. The 9×8 reduction
now area-averages every source pixel, and the default distance dropped
10 → 6.

Measured over 13,469 real files, against a same-shot oracle: precision
87.6% → 98.9%, worst single false group 54 → 4 extra files, at the cost
of 1.9% of true duplicate pairs. Grouping stayed complete linkage; on the
fixed hash star and complete linkage produce identical groups at 6.

Re-test: G → D on the first unsorted file, then Shift+D. **Check File ->
Settings… first** — a previously saved slider value wins over the new
default, so set Duplicate match distance to 6 by hand if it still says 10.

## ACTIVE DEVELOPMENT

## TODO

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
