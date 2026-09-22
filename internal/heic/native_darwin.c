//go:build darwin && cgo && (amd64 || arm64)

#include "native_darwin.h"
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <ImageIO/ImageIO.h>
#include <dlfcn.h>
#include <limits.h>
#include <stdio.h>
#include <stdlib.h>

#define FAILURE(category, text) do { result.code = category; snprintf(result.message, sizeof(result.message), "%s", text); goto done; } while (0)

static int integer_property(CFDictionaryRef properties, CFStringRef key, int64_t *value) {
    CFTypeRef number = CFDictionaryGetValue(properties, key);
    return number && CFGetTypeID(number) == CFNumberGetTypeID() &&
        CFNumberGetValue((CFNumberRef)number, kCFNumberSInt64Type, value);
}

static int heic_type(CFStringRef type) {
    return type && (CFEqual(type, CFSTR("public.heic")) || CFEqual(type, CFSTR("public.heif")));
}

static void provider_identity(char *buffer, size_t length) {
    char version[128] = "unknown version";
    CFBundleRef bundle = CFBundleGetBundleWithIdentifier(CFSTR("com.apple.ImageIO"));
    CFTypeRef value = bundle ? CFBundleGetValueForInfoDictionaryKey(bundle, kCFBundleVersionKey) : NULL;
    if (value && CFGetTypeID(value) == CFStringGetTypeID()) {
        if (!CFStringGetCString((CFStringRef)value, version, sizeof(version), kCFStringEncodingUTF8)) {
            snprintf(version, sizeof(version), "%s", "unknown version");
        }
    }
    snprintf(buffer, length, "Apple ImageIO %s", version);
}

picfetch_imageio_result picfetch_imageio_read(const uint8_t *data, size_t length,
                                             int64_t max_pixels, int pixels) {
    picfetch_imageio_result result = {0};
    CFDataRef encoded = NULL;
    CFArrayRef types = NULL;
    CGImageSourceRef source = NULL;
    CFDictionaryRef properties = NULL, options = NULL;
    CGImageRef image = NULL;
    CGColorSpaceRef destination = NULL;
    CGContextRef context = NULL;

    // This public API first shipped in macOS 10.14. An older OS is an
    // unavailable backend, never permission to assume that item zero is primary.
    typedef size_t (*primary_index_function)(CGImageSourceRef);
    primary_index_function primary_index = (primary_index_function)dlsym(RTLD_DEFAULT, "CGImageSourceGetPrimaryImageIndex");
    if (!primary_index) { FAILURE(1, "ImageIO primary-image selection requires macOS 10.14 or later"); }
    types = CGImageSourceCopyTypeIdentifiers();
    int supports_heic = 0;
    if (types) for (CFIndex index = 0; index < CFArrayGetCount(types); index++) {
        CFTypeRef type = CFArrayGetValueAtIndex(types, index);
        if (type && CFGetTypeID(type) == CFStringGetTypeID() && heic_type((CFStringRef)type)) supports_heic = 1;
    }
    if (!supports_heic) { FAILURE(1, "ImageIO does not advertise a system HEIC decoder"); }
    if (!data || length == 0 || length > LONG_MAX || max_pixels <= 0) { FAILURE(3, "invalid ImageIO request bounds"); }
    encoded = CFDataCreateWithBytesNoCopy(kCFAllocatorDefault, data, (CFIndex)length, kCFAllocatorNull);
    if (!encoded) { FAILURE(4, "allocating ImageIO source data failed"); }
    source = CGImageSourceCreateWithData(encoded, NULL);
    if (!source) { FAILURE(3, "ImageIO could not parse the HEIC source"); }
    if (!heic_type(CGImageSourceGetType(source))) { FAILURE(2, "source is not an HEIC still"); }
    size_t index = primary_index(source);
    if (index >= CGImageSourceGetCount(source)) { FAILURE(3, "invalid designated-primary image index"); }
    properties = CGImageSourceCopyPropertiesAtIndex(source, index, NULL);
    if (!properties) { FAILURE(3, "ImageIO primary properties are unavailable"); }
    int64_t width = 0, height = 0, orientation = 1;
    if (!integer_property(properties, kCGImagePropertyPixelWidth, &width) ||
        !integer_property(properties, kCGImagePropertyPixelHeight, &height) ||
        width <= 0 || height <= 0 || width > INT_MAX || height > INT_MAX || width > max_pixels / height) {
        FAILURE(3, "primary dimensions exceed pixel limit");
    }
    if (CFDictionaryContainsKey(properties, kCGImagePropertyOrientation) &&
        (!integer_property(properties, kCGImagePropertyOrientation, &orientation) || orientation < 1 || orientation > 8)) {
        FAILURE(3, "invalid primary orientation");
    }
    const void *keys[] = {kCGImageSourceShouldCacheImmediately, kCGImageSourceShouldAllowFloat};
    const void *values[] = {kCFBooleanTrue, kCFBooleanFalse};
    options = CFDictionaryCreate(kCFAllocatorDefault, keys, values, 2,
                                  &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    if (!options) { FAILURE(4, "allocating ImageIO options failed"); }
    image = CGImageSourceCreateImageAtIndex(source, index, options);
    if (!image) { FAILURE(3, "ImageIO full-primary decoding failed"); }
    if (CGImageSourceGetStatusAtIndex(source, index) != kCGImageStatusComplete) {
        FAILURE(3, "ImageIO primary decoding did not complete");
    }
    if (CGImageGetWidth(image) != (size_t)width || CGImageGetHeight(image) != (size_t)height) {
        FAILURE(3, "full-primary decode disagrees with source dimensions");
    }
    result.width = (int)width;
    result.height = (int)height;
    result.orientation = (int)orientation;
    provider_identity(result.provider, sizeof(result.provider));
    if (pixels) {
        result.pixel_bytes = (size_t)width * (size_t)height * 4;
        result.pixels = calloc(1, result.pixel_bytes);
        if (!result.pixels) { FAILURE(4, "allocating canonical RGBA pixels failed"); }
        destination = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
        if (!destination) { FAILURE(4, "creating sRGB color space failed"); }
        context = CGBitmapContextCreate(result.pixels, (size_t)width, (size_t)height, 8,
                                       (size_t)width * 4, destination,
                                       kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
        if (!context) { FAILURE(4, "creating the canonical bitmap context failed"); }
        CGContextSetBlendMode(context, kCGBlendModeCopy);
        CGContextSetInterpolationQuality(context, kCGInterpolationNone);
        CGContextDrawImage(context, CGRectMake(0, 0, (CGFloat)width, (CGFloat)height), image);
        CGContextFlush(context);
        // CoreGraphics' RGBA bitmap context is premultiplied; shared imaging
        // consumers require straight alpha, including transparent black.
        for (size_t offset = 0; offset < result.pixel_bytes; offset += 4) {
            uint8_t *pixel = result.pixels + offset;
            for (int channel = 0; channel < 3; channel++) {
                unsigned value = pixel[3] ? ((unsigned)pixel[channel] * 255 + pixel[3] / 2) / pixel[3] : 0;
                pixel[channel] = value > 255 ? 255 : (uint8_t)value;
            }
        }
    }
done:
    if (context) CGContextRelease(context);
    if (destination) CGColorSpaceRelease(destination);
    if (image) CGImageRelease(image);
    if (options) CFRelease(options);
    if (properties) CFRelease(properties);
    if (source) CFRelease(source);
    if (encoded) CFRelease(encoded);
    if (types) CFRelease(types);
    if (result.code) {
        free(result.pixels);
        result.pixels = NULL;
        result.pixel_bytes = 0;
    }
    return result;
}
