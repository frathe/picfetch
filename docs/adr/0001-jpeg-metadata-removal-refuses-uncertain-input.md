# Refuse uncertain JPEG metadata removal

Status: accepted

JPEG metadata removal is a privacy operation on the original file, so success
must cover identifying metadata and secondary media throughout that file.
We preserve encoded image data and color appearance for every orientation, accepting refusal
when complete removal or the required fidelity cannot be established instead of
partial removal or automatic recompression. This deliberately permits refusing a
file the viewer can display: decoder tolerance, recognized container identifiers
and an empty EXIF list do not establish that the retained data satisfies the
privacy contract.

The compatibility follow-up reconstructs only validated orientation, chroma
placement and enumerated color declarations in a minimal EXIF block. Keeping
these rendering instructions with unchanged image data avoids recompression and
removes the need to guess EXIF/ICC precedence. This replaces the original
quality-95 orientation re-encode, whose stricter declaration refusals made common
camera/editor output unavailable. No source EXIF directories, descriptive values,
previews, padding or unclaimed bytes survive. Metadata omission during export
remains a separate operation.

The tracked [qualification record](../jpeg-metadata-removal-qualification.md)
defines the supported behavior, acceptance checks and limits. The
[research](../jpeg-metadata-removal-research-2026-09-17.md) records the supporting
format facts.
