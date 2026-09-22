<!-- intro -->
# HEIC installation instructions

PicFetch uses your system's HEIC support. Reading this guide does not check
support, download software or install codecs. Links open only when you choose
them; following them and installing software requires internet access.

PicFetch uses the system decoder's rendition. Color correction is best effort;
ICC, wide-gamut and HDR photos may look different from other viewers.

<!-- macos -->
## macOS

HEIF/HEIC support is built-in to macOS High Sierra 10.13 and later. Use the
system's current updates; there is no separate codec download in this guide.
PicFetch's designated-primary image selection requires macOS Mojave 10.14 or later. PicFetch's own minimum macOS requirement still applies.

[Apple's HEIF and HEVC requirements](https://support.apple.com/en-us/116944)

<!-- windows -->
## Windows

Use Microsoft's official **HEIF Image Extensions** and **HEVC Video Extensions**
from Microsoft Store. HEVC is also needed for HEIC photos, even when you do not
play videos. The HEVC extension may cost money; check the current price and
system requirements, including compatibility with your device, in Store.

- [HEIF Image Extensions](https://apps.microsoft.com/detail/9PMMSR1CGPWG)
- [HEVC Video Extensions](https://apps.microsoft.com/detail/9NMZLZ57R3T7)
- [Microsoft's installation guidance](https://support.microsoft.com/en-us/windows/apps/photos/photos-app-video-editor-error-can-t-view-this-file-type)

Install or update the extensions for the account running PicFetch. A codec
supplied privately by another photo application is not a replacement for the
official Windows components.

<!-- linux -->
## Linux

PicFetch needs the system libheif library and an HEVC decoder usable by that
library, matching PicFetch's architecture. Having libheif installed without an
HEVC decoder is not enough. Packages inside another application's sandbox may
not be available to PicFetch.

<!-- debian13 -->
### Debian 13 (trixie), amd64 or arm64

With Debian's standard repositories configured, refresh package information:

`sudo apt update`

Then install the system library and its HEVC decoder plugin:

`sudo apt install libheif1 libheif-plugin-libde265`

[Debian package and architecture information](https://packages.debian.org/trixie/libheif-plugin-libde265)

<!-- ubuntu2404 -->
### Ubuntu 24.04 LTS (noble), amd64 or arm64

The decoder plugin is in Ubuntu's **universe** repository. If your administrator
has disabled that repository, follow Ubuntu's repository documentation first.
Refresh package information:

`sudo apt update`

Then install the system library and its HEVC decoder plugin:

`sudo apt install libheif1 libheif-plugin-libde265`

- [Ubuntu package and architecture information](https://packages.ubuntu.com/noble/libheif-plugin-libde265)
- [Ubuntu package management](https://ubuntu.com/server/docs/how-to/software/package-management/)

<!-- arch -->
### Arch Linux, rolling release, x86_64

The official Extra repository's libheif package depends on libde265 for HEVC
decoding. With the standard repositories configured:

`sudo pacman -Syu libheif`

This performs a full system upgrade as well as installing libheif. Read Arch's
current upgrade notices before proceeding; avoid partial upgrades.

- [Arch package and dependency information](https://archlinux.org/packages/extra/x86_64/libheif/)
- [Arch package management](https://wiki.archlinux.org/title/Pacman)

<!-- fedora -->
### Fedora

No installation command has been verified here for this release and
architecture. Check Fedora's current libheif packages and whether an HEVC
decoder is available to the system library on your edition.

RPM Fusion is an **optional external repository**, separate from Fedora's
official repositories. Enabling it is your choice; consult its current package
and release guidance before changing repositories.

- [Fedora libheif packages](https://packages.fedoraproject.org/pkgs/libheif/libheif/)
- [Fedora guidance about RPM Fusion](https://docs.fedoraproject.org/en-US/quick-docs/rpmfusion-setup/)
- [RPM Fusion multimedia guidance](https://rpmfusion.org/Howto/Multimedia)

<!-- generic -->
### Other distribution, release or architecture

No installation command has been verified here for this combination. Ask your
distribution's package documentation or administrator for the system libheif
library and a compatible HEVC decoding plugin for your architecture. Do not
copy a command intended for another distribution or release.

[libheif's codec and plugin requirements](https://github.com/strukturag/libheif)

<!-- unknown -->
## System requirements

No installation command has been verified here for this operating system.
Consult your system vendor's documentation about HEIC support. Availability in
another application does not mean this PicFetch build supports the system.

<!-- after -->

## After installation

Return to Settings -> General and choose **Check HEIC support**. You can check
again without closing PicFetch. If the newly installed component is not yet
visible to the system, restart PicFetch and check again. System updates may
also require a computer restart.

Another application may include its own private codecs. A file opening there
does not establish that PicFetch's system decoder can open it.
