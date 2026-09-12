"""Capture-independent history replay and the Docker-only Gource film pipeline."""

import argparse
import bisect
import datetime as dt
import functools
import hashlib
import json
import math
from pathlib import Path
import re
import shutil
import subprocess
import wave
from urllib.parse import quote

COLOURS = {
    "source": "63D8EA",
    "tests": "96E6A1",
    "docs": "F5C77E",
    "assets": "C79AFF",
    "build": "82A9FF",
}


def custom_field(value):
    """Escape Gource's record delimiters without conflating literal percent text."""
    return quote(value, safe=" /._-()[]@+", encoding="utf-8", errors="surrogateescape")


def category(path):
    if path.endswith("_test.go") or "/testdata/" in path:
        return "tests"
    if path.endswith((".go", ".m", ".h", ".py", ".sh", ".jq")):
        return "source"
    if path.endswith((".md", ".txt")):
        return "docs"
    if path.startswith("assets/") or path.endswith(
        (".png", ".jpg", ".gif", ".webp", ".svg", ".icns", ".ico", ".afphoto")
    ):
        return "assets"
    return "build"


def read_history(raw, merge_trees, expected):
    """Replay all ancestor commits; merge snapshots resolve divergent file sets."""
    fields = iter(raw.decode("utf-8", "surrogateescape").split("\0"))
    commits, events = [], []
    active = set()
    current = None
    clamps = reconciliations = 0

    def emit(action, path):
        if action == "D":
            if path not in active:
                return
            active.remove(path)
        else:
            action = "M" if path in active else "A"
            active.add(path)
        events.append((current["timestamp"], current["author"], action, path))

    def finish():
        nonlocal reconciliations
        if current is None:
            return
        tree = merge_trees.get(current["hash"])
        if tree is not None:
            for path in sorted(active - tree):
                emit("D", path)
                reconciliations += 1
            for path in sorted(tree - active):
                emit("A", path)
                reconciliations += 1
        current["files"] = len(active)
        commits.append(current)

    try:
        for field in fields:
            token = field.strip("\n")
            if not token:
                continue
            if token == "PICFETCH-COMMIT":
                finish()
                sha, timestamp, date, author, subject = [next(fields) for _ in range(5)]
                timestamp = int(timestamp)
                effective = max(timestamp, commits[-1]["timestamp"]) if commits else timestamp
                clamps += effective != timestamp
                current = {
                    "hash": sha,
                    "timestamp": effective,
                    "original_timestamp": timestamp,
                    "date": date,
                    "author": author,
                    "subject": subject,
                }
            elif current is not None and token in ("A", "M", "D", "T"):
                emit("M" if token == "T" else token, next(fields))
            else:
                raise ValueError(f"Unsupported Git history record: {token!r}")
        finish()
    except (StopIteration, TypeError) as error:
        raise ValueError("Incomplete Git history record") from error
    if not commits or not events:
        raise ValueError("Git history is empty or contains no file changes")
    if active != expected:
        raise ValueError(
            f"History does not match final tree: {len(expected-active)} missing, {len(active-expected)} extra"
        )
    return {
        "commits": commits,
        "events": events,
        "head": commits[-1]["hash"],
        "timestamp_clamps": clamps,
        "merge_reconciliations": reconciliations,
    }


def timing(start, end, seconds):
    seconds = float(seconds)
    if not math.isfinite(seconds) or not 30 <= seconds <= 900:
        raise ValueError("MOVIE_SECONDS must be between 30 and 900")
    return {
        "duration": seconds,
        "history_seconds": seconds - 19,
        "body_seconds": seconds - 11,
        "seconds_per_day": (seconds - 19) * 86400 / max(1, end - start),
    }


def path_set(path):
    return {
        item.decode("utf-8", "surrogateescape") for item in path.read_bytes().split(b"\0") if item
    }


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + "\n")


