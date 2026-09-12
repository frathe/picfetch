# Git history movie

Run from the repository root:

```sh
make movie
```

Requires a running Docker engine, Git, Bash and Make. No host Python, Gource,
FFmpeg or Homebrew installation is needed. The first run builds a local Docker
image from official Debian packages; subsequent runs reuse it.

The command prints the full path to `PicFetch-history.mp4` in a new folder under
`.scratch/history-movies/`. Each run gets its own folder, preserving older movies.

```sh
make movie MOVIE_SECONDS=120
make movie MOVIE_DIR="$HOME/Movies/PicFetch"
make movie-test
```

`MOVIE_SECONDS` is the total length, including titles and the final hold, and
accepts 30–900 seconds (default 180). The result is 1920 × 1080 at 60 fps with
H.264 video, AAC stereo audio, chapter markers, a file-category legend, live
file/commit counts, a growth chart and an original synthesized ambient score.
Caption candidates come from feature and release commits spaced along the timeline.
Dates, counts, graph scale, captions and commit identity are regenerated on each
run. The current source tree may contain uncommitted edits; only the captured
commit and its ancestry are visualized.

## History fidelity

Git is read on the host. The exporter captures HEAD once and reads its complete
ancestry in topological order, with NUL-delimited paths. A shallow checkout is
rejected with instructions to fetch its missing history. Renames are shown as
deletions and additions. Empty/merge commits count even when they emit no file
changes. Backward committer clocks are clamped to the preceding timestamp;
original timestamps are retained in `timeline.json`.

Branch histories can mention the same path more than once or modify a path
already deleted on another branch. The replay normalizes these into valid
visible-tree operations and reconciles every merge against its actual tree.
Reconciliation additions/deletions belong to the merge and are reported in
`summary.json`. The final set must exactly match `git ls-tree` at captured HEAD
before rendering begins. The movie describes file activity, not line counts.
Paths/names requiring Gource delimiter escaping are percent-encoded for display.

All files remain visible until a recorded deletion. Playback keeps the calendar
spacing between commits; quiet periods are not skipped. Titles use the final
commit's UTC offset consistently throughout the movie. Very dense bursts may
apply several commits between video frames; the source log retains every event.

## Tools and isolation

The base is the official `debian:trixie-slim` image pinned by digest in
`Dockerfile`. Apt verifies signed Debian repositories, including security updates.
[Gource](https://github.com/acaudwell/Gource) supplies the animation;
[FFmpeg](https://github.com/FFmpeg/FFmpeg) supplies encoding/assembly, as documented
in [Gource's video guide](https://github.com/acaudwell/Gource/wiki/Videos).
Pillow draws the titles and chart; NumPy synthesizes the original score directly
from tones. The soundtrack is provided without a music royalty fee and uses no
downloaded music, third-party recordings or samples. There is no external music
license attached to it. Software and font licenses are recorded separately below.

The render runs as a non-root user with networking disabled, all capabilities
dropped, no-new-privileges, a read-only root filesystem, a temporary `/tmp`,
and limits of six CPUs and 4 GiB RAM. It mounts only the scripts (read-only) and
the run's exported history/output. The repository and Docker socket are not
mounted. Ctrl+C removes the task's own container; partial output remains for
diagnosis. It does not change host software permissions or install host tools.

The tools are established projects, but neither popularity nor isolation proves
absence of vulnerabilities. Consult the Debian security trackers for
[Gource](https://security-tracker.debian.org/tracker/source-package/gource) and
[FFmpeg](https://security-tracker.debian.org/tracker/source-package/ffmpeg).

Gource is GPL-3.0-or-later; Debian's default FFmpeg build is GPL-2.0-or-later;
Pillow uses the HPND license; NumPy uses BSD with bundled notices; Xvfb and DejaVu
carry their own notices. They are development tools, not runtime dependencies
shipped with PicFetch. Every run retains the full installed-package manifest,
image identity and package copyright notices. A future image rebuild can receive
updated packages; its recorded versions identify what actually ran.

## Output and verification

Each run retains the movie, poster, history exports, replay summary, timeline,
chapters, intermediate media, exact render/assembly commands, recipe copy,
package versions/notices and logs. `verification.json` records the SHA-256,
stream format/duration, matching audio/video lengths and complete FFmpeg decode.
Intermediate assets can occupy several hundred megabytes for a three-minute film.

`make movie-test` runs the replay/timing regression tests inside the same image.
The tests cover merges, backward clocks, empty commits, unusual filenames,
invalid exports, duration bounds and single-timestamp histories. A real
`make movie` run verifies rendering and encoding in addition to those tests.
