# 07: Document, render, and verify Copy Selection

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Finish the user and architecture documentation, add the
real-viewer golden scenario and manual guard, then run every focused acceptance
command and PicFetch's complete verification gate.

**Blocked by:** 06

**Status:** ready-for-agent

## Documentation and evidence checklist

- [ ] Update `ARCHITECTURE.md` with `internal/ui/copyselection`, its narrow
      interface, the viewer adapter, and the new overlay's exact position in the
      load-bearing stack. Do not duplicate the feature spec there.
- [ ] Update English and German manuals with Actions -> Copy selection,
      `Option`/`Alt+Shift+C`, drawing/moving/resizing, `Escape`, `Return`,
      full-resolution PNG output, and the animated/SVG/RAW/large-crop limits.
- [ ] Use only ASCII `->` for menu paths and preserve all manual formatting
      constraints.
- [ ] Add `TestManualDocumentsCopySelection` so both manuals are checked for
      the feature and shortcut rather than relying on visual review alone.
- [ ] Add `TestE2E_CopySelection` through `newTestUI`, asserting state before
      comparing the real viewer render.
- [ ] Regenerate only with `make golden`; inspect
      `internal/ui/testdata/failed/*.png`, then accept only the intended Copy
      Selection baseline. Never commit failed renders.
- [ ] Run every AC command from `spec.md` and read its output.
- [ ] Negatively verify the new manual guard and one real-viewer mode guard,
      restore them, and rerun their commands.
- [ ] Run `make verify` as the final gate.
- [ ] Confirm no `TODO`/`FIXME` was added, `internal/clipboard` is unchanged,
      and the existing Copy image and Grid selection behaviors still pass.

## Files

- Modify: `ARCHITECTURE.md`
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `internal/ui/help/manual_test.go`
- Modify: `internal/ui/e2e_test.go`
- Add accepted golden under `internal/ui/testdata/`
- Do not create an ADR or change `CONTEXT.md`; terminology was resolved during
  specification.

## Verification

```sh
go test ./internal/ui/help -run 'Test(ManualDocumentsCopySelection|ManualHasNoUnicodeArrows)$'
go test ./internal/ui -run 'Test(E2E_CopySelection|TranslationsHaveNoUnicodeArrows)$'
rg -n 'internal/ui/copyselection' ARCHITECTURE.md
make verify
```

When every ticket and acceptance criterion is green, check every box, set
`Status: resolved`, and summarize documentation, golden inspection, focused
commands, negative guard checks, and `make verify` output under `## Answer`.
Do not commit.

## Answer

Pending.
