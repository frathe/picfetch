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
      Browse, compare and rediscover your photos with PicFetch. Explore GPS
      locations, find similar images locally and create photo mosaics.
      Free and open source for macOS, Windows and Linux.
  open_graph_title:
    id: metadata.open-graph-title
    text: PicFetch — a small, fast image viewer
  open_graph_description:
    id: metadata.open-graph-description
    text: >-
      Explore your photos by location or visual similarity, compare images and
      create mosaics. Free and open source, for macOS, Windows and Linux.
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
      video_id: '1231254824'
      width: 1920
      height: 1080
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
          width: 1904
          height: 1462
        alt:
          id: screenshots.viewer.alt
          text: A photo with the EXIF data window open, showing camera settings, capture date and an OpenStreetMap view of its recorded GPS location
        caption:
          id: screenshots.viewer.caption
          text: Inspect camera settings and the photo's recorded location
      - id: shot-location-map
        image:
          url: https://frathe.github.io/picfetch/screenshots/location-map.webp
          width: 1726
          height: 1083
        alt:
          id: screenshots.location-map.alt
          text: PicFetch's Location Map showing photo thumbnails and numbered clusters across Europe, with Fit All and Back to Viewer controls
        caption:
          id: screenshots.location-map.caption
          text: Browse GPS-tagged photos by place with Location Map
      - id: shot-similarity-explorer
        image:
          url: https://frathe.github.io/picfetch/screenshots/similarity-explorer.webp
          width: 1728
          height: 1083
        alt:
          id: screenshots.similarity-explorer.alt
          text: Similarity Explorer showing stacks of related photos, subject filters and a granularity control for adjusting the grouping
        caption:
          id: screenshots.similarity-explorer.caption
          text: Discover related photos in Similarity Explorer
      - id: side-by-side
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/side-by-side-compare.png
          width: 1208
          height: 645
        alt:
          id: screenshots.side-by-side.alt
          text: Two photos displayed side by side in PicFetch for comparing details
        caption:
          id: screenshots.side-by-side.caption
          text: Compare two photos side by side with linked zoom and pan
      - id: swipe-compare
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/swipe-compare.png
          width: 1208
          height: 645
        alt:
          id: screenshots.swipe-compare.alt
          text: Two photos overlaid in PicFetch with a movable swipe divider revealing the differences
        caption:
          id: screenshots.swipe-compare.caption
          text: Slide the divider to reveal differences between images
      - id: mosaic-settings
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/mosaic_setting.png
          width: 1726
          height: 956
        alt:
          id: screenshots.mosaic-settings.alt
          text: The Image Mosaic settings window showing a 403-image selection, target display picker, and advanced controls for image size, frame, size variation, overlap, rotation and drop shadows
        caption:
          id: screenshots.mosaic-settings.caption
          text: Choose a display and tune the look of the mosaic
      - id: mosaic-creation
        image:
          url: https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/mosaic_creation.png
          width: 3456
          height: 2166
        alt:
          id: screenshots.mosaic-creation.alt
          text: A generated image mosaic made from overlapping framed travel photographs, with controls to regenerate, save it, or set it as wallpaper
        caption:
          id: screenshots.mosaic-creation.caption
          text: Generate a display-sized mosaic, then save it or set it as wallpaper
  - id: features
    kind: features
    heading:
      id: sections.features.heading
      text: Features
    features:
      - id: location-map
        title:
          id: features.location-map.title
          text: Browse photos by location
        body: features.location-map.body
      - id: similarity-explorer
        title:
          id: features.similarity-explorer.title
          text: Similarity Explorer
        body: features.similarity-explorer.body
      - id: find-similar
        title:
          id: features.find-similar.title
          text: Find more photos like this
        body: features.find-similar.body
      - id: comparison
        title:
          id: features.comparison.title
          text: Compare every detail
        body: features.comparison.body
      - id: copy-selection
        title:
          id: features.copy-selection.title
          text: Copy just the part you need
        body: features.copy-selection.body
      - id: export
        title:
          id: features.export.title
          text: Export a copy to share
        body: features.export.body
      - id: picture-frame
        title:
          id: features.picture-frame.title
          text: Picture-frame mode
        body: features.picture-frame.body
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
      - id: mosaic-generator
        title:
          id: features.mosaic-generator.title
          text: Mosaic generator
        body: features.mosaic-generator.body
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
  - id: demo-basic-usage
    kind: video
    heading:
      id: sections.demo-basic-usage.heading
      text: Basic usage and image browsing
    video:
      id: vimeo-basic-usage
      video_id: '1220283616'
      width: 1000
      height: 660
      autoplay: false
      title:
        id: videos.demo-basic-usage.title
        text: PicFetch — basic usage and image browsing
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
        badge:
          name: Microsoft Store
          href: https://apps.microsoft.com/detail/9p0dm0kth01k?ocid=webpdpshare
          images:
            en:
              url: https://get.microsoft.com/images/en-us%20dark.svg
              width: 161
              height: 44
            de:
              url: https://get.microsoft.com/images/de%20dark.svg
              width: 183
              height: 44
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

