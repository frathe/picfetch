# PicFetch threat model

This public overview describes PicFetch's security goals, trust boundaries,
existing safeguards, and remaining limitations. It covers the desktop application,
its optional features, and software delivery. Data-handling details are documented
in the [privacy policy](PRIVACY.md).

## Project scope

PicFetch is a Go/Fyne image viewer for Windows, macOS, and Linux. It opens local
images and folders through file pickers, drag-and-drop, command-line arguments,
and operating-system file associations. It also supports metadata inspection and
removal, image export, clipboard operations, Trash, favorites, and local similarity
analysis.

PicFetch runs with the current user's permissions. It has no user accounts,
public API, listening network service, or privileged background service. Image
viewing and analysis happen on the device. Optional downloads, update operations,
and location-map requests contact external services.

## What needs protection

- **Private content:** images, metadata such as GPS coordinates, filenames,
  folder paths, previews, saved collections, and image-analysis results.
- **User files:** originals and export destinations must be protected from
  unintended modification, deletion, or loss.
- **Application integrity:** installed binaries, updates, dependencies, and
  downloaded analysis assets must come from their intended sources.
- **Availability:** opening an image or folder should not cause uncontrolled
  resource consumption or prevent the user from continuing normal work.
- **Release integrity:** source control, build systems, and publishing access
  must be protected against unauthorized changes.

## Trust boundaries and assumptions

**Files entering the application.** Selecting an image authorizes PicFetch to
process it; it does not make its contents trustworthy. Images, metadata, filenames,
and folder contents may originate from other people or shared storage. Parsing,
preview generation, and analysis must treat those inputs as untrusted.

**The application and the filesystem.** PicFetch relies on operating-system
permissions to protect user data and app storage. It is intended to run without
elevated privileges. A selected folder is not a filesystem sandbox: links may
refer to files elsewhere, and shared files can change while being processed.
Local caches and preferences are not a boundary against software already running
with the same user's access.

**The viewer and analysis workers.** Optional similarity features use local
worker processes. Process separation supports cancellation and contains some
failures, but it is not complete filesystem isolation. Network restrictions vary
by platform; the [privacy policy](PRIVACY.md#similarity-explorer-downloads)
describes those differences.

**External services and downloaded content.** Network responses and downloaded
assets require validation before use. HTTPS protects connections, while asset
checksums and update provenance checks provide additional integrity controls.
The operating system, trusted dependencies, and authorized release infrastructure
remain part of the application's trust assumptions.

## Main threats and safeguards

### Malformed images and excessive resource use

An image or collection can contain malformed data or require unreasonable amounts
of memory, processing time, or storage. PicFetch applies input and image-size
limits, format-specific validation, bounded caches, and limits on background work.
Unsupported or invalid content should produce a controlled error.

These measures reduce risk but do not guarantee that every decoder is free of
bugs or that all processing can be interrupted immediately. Ordinary image
viewing is not fully isolated in a sandbox.

### Unintended file changes and metadata disclosure

Saving, exporting, removing metadata, and moving files to Trash can affect
valuable originals. PicFetch uses coordinated file operations and temporary-file
replacement for image writes, and deletion uses the operating system's Trash
with confirmation. The security goal is to affect only the intended files and
report failures clearly.

Metadata handling depends on the format and the selected operation. A saved or
exported image should not be assumed anonymous. Visible image content, filenames,
and retained local copies can still disclose personal information.

### Local data and optional network features

Sessions, favorites, thumbnails, and analysis caches may retain sensitive paths
and derived image data. PicFetch does not encrypt these records itself; their
confidentiality depends on operating-system access controls and any device
encryption configured by the user.

PicFetch's normal image-viewing and analysis features do not upload images or
analysis results. Model downloads require consent, and automatic update checks
are optional. The location map makes requests only when expanded by the user;
the map provider receives the requested geographic area and ordinary connection
information, including the device's IP address. These requests are
privacy-relevant even though the image itself is not sent.

### Downloaded assets and software updates

Substituted analysis assets or application updates could compromise application
integrity. Analysis downloads are checked against pinned checksums. The standalone
in-app updater verifies release attestations and artifact integrity before
installing an update. Microsoft Store builds use Store-managed delivery and
updates instead.

These controls depend on trustworthy release and dependency sources. They do not
establish that correctly authenticated software is free of defects. Dependency
maintenance, automated security checks, and review of release changes remain
part of protecting the project.

## Assessing reports

Reports are assessed by demonstrated impact, required user interaction, affected
platforms and versions, and whether an existing trust boundary is crossed. Code
execution, unauthorized file changes, disclosure of private content, and update
integrity failures warrant particular attention. Repeatable crashes and excessive
resource use are also relevant.

Prior access to the same user's files or control of the launch environment matters
when assessing impact, but does not automatically make a report irrelevant.
Web-service threats such as account takeover and tenant separation are outside
PicFetch's current architecture. Revisit this model when features, data flows,
platform isolation, or distribution methods change.

## Reporting a vulnerability

Please report suspected vulnerabilities privately using the channels in the
[security policy](.github/SECURITY.md):

- **Preferred:** [GitHub Security Advisories](https://github.com/frathe/picfetch/security/advisories/new)
  for this repository.
- **Alternative:** email [florianrathe@gmail.com](mailto:florianrathe@gmail.com).

Do not publish vulnerability details or affected sample files in public issues,
discussions, or pull requests before coordinating disclosure with the maintainer.

Include the PicFetch version and distribution, operating system, affected feature,
a description of the impact, and minimal reproduction steps. Where safe, check
whether the issue remains present in the latest release; the security policy
lists supported versions. Share only the sample data and redacted diagnostics
needed to explain the issue. Remove credentials, private paths, GPS coordinates,
and unrelated personal content, and arrange a private transfer if sensitive
material is essential to the report.
