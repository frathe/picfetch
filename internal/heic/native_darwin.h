#ifndef PICFETCH_HEIC_DARWIN_H
#define PICFETCH_HEIC_DARWIN_H

#include <stddef.h>
#include <stdint.h>

typedef struct {
    int code, width, height, orientation;
    size_t pixel_bytes;
    uint8_t *pixels;
    char message[512], provider[256];
} picfetch_imageio_result;

picfetch_imageio_result picfetch_imageio_read(const uint8_t *data, size_t length,
                                             int64_t max_pixels, int pixels);
#endif
