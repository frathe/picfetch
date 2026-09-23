<!-- intro -->
# HEIC-Installationsanleitung

PicFetch verwendet die HEIC-Unterstützung Ihres Systems. Das Lesen dieser
Anleitung prüft die Unterstützung nicht und lädt oder installiert keine
Software. Links öffnen sich nur beim Anklicken. Dafür und für die Installation
ist eine Internetverbindung erforderlich.

PicFetch verwendet die Darstellung des Systemdecoders. Die Farbkorrektur ist
nicht garantiert; ICC-, Wide-Gamut- und HDR-Fotos können anders aussehen als in
anderen Bildbetrachtern.

<!-- macos -->
## macOS

HEIF/HEIC-Unterstützung ist ab macOS High Sierra 10.13 integriert. Installieren
Sie die aktuellen Systemupdates; ein separater Codec-Download ist hierfür nicht
vorgesehen. Die Auswahl des festgelegten Hauptbilds in PicFetch erfordert macOS Mojave 10.14 oder neuer. PicFetchs eigene Mindestanforderung an macOS gilt weiterhin.

[Apples Anforderungen für HEIF und HEVC](https://support.apple.com/en-us/116944)

<!-- windows -->
## Windows

Verwenden Sie Microsofts offizielle **HEIF-Bilderweiterungen** und
**HEVC-Videoerweiterungen** aus dem Microsoft Store. HEVC wird auch für
HEIC-Fotos benötigt. Die HEVC-Erweiterung kann kostenpflichtig sein; prüfen
Sie im Store den aktuellen Preis und die Systemanforderungen für Ihr Gerät.

- [HEIF-Bilderweiterungen](https://apps.microsoft.com/detail/9PMMSR1CGPWG)
- [HEVC-Videoerweiterungen](https://apps.microsoft.com/detail/9NMZLZ57R3T7)
- [Microsofts Installationshinweise](https://support.microsoft.com/en-us/windows/apps/photos/photos-app-video-editor-error-can-t-view-this-file-type)

Installieren oder aktualisieren Sie die Erweiterungen für das Benutzerkonto,
unter dem PicFetch läuft. Private Codecs anderer Fotoanwendungen ersetzen die
offiziellen Windows-Komponenten nicht.

<!-- linux -->
## Linux

PicFetch benötigt die systemweite libheif-Bibliothek und einen damit nutzbaren
HEVC-Decoder, passend zu PicFetchs Architektur. libheif allein reicht ohne
HEVC-Decoder nicht aus. Pakete in der Sandbox einer anderen Anwendung sind
möglicherweise für PicFetch nicht zugänglich.

<!-- debian13 -->
### Debian 13 (trixie), amd64 oder arm64

Wenn die Standard-Paketquellen von Debian eingerichtet sind, aktualisieren
Sie zunächst die Paketinformationen:

`sudo apt update`

Installieren Sie dann die Systembibliothek und ihr HEVC-Decoder-Plugin:

`sudo apt install libheif1 libheif-plugin-libde265`

[Debians Paket- und Architekturinformationen](https://packages.debian.org/trixie/libheif-plugin-libde265)

<!-- ubuntu2404 -->
### Ubuntu 24.04 LTS (noble), amd64 oder arm64

Das Decoder-Plugin liegt in Ubuntus Paketquelle **universe**. Wenn Ihre
Administration diese deaktiviert hat, folgen Sie zunächst Ubuntus
Dokumentation zu Paketquellen. Aktualisieren Sie die Paketinformationen:

`sudo apt update`

Installieren Sie dann die Systembibliothek und ihr HEVC-Decoder-Plugin:

`sudo apt install libheif1 libheif-plugin-libde265`

- [Ubuntus Paket- und Architekturinformationen](https://packages.ubuntu.com/noble/libheif-plugin-libde265)
- [Ubuntus Paketverwaltung](https://ubuntu.com/server/docs/how-to/software/package-management/)

<!-- arch -->
### Arch Linux, Rolling Release, x86_64

Das libheif-Paket in der offiziellen Paketquelle Extra benötigt libde265
zum Decodieren von HEVC. Bei eingerichteten Standard-Paketquellen:

`sudo pacman -Syu libheif`

Dies führt neben der Installation von libheif eine vollständige
Systemaktualisierung durch. Lesen Sie zuvor Archs aktuelle Update-Hinweise;
vermeiden Sie Teilaktualisierungen.

- [Archs Paket- und Abhängigkeitsinformationen](https://archlinux.org/packages/extra/x86_64/libheif/)
- [Archs Paketverwaltung](https://wiki.archlinux.org/title/Pacman)

<!-- fedora -->
### Fedora

Für diese Version und Architektur wurde hier kein Installationsbefehl geprüft.
Prüfen Sie Fedoras aktuelle libheif-Pakete und ob für Ihre Edition ein
HEVC-Decoder für die Systembibliothek verfügbar ist.

RPM Fusion ist eine **optionale externe Paketquelle**, getrennt von Fedoras
offiziellen Paketquellen. Sie entscheiden, ob Sie diese aktivieren. Lesen Sie
vor Änderungen die aktuellen Paket- und Versionshinweise des Anbieters.

- [Fedoras libheif-Pakete](https://packages.fedoraproject.org/pkgs/libheif/libheif/)
- [Fedoras Hinweise zu RPM Fusion](https://docs.fedoraproject.org/en-US/quick-docs/rpmfusion-setup/)
- [RPM Fusions Multimedia-Hinweise](https://rpmfusion.org/Howto/Multimedia)

<!-- generic -->
### Andere Distribution, Version oder Architektur

Für diese Kombination wurde hier kein Installationsbefehl geprüft. Fragen
Sie die Paketdokumentation Ihrer Distribution oder Ihre Administration nach
der systemweiten libheif-Bibliothek und einem passenden HEVC-Decoder-Plugin
für Ihre Architektur. Übernehmen Sie keine Befehle für andere Distributionen
oder Versionen.

[libheifs Anforderungen an Codecs und Plugins](https://github.com/strukturag/libheif)

<!-- unknown -->
## Systemanforderungen

Für dieses Betriebssystem wurde hier kein Installationsbefehl geprüft.
Lesen Sie die Dokumentation Ihres Systemanbieters zur HEIC-Unterstützung.
Wenn eine andere Anwendung HEIC öffnet, bedeutet das nicht, dass dieser
PicFetch-Build das System unterstützt.

<!-- after -->
## Nach der Installation

Gehen Sie zu Einstellungen -> Allgemein und wählen Sie **HEIC-Unterstützung
prüfen**. Sie können erneut prüfen, ohne PicFetch zu schließen. Erkennt das
System die neue Komponente noch nicht, starten Sie PicFetch neu und prüfen
Sie erneut. Systemupdates können auch einen Neustart des Computers erfordern.

Andere Anwendungen können eigene private Codecs enthalten. Wenn sich eine
Datei dort öffnen lässt, bestätigt das nicht, dass PicFetchs Systemdecoder
sie öffnen kann.