def prepare(root, seconds):
    merges = {path.stem: path_set(path) for path in (root / "merge-trees").glob("*.paths")}
    history = read_history(
        (root / "history.git").read_bytes(), merges, path_set(root / "expected-tree.paths")
    )
    if history["head"] != (root / "source-head.txt").read_text().strip():
        raise ValueError("Export does not end at the captured HEAD")
    commits = history["commits"]
    start, end = commits[0]["timestamp"], commits[-1]["timestamp"]
    config = timing(start, end, seconds)
    summary = {
        "head": history["head"],
        "commits": len(commits),
        "events": len(history["events"]),
        "first_files": commits[0]["files"],
        "last_files": commits[-1]["files"],
        "first_timestamp": start,
        "last_timestamp": end,
        "timestamp_clamps": history["timestamp_clamps"],
        "merge_reconciliations": history["merge_reconciliations"],
        "category_colours": COLOURS,
        **config,
    }
    write_json(root / "summary.json", summary)
    write_json(root / "timeline.json", commits)
    with (root / "history.gource").open("w") as log:
        for timestamp, author, action, path in history["events"]:
            log.write(
                f"{timestamp}|{custom_field(author)}|{action}|{custom_field(path)}|{COLOURS[category(path)]}\n"
            )
    # Prefer feature/release milestones, with room between captions to read them.
    candidates = {}
    previous_files = commits[0]["files"]
    for commit in commits[1:]:
        second = (commit["timestamp"] - start) / 86400 * config["seconds_per_day"]
        bucket = int(second // 12)
        weight = abs(commit["files"] - previous_files)
        previous_files = commit["files"]
        subject = commit["subject"].strip()
        if (
            second < 8
            or not re.match(r"(?i)^(feat(?:ure)?[(:/ ]|add\b|release\b)", subject)
        ):
            continue
        if bucket not in candidates or weight > candidates[bucket][0]:
            candidates[bucket] = (weight, second, commit)
    captions = [(start, f"FIRST COMMIT  /  {commits[0]['files']:,} files take shape")]
    previous_second = 0
    for _, second, commit in sorted(candidates.values(), key=lambda row: row[1]):
        if second - previous_second < 8:
            continue
        subject = " ".join(commit["subject"].replace("|", "/").split())
        subject = subject.encode("utf-8", "replace").decode("utf-8")
        captions.append((commit["timestamp"], subject[:77] + ("..." if len(subject) > 77 else "")))
        previous_second = second
    (root / "captions.txt").write_text(
        "".join(f"{stamp}|{caption}\n" for stamp, caption in captions)
    )
    return history, summary


def render(root, summary):
    command = [
        "xvfb-run",
        "-a",
        "-s",
        "-screen 0 1920x1080x24 -nolisten tcp",
        "gource",
        "/work/history.gource",
        "--log-format",
        "custom",
        "-1920x860",
        "--output-framerate",
        "60",
        "--output-ppm-stream",
        "-",
        "--seconds-per-day",
        str(summary["seconds_per_day"]),
        "--disable-auto-skip",
        "--no-time-travel",
        "--file-idle-time",
        "0",
        "--file-idle-time-at-end",
        "0",
        "--max-files",
        "0",
        "--max-file-lag",
        "0.15",
        "--background-colour",
        "0A111E",
        "--font-colour",
        "DFEAF6",
        "--dir-colour",
        "AAC0D8",
        "--filename-colour",
        "E8F0FA",
        "--font-file",
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "--file-font-size",
        "13",
        "--dir-font-size",
        "17",
        "--user-font-size",
        "17",
        "--dir-name-depth",
        "3",
        "--filename-time",
        "2",
        "--camera-mode",
        "overview",
        "--padding",
        "1.12",
        "--bloom-multiplier",
        "1.1",
        "--bloom-intensity",
        "0.65",
        "--caption-file",
        "/work/captions.txt",
        "--caption-size",
        "26",
        "--caption-colour",
        "E8F0FA",
        "--caption-duration",
        "5",
        "--hide",
        "date,mouse,progress",
        "--disable-input",
        "--no-vsync",
        "--dont-stop",
        "--stop-at-time",
        str(summary["body_seconds"]),
    ]
    encode = [
        "ffmpeg",
        "-hide_banner",
        "-loglevel",
        "warning",
        "-y",
        "-framerate",
        "60",
        "-f",
        "image2pipe",
        "-vcodec",
        "ppm",
        "-i",
        "pipe:0",
        "-an",
        "-c:v",
        "libx264",
        "-preset",
        "veryfast",
        "-crf",
        "16",
        "-threads",
        "4",
        "-pix_fmt",
        "yuv420p",
        "-progress",
        "/work/render-progress.txt",
        "/work/history-body.mp4",
    ]
    write_json(root / "render-commands.json", [command, encode])
    with (root / "gource.log").open("w") as g_log, (root / "render.log").open("w") as f_log:
        producer = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=g_log)
        consumer = None
        try:
            consumer = subprocess.Popen(encode, stdin=producer.stdout, stderr=f_log)
            producer.stdout.close()
            while consumer.poll() is None:
                try:
                    consumer.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    print("Rendering animation...", flush=True)
            if consumer.returncode or producer.wait():
                raise RuntimeError("Gource/FFmpeg rendering failed; see gource.log and render.log")
        finally:
            for child in (consumer, producer):
                if child is not None and child.poll() is None:
                    child.terminate()
                    try:
                        child.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        child.kill()
                        child.wait()


