# PicFetch — TODOs

## Done

### 3. Extract the directory-scan walker out of `handleDrop`

The recursive folder scan — symlink-cycle guard, per-scan dedupe, the
`maxScan` cap, the throttled progress callback — is now `internal/filescan`'s
`Images(ctx, uris, max, progress)`, tested with just `test.NewApp()` in
`TestMain` instead of a full viewer. `handleDrop` (`internal/ui/drop.go`) is
UI glue: snapshot merge mode and the cap, show the spinner, call `Images`,
apply the result. Both drop paths now share the one walker, so the `maxScan`
cap applies to loose-file drops too, not just recursive folder scans.

### 4. Split `internal/ui/grid/grid.go` into four files

`internal/ui/grid/grid.go` (995 lines, one file holding four separable
concerns) is now four files in the same package, with no API change:
`grid.go` keeps `Host`, `Overview`, construction, and the
Toggle/Close/Overlay lifecycle; `nav.go` took the highlight ring and its key
dispatch; `search.go` took the `/` filename filter and the display→host
index mapping; `thumbs.go` took the thumbnail cache and its bounded decode
pipeline. `grid_test.go` split the same four ways plus a new
`harness_test.go` for the shared fake `Host` and helpers (`openGrid`,
`typeQuery`) that `selection_test.go` also uses. Pure motion: every
declaration moved byte-identical, nothing renamed, no visibility change, no
exported API change.

### Extract the shared bounded decode pool into `internal/decodepool` (item 4's stretch goal)

The grid's thumbnail decode pool (`thumbs.go`) and the viewer's preload pool
(`preloadSem`/`preloading`/`preloadPending`) duplicated the same
semaphore/in-flight-claim/WaitGroup trio. Both now share one generic
`Pool[K, V]` in a new `internal/decodepool` package: `Claim`/`Release` for
the per-key in-flight guard, `Go`/`Wait` for the bounded worker pool and its
test-drainable completion count. `internal/ui/grid`'s `Overview` collapsed
`sem`/`pending`/`inflight` into one `decodes
*decodepool.Pool[*fyne.Container, int]`; `internal/ui`'s `viewer` collapsed
`preloadSem`/`preloading`/`preloadPending` into one `preloads
*decodepool.Pool[string, struct{}]`. `stillWanted` and `cellIDs` stayed
behind in `internal/ui/grid`: deciding whether a finished decode still
belongs on its cell needs the host generation, the filter generation, and
cell recycling, none of which a general-purpose pool should know about — it
answers only whether identical work is already in flight, not whether that
work still matters.

### 5. Unify the test-synchronization channels behind one small type

The viewer's nine ad-hoc `chan struct{}` fields that shared the same
replace-on-start / close-on-finish / wait-in-test contract — `loadDone`,
`animStopped`, the scan/sort `asyncOpUI` instances' own `done` field,
`toast.done`, `clipboardDone`, `chooserDone`, `wallpaperDone`, `favThumbDone`
— are now one type, `internal/completion.Signal`: `Begin() (done func())`,
`Wait(ctx) error`, `Begun() bool`, and `Current() Handle` for the one case a
bare `Wait` can't cover — proving a *specific*, since-superseded
generation's goroutine actually exited, not just that whatever generation is
current has finished. Mirrors `internal/decodepool`'s precedent exactly: one
audited type replacing N hand-rolled copies of the same contract, with the
type owning the mechanism and each caller keeping its own staleness rules on
top of it. Nine field-comment restatements of "a superseded generation must
still finish its own channel" and eleven hand-rolled `select`/`time.After`
waiters collapsed into the type itself plus one pair of test helpers,
`waitFor`/`waitHandle` in `harness_test.go`.

Three things deliberately stayed behind, not folded into `Signal`: `animFrame`
stayed a plain `atomic.Uint64` — it's an N-event counter a test polls (every
frame `animate` writes one), not a one-shot completion. `toast.stop` stayed a
raw cancel channel next to the new `toast.hidden` Signal — a cancel is a
different contract from a completion, and `cancelAutoHide`'s nil-out of
`stop` is load-bearing: it's what answers "is an auto-hide currently
pending", which a Signal (monotonic, never un-begins) can't express.
`vector.pending`, `preloads`, and `grid.Settle`/`slides.Settle` stayed
N-goroutine `sync.WaitGroup` waits — `Signal` is a one-shot, not a shape that
generalizes to "wait out however many goroutines happen to be in flight".

### 6. Preserve JPEG metadata on Save Changes and JPEG export

Rotated JPEG saves and JPEG exports from a JPEG source no longer drop EXIF
and other APP-segment metadata. `internal/imaging/jpegexif.go` re-reads the
source file, splices COM/APPn segments after SOI (skipping JFIF APP0 and MPF
APP2), sets Exif Orientation to 1, and unlinks IFD1 so a stale thumbnail
cannot show the unrotated photo. `SaveRotated` uses this on JPEG save;
`Export` uses it when both source and destination are JPEG (`internal/ui/export.go`
passes the source URI; wallpaper passes nil). Other formats and PNG exports
remain a plain re-encode with no metadata carry-over.

### 7. Remove Metadata from JPEG in the EXIF window

The EXIF panel (`E` / info overlay's "Show EXIF data" link) now has a
**Remove Metadata** button at the bottom for JPEGs only — hidden for HEIC,
RAW, PNG, and WebP. A `widgets.ChoicePanel` confirmation on the panel's own
window (`confirm.go`, not the main-window `ChoiceCard`) asks before
`internal/imaging`'s `StripJPEGMetadata` rewrites the original file in place:
COM/APPn privacy segments (camera, date, GPS, XMP, IPTC, comments) go via
`jpegexif.go`'s `stripJPEGSegments`/`keepOnStrip` (the inverse of the
preserve-metadata splice), while JFIF APP0, Adobe APP14, and ICC APP2 stay.
Exif orientation 2–8 triggers a one-time quality-95 re-encode so the photo
stays upright without the tag, then splices the original ICC profile back
(`encodeJPEGKeepingICC`); orientation 1 is a lossless header strip.
View-only `R` rotation is ignored — Save Changes first. `Host` supplies
`DisplayedFile`, `AfterMetadataRemoved(u)`, and `ShowToast`; cache and info
overlay refresh through the host callback.

## ACTIVE DEVELOPMENT

## TODO

### Truncate MPF / motion-photo trailers when stripping JPEG metadata

`StripJPEGMetadata` drops the MPF APP2 index in the JPEG header but copies
the scan through EOF, so a second image concatenated after the primary EOI
(and that copy's own Exif/GPS) can survive a privacy strip. Truncating at
the primary EOI is a separate change: it needs fixtures and a careful look
at which trailers are safe to drop.

### Migrate `internal/ui/exifwin`'s `warmDone` onto `internal/completion.Signal`

`internal/ui/exifwin/exifwin.go`'s `warmDone chan struct{}` (field at
`exifwin.go:108`, set in `startWarm` at `exifwin.go:400`) is a tenth
hand-rolled copy of the same replace-on-start / close-on-finish / wait-in-test
contract that item 5's nine fields on `viewer` collapsed into
`internal/completion.Signal` — `exifwin_test.go:290`'s `waitForWarm` still
hand-rolls the nil-guard-plus-`select` that `waitFor` replaced everywhere
else. `internal/completion` is viewer-independent (no Fyne types, no
`fyne.Do`), so `exifwin` could import it with no cycle. It stayed out of item
5's scope because that plan named only the nine fields on `viewer`. Not
migrated here — a separate change.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)

