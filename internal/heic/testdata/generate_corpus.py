#!/usr/bin/env python3
"""Generate project-owned native HEIC qualification fixtures; no codec installs."""

import ctypes
import pathlib
import struct
import subprocess
import sys
import tempfile

sys.dont_write_bytecode = True
from generate import box, fullbox
from generate_primary import extract

ROOT = pathlib.Path(__file__).parent


COLORS = ((192, 32, 64), (32, 160, 64), (32, 64, 192), (160, 128, 32))


def icc_profile(wide=False):
    library = ctypes.CDLL("/lib/x86_64-linux-gnu/liblcms2.so.2")
    library.cmsCreate_sRGBProfile.restype = ctypes.c_void_p
    library.cmsSaveProfileToMem.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.POINTER(ctypes.c_uint32)]
    library.cmsSaveProfileToMem.restype = ctypes.c_int
    library.cmsCloseProfile.argtypes = [ctypes.c_void_p]
    curves = []
    if wide:
        class Chromaticity(ctypes.Structure):
            _fields_ = [(name, ctypes.c_double) for name in ("x", "y", "Y")]

        class Primaries(ctypes.Structure):
            _fields_ = [(name, Chromaticity) for name in ("red", "green", "blue")]

        library.cmsBuildGamma.argtypes = [ctypes.c_void_p, ctypes.c_double]
        library.cmsBuildGamma.restype = ctypes.c_void_p
        library.cmsFreeToneCurve.argtypes = [ctypes.c_void_p]
        library.cmsCreateRGBProfile.argtypes = [ctypes.POINTER(Chromaticity), ctypes.POINTER(Primaries), ctypes.POINTER(ctypes.c_void_p)]
        library.cmsCreateRGBProfile.restype = ctypes.c_void_p
        white = Chromaticity(0.3127, 0.3290, 1.0)
        primaries = Primaries(Chromaticity(0.68, 0.32, 1.0), Chromaticity(0.265, 0.690, 1.0), Chromaticity(0.15, 0.06, 1.0))
        curves = [library.cmsBuildGamma(None, 1.0) for _ in range(3)]
        if not all(curves):
            raise RuntimeError("cannot create linear transfer curves")
        profile = library.cmsCreateRGBProfile(ctypes.byref(white), ctypes.byref(primaries), (ctypes.c_void_p * 3)(*curves))
    else:
        profile = library.cmsCreate_sRGBProfile()
    if not profile:
        raise RuntimeError("cannot create owned sRGB ICC profile")
    try:
        size = ctypes.c_uint32()
        if not library.cmsSaveProfileToMem(profile, None, ctypes.byref(size)):
            raise RuntimeError("cannot measure ICC profile")
        output = ctypes.create_string_buffer(size.value)
        if not library.cmsSaveProfileToMem(profile, output, ctypes.byref(size)):
            raise RuntimeError("cannot serialize ICC profile")
        data = bytearray(output.raw)
        # Pin the ICC creation timestamp, leaving profile tags unchanged.
        data[24:36] = struct.pack(">6H", 2026, 9, 22, 0, 0, 0)
        return bytes(data)
    finally:
        library.cmsCloseProfile(profile)
        for curve in curves:
            library.cmsFreeToneCurve(curve)


def color_property(profile=None, full_range=False):
    if profile is not None:
        return box(b"colr", b"prof" + profile)
    return box(b"colr", b"nclx" + struct.pack(">HHHB", 1, 13, 1, 0x80 if full_range else 0))


def coded_item(configuration, sample, bits, profile=None, extra=(), full_range=False):
    return {"kind": b"hvc1", "sample": sample, "properties": [
        box(b"hvcC", configuration), fullbox(b"ispe", struct.pack(">II", 64, 64)),
        fullbox(b"pixi", bytes([3, bits, bits, bits])), color_property(profile, full_range), *extra,
    ]}


def write_container(name, items, bits=8, primary_id=1, references=()):
    brand = b"heic" if bits == 8 else b"heix"
    ftyp = box(b"ftyp", brand + bytes(4) + brand + b"mif1")
    handler = fullbox(b"hdlr", bytes(4) + b"pict" + bytes(12) + b"PicFetch authored corpus\0")
    primary = fullbox(b"pitm", struct.pack(">H", primary_id))
    descriptions, properties = b"", []
    assignments = struct.pack(">I", len(items))
    for item_id, item in enumerate(items, 1):
        header = bytes([2, 0, 0, int(item.get("hidden", False))])
        descriptions += box(b"infe", header + struct.pack(">HH4s", item_id, 0, item["kind"]) + b"image\0")
        assigned = []
        for prop in item["properties"]:
            properties.append(prop)
            essential = prop[4:8] in (b"hvcC", b"pixi", b"irot", b"imir")
            assigned.append(len(properties) | (0x80 if essential else 0))
        assignments += struct.pack(">HB", item_id, len(assigned)) + bytes(assigned)
    info = fullbox(b"iinf", struct.pack(">H", len(items)) + descriptions)
    iprp = box(b"iprp", box(b"ipco", b"".join(properties)) + fullbox(b"ipma", assignments))
    reference_boxes = b"".join(box(kind, struct.pack(">HH", source, len(targets))
                                   + b"".join(struct.pack(">H", target) for target in targets))
                               for kind, source, targets in references)
    refs = fullbox(b"iref", reference_boxes) if references else b""

    def meta(offset):
        locations = bytes([0x44, 0]) + struct.pack(">H", len(items))
        for item_id, item in enumerate(items, 1):
            locations += struct.pack(">HHHII", item_id, 0, 1, offset, len(item["sample"]))
            offset += len(item["sample"])
        return fullbox(b"meta", handler + primary + fullbox(b"iloc", locations) + info + refs + iprp)

    metadata = meta(0)
    metadata = meta(len(ftyp) + len(metadata) + 8)
    (ROOT / name).write_bytes(ftyp + metadata + box(b"mdat", b"".join(item["sample"] for item in items)))


