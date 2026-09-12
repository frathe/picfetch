# Reusable Git history movie

Route: Standard, lead implementation and review, no delegation. The deliverable
is `make movie`: the previously accepted Gource film generated from the current
committed history, without host graphics/Python/Homebrew dependencies.

## Decisions and scope

- Capture HEAD once; include its full ancestry, preserve topological order and
  clamp backward committer times. Reconcile merged branch file sets against the
  actual merge tree; verify the final set against the captured HEAD.
- Use Docker, Git, Bash and Make on the host. Build the official Debian render
  image locally; run it without networking, as a non-root user, with a read-only
  root filesystem and bounded resources. Mount scripts read-only and generated
  history/output only. Never mount the repository or Docker socket in the renderer.
- Produce 1080p60 H.264/AAC, matching the original titles, glowing file tree,
  counters, chart and original synth score. `MOVIE_SECONDS=180` controls duration;
  `MOVIE_DIR=.scratch/history-movies` controls the parent of unique run folders.
- Retain raw history, metadata, exact tool/package versions and verification.
  Derive captions from feature/release commit subjects with enough spacing to remain readable.
  Do not change app runtime dependencies or translate development-only tooling.
- A movie visualizes file operations, not source lines or uncommitted work.
  Tool popularity and isolation do not establish vulnerability-free software.

## Tasks and acceptance evidence

| Task | Files / contract | Verification | Owner / budget |
| --- | --- | --- | --- |
| History and timing | `scripts/historymovie/movie.py`, `test_movie.py`: NUL-delimited Git input, merge reconciliation, dynamic counts, safe display names, bounded duration | `make movie-test`; fixtures for backward times, merge deletion, unusual filenames and a single timestamp | Lead; no spawns |
| Docker entry point and render | `run.sh`, `Dockerfile`, Makefile: capture one HEAD, unique output, cleanup on interruption, Docker-only render | `bash -n scripts/historymovie/run.sh`; `make movie`; full FFmpeg decode and final tree check | Lead; no spawns |
| Documentation and final gate | script README, architecture index, `todos.md`, this record | GoLand changed-file inspections, `make verify`, final film stills | Lead; one full suite |

## Tool provenance and licenses

Reuse the original film's researched toolchain. Official Debian base:
`debian:trixie-slim@sha256:d7e12182ce18b85b93007c1dedf31f2d29e01ccf3182cc4017c709b6259bc132`.
Debian signed repositories, including security updates, supply Gource 0.54-1+b3
(GPL-3.0-or-later), FFmpeg 7:7.1.5-0+deb13u1 (Debian default GPL-2.0-or-later),
Xvfb 2:21.1.16-1.3+deb13u4 (X/MIT notices), Pillow 11.1.0-5+deb13u4
(HPND), NumPy 1:2.2.4+ds-1 (BSD and bundled notices), and DejaVu 2.37-8
(font notices). These are development tools; no binaries/fonts are shipped in
PicFetch. Keep package copyright notices and the full installed-package manifest
with each movie. New builds may receive security updates; record actual versions
and image ID rather than claiming byte-for-byte reproducibility across rebuilds.

Sources: <https://github.com/acaudwell/Gource>,
<https://github.com/acaudwell/Gource/wiki/Videos>,
<https://packages.debian.org/trixie/gource>,
<https://packages.debian.org/trixie/ffmpeg>,
<https://security-tracker.debian.org/tracker/source-package/gource>,
<https://security-tracker.debian.org/tracker/source-package/ffmpeg>.
Original detailed research is retained with the September 12 film under Movies.

## Evidence

- Regression tests were first observed failing on unimplemented history/timing
  behavior, then passing. A later caption regression failed when a late feature
  milestone was dropped, and passed after the selection fix.
- `make movie-test`: seven tests pass inside Docker. `bash -n` and the Make help
  entry pass; `MOVIE_SECONDS=0` is rejected before rendering.
- `make movie`: generated and fully decoded a 180.0-second 1080p60 H.264/AAC
  film, with exactly 355 commits and a 121-to-1,238-file replay matching HEAD.
  Evidence: `.scratch/history-movies/20260912-165835-b76593c-Bfzjzc/`.
- GoLand reports no errors/warnings for `movie.py`, `test_movie.py`, `run.sh`,
  `Dockerfile`, and Makefile. Its build tool returns success with the caveat
  "Build messages cannot be collected"; independent Make vet/build checks pass.
- Black formatting ran in a disposable Debian container; nothing was installed
  on the host or added to the render image for formatting.
- A final-code 30.0-second smoke movie passed full decode and matching-tree
  verification with `MOVIE_DIR='.scratch/history movies smoke'`, proving the
  custom duration and paths containing spaces. Evidence:
  `.scratch/history movies smoke/20260912-170307-b76593c-6FbW91/`.
  The first-title and completed-tree stills were visually checked for framing,
  legible counters/legend and the final 355 commits / 1,238 files.
- Five guard checks rejected deliberate faults in isolated temporary copies:
  final-tree validation, clock clamping, merge reconciliation, delimiter
  encoding and duration validation. Production source was unchanged; evidence
  is `.scratch/history-movies/guard-checks.json`.
- `make verify` completed. Formatting, TUF, Qodana exclusions, vet and build
  passed. All 681 UI race tests passed across the three shards (408.567,
  418.239 and 438.664 seconds). The non-UI partition passed 1,919 top-level tests
  and reproduced only the two previously documented local amd64 seccomp
  failures: `TestLinuxWorkerIsolation` and
  `TestAssetInstall/worker_reaches_asset_check_and_exits`, both reporting
  `offline worker seccomp: invalid argument`. Thus the overall gate exits 2;
  it is not claimed clean. Full evidence:
  `.scratch/race-runs/20260912T145942Z-dEE7tc/` and
  `.scratch/history-movies/make-verify.log`.
- Final GoLand inspections of both Python files remain clear after caption
  changes; `git diff --check` is clean. No app source changed and no commit
  was created. Suggested commit: `Add Docker Git history movie target`.

| Task | Spawns (budget/actual) | Review rounds | Full suite |
| --- | --- | --- | --- |
| Implementation and movie QA | 0 / 0 | 2 | no |
| Repository gate | 0 / 0 | 1 | once, complete with known baseline failures |
