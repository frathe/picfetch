---
site:
  base_url: https://frathe.github.io/picfetch/
  product_name: PicFetch
  protected_terms:
    - PicFetch
    - macOS
    - Windows
    - Linux
    - Apple Silicon
    - Intel
    - ARM64
    - arm64
    - x64
    - amd64
    - x86_64
    - Control-click
    - Go
    - GitHub
    - JPEG
    - JPEGs
    - PNG
    - GIF
    - GIFs
    - WebP
    - BMP
    - TIFF
    - ICO
    - XPM
    - SVG
    - HEIC
    - AVIF
    - RAW
    - WASM
    - EXIF
    - ISO
    - GPS
    - OpenStreetMap
    - Gatekeeper
    - Apple Developer ID
    - DeepL
    - Terminal
    - MIT
    - Fyne
metadata:
  title:
    id: metadata.title
    text: PicFetch — a small, fast image viewer for macOS, Windows and Linux
  description:
    id: metadata.description
    text: >-
      PicFetch is a small multi-platform desktop app for quickly viewing and
      browsing images. Drop one or more images onto the window and step through
      them with the keyboard.
  open_graph_title:
    id: metadata.open-graph-title
    text: PicFetch — a small, fast image viewer
  open_graph_description:
    id: metadata.open-graph-description
    text: >-
      Drop one or more images onto the window and step through them with the
      keyboard. Free and open source, for macOS, Windows and Linux.
  open_graph_image: https://raw.githubusercontent.com/frathe/picfetch/main/assets/social_logo.jpg
icons:
  - rel: icon
    type: image/png
    sizes: 32x32
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-32.png
  - rel: icon
    type: image/png
    sizes: 36x36
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-36.png
  - rel: icon
    type: image/png
    sizes: 48x48
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-48.png
  - rel: icon
    type: image/png
    sizes: 72x72
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-72.png
  - rel: icon
    type: image/png
    sizes: 96x96
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-96.png
  - rel: icon
    type: image/png
    sizes: 144x144
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-144-precomposed.png
  - rel: icon
    type: image/png
    sizes: 192x192
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-192.png
  - rel: apple-touch-icon
    sizes: 180x180
    href: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/Favicons%20%28TaneWithFrame%29/favicon-180-precomposed.png
language_flags:
  english: '🇬🇧'
  german: '🇩🇪'
labels:
  language_selector:
    id: labels.language-selector
    text: Choose language
  english:
    id: labels.english
    text: English
  german:
    id: labels.german
    text: German
  lightbox_close:
    id: labels.lightbox-close
    text: Close
  deepl_disclosure:
    id: labels.deepl-disclosure
    text: This page was translated with DeepL and has not been edited.
hero:
  image:
    url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/header.jpg
    width: 1200
    height: 400
  alt:
    id: hero.image-alt
    text: PicFetch — the app window showing a 'Drop images here' prompt beside the mascot artwork
  tagline: hero.tagline
  actions:
    - id: download
      label:
        id: hero.actions.download
        text: Download
      href: '#downloads'
      primary: true
    - id: github
      label:
        id: hero.actions.github
        text: View on GitHub
      href: https://github.com/frathe/picfetch
