# JPEG metadata removal: format facts and contract limits

Research date: 2026-09-17. Primary-source research for the metadata-removal
design discussion. This note makes no implementation or dependency selection,
does not validate the current PicFetch implementation, and contains no new
vulnerability reproduction. Upstream source links below were inspected on this
date and may subsequently change.

## JPEG boundaries and multiple scans

**Established:** JPEG's non-hierarchical interchange syntax contains SOI, one
frame, and EOI. A frame may contain multiple scans. Each scan header may be
preceded by miscellaneous segments, which include COM and APPn. The first SOS
therefore does not mark the end of all metadata. JPEG also distinguishes marker
segments from entropy-coded data; locating boundaries requires respecting that
syntax. [ITU-T T.81, Annex B, especially B.1.1, B.2.1 and B.2.4](https://www.w3.org/Graphics/JPEG/itu-t81.pdf).

**Inference:** Removing only header segments cannot establish removal throughout
the primary image. A contract that keeps only the primary image also needs a
defined policy for bytes beyond its EOI. This follows from the format boundary;
it does not establish the meaning or sensitivity of every possible trailer.

## JFIF and JFXX are not just color hints

**Established:** JFIF APP0 carries version and density/aspect information and
may also carry an RGB thumbnail. JFXX APP0 extensions support JPEG-compressed,
palette RGB, and uncompressed RGB thumbnails. Thus preserving whole recognized
APP0 segments may preserve a separate image. The format does not establish
that every stored thumbnail reflects subsequent edits to the main image.
[ITU-T T.871, 6.3–6.4 and 10.1–10.5](https://www.itu.int/rec/dologin_pub.asp?id=T-REC-T.871-201105-I!!PDF-E&lang=e&type=items).

**Inference:** Thumbnail removal and preservation of essential interpretation
information are separate requirements. A recognized marker identifier alone
does not prove its complete contents satisfy a privacy contract.

## Adobe APP14 and ICC profiles

**Established:** The registered Adobe APP14 marker is component-decorrelation
control information. APPn markers are extensible application data; recognized
markers can affect image utility even when an image remains expandable without
them. [ITU-T T.86 (2024), 6–7 and Annex A](https://www.itu.int/epublications/publication/itu-t-t-86-v2-2024-02-information-technology-digital-compression-and-coding-of-continuous-tone-still-images-appn-markers).

Go's JPEG reader uses Adobe's transform value to interpret color components;
its four-component path can reject an image without that information. The
reader ignores unused APP14 bytes rather than certifying their contents.
[Go JPEG reader, processApp14Marker and applyBlack](https://go.dev/src/image/jpeg/reader.go).

ICC profiles describe color transforms but also have creation date,
manufacturer/model and creator fields, textual descriptions and copyright,
a metadata tag, and private tags. Profile creation time is not necessarily
photo capture time. Wholesale ICC preservation therefore cannot establish
absence of descriptive or potentially identifying metadata.
[ICC.1:2022, Table 17, 9.2.22, 9.2.35, 9.2.43 and 10.1](https://www.color.org/specification/ICC.1-2022-05.pdf).

**Inference:** Keeping JPEG coefficients unchanged and preserving rendered color
are distinct promises. Dropping interpretation information can change appearance
without changing encoded samples. Preserving color transforms while removing
descriptive ICC content is a proposed contract, not a sanitizer implemented or
proven by this research. Supported profile classes, required tags and output
validity would need explicit qualification.

## Lossless removal versus decode and re-encode

**Established:** Coefficient-level JPEG transformation can preserve image data
without another lossy encoding step. libjpeg-turbo documents separate marker
copy policies and explicitly notes that copied thumbnails are not transformed.
It distinguishes this from decompressing and recompressing, which can degrade
the image. This is evidence of a technical distinction, not a recommendation
to add that dependency. [libjpeg-turbo jpegtran manual](https://raw.githubusercontent.com/libjpeg-turbo/libjpeg-turbo/main/doc/jpegtran.1).

Go's JPEG encoder accepts decoded image data plus a quality setting and writes
a new baseline JPEG; its output sequence does not copy the source's APP/COM
segments or trailer. **Inference:** A carefully bounded decode/re-encode path
can avoid transporting those original structures, but it changes the encoded
image and needs explicit orientation and color handling. It is not, by itself,
a promise of unchanged appearance. [Go JPEG encoder, Encode](https://go.dev/src/image/jpeg/writer.go).

## What rejection can and cannot establish

**Established:** Go's decoder intentionally accepts some extraneous input
bytes. A successful decode is consequently not proof of strict format
conformance or absence of ignored data.
[Go JPEG reader, decode](https://go.dev/src/image/jpeg/reader.go).

**Inference:** Rejecting input whose boundaries or supported structures cannot
be established can support the narrow statement that an uncertain file was
left untouched. Structural validation alone does not prove semantic validity
of compressed content, nor absence of information embedded in pixels. No
metadata-removal promise here covers visible identifying content, steganography,
filesystem metadata, sidecars, backups, or copies outside the rewritten file.

Open qualification work includes the supported JPEG processes, the exact
retained-marker policy, ICC normalization validity and color equivalence,
malformed-input behavior, and consistency between eligibility and rewriting.
This research supplies format constraints; it does not establish those
properties for PicFetch.