def artwork(root, history, summary):
    import numpy as np
    from PIL import Image, ImageDraw, ImageFont

    ROOT = root
    S = summary
    TIMELINE = history["commits"]
    START, END = S["first_timestamp"], S["last_timestamp"]
    BODY_SECONDS = S["body_seconds"]
    INTRO, OUTRO = 5, 6
    DURATION = INTRO + BODY_SECONDS + OUTRO
    display_zone = dt.datetime.fromisoformat(TIMELINE[-1]["date"]).tzinfo
    first_date = dt.datetime.fromtimestamp(START, display_zone)
    last_date = dt.datetime.fromtimestamp(END, display_zone)
    days_label = f"{(END-START)/86400:.0f} days" if END - START >= 86400 else "Less than a day"
    date_range = f"{first_date:%d %B %Y} — {last_date:%d %B %Y}".upper()
    peak_files = max(1, max(c["files"] for c in TIMELINE) * 1.09)
    W, H = 1920, 1080
    BG = "#0A111E"
    FG = "#EDF4FC"
    MUTED = "#91A6BF"
    CYAN = "#63D8EA"
    FONT_DIR = Path("/usr/share/fonts/truetype/dejavu")

    @functools.lru_cache(maxsize=32)
    def font(size, bold=False):
        return ImageFont.truetype(
            str(FONT_DIR / ("DejaVuSans-Bold.ttf" if bold else "DejaVuSans.ttf")), size
        )

    def text(draw, pos, value, size=20, colour=FG, bold=False, anchor=None):
        draw.text(pos, value, fill=colour, font=font(size, bold), anchor=anchor)

    def graph(draw, box, timestamp, glow=False):
        x, y, width, height = box
        points = [
            (
                x + (p["timestamp"] - START) / max(1, END - START) * width,
                y + height - p["files"] / peak_files * height,
            )
            for p in TIMELINE
        ]
        if len(points) > 1:
            draw.line(points, fill="#263850", width=3)
        done = [p for p, c in zip(points, TIMELINE) if c["timestamp"] <= timestamp]
        if len(done) > 1:
            draw.line(done, fill=CYAN, width=4 if glow else 3)
        if done:
            px, py = done[-1]
            draw.ellipse((px - 5, py - 5, px + 5, py + 5), fill=FG)

    def backdrop():
        yy, xx = np.mgrid[0:H, 0:W]
        radial = np.exp(-(((xx - W * 0.67) / 800) ** 2 + ((yy - H * 0.42) / 550) ** 2))
        base = np.empty((H, W, 3), dtype=np.uint8)
        for ch, (low, extra) in enumerate([(10, 7), (17, 16), (30, 23)]):
            base[:, :, ch] = low + radial * extra
        im = Image.fromarray(base)
        d = ImageDraw.Draw(im)
        for x in range(48, W, 64):
            for y in range(50, H, 64):
                d.ellipse((x, y, x + 1, y + 1), fill="#24374A")
        return im

    def cards():
        im = backdrop()
        d = ImageDraw.Draw(im)
        d.rounded_rectangle((96, 112, 286, 156), radius=22, fill="#18303F")
        text(d, (191, 134), f"GIT / {first_date:%Y}", 18, CYAN, True, "mm")
        text(d, (96, 205), "PICFETCH", 96, FG, True)
        text(d, (100, 331), "A codebase comes alive.", 46)
        text(d, (100, 419), f"{days_label} of building, editing, and growing.", 26, MUTED)
        graph(d, (102, 520, 1710, 225), END, True)
        text(d, (100, 792), f"{S['first_files']:,}", 56, FG, True)
        text(d, (100, 863), "FILES AT THE FIRST COMMIT", 16, MUTED)
        text(d, (760, 792), f"{S['commits']:,}", 56, FG, True)
        text(d, (760, 863), "COMMITS IN THE HISTORY", 16, MUTED)
        text(d, (1490, 792), f"{S['last_files']:,}", 56, CYAN, True)
        text(d, (1490, 863), "FILES AT THE LATEST COMMIT", 16, MUTED)
        d.line((100, 956, 1820, 956), fill="#314258", width=1)
        text(d, (100, 982), date_range, 17, MUTED)
        text(d, (1820, 982), "A PICFETCH HISTORY FILM", 17, MUTED, anchor="ra")
        im.save(ROOT / "intro.png")
        im = backdrop()
        d = ImageDraw.Draw(im)
        text(d, (960, 174), "STILL GROWING.", 76, FG, True, "mm")
        text(d, (960, 258), f"PicFetch · {last_date:%d %B %Y}", 28, MUTED, anchor="mm")
        graph(d, (240, 355, 1440, 235), END, True)
        for x, number, label in [
            (365, f"{S['commits']:,}", "COMMITS"),
            (960, f"{S['events']:,}", "FILE CHANGE EVENTS"),
            (1555, f"{S['last_files']:,}", "TRACKED FILES"),
        ]:
            text(d, (x, 704), number, 64, CYAN, True, "mm")
            text(d, (x, 765), label, 18, MUTED, anchor="mm")
        text(d, (960, 865), "Rendered with Gource + FFmpeg", 22, FG, anchor="mm")
        text(
            d,
            (960, 909),
            "Film design & original procedural soundtrack: Pico",
            18,
            MUTED,
            anchor="mm",
        )
        text(
            d,
            (960, 968),
            f"Committed history through {S['head'][:7]} · Uncommitted edits are outside Git history",
            16,
            MUTED,
            anchor="mm",
        )
        im.save(ROOT / "outro.png")

    def hud():
        directory = ROOT / "hud"
        directory.mkdir(exist_ok=True)
        times = [c["timestamp"] for c in TIMELINE]
        for index in range(math.ceil(BODY_SECONDS * 6) + 2):
            seconds = index / 6
            timestamp = min(END, START + seconds / S["seconds_per_day"] * 86400)
            count = bisect.bisect_right(times, timestamp)
            current = TIMELINE[max(0, count - 1)]
            im = Image.new("RGBA", (W, H), (0, 0, 0, 0))
            d = ImageDraw.Draw(im)
            d.rectangle((0, 0, W, 111), fill=BG)
            d.rectangle((0, 972, W, H), fill=BG)
            d.line((48, 111, 1872, 111), fill="#2B3B51", width=1)
            d.line((48, 973, 1872, 973), fill="#2B3B51", width=1)
            text(d, (48, 17), "PICFETCH", 35, FG, True)
            text(d, (50, 66), "A CODEBASE COMES ALIVE", 14, MUTED)
            text(
                d,
                (610, 28),
                f"{S['commits']:,} commits  /  {days_label.lower()}  /  one growing codebase",
                20,
                MUTED,
            )
            text(
                d,
                (610, 64),
                "Files branch out. Contributors create, edit, and remove them.",
                16,
                MUTED,
            )
            date = dt.datetime.fromtimestamp(timestamp, display_zone)
            text(d, (1872, 19), date.strftime("%d %b %Y").upper(), 28, FG, True, "ra")
            text(
                d,
                (1872, 63),
                "HISTORY COMPLETE" if timestamp >= END else "COMMITTED HISTORY",
                14,
                CYAN,
                anchor="ra",
            )
            text(d, (48, 985), f"{current['files']:,}", 31, FG, True)
            text(d, (50, 1031), "TRACKED FILES", 13, MUTED)
            text(d, (265, 985), f"{count:,} / {S['commits']:,}", 31, FG, True)
            text(d, (267, 1031), "COMMITS", 13, MUTED)
            labels = [
                ("source", "Code"),
                ("tests", "Tests"),
                ("docs", "Docs"),
                ("assets", "Assets"),
                ("build", "Config"),
            ]
            for j, (key, label) in enumerate(labels):
                x = 550 + j * 128
                d.ellipse((x, 999, x + 10, 1009), fill="#" + S["category_colours"][key])
                text(d, (x + 19, 991), label, 16, MUTED)
            text(
                d,
                (550, 1031),
                f"{S['first_files']:,} files at the start · every committed path included",
                14,
                MUTED,
            )
            graph(d, (1268, 995, 603, 61), timestamp)
            im.save(directory / f"{index:05d}.png", compress_level=3)
        print("Title cards and timeline HUD created.", flush=True)

    def soundtrack():
        rate = 48000
        length = math.ceil(DURATION * rate)
        score = np.zeros((length, 2), np.float32)
        beat = 60 / 90

        def put(start, signal, amplitude, pan=0):
            offset = round(start * rate)
            if offset >= length:
                return
            signal = signal[: length - offset] * amplitude
            score[offset : offset + len(signal), 0] += signal * math.sqrt((1 - pan) / 2)
            score[offset : offset + len(signal), 1] += signal * math.sqrt((1 + pan) / 2)

        def frequency(midi):
            return 440 * 2 ** ((midi - 69) / 12)

        chords = [[45, 52, 60, 64], [41, 48, 57, 60], [48, 55, 59, 64], [43, 50, 57, 62]]
        bar = beat * 8
        for bar_index, start in enumerate(np.arange(0, DURATION, bar)):
            chord = chords[bar_index % len(chords)]
            t = np.arange(round((bar + 1.5) * rate), dtype=np.float32) / rate
            env = np.minimum(t / 1.5, 1) * np.minimum((t[-1] - t) / 2.2, 1)
            for j, note in enumerate(chord):
                hz = frequency(note)
                signal = (
                    np.sin(2 * np.pi * hz * t + 0.012 * np.sin(t * 1.2))
                    + 0.18 * np.sin(2 * np.pi * hz * 2.002 * t)
                ) * env
                put(start, signal, 0.045, (j - 1.5) * 0.35)
            for step in range(16):
                t = np.arange(round(1.7 * rate), dtype=np.float32) / rate
                hz = frequency(chord[[0, 2, 1, 3, 2, 1, 3, 2][step % 8]] + 24)
                env = (1 - np.exp(-t * 150)) * np.exp(-t * 3.7)
                signal = (np.sin(2 * np.pi * hz * t) + 0.15 * np.sin(2 * np.pi * hz * 2 * t)) * env
                at = start + step * beat / 2
                pan = math.sin(step * 0.8) * 0.55
                put(at, signal, 0.039, pan)
                put(at + beat * 0.75, signal, 0.012, -pan)
            for step in range(0, 8, 2):
                t = np.arange(round(0.36 * rate), dtype=np.float32) / rate
                signal = np.sin(
                    2 * np.pi * (46 * t + 38 * 0.022 * (1 - np.exp(-t / 0.022)))
                ) * np.exp(-t * 15)
                put(start + step * beat, signal, 0.075)
        fades = np.minimum(np.arange(length) / (rate * 4), 1) * np.minimum(
            (length - 1 - np.arange(length)) / (rate * 5), 1
        )
        score *= fades[:, None]
        score = np.tanh(score * 1.7) * 0.65
        with wave.open(str(ROOT / "original-score.wav"), "wb") as wav:
            wav.setnchannels(2)
            wav.setsampwidth(2)
            wav.setframerate(rate)
            wav.writeframes((score * 32767).astype("<i2").tobytes())
        print("Original ambient score created.", flush=True)

    cards()
    hud()
    soundtrack()