JPEG, PNG, GIF, WebP, BMP, TIFF, ICO, XPM, SVG, AVIF and camera RAW (embedded JPEG preview). HEIC/HEIF opens with a compatible system decoder; Settings includes a support check and installation instructions. AVIF support is built in.

## Browse photos by location {#features.location-map.body}

<kbd>Shift</kbd>+<kbd>L</kbd> opens Location Map for your loaded collection. Explore recorded GPS locations, open nearby photo groups and return to the same map position. Pan, zoom and navigate by keyboard, with light and dark map styles. OpenStreetMap tiles need an internet connection.

## Similarity Explorer {#features.similarity-explorer.body}

<kbd>Shift</kbd>+<kbd>S</kbd> groups visually related photos into stacks. Filter by 75 subjects and scenes, adjust the grouping and save reusable presets. Available on supported macOS, Linux and Windows 11 systems. After an optional first-use model download, image analysis runs locally and works offline — your images are never uploaded.

## Find more like this {#features.find-similar.body}

Choose a reference image and press <kbd>Cmd/Ctrl</kbd>+<kbd>Shift</kbd>+<kbd>L</kbd> to find up to 30 similar pictures in your loaded collection. Browse the matches, compare your picks and save selected results to Favorites. Uses the same local analysis as Similarity Explorer.

## Compare every detail {#features.comparison.body}

Compare two images side by side or overlay them with a movable swipe divider. Link zoom and pan to inspect the same detail in both, adjust each image independently and swap sides.

## Copy just the part you need {#features.copy-selection.body}

<kbd>Opt/Alt</kbd>+<kbd>Shift</kbd>+<kbd>C</kbd> starts Copy Selection. Draw a rectangle, move or resize it, then copy that image region as PNG pixels at its own resolution. Paste it into another app without changing the original image.

## Export a copy to share {#features.export.body}

Export PNG or JPEG copies with an optional longest-edge limit of 2400, 1600 or 1000 pixels. Leave camera, capture-date and GPS metadata out of JPEG copies while keeping the original untouched.

## Picture-frame mode {#features.picture-frame.body}

<kbd>P</kbd> turns your collection into a full-screen slideshow with crossfades. Adjust the timing with the up and down arrows, or use <kbd>Shift</kbd>+<kbd>P</kbd> to shuffle the order.

## Keyboard browsing {#features.keyboard-browsing.body}

Step through a set with the arrow keys (wrapping at both ends), or jump straight to the ends with <kbd>Home</kbd> and <kbd>End</kbd>.

## Thumbnail grid {#features.thumbnail-grid.body}

<kbd>G</kbd> opens a full-window grid for finding an image by sight. Thumbnails load lazily, so a folder of several thousand files stays responsive.

## Mosaic generator {#features.mosaic-generator.body}

Turn your Grid View selection — or the current filtered grid when nothing is selected — into a display-sized image mosaic. Choose Random or Shelf layout, tune frames, overlap and shadows, then regenerate, save it as PNG or JPEG, or set it as wallpaper.

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
