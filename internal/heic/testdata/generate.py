#!/usr/bin/env python3
"""Reproduce the project-owned HEVC probe fixtures with installed FFmpeg/x265.

The source is a 64x64, limited-range, neutral-chroma grayscale image: left
half black, right half white. Its independent RGBA oracle is (0,0,0,255)
and (255,255,255,255), allowing two levels for integer color conversion.
No downloaded photographs, decoder, or encoder binaries are distributed.
"""

import pathlib
import struct
import subprocess
import tempfile


def box(kind, payload):
    return struct.pack(">I4s", len(payload) + 8, kind) + payload


def fullbox(kind, payload, version=0):
    return box(kind, bytes([version, 0, 0, 0]) + payload)


def generate(bits, rotation=None, exif_orientation=None, name=None):
    width = height = 64
    scale = 1 << (bits - 8)
    samples = [scale * (16 if x < 32 else 235)
               for _ in range(height) for x in range(width)]
    samples += [128 * scale] * (width * height // 2)
    raw = bytes(samples) if bits == 8 else struct.pack("<" + "H" * len(samples), *samples)
    with tempfile.TemporaryDirectory(prefix="picfetch-heic-fixture-") as tmp:
        output = pathlib.Path(tmp) / "source.mp4"
        subprocess.run([
            "ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "rawvideo",
            "-pixel_format", "yuv420p" if bits == 8 else "yuv420p10le",
            "-video_size", "64x64", "-i", "pipe:0", "-frames:v", "1",
            "-c:v", "libx265", "-profile:v", "main" if bits == 8 else "main10", "-x265-params",
            "qp=1:pools=none:frame-threads=1:keyint=2:log-level=error:info=0",
            "-color_range", "tv", "-color_primaries", "bt709",
            "-color_trc", "iec61966-2-1", "-colorspace", "bt709",
            "-tag:v", "hvc1", str(output),
        ], input=raw, check=True, timeout=30)
        encoded = output.read_bytes()
    # The generator controls this single-sample MP4. Extract its configuration
    # and length-prefixed HEVC access unit without adopting movie decoding.
    index = encoded.index(b"hvcC")
    size = struct.unpack_from(">I", encoded, index - 4)[0]
    configuration = encoded[index + 4:index - 4 + size]
    position = 0
    sample = None
    while position < len(encoded):
        size, kind = struct.unpack_from(">I4s", encoded, position)
        if kind == b"mdat":
            sample = encoded[position + 8:position + size]
        position += size
    assert sample and configuration
    brand = b"heic" if bits == 8 else b"heix"
    ftyp = box(b"ftyp", brand + bytes(4) + brand + b"mif1")
    handler = fullbox(b"hdlr", bytes(4) + b"pict" + bytes(12) + b"PicFetch fixture\0")
    primary = fullbox(b"pitm", struct.pack(">H", 1))
    item = fullbox(b"infe", struct.pack(">HH4s", 1, 0, b"hvc1") + b"probe\0", 2)
    exif = b""
    references = b""
    if exif_orientation:
        exif = bytes(4) + b"II" + struct.pack("<HIH", 42, 8, 1)
        exif += struct.pack("<HHIHHI", 0x112, 3, 1, exif_orientation, 0, 0)
        item += fullbox(b"infe", struct.pack(">HH4s", 2, 0, b"Exif") + b"orientation\0", 2)
        references = fullbox(b"iref", box(b"cdsc", struct.pack(">HHH", 2, 1, 1)))
    info = fullbox(b"iinf", struct.pack(">H", 2 if exif else 1) + item)
    properties = (box(b"hvcC", configuration)
                  + fullbox(b"ispe", struct.pack(">II", width, height))
                  + fullbox(b"pixi", bytes([3, bits, bits, bits]))
                  + box(b"colr", b"nclx" + struct.pack(">HHHB", 1, 13, 1, 0)))
    associations = bytes([0x81, 2, 0x83, 4])
    if rotation is not None:
        properties += box(b"irot", bytes([rotation]))
        associations += bytes([0x85])
    associations = fullbox(b"ipma", struct.pack(">IHB", 1, 1, len(associations)) + associations)
    iprp = box(b"iprp", box(b"ipco", properties) + associations)

    def meta(offset):
        items = struct.pack(">HHHII", 1, 0, 1, offset, len(sample))
        if exif:
            items += struct.pack(">HHHII", 2, 0, 1, offset + len(sample), len(exif))
        locations = fullbox(b"iloc", bytes([0x44, 0]) + struct.pack(">H", 2 if exif else 1) + items)
        return fullbox(b"meta", handler + primary + locations + info + references + iprp)

    metadata = meta(0)
    metadata = meta(len(ftyp) + len(metadata) + 8)
    result = ftyp + metadata + box(b"mdat", sample + exif)
    (pathlib.Path(__file__).parent / (name or f"probe{bits}.heic")).write_bytes(result)


if __name__ == "__main__":
    generate(8)
    generate(10)
    generate(8, rotation=1, name="container-rotate.heic")
    generate(8, exif_orientation=6, name="exif-rotate.heic")
    generate(8, rotation=1, exif_orientation=6, name="container-and-exif.heic")
