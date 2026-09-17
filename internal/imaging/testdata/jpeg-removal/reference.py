#!/usr/bin/env python3
"""Generate synthetic fixtures and independently compare ICC color transforms."""

import argparse
import ctypes
import itertools
import math
import os
from pathlib import Path
import struct
import subprocess
import sys


CJPEG = Path("/home/linuxbrew/.linuxbrew/Cellar/jpeg-turbo/3.2.0/bin/cjpeg")
LCMS = Path("/home/linuxbrew/.linuxbrew/Cellar/little-cms2/2.19.1/lib/liblcms2.so")
RGB_SIGNATURE = int.from_bytes(b"RGB ", "big")
GRAY_SIGNATURE = int.from_bytes(b"GRAY", "big")
RGB_DOUBLE = (1 << 22) | (4 << 16) | (3 << 3)
GRAY_DOUBLE = (1 << 22) | (3 << 16) | (1 << 3)
XYZ_DOUBLE = (1 << 22) | (9 << 16) | (3 << 3)
NO_OPTIMIZE = 0x0100
TOLERANCE = 1e-7
INTENTS = ("perceptual", "relative", "saturation", "absolute")
PROFILES = ("rgb-v2.icc", "rgb-v4.icc", "gray-v2.icc", "gray-v4.icc")


class LittleCMS:
    def __init__(self):
        self.lib = ctypes.CDLL(str(LCMS))
        pointer = ctypes.c_void_p
        uint = ctypes.c_uint32
        double = ctypes.c_double
        string = ctypes.c_char_p
        bindings = (
            ("cmsGetEncodedCMMversion", ctypes.c_int, ()),
            ("cmsCreate_sRGBProfile", pointer, ()),
            ("cmsCreateGrayProfile", pointer, (pointer, pointer)),
            ("cmsCreateXYZProfile", pointer, ()),
            ("cmsBuildGamma", pointer, (pointer, double)),
            ("cmsD50_xyY", pointer, ()),
            ("cmsFreeToneCurve", None, (pointer,)),
            ("cmsSetProfileVersion", None, (pointer, double)),
            ("cmsSaveProfileToFile", ctypes.c_int, (pointer, string)),
            ("cmsOpenProfileFromFile", pointer, (string, string)),
            ("cmsCloseProfile", ctypes.c_int, (pointer,)),
            ("cmsGetColorSpace", uint, (pointer,)),
            ("cmsMLUalloc", pointer, (pointer, uint)),
            ("cmsMLUsetASCII", ctypes.c_int, (pointer, string, string, string)),
            ("cmsMLUfree", None, (pointer,)),
            ("cmsWriteTag", ctypes.c_int, (pointer, uint, pointer)),
            ("cmsCreateTransform", pointer, (pointer, uint, pointer, uint, uint, uint)),
            ("cmsDoTransform", None, (pointer, pointer, pointer, uint)),
            ("cmsDeleteTransform", None, (pointer,)),
        )
        for name, result, arguments in bindings:
            function = getattr(self.lib, name)
            function.restype = result
            function.argtypes = arguments
        version = self.lib.cmsGetEncodedCMMversion()
        if version != 2190:
            raise RuntimeError(f"expected LittleCMS API version 2190, got {version}")

    def write_text_tag(self, profile, signature, text):
        value = self.lib.cmsMLUalloc(None, 1)
        if not value:
            raise RuntimeError("LittleCMS could not allocate a text tag")
        try:
            if not self.lib.cmsMLUsetASCII(value, b"en", b"US", text.encode("ascii")):
                raise RuntimeError("LittleCMS could not populate a text tag")
            if not self.lib.cmsWriteTag(profile, int.from_bytes(signature, "big"), value):
                raise RuntimeError(f"LittleCMS could not write {signature!r}")
        finally:
            self.lib.cmsMLUfree(value)

    def generate_profile(self, destination, gray, version):
        if gray:
            curve = self.lib.cmsBuildGamma(None, 2.2)
            if not curve:
                raise RuntimeError("LittleCMS could not create a gray gamma curve")
            try:
                profile = self.lib.cmsCreateGrayProfile(self.lib.cmsD50_xyY(), curve)
            finally:
                self.lib.cmsFreeToneCurve(curve)
        else:
            profile = self.lib.cmsCreate_sRGBProfile()
        if not profile:
            raise RuntimeError("LittleCMS could not create a profile")
        try:
            self.lib.cmsSetProfileVersion(profile, version)
            color = "D50 gray gamma 2.2" if gray else "sRGB matrix/TRC"
            self.write_text_tag(profile, b"desc", f"PicFetch synthetic {color} ICC v{version}")
            self.write_text_tag(profile, b"cprt", "Copyright 2026 Florian Rathe; MIT License")
            if not self.lib.cmsSaveProfileToFile(profile, os.fsencode(destination)):
                raise RuntimeError(f"LittleCMS could not save {destination}")
        finally:
            self.lib.cmsCloseProfile(profile)
        # Fix the creation date so rerunning the generator produces the same data.
        data = bytearray(destination.read_bytes())
        data[24:36] = struct.pack(">6H", 2026, 9, 17, 0, 0, 0)
        destination.write_bytes(data)

    def transform(self, profile, xyz, pixel_format, values, count, intent):
        transform = self.lib.cmsCreateTransform(
            profile, pixel_format, xyz, XYZ_DOUBLE, intent, NO_OPTIMIZE
        )
        if not transform:
            raise RuntimeError(f"LittleCMS could not create {INTENTS[intent]} transform")
        try:
            source = (ctypes.c_double * len(values))(*values)
            result = (ctypes.c_double * (count * 3))()
            self.lib.cmsDoTransform(transform, source, result, count)
            if not all(math.isfinite(value) for value in result):
                raise RuntimeError("LittleCMS produced a non-finite XYZ value")
            return result
        finally:
            self.lib.cmsDeleteTransform(transform)

    def check(self, original, output):
        handles = []
        try:
            for path in (original, output):
                profile = self.lib.cmsOpenProfileFromFile(os.fsencode(path), b"r")
                if not profile:
                    raise RuntimeError(f"LittleCMS could not open {path}")
                handles.append(profile)
            color = self.lib.cmsGetColorSpace(handles[0])
            if self.lib.cmsGetColorSpace(handles[1]) != color:
                raise RuntimeError("profile input color spaces differ")
            if color == RGB_SIGNATURE:
                levels = (0.0, 0.1, 0.25, 0.5, 0.75, 0.9, 1.0)
                samples = list(itertools.product(levels, repeat=3))
                pixel_format = RGB_DOUBLE
            elif color == GRAY_SIGNATURE:
                samples = [(index / 256.0,) for index in range(257)]
                pixel_format = GRAY_DOUBLE
            else:
                raise RuntimeError("reference checker requires RGB or gray profiles")
            xyz = self.lib.cmsCreateXYZProfile()
            if not xyz:
                raise RuntimeError("LittleCMS could not create XYZ output profile")
            handles.append(xyz)
            values = list(itertools.chain.from_iterable(samples))
            maximum = 0.0
            for intent, name in enumerate(INTENTS):
                before, after = (
                    self.transform(profile, xyz, pixel_format, values, len(samples), intent)
                    for profile in handles[:2]
                )
                difference = max(abs(a - b) for a, b in zip(before, after))
                maximum = max(maximum, difference)
                print(f"{name}: samples={len(samples)} max_abs_xyz={difference:.12g}")
            print(f"{original.name} -> {output.name}: max_abs_xyz={maximum:.12g}")
            if maximum > TOLERANCE:
                raise RuntimeError(f"XYZ difference {maximum:.12g} exceeds {TOLERANCE:g}")
        finally:
            for profile in reversed(handles):
                self.lib.cmsCloseProfile(profile)


