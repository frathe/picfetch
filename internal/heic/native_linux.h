#ifndef PICFETCH_HEIC_NATIVE_H
#define PICFETCH_HEIC_NATIVE_H
#include <stddef.h>
#include <stdint.h>

typedef struct {
    int code, width, height;
    size_t pixel_bytes, exif_bytes;
    uint8_t *pixels, *exif;
    char message[512], provider[256];
} picfetch_heic_result;

picfetch_heic_result picfetch_heic_read(const uint8_t *data, size_t length,
                                      int64_t max_pixels, int pixels);
#endif
