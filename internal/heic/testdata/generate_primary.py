#!/usr/bin/env python3
"""Generate an authored two-image HEIF whose second stored item is primary."""

import pathlib
import struct
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
from generate import box, fullbox


def extract(data):
    index = data.index(b"hvcC")
    size = struct.unpack_from(">I", data, index - 4)[0]
    configuration = data[index + 4:index - 4 + size]
    position = 0
    while position < len(data):
        size, kind = struct.unpack_from(">I4s", data, position)
        if kind == b"mdat":
            return configuration, data[position + 8:position + size]
        position += size
    raise ValueError("missing sample")


def generate():
    root = pathlib.Path(__file__).parent
    first_configuration, first_sample = extract((root / "probe8.heic").read_bytes())
    # First image is black/white. The designated second image is white/black.
    raw = bytes([235 if x < 32 else 16 for _ in range(64) for x in range(64)])
    raw += bytes([128]) * (64 * 64 // 2)
    with tempfile.TemporaryDirectory(prefix="picfetch-heic-primary-") as tmp:
        output = pathlib.Path(tmp) / "source.mp4"
        subprocess.run([
            "ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "rawvideo",
            "-pixel_format", "yuv420p", "-video_size", "64x64", "-i", "pipe:0",
            "-frames:v", "1", "-c:v", "libx265", "-profile:v", "main",
            "-x265-params", "qp=1:pools=none:frame-threads=1:keyint=2:log-level=error:info=0",
            "-color_range", "tv", "-color_primaries", "bt709",
            "-color_trc", "iec61966-2-1", "-colorspace", "bt709", "-tag:v", "hvc1", str(output),
        ], input=raw, check=True, timeout=30)
        second_configuration, second_sample = extract(output.read_bytes())
    ftyp = box(b"ftyp", b"heic" + bytes(4) + b"heicmif1")
    handler = fullbox(b"hdlr", bytes(4) + b"pict" + bytes(12) + b"PicFetch primary oracle\0")
    primary = fullbox(b"pitm", struct.pack(">H", 2))
    items = b"".join(fullbox(b"infe", struct.pack(">HH4s", item, 0, b"hvc1") + b"picture\0", 2)
                     for item in (1, 2))
    info = fullbox(b"iinf", struct.pack(">H", 2) + items)
    properties = (box(b"hvcC", first_configuration) + box(b"hvcC", second_configuration)
                  + fullbox(b"ispe", struct.pack(">II", 64, 64))
                  + fullbox(b"pixi", bytes([3, 8, 8, 8]))
                  + box(b"colr", b"nclx" + struct.pack(">HHHB", 1, 13, 1, 0)))
    assignments = struct.pack(">I", 2)
    for item in (1, 2):
        assignments += struct.pack(">HB", item, 4) + bytes([0x80 | item, 3, 0x84, 5])
    iprp = box(b"iprp", box(b"ipco", properties) + fullbox(b"ipma", assignments))

    def meta(offset):
        items = struct.pack(">HHHII", 1, 0, 1, offset, len(first_sample))
        items += struct.pack(">HHHII", 2, 0, 1, offset + len(first_sample), len(second_sample))
        locations = fullbox(b"iloc", bytes([0x44, 0]) + struct.pack(">H", 2) + items)
        return fullbox(b"meta", handler + primary + locations + info + iprp)

    metadata = meta(0)
    metadata = meta(len(ftyp) + len(metadata) + 8)
    (root / "nonfirst-primary.heic").write_bytes(ftyp + metadata + box(b"mdat", first_sample + second_sample))


if __name__ == "__main__":
    generate()
