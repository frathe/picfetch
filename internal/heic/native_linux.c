//go:build linux && cgo && (amd64 || arm64)

#include "native_linux.h"
#include "libheif_abi.h"
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

/* No mandatory libheif link: every symbol is resolved only in the child. */
#define SYMBOL(type, name, args) \
    typedef type (*name##_function) args; \
    name##_function name = (name##_function)dlsym(library, #name); \
    if (!name) { result.code = 1; snprintf(result.message, sizeof(result.message), "missing libheif ABI: %s", #name); goto done; }
#define FAILURE(category, text) do { result.code = category; snprintf(result.message, sizeof(result.message), "%s", text); goto done; } while (0)
#define CHECK(call) do { struct heif_error error = (call); if (error.code) { \
    result.code = (error.code == 3 || error.code == 4) ? 2 : ((error.code == 2 || error.subcode == 1000) ? 3 : 4); \
    snprintf(result.message, sizeof(result.message), "libheif error %d/%d: %.400s", error.code, error.subcode, error.message); goto done; } } while (0)

picfetch_heic_result picfetch_heic_read(const uint8_t *data, size_t length, int64_t max_pixels, int pixels) {
    picfetch_heic_result result = {0};
    void *library = dlopen("libheif.so.1", RTLD_NOW | RTLD_LOCAL);
    void *context = NULL, *handle = NULL, *image = NULL;
    struct heif_decoding_options *options = NULL;
    int initialized = 0;
    void (*context_free)(void *) = NULL, (*handle_release)(const void *) = NULL;
    void (*image_release)(const void *) = NULL, (*deinit)(void) = NULL;
    void (*options_free)(struct heif_decoding_options *) = NULL;
    if (!library) { FAILURE(1, "installed libheif.so.1 could not be loaded"); }
    SYMBOL(const char *, heif_get_version, (void));
    SYMBOL(uint32_t, heif_get_version_number, (void));
    if (heif_get_version_number() < 0x01110600 || heif_get_version_number() >= 0x02000000) {
        FAILURE(1, "libheif ABI requires qualified version 1.17.6 or newer 1.x");
    }
    SYMBOL(struct heif_error, heif_init, (void *));
    SYMBOL(void, heif_deinit, (void)); deinit = heif_deinit;
    SYMBOL(void *, heif_context_alloc, (void));
    SYMBOL(void, heif_context_free, (void *)); context_free = heif_context_free;
    SYMBOL(void, heif_context_set_maximum_image_size_limit, (void *, int));
    SYMBOL(void, heif_context_set_max_decoding_threads, (void *, int));
    SYMBOL(struct heif_error, heif_context_read_from_memory_without_copy, (void *, const void *, size_t, const void *));
    SYMBOL(struct heif_error, heif_context_get_primary_image_handle, (void *, void **));
    SYMBOL(void, heif_image_handle_release, (const void *)); handle_release = heif_image_handle_release;
    SYMBOL(int, heif_image_handle_get_width, (const void *));
    SYMBOL(int, heif_image_handle_get_height, (const void *));
    SYMBOL(int, heif_image_handle_is_premultiplied_alpha, (const void *));
    SYMBOL(int, heif_get_decoder_descriptors, (int, const void **, int));
    SYMBOL(const char *, heif_decoder_descriptor_get_id_name, (const void *));
    SYMBOL(const char *, heif_decoder_descriptor_get_name, (const void *));
    SYMBOL(struct heif_decoding_options *, heif_decoding_options_alloc, (void));
    SYMBOL(void, heif_decoding_options_free, (struct heif_decoding_options *)); options_free = heif_decoding_options_free;
    SYMBOL(struct heif_error, heif_decode_image, (const void *, void **, int, int, const struct heif_decoding_options *));
    SYMBOL(void, heif_image_release, (const void *)); image_release = heif_image_release;
    SYMBOL(int, heif_image_get_primary_width, (const void *));
    SYMBOL(int, heif_image_get_primary_height, (const void *));
    SYMBOL(const uint8_t *, heif_image_get_plane_readonly, (const void *, int, int *));
    SYMBOL(int, heif_image_handle_get_list_of_metadata_block_IDs, (const void *, const char *, uint32_t *, int));
    SYMBOL(size_t, heif_image_handle_get_metadata_size, (const void *, uint32_t));
    SYMBOL(struct heif_error, heif_image_handle_get_metadata, (const void *, uint32_t, void *));

    CHECK(heif_init(NULL)); initialized = 1;
    const void *descriptor = NULL;
    if (heif_get_decoder_descriptors(1, &descriptor, 1) != 1 || !descriptor) { FAILURE(1, "no installed HEVC decoder plugin"); }
    const char *decoder_id = heif_decoder_descriptor_get_id_name(descriptor);
    if (!decoder_id || !decoder_id[0]) { FAILURE(1, "HEVC decoder has no stable provider identity"); }
    snprintf(result.provider, sizeof(result.provider), "libheif %.40s; %.160s", heif_get_version(), heif_decoder_descriptor_get_name(descriptor));
    context = heif_context_alloc();
    if (!context) { FAILURE(4, "allocating libheif context failed"); }
    heif_context_set_maximum_image_size_limit(context, (int)max_pixels);
    heif_context_set_max_decoding_threads(context, 0);
    CHECK(heif_context_read_from_memory_without_copy(context, data, length, NULL));
    CHECK(heif_context_get_primary_image_handle(context, &handle));
    int width = heif_image_handle_get_width(handle), height = heif_image_handle_get_height(handle);
    if (width <= 0 || height <= 0 || (int64_t)width * height > max_pixels) { FAILURE(3, "primary dimensions exceed pixel limit"); }
    options = heif_decoding_options_alloc();
    if (!options || options->version < 4) { FAILURE(1, "incompatible libheif decoding options ABI"); }
    options->ignore_transformations = 0;
    options->strict_decoding = 1;
    // Let the native decoder render color and bit depth into the viewer format.
    options->convert_hdr_to_8bit = 1;
    options->decoder_id = decoder_id;
    CHECK(heif_decode_image(handle, &image, 1, 11, options));
    result.width = heif_image_get_primary_width(image);
    result.height = heif_image_get_primary_height(image);
    if (result.width <= 0 || result.height <= 0 || (int64_t)result.width * result.height > max_pixels) { FAILURE(3, "decoded dimensions exceed pixel limit"); }
    int stride = 0;
    const uint8_t *plane = heif_image_get_plane_readonly(image, 10, &stride);
    if (!plane || stride < (int64_t)result.width * 4 || (int64_t)stride * result.height > 4LL * 1024 * 1024 * 1024) { FAILURE(3, "invalid decoded RGBA plane"); }
    if (pixels) {
        result.pixel_bytes = (size_t)result.width * result.height * 4;
        result.pixels = malloc(result.pixel_bytes);
        if (!result.pixels) { FAILURE(4, "allocating canonical pixels failed"); }
        int premultiplied = heif_image_handle_is_premultiplied_alpha(handle);
        for (int y = 0; y < result.height; y++) {
            uint8_t *row = result.pixels + (size_t)y * result.width * 4;
            memcpy(row, plane + (size_t)y * stride, (size_t)result.width * 4);
            if (premultiplied) for (int x = 0; x < result.width; x++) {
                uint8_t *pixel = row + x * 4;
                for (int channel = 0; channel < 3; channel++) {
                    unsigned value = pixel[3] ? ((unsigned)pixel[channel] * 255 + pixel[3] / 2) / pixel[3] : 0;
                    pixel[channel] = value > 255 ? 255 : (uint8_t)value;
                }
            }
        }
    }
    uint32_t metadata_id = 0;
    if (heif_image_handle_get_list_of_metadata_block_IDs(handle, "Exif", &metadata_id, 1) == 1) {
        size_t bytes = heif_image_handle_get_metadata_size(handle, metadata_id);
        if (bytes > 0 && bytes <= 1024 * 1024) {
            result.exif = malloc(bytes);
            if (!result.exif) { FAILURE(4, "allocating EXIF metadata failed"); }
            struct heif_error metadata_error = heif_image_handle_get_metadata(handle, metadata_id, result.exif);
            if (!metadata_error.code) result.exif_bytes = bytes;
        }
    }
done:
    if (options && options_free) options_free(options);
    if (image && image_release) image_release(image);
    if (handle && handle_release) handle_release(handle);
    if (context && context_free) context_free(context);
    if (initialized && deinit) deinit();
    if (library) dlclose(library);
    if (result.code) {
        free(result.pixels); free(result.exif);
        result.pixels = NULL; result.exif = NULL;
        result.pixel_bytes = result.exif_bytes = 0;
    }
    return result;
}
