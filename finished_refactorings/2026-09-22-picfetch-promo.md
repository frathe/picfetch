# PicFetch three-minute promotion film

Route: Standard artifact production. Owner: Pico, inline; zero subagents.

Deliver a 180-second 1920x1080, 30 fps H.264/AAC film with upbeat original
electronic music, readable animated feature stories, PicFetch branding and a
download call to action. Use supplied repository screenshots and artwork.
Application code, releases and publishing are outside this task.

Decisions: English captions; widescreen; no voice-over; 120 BPM original score;
cyan/violet/amber on deep navy; real screenshots plus clearly identified feature
illustrations. Do not advertise in-progress HEIC qualification or imply that all
network activity is blocked on Windows. Local AI setup receives a short qualifier.

1. Create `.scratch/picfetch-promo/produce.py`, production/source notes and scene
   timeline. Verify descriptions against README and the embedded manual. The
   existing brainstorming skill named by the process is unavailable; design and
   storyboarding are done inline. No product behavior is changed, so artifact
   validation replaces application TDD and the full application suite.
2. Render a contact sheet and inspect every scene before full encoding. Synthesize
   drums, bass, pads and melodic leads with no external recordings or samples.
3. Render and verify the complete film; retain a poster, standalone music,
   timestamps, source recipe and machine-readable verification alongside it.

Acceptance evidence:

- `produce.py preview`: all 22 scenes fit 1080p, with readable titles and captions;
  inspect the resulting contact sheets and selected full-size frames.
- `produce.py render`: 5,400 frames, exactly 180 seconds, stereo AAC and H.264,
  fast-start playback and chapters; scene audio cuts align with 120 BPM bars.
- `produce.py verify`: ffprobe geometry/timing/streams and full ffmpeg decode;
  loudness and peak analysis; no unintended black or frozen spans; artifact hash.
- GoLand inspection of the production script, with availability recorded.

Rendering tools are development-only: local FFmpeg 9.0.1 (GPL-enabled), Pillow
12.3.0 (HPND) and NumPy 2.5.3 (BSD-3-Clause and bundled notices). Pillow/NumPy are
installed only in the artifact's isolated venv. No new shipped app dependency.
Fonts are locally installed Arial faces, used as rendered pixels, not bundled.
Existing branded art and screenshots retain their repository provenance.

Honest limit: animated screenshots and labeled illustrations are a promotional
presentation, not a recording of a continuous live app session. No voice-over.

Budget: zero spawns; one visual review plus necessary corrections; one complete
media decode. Application race/build suites are not relevant to media-only output.

Status: complete. Output remains local and is not published.

## Final evidence

Delivered `.scratch/picfetch-promo/PicFetch-See-More-1080p.mp4`, approximately
24 MB, plus poster, standalone AAC music, WAV source, editable production script,
storyboards, source/rights notes and timing/verification records.

- Final ffprobe: 180.000000 seconds, 1920x1080, 30/1 fps, 5,400 H.264 frames,
  yuv420p, 48 kHz stereo AAC and 22 chapters. Audio and video durations match.
- Complete FFmpeg decode with `-xerror`: passed.
- All 22 authored preview scenes and the encoded contact sheet were inspected;
  enlarged encoded comparison/closing frames were also reviewed during production.
- Black detection found no black intervals. Sensitive freeze detection marked
  five low-motion intervals; these contain the intended reading holds/slow
  graphics, not failed decoding. Frame-hash sampling at 2 fps confirmed 360
  distinct frames across all 360 samples.
- EBU R128 integrated loudness: -15.0 LUFS; true peak: -1.2 dBFS; LRA: 3.1 LU.
- GoLand inspection of the final production script returned no findings.
- `git diff --check`: passed. No PicFetch runtime code or dependencies changed;
  application build/race suites were not run for this artifact-only task.
- Final SHA-256:
  `d4ff58f7d53fab4e3cc17c203a186b9b3a89a80163c3954fe8deba8c82bd4280`.

Actual budget: zero agent spawns, two visual correction passes, two complete
media checks because the final motion pass changed the encoded output. Tool
dependencies stayed within the isolated production directory. No commits.
