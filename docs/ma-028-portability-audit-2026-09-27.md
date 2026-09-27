# MA-028 portability and privacy audit

Date: 2026-09-27. Scope: the local `.scratch/ma-028` folder before the new
native-CI evidence was added (106 files, about 14 MiB, including 40 PNGs).
This is a migration assessment, not approval to publish raw desktop evidence.
No folder move or upload has been performed.

The linked, already-tracked Linux qualification summary also contained the
username in its FAT mount path. Its current text now uses `<user>` instead;
the original local evidence and checksums are unchanged. This is a privacy
cleanup, not a credential rotation. Earlier local commits still contain the
original path; no history rewrite has been authorized or performed.

## Recommendation

Move the 22 planning documents into tracked `docs/ma-028`: `spec.md`,
`interview.md`, the ten current `issues/*.md`, and the ten historical
`draft-issues/*.md`. Keep drafts explicitly historical and ticket 10 open.
These documents were read in full; the only personal machine path in that
subset is the pinned Fyne module reference in `interview.md`. Replace its
home-directory prefix with `$GOMODCACHE/`, preserving the exact module version
and source location.

Before committing a move, update inbound links in the plan, todos and design
records, and outbound links to tracked docs/plans. Audit links to other ignored
specs: either provide sufficient context in the portable document or label the
link local-only. Add a short index distinguishing accepted spec, current tickets,
historical drafts and tracked qualification summaries. Verify the resulting
link targets from a checkout without `.scratch`. No machine-specific launcher
is needed merely to carry the acceptance procedure to another desktop.

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
remaining acceptance checks. Publishing the branch and executing remote CI still
require separate authorization.

The curated move is awaiting the user's choice between moving portable Markdown
and leaving the folder in place. A wholesale move is not recommended.