sections:
  - id: demo-main
    kind: video
    heading:
      id: sections.demo-main.heading
      text: See it in action
    video:
      id: vimeo-main
      video_id: '1220283616'
      width: 1000
      height: 660
      autoplay: true
      title:
        id: videos.demo-main.title
        text: PicFetch
  - id: screenshots
    kind: screenshots
    heading:
      id: sections.screenshots.heading
      text: Screenshots
    screenshots:
      - id: main
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/main_screen.png
          width: 520
          height: 372
        alt:
          id: screenshots.main.alt
          text: PicFetch's main window showing the 'Drop images here' prompt before any images are loaded
        caption:
          id: screenshots.main.caption
          text: Drop images onto the window to get started
      - id: gallery
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/picture_galery.png
          width: 2758
          height: 1772
        alt:
          id: screenshots.gallery.alt
          text: Thumbnail grid view showing dozens of photos at once, with the current image highlighted
        caption:
          id: screenshots.gallery.caption
          text: The thumbnail grid for finding an image by sight
      - id: viewer
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/viewer.png
          width: 2088
          height: 1160
        alt:
          id: screenshots.viewer.alt
          text: A zoomed-in photo with the file info overlay and the EXIF data window open, showing camera, aperture, focal length and capture date
        caption:
          id: screenshots.viewer.caption
          text: Zoomed in, with file info and EXIF data on show
  - id: features
    kind: features
    heading:
      id: sections.features.heading
      text: Features
    features:
      - id: drop-anything
        title:
          id: features.drop-anything.title
          text: Drop almost anything
        body: features.drop-anything.body
      - id: keyboard-browsing
        title:
          id: features.keyboard-browsing.title
          text: Keyboard browsing
        body: features.keyboard-browsing.body
      - id: thumbnail-grid
        title:
          id: features.thumbnail-grid.title
          text: Thumbnail grid
        body: features.thumbnail-grid.body
      - id: zoom-pan
        title:
          id: features.zoom-pan.title
          text: Zoom and pan
        body: features.zoom-pan.body
      - id: animated-gifs
        title:
          id: features.animated-gifs.title
          text: Animated GIFs
        body: features.animated-gifs.body
      - id: exif-aware
        title:
          id: features.exif-aware.title
          text: EXIF aware
        body: features.exif-aware.body
      - id: sorting
        title:
          id: features.sorting.title
          text: Sorting that makes sense
        body: features.sorting.body
      - id: folders-merging
        title:
          id: features.folders-merging.title
          text: Folders and merging
        body: features.folders-merging.body
      - id: named-favorites
        title:
          id: features.named-favorites.title
          text: Named favorites
        body: features.named-favorites.body
      - id: hide-duplicates
        title:
          id: features.hide-duplicates.title
          text: Hide duplicate images
        body: features.hide-duplicates.body
  - id: demo-compare
    kind: video
    heading:
      id: sections.demo-compare.heading
      text: Compare images with ease
    video:
      id: vimeo-compare
      video_id: '1223380739'
      width: 1000
      height: 660
      title:
        id: videos.demo-compare.title
        text: image_compare
  - id: downloads
    kind: downloads
    anchor: downloads
    heading:
      id: sections.downloads.heading
      text: Download
    body: downloads.introduction
    download_groups:
      - id: macos
        title:
          id: downloads.macos.title
          text: macOS
        links:
          - id: macos-arm64
            label:
              id: downloads.macos.arm64
              text: Apple Silicon (arm64)
            href: https://github.com/frathe/picfetch/releases/latest/download/picfetch-macos-arm64.zip
          - id: macos-x86-64
            label:
              id: downloads.macos.x86-64
              text: Intel (x86_64)
            href: https://github.com/frathe/picfetch/releases/latest/download/picfetch-macos-x86_64.zip
      - id: windows
        title:
          id: downloads.windows.title
          text: Windows
        links:
          - id: windows-amd64
            label:
              id: downloads.windows.amd64
              text: x64 (amd64)
            href: https://github.com/frathe/picfetch/releases/latest/download/picfetch-windows-amd64.zip
          - id: windows-arm64
            label:
              id: downloads.windows.arm64
              text: ARM64
            href: https://github.com/frathe/picfetch/releases/latest/download/picfetch-windows-arm64.zip
      - id: linux
        title:
          id: downloads.linux.title
          text: Linux
        links:
          - id: linux-amd64
            label:
              id: downloads.linux.amd64
              text: x64 (amd64)
            href: https://github.com/frathe/picfetch/releases/latest/download/picfetch-linux-amd64.tar.gz
          - id: linux-arm64
            label:
              id: downloads.linux.arm64
              text: ARM64
            href: https://github.com/frathe/picfetch/releases/latest/download/picfetch-linux-arm64.tar.gz
    notice:
      title:
        id: downloads.warning.title
        text: 'macOS: “app is damaged” warning'
      body: downloads.warning.body
