# Archive privacy cleanup

Scope: recursively inspect and sanitize `finished_refactorings/`, including text,
structured logs, screenshots, and hidden metadata. Preserve useful technical
history. Git history and other folders are outside this cleanup.

Route: Standard documentation maintenance; no application behavior changes.
The archive has 477 files, including 51 images and two Finder metadata files.
Two read-only scouts cover infrastructure records and image/OCR discovery;
the lead owns all decisions, edits, and final review. The configured Scout tier
is unavailable, so the existing agents perform bounded read-only searches.

Acceptance criteria and verification:

- Personal paths, account details, local machine identifiers, private addresses,
  and any discovered credentials are replaced with explicit placeholders.
  Verify: local privacy scanner, including an observed failing baseline and
  a second independent pattern sweep after edits.
- Images are checked for embedded metadata and visible private information.
  Sensitive screenshots are withdrawn with an explicit notice and references
  updated; generated artwork and clean technical captures remain intact.
  Verify: image inventory, local OCR, lead inspection, reference check.
- JSON/JSONL retains its original parseability, and archived scripts retain
  their syntax. Verify: baseline versus final format checks and shell/Python
  syntax checks; `git diff --check`.
- Only the requested archive is changed at handoff. Verify: scoped Git status
  and a final redaction report. Public repository/dependency identifiers,
  versions, and technical checksums remain where they explain the evidence.

Tasks: inventory and failing privacy checks -> sanitize text and images ->
rescan and review -> archive this report. All edits and review are inline.
No production tests are needed for this documentation-only cleanup.

Results: complete. Scanned 477 original files: 424 text artifacts, 51 images,
and two hidden Finder metadata files. Updated 63 existing text files with
redactions and screenshot references. Removed both `.DS_Store` files.

Redacted personal home paths (including shortened Windows account paths),
user/host names, container identifiers, per-user temporary directory names,
private session/VM UUIDs, publisher personal details, and reviewer account
references. `REDACTED_*` values, zero UUIDs, and reviewer ID `0` are placeholders,
not working environment values. Archived commands containing them require
local configuration before reuse. Public project/dependency namespaces,
package identities, checksums, versions, and localhost test endpoints remain.
No exposed credential literal or remote private network address was found.
Email-shaped matches were Go module paths and one systemd unit name.

All 37 screenshots were visually inspected. Local Tesseract OCR covered all
114 image frames with no frame errors; image metadata was checked separately.
Six Windows screenshots exposed account or private desktop content and were
replaced by `.redacted.md` notices with their report references updated:

- `26-windows-linked-pan-occluded`
- `27-windows-sdk-unavailable`
- `27-windows-second-empty`
- `27-windows-first-alpha`
- `27-windows-first-beta`
- `27-windows-third-alpha`

The 31 other screenshots and 14 generated artwork/QA images remain unchanged.
No identifying embedded metadata was found. Original observations in the
reports remain historical claims; the withdrawn captures can no longer serve
as visual evidence. No synthetic replacement screenshots were created.

Verification completed:

- `python3 /private/tmp/picfetch-redact-archive.py discover` failed before
  redaction; `python3 /private/tmp/picfetch-redact-archive.py check` passed
  afterward with zero remaining discovered private values or Finder metadata.
- A separate pattern sweep found no remaining personal home paths, private
  temporary identifiers, private IP addresses, or credential-shaped literals.
  The remaining nonzero UUID identifies a public Microsoft SDK download.
- `python3 /private/tmp/picfetch-privacy-verify.py` verified all 71 JSON files
  against their original parseability: 43 JSON documents and 28 JSONL streams.
  UTF-8 BOMs and the legacy text encoding were handled explicitly.
- Python syntax checks passed for six archived scripts; `bash -n` passed for
  seven shell scripts. No archived script was executed.
- All 45 retained image files match their original Git bytes. All six
  screenshot-notice links resolve. Every existing text edit matches the
  specified substitutions/reference updates, with indentation preserved
  visually on redacted diagnostic lines.
- `git diff --check` passed. Cleanup edits are confined to this archive;
  unrelated working-tree changes were left intact.

Audit helpers ran locally. Temporary original-value maps and raw OCR were
deleted after verification. This cleanup covers the current folder only;
older Git commits, published copies, and other directories still retain their
original contents. No history rewrite or commit was performed.
