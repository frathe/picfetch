# MA-028 portability and privacy audit

Date: 2026-09-27. Scope: the local `.scratch/ma-028` folder before the new
native-CI evidence was added (106 files, about 14 MiB, including 40 PNGs).
The user subsequently approved the curated move, commit, push and PR creation.
The 22 reviewed planning documents now live under [docs/ma-028](ma-028/README.md),
with one home-path redaction and updated links. Raw desktop evidence remains
ignored and is excluded from publication.

The linked, already-tracked Linux qualification summary also contained the
username in its FAT mount path. Its current text now uses `<user>` instead;
the original local evidence and checksums are unchanged. This is a privacy
cleanup, not a credential rotation. Earlier local commits still contain the
original path; no history rewrite has been authorized or performed.

## Approved migration

Moved the 22 planning documents into tracked `docs/ma-028`: `spec.md`,
`interview.md`, the ten current `issues/*.md`, and the ten historical
`draft-issues/*.md`. Drafts remain explicitly historical and ticket 10 open.
These documents were read in full; the only personal machine path in that
subset was the pinned Fyne module reference in `interview.md`. Its home-directory
prefix is now `$GOMODCACHE/`, preserving the exact module version and source
location. A direct comparison confirmed all 22 moves contained only the planned
path/link substitutions before current ticket/publication status was updated.

Inbound links in the plan, todos, design and backlog now point to the tracked
files. The index separates accepted spec, current tickets, historical drafts
and qualification summaries. The tracker guide records this feature-specific
location. Raw-artifact references are labeled optional local evidence rather
than broken links into an ignored folder. No machine-specific launcher is
needed merely to carry the acceptance procedure to another desktop.

## Findings and treatment

| Material | Observation | Treatment |
| --- | --- | --- |
| Planning Markdown | No credential-pattern matches; one personal home path in the interview | Redact that prefix, fix links, retain decisions and evidence limitations |
| Raw logs, shell scripts and overlay JSON | Personal workspace/home paths and disposable mount/temp paths occur in 12 text files across the original folder | Keep ignored originals; any later export needs a separately redacted copy |
| Favorite JSON and clipboard text | Favorite records point only to the two temporary fixtures; retained editor text is the synthetic marker `ma028_editor_marker` | No personal collection or clipboard text found in these files; raw files are unnecessary for the planning move |
| Fixture images | The two input images are byte-identical to the repository's astronaut/coffee fixtures | They add no new source-photo content; do not infer that every screenshot is similarly safe |
| PNG metadata | No output for the inspected author/owner/comment/description/host, GPS, XMP or IPTC fields | Limited metadata check only; all 40 screenshot/image pixel contents have **not** been visually cleared for publication |
| Native helper | C source plus a non-stripped ELF executable | Keep binary and machine-specific harness outside tracked docs |
| Mutation evidence | Includes `mutant-viewer.go`, an intentionally broken Go source copy, and an absolute-path overlay | Keep ignored; moving it wholesale into docs would expose it to recursive Go package discovery |
| Checksums | Existing manifests identify the original raw evidence | Preserve originals; sanitized exports need their own new manifests |

The text scan checked private-key headers, common GitHub/AWS/OpenAI credential
formats and assignments to password, secret, token, API-key, authorization and
cookie names. It found no matches in the scanned text files. Binary/image data
and checksum manifests were excluded from that scan. This is not a guarantee
that arbitrary sensitive information is absent. Email/credential-bearing URL
checks found none; observed external documentation links are public sources.

The tracked implementation and Linux qualification records already carry the
meaningful observations and limitations without requiring the raw artifacts on
every desktop. Keep those summaries portable; raw artifact links must be labeled
as optional local evidence, not prerequisites for understanding or running the
remaining acceptance checks. The user has now authorized publishing the branch
and opening a PR; remote CI results remain unverified until inspected.

The curated move preserves the raw evidence and its original checksums. No
binary, image, log or intentionally broken mutation source was added to docs.

## Migration verification

- All 22 expected destination documents were initially absent; the move then
  passed direct comparison against the original text with only the approved
  redaction/link substitutions. Current ticket/publication status was updated
  separately afterwards; historical drafts retain their original meaning.
- The portable index adds one Markdown file. The staged migration contains no
  non-Markdown files or `.scratch` artifacts.
- A relative-link check against the Git index passed all 117 scoped links;
  each target is available from tracked files without the original `.scratch`.
- The credential/home-path scan found no matches in `docs/ma-028`.
  `git diff --cached --check` and retained raw-evidence checksums passed.
- Source/workflow hashes still match the inspected and fully verified
  `ba43d1b` revision. This documentation-only move carries that evidence forward;
  it does not rerun or claim completed remote/native acceptance.