footer:
  image:
    url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/trane_comparing_images.webp
    width: 1024
    height: 935
  alt:
    id: footer.image-alt
    text: PicFetch mascot trane, at work comparing images for you.
  links:
    - id: source
      label: {id: footer.links.source, text: Source code}
      href: https://github.com/frathe/picfetch
    - id: privacy
      label: {id: footer.links.privacy, text: Privacy policy}
      href: https://github.com/frathe/picfetch/blob/main/PRIVACY.md
    - id: releases
      label: {id: footer.links.releases, text: Releases}
      href: https://github.com/frathe/picfetch/releases
    - id: issues
      label: {id: footer.links.issues, text: Issues}
      href: https://github.com/frathe/picfetch/issues
    - id: manual
      label: {id: footer.links.manual, text: Manual}
      href: https://github.com/frathe/picfetch/blob/main/internal/ui/help/manual.md
    - id: contributing
      label: {id: footer.links.contributing, text: Contributing}
      href: https://github.com/frathe/picfetch/blob/main/.github/CONTRIBUTING.md
    - id: security
      label: {id: footer.links.security, text: Security}
      href: https://github.com/frathe/picfetch/blob/main/.github/SECURITY.md
    - id: license
      label: {id: footer.links.license, text: License}
      href: https://github.com/frathe/picfetch/blob/main/LICENSE
    - id: buy-coffee
      label: {id: footer.links.buy-coffee, text: Buy me a coffee}
      href: https://buymeacoffee.com/gcobnk0grj
  colophon: footer.colophon
---

## Tagline {#hero.tagline}

A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard.

## Drop almost anything {#features.drop-anything.body}

JPEG, PNG, GIF, WebP, BMP, TIFF, ICO, XPM, SVG, HEIC, AVIF and camera RAW (embedded JPEG preview). HEIC and AVIF decode through embedded WASM, so they need no system libraries.

## Keyboard browsing {#features.keyboard-browsing.body}

Step through a set with the arrow keys (wrapping at both ends), or jump straight to the ends with <kbd>Home</kbd> and <kbd>End</kbd>.

## Thumbnail grid {#features.thumbnail-grid.body}

<kbd>G</kbd> opens a full-window grid for finding an image by sight. Thumbnails load lazily, so a folder of several thousand files stays responsive.

## Zoom and pan {#features.zoom-pan.body}

<kbd>+</kbd> <kbd>−</kbd> <kbd>1</kbd> <kbd>0</kbd>, or scroll to zoom at the cursor. Click-drag or <kbd>Shift</kbd>+scroll to pan once zoomed in.

## Animated GIFs {#features.animated-gifs.body}

Played back frame by frame at their encoded speed, composited correctly per frame so partial updates never leave stale pixels.

## EXIF aware {#features.exif-aware.body}

JPEGs auto-rotate to their orientation tag, and <kbd>E</kbd> shows camera, lens, exposure, aperture, ISO, capture date and coordinates — plus a collapsible OpenStreetMap view pinned at the spot a GPS-tagged photo was taken, collapsed until you ask for it.

## Sorting that makes sense {#features.sorting.body}

Natural name order by default, so <code>IMG_2</code> comes before <code>IMG_10</code>. <kbd>S</kbd> cycles through capture date, modified time, size and raw drop order.

## Folders and merging {#features.folders-merging.body}

Drop folders to scan them recursively, with a live counter for large trees. <kbd>M</kbd> makes further drops add to the set instead of replacing it.

## Named favorites {#features.named-favorites.body}

Save the current file list as a collection and reopen it from the Favorites menu after a restart. The first ten are one shortcut away with <kbd>Cmd/Ctrl</kbd>+<kbd>1</kbd>–<kbd>9</kbd> and <kbd>Cmd/Ctrl</kbd>+<kbd>0</kbd>.

## Hide duplicate images {#features.hide-duplicates.body}

Browse large quantities of images with ease by hiding duplicates and filtering out noise. <kbd>d</kbd> and <kbd>Shift</kbd>+<kbd>D</kbd> (to show duplicate variants).

## Download introduction {#downloads.introduction}

Pre-built binaries, no Go toolchain required — the links below always fetch the newest release. Prefer to build from source? The [build instructions](https://github.com/frathe/picfetch#building) are in the repository.

## macOS warning {#downloads.warning.body}

The release build isn’t signed with an Apple Developer ID or notarized, so Gatekeeper quarantines it after download and claims it’s damaged. It isn’t — to open it anyway, either right-click (Control-click) <code>PicFetch.app</code> and choose <strong>Open</strong>, confirming the dialog that appears, or run <code>xattr -cr "/path/to/PicFetch.app"</code> in Terminal to clear the quarantine flag and then open it normally.

## Colophon {#footer.colophon}

PicFetch is free and open source under the [MIT licence](https://github.com/frathe/picfetch/blob/main/LICENSE), and built with [Fyne](https://fyne.io/).