def encode(bits, colors=COLORS, alpha=None):
    scale = 1 << (bits - 8)
    planes = [[], [], []]
    for plane in range(3):
        size = 64 if plane == 0 else 32
        for y in range(size):
            for x in range(size):
                index = (y // (size // 2)) * 2 + x // (size // 2)
                if alpha is not None:
                    value = alpha[index] * ((1 << bits) - 1) / 255 if plane == 0 else 128 * scale
                else:
                    red, green, blue = colors[index]
                    luma = 0.2126 * red + 0.7152 * green + 0.0722 * blue
                    values = (16 + 219 * luma / 255,
                              128 + 224 * (blue - luma) / (2 * (1 - 0.0722) * 255),
                              128 + 224 * (red - luma) / (2 * (1 - 0.2126) * 255))
                    value = values[plane] * scale
                planes[plane].append(round(value))
    samples = sum(planes, [])
    raw = bytes(samples) if bits == 8 else struct.pack("<" + "H" * len(samples), *samples)
    with tempfile.TemporaryDirectory(prefix="picfetch-heic-corpus-") as tmp:
        output = pathlib.Path(tmp) / "source.mp4"
        subprocess.run([
            "ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "rawvideo",
            "-pixel_format", "yuv420p" if bits == 8 else "yuv420p10le", "-video_size", "64x64",
            "-i", "pipe:0", "-frames:v", "1", "-c:v", "libx265", "-profile:v", "main" if bits == 8 else "main10",
            "-x265-params", "qp=1:pools=none:frame-threads=1:keyint=2:log-level=error:info=0",
            "-color_range", "pc" if alpha is not None else "tv", "-color_primaries", "bt709",
            "-color_trc", "iec61966-2-1", "-colorspace", "bt709", "-tag:v", "hvc1", str(output),
        ], input=raw, check=True, timeout=30)
        return extract(output.read_bytes())


def generate():
    profile, wide = icc_profile(), icc_profile(wide=True)
    for bits in (8, 10):
        configuration, sample = extract((ROOT / f"probe{bits}.heic").read_bytes())
        write_container(f"icc-srgb{bits}.heic", [coded_item(configuration, sample, bits, profile)], bits)
        configuration, sample = encode(bits)
        write_container(f"color{bits}.heic", [coded_item(configuration, sample, bits)], bits)
        write_container(f"icc-p3-linear{bits}.heic", [coded_item(configuration, sample, bits, wide)], bits)
        write_container(f"mirror-horizontal{bits}.heic", [coded_item(configuration, sample, bits, extra=[box(b"imir", bytes([1]))])], bits)
        if bits == 8:
            write_container("mirror-rotate8.heic", [coded_item(configuration, sample, bits,
                            extra=[box(b"irot", bytes([1])), box(b"imir", bytes([1]))])])

        tiles = []
        for color in COLORS:
            configuration, sample = encode(bits, colors=[color] * 4)
            item = coded_item(configuration, sample, bits)
            item["hidden"] = True
            tiles.append(item)
        grid = {"kind": b"grid", "sample": bytes([0, 0, 1, 1]) + struct.pack(">HH", 128, 128),
                "properties": [fullbox(b"ispe", struct.pack(">II", 128, 128)),
                               fullbox(b"pixi", bytes([3, bits, bits, bits])), color_property()]}
        write_container(f"grid{bits}.heic", tiles + [grid], bits, 5, [(b"dimg", 5, [1, 2, 3, 4])])

        alpha_values = (0, 128, 192, 255)
        alpha_configuration, alpha_sample = encode(bits, alpha=alpha_values)
        alpha_item = coded_item(alpha_configuration, alpha_sample, bits, full_range=True,
                                extra=[fullbox(b"auxC", b"urn:mpeg:hevc:2015:auxid:1\0")])
        for premultiplied in (False, True):
            color = (160, 80, 40)
            source_colors = [tuple(round(channel * alpha / 255) for channel in color)
                             if premultiplied else color for alpha in alpha_values]
            configuration, sample = encode(bits, colors=source_colors)
            references = [(b"auxl", 2, [1])]
            if premultiplied:
                references.append((b"prem", 1, [2]))
            prefix = "alpha-premultiplied" if premultiplied else "alpha-straight"
            write_container(f"{prefix}{bits}.heic", [coded_item(configuration, sample, bits), alpha_item], bits, references=references)


if __name__ == "__main__":
    generate()
