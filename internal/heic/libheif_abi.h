/* ABI declarations adapted from libheif v1.17.6 libheif/heif.h.
 * Copyright (c) 2017-2023 Dirk Farin <dirk.farin@gmail.com>
 * SPDX-License-Identifier: LGPL-3.0-or-later
 * Source and license delivery: notices/README.md and notices/LGPL-3.txt.
 * The opaque public objects are represented by void pointers only in the
 * dynamically loaded function signatures in native_linux.c.
 */
#ifndef PICFETCH_LIBHEIF_ABI_H
#define PICFETCH_LIBHEIF_ABI_H
#include <stdint.h>

struct heif_error {
    int code;
    int subcode;
    const char *message;
};

struct heif_color_conversion_options {
    uint8_t version;
    int preferred_chroma_downsampling_algorithm;
    int preferred_chroma_upsampling_algorithm;
    uint8_t only_use_preferred_chroma_algorithm;
};

struct heif_decoding_options {
    uint8_t version;
    uint8_t ignore_transformations;
    void (*start_progress)(int, int, void *);
    void (*on_progress)(int, int, void *);
    void (*end_progress)(int, void *);
    void *progress_user_data;
    uint8_t convert_hdr_to_8bit;
    uint8_t strict_decoding;
    const char *decoder_id;
    struct heif_color_conversion_options color_conversion_options;
};
#endif
