# HEIC installation guide source record

Reviewed 2026-09-22 for ticket 01. The embedded English and German guides are
`internal/ui/help/heic_guide.md` and `heic_guide_de.md`. These are documentation
checks against primary sources, not native decoder qualification or proof of a
successful installation. No package or codec was installed during this review.

## Selection and boundaries

The guide selects the current OS. Linux recipes require an exact distribution,
release and PicFetch process architecture match. `ID_LIKE` does not qualify a
derivative. Unlisted releases and architectures receive generic requirements
and links. `/etc/os-release` takes precedence over `/usr/lib/os-release`; the
latter is read only when the former is missing. Reads are bounded to 64 KiB.
Values are read as data and never executed. The source for precedence and
field semantics is the [systemd os-release manual](https://manpages.debian.org/trixie/systemd/os-release.5.en.html).

Guide text and links are embedded, so reading it does not need a network
request, installation, support probe or subprocess. Links require a user click.
German locales use the German document; other locales use English. There is
no new dependency or bundled codec in this ticket.

## Qualified documentation recipes

- **Debian 13 / trixie, amd64 and arm64:**
  [Debian's libheif-plugin-libde265 package](https://packages.debian.org/trixie/libheif-plugin-libde265)
  lists version `1.19.8-1+deb13u1`, both architectures, and dependencies on
  matching `libheif1` and `libde265-0`. The guide offers `sudo apt update` then
  `sudo apt install libheif1 libheif-plugin-libde265` with standard repositories
  configured. Command semantics were checked against [APT's manual](https://manpages.debian.org/trixie/apt/apt.8.en.html).
- **Ubuntu 24.04 / noble, amd64 and arm64:**
  [Ubuntu's plugin package](https://packages.ubuntu.com/noble/libheif-plugin-libde265)
  places the plugin in universe and lists both architectures. Observed versions
  were `1.17.6-1ubuntu4.8` for amd64 and `1.17.6-1ubuntu4` for arm64, with
  matching `libheif1` dependencies. The guide offers the same APT commands and
  identifies universe as a prerequisite. [Ubuntu's package management guide](https://ubuntu.com/server/docs/how-to/software/package-management/)
  supplies the package installation context. No command enables extra sources.
- **Arch Linux rolling, x86_64 only:**
  [Arch's official Extra libheif package](https://archlinux.org/packages/extra/x86_64/libheif/)
  lists `1.23.4-1` and a `libde265` dependency for x86_64. The guide offers
  `sudo pacman -Syu libheif`, explicitly identifying the full system upgrade.
  Command semantics were checked against [the pacman manual](https://man.archlinux.org/man/pacman.8.en).
  [Arch's package management documentation](https://wiki.archlinux.org/title/Pacman)
  was available through its indexed primary-source content; direct access was
  blocked by its bot challenge. Arch ARM and derivatives are not qualified by
  the x86_64 package page.

These commands are guidance assembled from the named package records and the
documented package-manager syntax. They were not executed. They do not pin
package versions; users receive their distribution's current updates.

## Platform requirements and generic guidance

- **macOS:** [Apple's HEIF/HEVC support article](https://support.apple.com/en-us/116944),
  published 2025-12-05, describes built-in support from macOS High Sierra
  10.13. This is a media requirement, not PicFetch's minimum deployment target.
  The guide offers no separate codec download.
- **Windows:** [Microsoft's HEIF/HEVC guidance](https://support.microsoft.com/en-us/windows/apps/photos/photos-app-video-editor-error-can-t-view-this-file-type)
  identifies both official extensions, including HEVC for photos and its cost.
  The guide links its official product IDs
  [HEIF `9PMMSR1CGPWG`](https://apps.microsoft.com/detail/9PMMSR1CGPWG) and
  [HEVC `9NMZLZ57R3T7`](https://apps.microsoft.com/detail/9NMZLZ57R3T7).
  Store price/device requirements are left to the live listing. The HEVC
  listing resolved; automated HEIF listing retrieval failed. No claim about
  a particular Store price or device architecture was inferred from that page.
- **Fedora:** [Fedora's libheif package index](https://packages.fedoraproject.org/pkgs/libheif/libheif/)
  confirms the system package, but does not establish a complete HEVC decoder
  recipe for each release/architecture. All Fedora combinations therefore get
  generic requirements without installation commands. [Fedora's third-party
  application policy](https://fedoraproject.org/wiki/Workstation/3rdPartyApps)
  distinguishes external repositories and user opt-in. RPM Fusion is labeled
  optional and external. The guide links [Fedora's RPM Fusion guidance](https://docs.fedoraproject.org/en-US/quick-docs/rpmfusion-setup/)
  and [RPM Fusion multimedia guidance](https://rpmfusion.org/Howto/Multimedia);
  direct retrieval of both was blocked by bot challenges. No package name or
  command was inferred from those blocked pages.
- **Other Linux combinations:** [libheif upstream](https://github.com/strukturag/libheif)
  documents HEVC decoding via libde265 and codec plugin configurations. The
  guide explains the need for an available system decoder, including the
  distinction from codecs private to another application.

Changing any recipe requires a new dated primary-source review and a matching
actual-window selection case in `TestHEICGuideCurrentSystemSelection`. Native
qualification belongs to the decoder tickets and remains separate.