def metadata_field(value):
    return "".join("\\" + char if char in "\\;#=" else char for char in value)


def assemble(root, summary):
    probe = json.loads(
        subprocess.check_output(
            [
                "ffprobe",
                "-v",
                "error",
                "-show_format",
                "-show_streams",
                "-of",
                "json",
                str(root / "history-body.mp4"),
            ]
        )
    )
    recorded_body = float(probe["format"]["duration"])
    body = min(recorded_body, summary["body_seconds"])
    total = 5 + body + 6
    chapters = [(0, "A codebase comes alive"), (5, "The first files")]
    for line in (root / "captions.txt").read_text().splitlines()[1:]:
        timestamp, title = line.split("|", 1)
        offset = (
            5 + (int(timestamp) - summary["first_timestamp"]) / 86400 * summary["seconds_per_day"]
        )
        chapters.append((offset, title.split("  /  ")[0].title()))
    chapters.append((5 + body, "Still growing"))
    metadata = [
        ";FFMETADATA1",
        "title=PicFetch — A codebase comes alive",
        "artist=Pico",
        f"comment=All {summary['commits']} commits reachable from {summary['head']}; generated locally with Gource and FFmpeg.",
    ]
    for index, (start, title) in enumerate(chapters):
        end = chapters[index + 1][0] if index + 1 < len(chapters) else total
        metadata += [
            "[CHAPTER]",
            "TIMEBASE=1/1000",
            f"START={round(start*1000)}",
            f"END={round(end*1000)}",
            f"title={metadata_field(title)}",
        ]
    (root / "chapters.ffmeta").write_text("\n".join(metadata) + "\n")
    filters = (
        f"[0:v]setpts=PTS-STARTPTS,pad=1920:1080:0:112:color=0x0A111E[base];"
        f"[base][1:v]overlay=shortest=1:format=auto,format=yuv420p,setsar=1,"
        f"fade=t=in:d=0.6,fade=t=out:st={body-0.6}:d=0.6[b];"
        "[2:v]trim=duration=5,setpts=PTS-STARTPTS,format=yuv420p,setsar=1,fade=t=in:d=0.8,fade=t=out:st=4.4:d=0.6[i];"
        "[3:v]trim=duration=6,setpts=PTS-STARTPTS,format=yuv420p,setsar=1,fade=t=in:d=0.6,fade=t=out:st=5:d=1[o];"
        "[i][b][o]concat=n=3:v=1:a=0[v]"
    )
    command = [
        "ffmpeg",
        "-hide_banner",
        "-loglevel",
        "warning",
        "-y",
        "-t",
        str(body),
        "-i",
        "/work/history-body.mp4",
        "-framerate",
        "6",
        "-i",
        "/work/hud/%05d.png",
        "-loop",
        "1",
        "-framerate",
        "60",
        "-t",
        "5",
        "-i",
        "/work/intro.png",
        "-loop",
        "1",
        "-framerate",
        "60",
        "-t",
        "6",
        "-i",
        "/work/outro.png",
        "-i",
        "/work/original-score.wav",
        "-i",
        "/work/chapters.ffmeta",
        "-filter_complex_threads",
        "2",
        "-filter_complex",
        filters,
        "-map",
        "[v]",
        "-map",
        "4:a",
        "-map_metadata",
        "5",
        "-map_chapters",
        "5",
        "-c:v",
        "libx264",
        "-preset",
        "medium",
        "-crf",
        "17",
        "-threads",
        "4",
        "-r",
        "60",
        "-pix_fmt",
        "yuv420p",
        "-color_primaries",
        "bt709",
        "-color_trc",
        "bt709",
        "-colorspace",
        "bt709",
        "-c:a",
        "aac",
        "-b:a",
        "192k",
        "-af",
        "loudnorm=I=-23:TP=-2:LRA=7",
        "-ar",
        "48000",
        "-t",
        str(total),
        "-movflags",
        "+faststart",
        "-progress",
        "/work/final-progress.txt",
        "/work/PicFetch-history.mp4",
    ]
    (root / "assembly-command.json").write_text(json.dumps(command, indent=2))
    with (root / "assembly.log").open("w") as log:
        subprocess.run(command, stderr=log, check=True)
    print(f"Final movie created: {total:.2f} seconds.", flush=True)


