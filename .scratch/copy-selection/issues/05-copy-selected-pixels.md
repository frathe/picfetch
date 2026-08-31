# 05: Copy the selected pixels

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Capture the image shown when Copy Selection mode begins,
freeze animation when necessary, prepare the correct oriented full-resolution
pixel source for raster, SVG, and RAW inputs, and run crop, PNG encoding, and
the existing clipboard dispatcher off the UI thread with deterministic
completion and retry behavior.

**Blocked by:** 04

**Status:** ready-for-agent

## Copy contract

- Capture the source frame and its oriented coordinate bounds on the UI thread
  at mode entry. The rectangle callback later refers to that stable source.
- For raster and RAW inputs, use the oriented displayed frame; RAW remains the
  embedded preview.
- For SVG, rasterize at oriented logical dimensions through the existing vector
  path and safety cap, not at the current zoom-dependent canvas raster size.
- Pause animated frame advancement before capturing the visible frame and
  resume it when the mode ends. Exact timing phase need not survive.
- Use the Copy Selection module's PNG function, then call the unchanged
  `clipboard.CopyImage` dispatcher.
- Reuse `viewer.clipboard` completion signaling. Completion covers encoding,
  OS dispatch, error reporting, and the final Fyne UI update so tests can wait
  without sleeping.

## Behavior checklist

- [ ] Start with a failing pixel-level viewer test using a literal multicolor
      image and a nontrivial rectangle; verify exact PNG dimensions and pixels.
- [ ] Prove output is independent of fit/manual zoom, pan, window size, and
      device scaling and contains no overlay/button pixels.
- [ ] Preserve alpha and apply EXIF orientation plus unsaved view rotation.
- [ ] Freeze and copy the frame visible at activation for an animated image,
      then resume animation after success and cancel.
- [ ] Cover SVG logical-resolution output under the existing cap and RAW
      preview output with real synthetic inputs from `internal/uitest`.
- [ ] Disable editing and repeated copy while the worker is pending.
- [ ] On success, finish the mode silently. On recoverable crop, encode, or
      clipboard failure, show the existing error-style toast, retain the
      rectangle, and unlock retry.
- [ ] Add no new crop limit and do not promise recovery from process-level
      memory exhaustion.
- [ ] Add the worker to `newTestUI` drain cleanup and give every goroutine a
      staleness/cancellation path plus an observable completion signal.

## Files

- Modify: `internal/ui/copyselection.go`
- Modify: `internal/ui/clipboard.go` or create one focused viewer-side file
- Modify animation/display glue only where required
- Modify: `internal/ui/harness_test.go`
- Create or modify focused tests under `internal/ui/`
- Do not modify: `internal/clipboard`, menus, translations, manuals, or
  `ARCHITECTURE.md`

## Verification

```sh
go test -race ./internal/ui/copyselection ./internal/ui -run 'TestCopySelection(Pixels|Transparency|Rotation|AnimatedFrame|SVG|RAWPreview|Busy|Success|EncodeFailure|ClipboardFailure)$'
```

Negatively verify at least one pixel-fidelity guard, restore it, rerun the
command, set `Status: resolved`, and record the result under `## Answer`. Do not
commit.

## Answer

Pending.
