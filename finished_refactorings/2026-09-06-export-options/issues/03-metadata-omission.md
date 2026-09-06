# 03: Metadata omission

**What to build:** The export prompt gains a checkbox reading "Include camera
metadata (JPEG only)", checked by default. Checked is exactly today's behaviour:
the source JPEG's segments are copied onto the exported copy with orientation
normalized. Unchecking it performs **metadata omission** — the exported copy is
written without the source's identifying tags, and the source file keeps
everything it had.

This is deliberately not **metadata removal**, which is the EXIF window's
irreversible in-place rewrite of the original behind a "cannot be undone"
confirmation. The two operations must not share wording: export never says
"strip" or "remove", and never asks for confirmation, because it cannot harm
anything.

Omission reuses the imaging module's existing ICC-preserving encode path, so the
colour profile survives even though the identifying tags do not, and a source
whose orientation tag is not 1 still comes out upright. Note the existing
constraint that path documents: the Adobe APP14 segment must not be spliced back,
since it would misdeclare the encoder's colour transform.

PNG carries no metadata either way, which is why the label states the JPEG-only
scope permanently rather than the control greying itself out when the format
changes. The format buttons are the committing action, so the format is not known
until the moment of commit — a static caveat stays honest without costing the
fast path a keystroke.

The completion toast reports omission only when the box was unchecked, so a
routine export keeps today's short message.

**Blocked by:** 01

**Status:** done

- [x] The export prompt offers an "Include camera metadata (JPEG only)" checkbox, checked by default
- [x] Checked produces a file identical to what export writes today
- [x] Unchecked writes a JPEG with no identifying tags, verified by reading tags back out of the written file
- [x] The source file is never modified, whichever way the box is set
- [x] An embedded colour profile survives omission
- [x] A source whose orientation tag is not 1 is written upright when metadata is omitted
- [x] The Adobe APP14 segment is not spliced back onto an omitted-metadata export
- [x] Exporting to PNG behaves identically regardless of the checkbox
- [x] No confirmation dialog is shown; the wording never says "strip" or "remove"
- [x] The completion toast reports omission only when the box was unchecked
- [x] The option resets to checked every time the prompt opens, and is never persisted
- [x] New labels have translation entries in both shipped languages

## Comments

2026-09-06 — Delivered: "Include camera metadata (JPEG only)", checked by default, unchecking routes the export through the existing ICC-preserving encode. Drawn as a tappable label rather than a widget.Check, which grabs canvas focus on tap and would swallow Return and Escape from the prompt.