def verify(root, summary):
    movie = root / "PicFetch-history.mp4"
    probe = json.loads(
        subprocess.check_output(
            [
                "ffprobe",
                "-v",
                "error",
                "-show_format",
                "-show_streams",
                "-show_chapters",
                "-of",
                "json",
                str(movie),
            ]
        )
    )
    video = next(stream for stream in probe["streams"] if stream["codec_type"] == "video")
    audio = next(stream for stream in probe["streams"] if stream["codec_type"] == "audio")
    if (video["width"], video["height"], video["r_frame_rate"], video["pix_fmt"]) != (
        1920,
        1080,
        "60/1",
        "yuv420p",
    ):
        raise ValueError("Movie does not have the expected 1080p60 format")
    if video["codec_name"] != "h264" or audio["codec_name"] != "aac" or audio["channels"] != 2:
        raise ValueError("Movie does not have H.264 video and AAC stereo audio")
    duration = float(probe["format"]["duration"])
    if (
        abs(duration - summary["duration"]) > 0.1
        or abs(float(video["duration"]) - float(audio["duration"])) > 0.1
    ):
        raise ValueError(
            "Movie duration or audio synchronization differs from the requested duration"
        )
    with (root / "decode-check.log").open("w") as log:
        subprocess.run(
            [
                "ffmpeg",
                "-hide_banner",
                "-v",
                "error",
                "-xerror",
                "-i",
                str(movie),
                "-f",
                "null",
                "-",
            ],
            stderr=log,
            check=True,
        )
    subprocess.run(
        [
            "ffmpeg",
            "-hide_banner",
            "-loglevel",
            "error",
            "-y",
            "-ss",
            str(5 + summary["history_seconds"] + 3),
            "-i",
            str(movie),
            "-frames:v",
            "1",
            str(root / "poster.png"),
        ],
        check=True,
    )
    with movie.open("rb") as file:
        digest = hashlib.file_digest(file, "sha256").hexdigest()
    write_json(
        root / "verification.json",
        {
            "file": movie.name,
            "duration_seconds": duration,
            "bytes": movie.stat().st_size,
            "sha256": digest,
            "video": video,
            "audio": audio,
            "chapter_count": len(probe["chapters"]),
            "full_decode_passed": True,
            "final_tree_matches_captured_head": True,
            "head": summary["head"],
            "commits": summary["commits"],
            "final_files": summary["last_files"],
        },
    )


