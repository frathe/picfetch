# Refuse uncertain JPEG metadata removal

Status: accepted

JPEG metadata removal is a privacy operation on the original file, so success
must cover identifying metadata and secondary media throughout that file.
We preserve lossless upright images and their color appearance, accepting refusal
when complete removal or the required fidelity cannot be established instead of
partial removal or automatic recompression. This deliberately permits refusing a
file the viewer can display: decoder tolerance, recognized container identifiers
and an empty EXIF list do not establish that the retained data satisfies the
privacy contract.

The existing orientation-correcting re-encode for sideways photos retains its
documented quality qualification and must satisfy the same metadata/profile
policy. Metadata omission during export remains a separate operation.

The [specification](../../.scratch/jpeg-metadata-privacy/spec.md) defines the
behavioral requirements; the [research](../jpeg-metadata-removal-research-2026-09-17.md)
records the supporting format facts and limits.
