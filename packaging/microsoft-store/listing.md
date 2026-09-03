# Microsoft Store listing handoff

Use this file as the reviewed source of truth when filling Partner Center for
Store ID `9P0DM0KTH01K`. Do not paste this introductory text into the listing.

## Shared URLs and declarations

- Website: https://frathe.github.io/picfetch/
- Support: https://github.com/frathe/picfetch/issues
- Privacy policy: https://github.com/frathe/picfetch/blob/main/PRIVACY.md
- License: https://github.com/frathe/picfetch/blob/main/LICENSE
- License terms: MIT License
- Pricing: Free
- Advertising: None
- Accounts or sign-in: None
- Privacy declaration: Yes, the product accesses or transmits personal
  information. PicFetch reads GPS coordinates embedded in an image locally. It
  transmits the corresponding map-tile coordinates and the device IP address to
  OpenStreetMap only after the user explicitly expands the Location map. Images,
  filenames, and other EXIF fields are not transmitted.

## Restricted-capability explanation

`runFullTrust` is required because PicFetch is a native Fyne/Win32 desktop image
viewer. It uses normal desktop process access to open user-selected files and
folders, save or export images, copy files and pixels, move selected files to
the Recycle Bin, and set the desktop wallpaper. PicFetch does not elevate,
install services, create scheduled tasks, or run background processes after the
app exits.

## Certification notes

PicFetch requires no account or sign-in. Open an image with File -> Open, by
dragging an image or folder onto the window, or through a registered image-file
association. The app can then be exercised entirely with local files.

The EXIF Location map is collapsed by default. Network requests to
`tile.openstreetmap.org` begin only when the user expands Location for a photo
that contains GPS coordinates. The Microsoft Store build does not contact
GitHub for application updates because Microsoft Store manages them.

## English (United States)

### Product name

PicFetch

### Short description

A fast, private, open-source desktop image viewer for browsing local photos.

### Description

PicFetch is a small, fast, free and open-source desktop image viewer. Open an
image, a group of files, or an entire folder and browse without importing your
photos into a library or creating an account.

Navigate with the keyboard, jump through large collections in the thumbnail
grid, compare two images side by side or with an interactive swipe, and inspect
EXIF details without interrupting your flow. PicFetch supports common image
formats, modern HEIC and AVIF files, scalable SVG artwork, animated GIFs, and
embedded previews from many camera RAW formats.

Images are decoded and processed on your device. PicFetch contains no ads,
analytics, or telemetry. The optional EXIF Location map is collapsed by default
and contacts OpenStreetMap only when you explicitly open it.

### Features

Fast browsing of individual images, file selections, and recursively scanned folders

Thumbnail grid with search, multi-selection, favorites, and duplicate handling

Side-by-side and swipe comparison with linked or independent zoom and pan

JPEG, PNG, GIF, WebP, BMP, TIFF, ICO, XPM, HEIC, AVIF, SVG, and camera RAW previews

Animated GIF playback and sharp on-demand SVG rendering

EXIF details, orientation correction, and an optional OpenStreetMap location view

Zoom, pan, rotate, slideshow, wallpaper, clipboard, export, and metadata removal tools

No account, advertising, analytics, or telemetry

English and German user interface

### What's new

PicFetch 1.0.0 is the first Microsoft Store release. Store builds receive
installation and updates through Microsoft Store while retaining the complete
desktop image viewer experience.

### Screenshot 1

- File: `assets/screens/picture_galery.png`
- Caption: Browse large folders visually in the fast thumbnail grid.

### Screenshot 2

- File: `assets/screens/viewer.png`
- Caption: View local images with quick navigation, zoom, and image details.

## German (Germany)

### Produktname

PicFetch

### Kurzbeschreibung

Ein schneller, privater Open-Source-Bildbetrachter für lokale Fotos.

### Beschreibung

PicFetch ist ein kleiner, schneller, kostenloser und quelloffener
Bildbetrachter für den Desktop. Öffnen Sie ein Bild, mehrere Dateien oder einen
ganzen Ordner und blättern Sie durch Ihre Fotos, ohne sie in eine Bibliothek zu
importieren oder ein Konto anzulegen.

Navigieren Sie mit der Tastatur, wechseln Sie in der Miniaturansicht schnell
durch große Sammlungen, vergleichen Sie zwei Bilder nebeneinander oder mit
einem interaktiven Schieberegler und prüfen Sie EXIF-Daten ohne Unterbrechung.
PicFetch unterstützt gängige Bildformate, moderne HEIC- und AVIF-Dateien,
skalierbare SVG-Grafiken, animierte GIFs und eingebettete Vorschauen vieler
Kamera-RAW-Formate.

Bilder werden ausschließlich auf Ihrem Gerät dekodiert und verarbeitet.
PicFetch enthält keine Werbung, Analysefunktionen oder Telemetrie. Die
optionale Standortkarte für EXIF-Daten ist standardmäßig eingeklappt und
kontaktiert OpenStreetMap nur, wenn Sie sie ausdrücklich öffnen.

### Features

Schnelles Blättern durch einzelne Bilder, Dateiauswahlen und rekursiv gelesene Ordner

Miniaturansicht mit Suche, Mehrfachauswahl, Favoriten und Duplikatverwaltung

Vergleich nebeneinander oder per Schieberegler mit gemeinsamem oder unabhängigem Zoom und Verschieben

JPEG, PNG, GIF, WebP, BMP, TIFF, ICO, XPM, HEIC, AVIF, SVG und Vorschauen von Kamera-RAW-Dateien

Wiedergabe animierter GIFs und scharfe SVG-Darstellung in jeder Zoomstufe

EXIF-Daten, Orientierungskorrektur und optionale Standortkarte von OpenStreetMap

Zoom, Verschieben, Drehen, Diashow, Hintergrundbild, Zwischenablage, Export und Entfernen von Metadaten

Kein Konto, keine Werbung, keine Analysefunktionen und keine Telemetrie

Deutsche und englische Benutzeroberfläche

### Neuigkeiten

PicFetch 1.0.0 ist die erste Microsoft-Store-Version. Store-Builds werden über
den Microsoft Store installiert und aktualisiert und bieten den vollständigen
Desktop-Bildbetrachter.

### Screenshot 1

- Datei: `assets/screens/picture_galery.png`
- Beschriftung: Große Ordner schnell in der übersichtlichen Miniaturansicht durchsuchen.

### Screenshot 2

- Datei: `assets/screens/viewer.png`
- Beschriftung: Lokale Bilder mit schneller Navigation, Zoom und Bildinformationen anzeigen.