def record_tools(root):
    with (root / "installed-packages.txt").open("w") as manifest:
        subprocess.run(
            ["dpkg-query", "-W", "-f", "${Package}\t${Version}\n"], stdout=manifest, check=True
        )
    notices = root / "notices"
    notices.mkdir()
    for source in Path("/usr/share/doc").glob("*/copyright"):
        shutil.copyfile(source, notices / (source.parent.name + ".txt"))
    recipe = root / "recipe"
    recipe.mkdir()
    for source in Path("/scripts").iterdir():
        if source.is_file():
            shutil.copyfile(source, recipe / source.name)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--seconds", default="180")
    args = parser.parse_args()
    root = Path("/work")
    history, summary = prepare(root, args.seconds)
    record_tools(root)
    print(
        f"History verified: {summary['commits']} commits, {summary['first_files']} to {summary['last_files']} files.",
        flush=True,
    )
    print("Rendering Gource at 1080p60...", flush=True)
    render(root, summary)
    print("Drawing titles, dates and growth chart; synthesizing the soundtrack...", flush=True)
    artwork(root, history, summary)
    print("Assembling the final movie...", flush=True)
    assemble(root, summary)
    print("Checking the complete video and audio streams...", flush=True)
    verify(root, summary)
    print("Movie verification passed.", flush=True)


if __name__ == "__main__":
    main()