def generate(destination, lcms):
    version = subprocess.run(
        [str(CJPEG), "-version"], check=True, capture_output=True, text=True
    )
    version_text = (version.stdout + version.stderr).strip()
    if not version_text.startswith("libjpeg-turbo version 3.2.0 "):
        raise RuntimeError(f"unexpected cjpeg version: {version_text}")
    destination.mkdir(parents=True, exist_ok=True)
    width, height = 16, 12
    rgb = bytes(
        channel
        for y in range(height)
        for x in range(width)
        for channel in (
            (17 * x + 29 * y) % 256,
            (31 * x + 7 * y) % 256,
            (11 * x + 43 * y) % 256,
        )
    )
    gray = bytes((19 * x + 37 * y) % 256 for y in range(height) for x in range(width))
    rgb_source = destination / "pattern-rgb.ppm"
    gray_source = destination / "pattern-gray.pgm"
    rgb_source.write_bytes(f"P6\n{width} {height}\n255\n".encode("ascii") + rgb)
    gray_source.write_bytes(f"P5\n{width} {height}\n255\n".encode("ascii") + gray)
    scans = destination / "sequential.scans"
    scans.write_text("0: 0 63 0 0;\n1: 0 63 0 0;\n2: 0 63 0 0;\n", encoding="ascii")
    variants = (
        ("baseline-rgb.jpg", rgb_source, ["-baseline"]),
        ("progressive-rgb.jpg", rgb_source, ["-progressive"]),
        ("multiscan-rgb.jpg", rgb_source, ["-baseline", "-scans", str(scans)]),
        ("baseline-gray.jpg", gray_source, ["-baseline", "-grayscale"]),
        ("progressive-gray.jpg", gray_source, ["-progressive", "-grayscale"]),
    )
    for name, source, options in variants:
        subprocess.run(
            [
                str(CJPEG), "-quality", "90", *options,
                "-outfile", str(destination / name), str(source),
            ],
            check=True,
        )
    for name in PROFILES:
        lcms.generate_profile(
            destination / name, name.startswith("gray"), 2.1 if "v2" in name else 4.3
        )
    print(f"Generated 5 JPEGs and 4 ICC profiles in {destination}")
    print(f"{version_text}; LittleCMS package 2.19.1, encoded API version 2190")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    for name in ("generate", "self-check"):
        command = commands.add_parser(name)
        command.add_argument("destination", type=Path)
    check = commands.add_parser("check")
    check.add_argument("original", type=Path)
    check.add_argument("output", type=Path)
    args = parser.parse_args()
    lcms = LittleCMS()
    if args.command == "check":
        lcms.check(args.original, args.output)
    else:
        generate(args.destination, lcms)
        if args.command == "self-check":
            for name in PROFILES:
                path = args.destination / name
                lcms.check(path, path)


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
        print(f"reference: {error}", file=sys.stderr)
        sys.exit(1)
