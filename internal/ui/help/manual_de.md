# PicFetch — Benutzerhandbuch

Version 0.1.3

PicFetch ist ein kleiner, schneller Bildbetrachter für macOS, Windows und
Linux. Es gibt keine Werkzeugleiste und keinen eingebauten Datei-Browser: Sie
ziehen Bilder auf das Fenster und sehen sie sich an. Das ist die ganze Idee —
wenn Sie aber nicht per Drag & Drop arbeiten möchten, öffnet ein Klick auf das
Fenster oder `Cmd`/`Strg+O` stattdessen den Dateiauswahldialog Ihres Systems
(siehe unten).

---

![Trane mit Bilderrahmen](TaneWithFrame.webp)

## 1. Erste Schritte

1. Starten Sie **PicFetch**. Ein kleines, leeres Fenster (etwa 520 × 340)
   erscheint mit einem abgerundeten Rahmen und dem Text **„Bilder hier
   ablegen“**.
2. Ziehen Sie ein oder mehrere Bilddateien aus Ihrem Dateimanager (Finder,
   Explorer, Nautilus, …) auf das Fenster und lassen Sie los.
3. Das erste Bild wird angezeigt, und das Fenster passt seine Größe
   automatisch daran an.

Sie können die Dateien an beliebiger Stelle auf dem Fenster ablegen — der
umrandete Bereich ist nur ein visueller Hinweis, kein Ziel, das Sie genau
treffen müssen.

**Keine Maus für Drag & Drop zur Hand, oder Sie bevorzugen einen
Dateiauswahldialog?** Klicken Sie irgendwo in den Ablegebereich, oder drücken
Sie `Cmd`/`Strg+O` (`Cmd`/`Strg+Shift+O` macht dasselbe — es ist eine zweite
Tastenkombination für denselben Dialog, kein eigener), um den Datei-Browser
Ihres Systems zu öffnen. Unter macOS und Linux können Sie damit eine beliebige
Mischung aus Dateien und Ordnern auf einmal auswählen, genau wie per Drag &
Drop; Ordner werden auf dieselbe Weise eingelesen (siehe „Ordner einlesen“
unten). **Unter Windows bietet der Dialog nur Dateien an** — der
Windows-eigene Dateidialog hat keinen Modus, der Ordner- und Mehrfachauswahl
kombiniert —, Ordner fügen Sie dort per Drag & Drop hinzu. Das funktioniert
jederzeit, nicht nur auf dem leeren Ablegebildschirm — weitere Bilder zu
öffnen, während bereits eines angezeigt wird, ersetzt die aktuelle Auswahl
oder ergänzt sie, wenn der Zusammenführen-Modus (`M`) aktiv ist, genau wie
beim Ablegen neuer Dateien.

---

## 2. Dieses Handbuch öffnen

Sie lesen es gerade, haben es also vermutlich schon gefunden, aber der
Vollständigkeit halber:

- Drücken Sie jederzeit **`F1`**, oder
- wählen Sie **Hilfe -> Handbuch** aus dem Menü.

Das Handbuch öffnet sich in einem eigenen, scrollbaren Fenster. Oben bleibt
ein Suchfeld stehen: geben Sie einen Begriff ein und drücken Sie **Enter**,
um Treffer hervorzuheben und den ersten in den Blick zu scrollen. **Enter**
mit demselben Begriff springt zum nächsten Treffer. Drücken Sie
**`Esc`** oder schließen Sie das Fenster, um es zu verlassen; das Bild, das
Sie betrachtet haben, bleibt unverändert. Ein erneuter Druck auf `F1` holt das
bereits geöffnete Handbuchfenster nach vorne, statt eine zweite Kopie zu
öffnen.

---

## 3. Unterstützte Dateiformate

- **JPEG** — `.jpg`, `.jpeg`, `.jpe`, `.jfif` (EXIF-Rotation wird angewendet)
- **PNG** — `.png` (Transparenz wird unterstützt)
- **GIF** — `.gif` (animierte GIFs werden abgespielt)
- **WebP** — `.webp` (statische Bilder)
- **BMP** — `.bmp`
- **TIFF** — `.tif`, `.tiff`
- **ICO** — `.ico` (Windows-Symbol; das größte enthaltene Bild wird angezeigt)
- **XPM** — `.xpm` (X Pixmap)
- **HEIC/HEIF** — `.heic`, `.heif` (iPhone-Fotos; EXIF-Rotation wird
  angewendet)
- **AVIF** — `.avif` (eingebaute Rotation/Spiegelung wird angewendet)
- **SVG** — `.svg` (Vektorgrafik; kleine Symbole werden groß genug geöffnet,
  um das Fenster zu füllen, und das Bild wird bei jeder Zoomstufe scharf neu
  gerendert, statt hochskaliert zu werden). Das Neu-Rendern nutzt die Pixel
  des Bildschirms (einschließlich Retina), nicht nur die Fenstergröße in
  Punkten.
- **RAW** — `.cr2`, `.cr3`, `.nef`, `.nrw`, `.arw`, `.dng`, `.orf`, `.rw2`,
  `.raf`, `.pef`, `.srw`, `.raw` (es wird die vom Fotoapparat eingebettete
  JPEG-Vorschau gezeigt, in Titelzeile und Infokarte als `(Vorschau)`
  gekennzeichnet; es gibt kein Demosaic, und Datei -> Änderungen speichern
  bleibt aus)

Eine Datei wird auch akzeptiert, wenn Ihr System sie als `image/jpeg`,
`image/png`, `image/gif`, `image/webp`, `image/bmp`, `image/tiff`,
`image/x-icon`, `image/vnd.microsoft.icon`, `image/x-xpixmap`, `image/heic`,
`image/heif`, `image/avif`, `image/svg+xml` oder einen Kamera-RAW-MIME-Typ
wie `image/x-adobe-dng` / `image/x-canon-cr2` meldet, auch wenn die
Dateiendung fehlt oder ungewöhnlich ist.

Alles andere — PDFs, Videos — wird **nicht** unterstützt.

---

## 4. Bilder betrachten

### Automatische Fenstergröße

Jedes Mal, wenn ein Bild angezeigt wird, passt sich die Fenstergröße daran
an:

- **Große Bilder** werden so verkleinert, dass sie in **1500 × 950** Pixel
  passen, das Seitenverhältnis bleibt erhalten.
- **Kleine Bilder** verkleinern das Fenster nie unter die Startgröße
  (520 × 340). Ein winziges Vorschaubild wird zentriert, mit leerem Platz
  drumherum, statt ein Fenster zu erzeugen, das zu klein zum Anfassen wäre.
- Sie können das Fenster jederzeit selbst durch Ziehen am Rand vergrößern
  oder verkleinern. Das Bild wird passend skaliert und nie beschnitten oder
  verzerrt.

### Die Fenstertitelzeile

Die Titelzeile zeigt, was Sie gerade betrachten, zum Beispiel:

`sunset.jpg — 4032 x 3024  (2/7)`

- **Dateiname** des aktuellen Bildes
- **Pixelabmessungen** des Bildes (nach etwaiger Rotationskorrektur)
- **`(animated)`**, wenn es sich um ein animiertes GIF handelt
- **`(Vorschau)`**, wenn es sich um eine Kamera-RAW-Datei handelt, die über
  ihr eingebettetes JPEG gezeigt wird
- **`(2/7)`** — Position in der aktuellen Auswahl, wird nur angezeigt, wenn
  Sie mehr als ein Bild abgelegt haben
- **`[Zusammenführen]`** ganz vorne, nur solange der Zusammenführen-Modus
  (`M`) aktiv ist — siehe „Mehrere Bilder durchblättern“ unten

Im Raster steht normalerweise der Dateiname in der Titelzeile; in der
Variantenansicht gilt `(n/m) [BxH] /Pfad` ohne Modus-Präfixe.

### Fotorotation

Fotos, die im Hoch- oder Querformat mit einem Handy oder einer Kamera
aufgenommen wurden, tragen ein EXIF-Orientierungs-Tag. PicFetch liest
dieses Tag und dreht oder spiegelt das Foto automatisch, sodass
Hochformataufnahmen aufrecht statt seitlich liegend erscheinen. Alle acht
EXIF-Orientierungen werden unterstützt, und die Abmessungen in der Titelzeile
spiegeln das korrigierte Bild wider.

Wenn ein Foto trotzdem nicht so ausgerichtet ist, wie Sie es möchten —
EXIF-korrigiert, aber aus einem anderen Grund seitlich, oder Sie möchten es
einfach gedreht betrachten —, drücken Sie **`R`**, um es um weitere 90° im
Uhrzeigersinn zu drehen, oder **`Shift+R`** für gegen den Uhrzeigersinn. Das
ist zunächst nur eine Ansichtsoption: Sie wird mit **`0`** (zusammen mit dem
Zoom) wieder auf aufrecht zurückgesetzt und beginnt — wie beim Wechsel zum
nächsten Bild — beim nächsten betrachteten Foto wieder aufrecht. Um sie zu
behalten, schreiben Sie sie mit **Datei -> Änderungen speichern**
(`Cmd/Strg+S`) in die Datei zurück oder legen mit **Datei -> Bild
exportieren** (`Cmd/Strg+E`) eine gedrehte Kopie in einem Format Ihrer Wahl
an; siehe „Menü“ unten. Beim Drehen wechselt das Fenster zwischen Quer- und
Hochformat, um sich der neuen Ausrichtung des Bildes anzupassen, genau wie
beim Laden eines anderen Bildes.

### Animierte GIFs

Animierte GIFs beginnen zu spielen, sobald sie angezeigt werden:

- Die Einzelbilder laufen mit der in der Datei gespeicherten Geschwindigkeit.
- Einzelbilder, die nur einen Teil des Bildes aktualisieren, werden korrekt
  zusammengesetzt, sodass Sie keine Bildreste oder flackernde leere Stellen
  sehen.
- Einzelbilder ohne Verzögerung (oder mit einer Verzögerung von null) werden
  0,1 s lang angezeigt, damit die Wiedergabe flüssig bleibt.
- Die Animation läuft in einer Schleife, bis Sie weiterblättern oder neue
  Dateien ablegen — dann stoppt sie von selbst.

---

## 5. Zoom und Verschieben

Standardmäßig wird ein Bild immer **fensterfüllend eingepasst**, wie oben
beschrieben. Vier Tasten wechseln zu einer manuellen Zoomstufe:

- **`+`** — vergrößern
- **`-`** — verkleinern
- **`1`** — direkt auf **100 %** springen (ein Bildpixel pro Bildschirmpixel)
- **`0`** — zurück zur Fenstereinpassung

Der erste Druck auf `+` oder `-` zoomt von der aktuell eingepassten Ansicht
aus, statt zuerst auf 100 % zu springen, damit sich das Zoomen fließend
anfühlt. Wiederholtes Drücken skaliert weiter hoch oder herunter, begrenzt
auf 5 % bis 1600 %. Das Fenster wächst und schrumpft mit dem Bild: nie
kleiner als die Größe, mit der PicFetch startet, und nie größer als die
maximale Fensterbreite und -höhe unter **Datei -> Einstellungen…**. `0`
stellt sowohl die Einpassung als auch diese Standard-Fenstergröße wieder her.

**Scrollen** mit Mausrad oder Trackpad über dem Bild zoomt ebenfalls, und
anders als die Tastenkürzel zoomt es um den Punkt unter dem Mauszeiger statt
um die Bildmitte, sodass das, worauf Sie zeigen, beim Zoomen an derselben
Stelle auf dem Bildschirm bleibt.

Sobald das Bild so weit hineingezoomt ist, dass es nicht mehr ins Fenster
passt (das Fenster hat bereits seine Maximalgröße), wird der Mauszeiger zu
einer Hand, um anzuzeigen, dass es verschoben werden kann; **klicken und
ziehen** Sie, um es zu verschieben — die Bewegung
ist begrenzt, sodass Sie das Bild nicht so weit ziehen können, dass leerer
Platz dahinter entsteht. Solange das Bild eingepasst ist oder eine Zoomstufe
hat, bei der es noch ins Fenster passt, bewirkt Verschieben nichts, und der
Mauszeiger bleibt der normale Pfeil.

Wenn Sie **Shift** beim Scrollen gedrückt halten, wird verschoben statt
gezoomt, in welche Richtung auch immer Sie scrollen — praktisch für eine
Zweifinger-Wischgeste auf dem Trackpad, sobald Sie hineingezoomt haben, ohne
klicken und ziehen zu müssen.

Zoom und Verschiebung gelten pro Bild: Der Wechsel zu einem anderen Bild (mit
den Pfeiltasten, `Home`/`End` oder einem neuen Ablegen) setzt dieses Bild
immer wieder auf Fenstereinpassung zurück. Die Ansichtsrotation (siehe
„Fotorotation“ oben) wird auf dieselbe Weise zurückgesetzt, und das Drehen
setzt auch den Zoom auf die Einpassung zurück — eine vor der Drehung gewählte
manuelle Zoomstufe ergibt selten noch Sinn, sobald die Hoch-/Querformatachsen
des Bildes vertauscht wurden.

---

## 6. Info-Overlay

Drücken Sie **`I`**, um in der oberen linken Ecke des Fensters eine kleine
Karte mit allen Informationen zum aktuellen Bild auf einen Blick
einzublenden:

- den **Dateinamen** und seine Position in der Auswahl (z. B. `3 / 47`), wenn
  Sie mehr als ein Bild abgelegt haben
- seine **Pixelabmessungen** (z. B. `1920 x 1080`)
- seine **Dateigröße** auf der Festplatte
- die aktuelle **Zoomstufe** in Prozent

Es wird live aktualisiert, während Sie navigieren oder den Zoom ändern, und
bleibt — anders als eine Toast-Meldung — so lange sichtbar, bis Sie `I`
erneut drücken, um es auszublenden. Es ist eine dauerhafte Einstellung wie
die Sortierreihenfolge oder der Zusammenführen-Modus: Einmal eingeschaltet,
bleibt es über Navigation und weitere Ablagen hinweg an und erscheint wieder,
sobald das nächste Bild geladen wird, selbst wenn Sie zwischendurch kurz zum
leeren Ablegebildschirm zurückkehren.

Unter dieser Übersicht öffnet ein Link **„EXIF-Daten anzeigen“** ein
separates Fenster mit den Exif-Metadaten des aktuellen Bildes —
Kamerahersteller und -modell, Objektiv, Belichtungszeit, Blende, ISO,
Brennweite und Aufnahmedatum sowie — bei einem Foto mit GPS-Tags —
**Breitengrad** und **Längengrad** in Dezimalgrad, eine Zeile pro Tag, das
tatsächlich in der
Datei vorhanden ist. `E` öffnet dasselbe Fenster direkt, ohne dass das
Info-Overlay vorher geöffnet sein muss. Das Fenster aktualisiert sich, wenn Sie bei
geöffnetem Fenster zu einem anderen Bild wechseln (aus dem Bildfenster, oder mit
`Left`/`Right`, solange das EXIF-Fenster selbst den Fokus hat), und — wie beim Handbuch-
und Info-Fenster — schließt `Esc` nur dieses Fenster, und ein erneuter Druck
auf `E`, während es bereits offen ist, holt es nach vorne, statt eine zweite
Kopie zu öffnen. Dateien ohne Exif-Daten (die meisten PNGs, GIFs und WebPs
sowie jedes JPEG ohne von einer Kamera geschriebenes Exif-Segment) zeigen
stattdessen die Meldung „keine Metadaten gefunden“.

Unterhalb der Tag-Liste erscheint bei JPEGs die Schaltfläche
**Metadaten entfernen** (oberhalb der Karte, falls eine da ist). Sie fragt
nach (**Abbrechen** vorausgewählt; **`Left`**/**`Right`** und **`Return`** /
**`Esc`**, dieselben Tastatur-Regeln wie bei anderen PicFetch-Bestätigungen).
Sie schreibt die **ursprüngliche JPEG-Datei** direkt um. Die Schaltfläche
fehlt, sobald nichts mehr zu entfernen ist — auch nach einem erfolgreichen
Strip — und immer dann, wenn das Fenster „keine Metadaten gefunden“ anzeigt.
Identifizierende Bytes, die die Liste nicht zeigt (Kommentare, XMP, IPTC,
ein zweites Bild nach dem Bild), fallen beim Strip trotzdem weg, wenn die
Datei Tags *listet*. Die Schaltfläche selbst
ist eine kompakte Steuerung, keine durchgehende Leiste.

- Nur JPEG. Bei HEIC, RAW, PNG und WebP fehlt die Schaltfläche.
- Entfernt Kamera, Datum, GPS, XMP, IPTC und Kommentare. Farbprofil (ICC)
  und die eigene Farbtransformation des JPEG bleiben, das Bild sollte also
  gleich aussehen.
- Ein seitlich aufgenommenes Foto (Exif-Orientierung 2–8) wird einmal neu
  gespeichert, damit es ohne Orientierungs-Tag aufrecht bleibt; das ist eine
  normale JPEG-Neukodierung (Qualität 95), kein verlustfreier Kopiervorgang.
  Das ursprüngliche ICC-Profil wird auf diese Neukodierung übernommen.
- Ein bereits aufrechtes Foto wird ohne Neukodierung der Pixel bereinigt.
- Nur-Ansicht-Drehung (`R`) wird nicht mitgeschrieben; nutzen Sie zuerst
  **Datei -> Änderungen speichern**, wenn diese Drehung auf die Platte
  soll.
- Ein zweites JPEG oder Motion-Photo-Video hinter dem Hauptbild wird
  verworfen, damit auch Tags in dieser Kopie verschwinden. Das Standbild
  bleibt; das Extra-Bild oder Video nicht.
- Nicht rückgängig zu machen außer über Backups / Papierkorb — kein
  Verschieben in den Papierkorb.

Unterhalb der Tag-Liste erhält ein Foto mit GPS-Koordinaten einen
ausklappbaren Bereich **„Ort“**: aufgeklappt zeigt er eine Karte, die auf
die Aufnahmestelle zentriert ist und sie mit einer Nadel markiert. Er ist
bei jedem Öffnen des Fensters zunächst eingeklappt, und erst im
aufgeklappten Zustand lädt PicFetch Kartenkacheln — das Öffnen des
EXIF-Fensters allein schickt den Aufnahmeort also nie ins Netz.

Beim ersten Aufklappen erscheint **„Karte wird geladen…“**, während die
Kacheln rund um den Aufnahmeort geladen werden; die Karte wird vollständig
angezeigt, sobald sie da sind, und das Fenster bleibt die ganze Zeit
bedienbar. Verschieben und Zoomen über den geladenen Bereich hinaus füllt
sich nach und nach auf, ebenfalls ohne zu blockieren. Die Karte nimmt die
Höhe ein, die das Fenster ihr lässt — ziehen Sie das EXIF-Fenster größer,
wird auch die Karte größer. Die Karte besteht aus
Kacheln von [OpenStreetMap](https://openstreetmap.org)
(© OpenStreetMap-Mitwirkende); für die große Mehrheit der Dateien, die
überhaupt keine GPS-Tags tragen, fehlt der Bereich vollständig.

---

## 7. Mehrere Bilder durchblättern

Legen Sie mehrere Dateien auf einmal ab und blättern Sie mit der Tastatur
durch:

- **`Right`** oder **`Down`** — nächstes Bild
- **`Left`** oder **`Up`** — vorheriges Bild
- **`Home`** — zum ersten Bild springen
- **`End`** — zum letzten Bild springen

Die Navigation **läuft im Kreis**: `Right` beim letzten Bild springt zurück zum
ersten, `Left` beim ersten Bild zum letzten.

Solange das **EXIF-Datenfenster** den Fokus hat, blättern `Left` und `Right` genau
so weiter (einschließlich im Kreis). `Esc` schließt weiterhin nur dieses
Fenster. Während **Metadaten entfernen** nachfragt, bewegen `Left`/`Right` die
Bestätigungswahl, nicht das Bild.

Hinweise:

- Die Pfeiltasten blättern durch die aktuelle Auswahl und laufen im Kreis.
  Wenn Sie **eine Bilddatei** geöffnet haben (per Ablegen, Dateidialog oder
  `picfetch photo.jpg`), lädt PicFetch auch die anderen Bilder im Ordner
  dieser Datei — keine Unterordner — und bleibt auf der geöffneten Datei
  stehen, sodass Links/Rechts zu ihren Nachbarn führen. Zwei oder mehr
  Dateien zu öffnen behält genau diese Teilmenge. Einen Ordner zu öffnen
  durchsucht ihn weiterhin rekursiv (siehe „Ordner einlesen“). Ein Ordner,
  der nur dieses eine Bild enthält, hat weiterhin nichts zum Blättern.
  Solange Duplikate ausgeblendet sind (`D`, siehe „Rasteransicht“),
  überspringen die Pfeiltasten die versteckten Extra-Kopien und laufen
  im Kreis durch die übrigen Dateien; `Home`/`End` springen zur ersten und
  letzten übrigen Datei.
- Standardmäßig ist die Auswahl **natürlich sortiert** nach Dateiname, sodass
  `IMG_2.jpg` vor `IMG_10.jpg` kommt, obwohl eine reine Textsortierung sie in
  der anderen Reihenfolge anordnen würde. Drücken Sie **`S`**, um durch vier
  weitere Sortierungen und zurück zum Namen zu wechseln:
  - **Aufnahmedatum** — das Exif-Datum/-Zeit des Fotos (derselbe Wert, den
    das Exif-Fenster als „Aufnahmedatum“ zeigt); eine Datei ohne
    Exif-Aufnahmedatum — ein Screenshot, die meisten PNGs/GIFs/WebPs, oder
    ein JPEG, das eine Kamera nie getaggt hat — greift stattdessen auf den
    Änderungszeitpunkt im Dateisystem zurück, statt sich ganz am Anfang der
    Liste zu häufen.
  - **Geändert** — Änderungszeitpunkt im Dateisystem.
  - **Größe** — Dateigröße, kleinste zuerst.
  - **Unsortiert** — die Reihenfolge, in der Ihr Dateimanager die Dateien
    roh übergeben hat („dumme Sortierung“ — gar keine Sortierung).

  Die Titelzeile zeigt an, welcher Modus aktiv ist (`[Sortierung: Datum]`,
  `[Sortierung: Geändert]`, `[Sortierung: Größe]`, `[unsortiert]`) — für die
  Standard-Namenssortierung wird nichts angezeigt. Das Bild, das Sie gerade
  betrachten, bleibt bei jedem Wechsel auf dem Bildschirm. Die Sortierung
  entfernt nie doppelte Dateien, und die Einstellung bleibt bis zum nächsten
  Ablegen und über Neustarts hinweg erhalten, bis Sie sie wieder ändern.

  Aufnahmedatum, Änderungszeitpunkt und Größe müssen jeweils jede Datei
  einmal lesen, um danach zu sortieren — ein Stat-Aufruf für
  Änderungszeitpunkt/Größe, ein Rohdatei-Lesevorgang für das Aufnahmedatum —,
  was bei einem sehr großen rekursiven Ablegen spürbar pausieren kann, ohne
  Fortschrittsanzeige oder Möglichkeit, es nach dem Start abzubrechen.
- Neue Dateien abzulegen **ersetzt** die aktuelle Auswahl und beginnt wieder
  beim ersten gerade abgelegten Bild, sofern der **Zusammenführen-Modus**
  nicht aktiv ist. Drücken Sie **`M`**, um den Zusammenführen-Modus ein- oder
  auszuschalten; solange er aktiv ist, beginnt die Titelzeile mit
  **`[Zusammenführen]`**, sodass Sie immer erkennen, in welchem Modus Sie
  sich befinden. Mit aktivem Zusammenführen-Modus **ergänzt** ein neues
  Ablegen seine Dateien zur aktuellen Auswahl, statt sie zu ersetzen — die
  Anzeige springt zur ersten gerade hinzugefügten Datei, die Sortierung gilt
  weiterhin, und nichts wird dedupliziert, sodass das zweimalige Ablegen
  derselben Datei sie auch zweimal hinzufügt. Enthält ein Ablegen im
  Zusammenführen-Modus nichts Unterstütztes, bleibt die bestehende Auswahl
  unverändert, und Sie erhalten nur eine Fehler-Toast-Meldung, keine
  Löschung. Der Zusammenführen-Modus ist eine dauerhafte Einstellung wie die
  Sortierreihenfolge — er bleibt über mehrere Ablagen hinweg ein- (oder
  aus-)geschaltet, bis Sie `M` erneut drücken, sodass Sie beim Ziehen nichts
  gedrückt halten müssen.

---

![Trane beim Graben](trane_digging.webp)

## 8. Rasteransicht

Drücken Sie **`G`**, um zu einem fensterfüllenden Raster von
Miniaturansichten der aktuellen Auswahl zu wechseln — praktisch, um bei
einem großen Ablegen ein bestimmtes Bild visuell zu finden, statt sich
einzeln durchzublättern.

- Klicken Sie auf eine Miniaturansicht, um direkt zu ihr zu springen und zur
  normalen Ansicht zurückzukehren, oder nutzen Sie die Tastatur: Die
  Pfeiltasten bewegen einen hervorgehobenen Rahmen durch das Raster
  (beginnend bei dem Bild, das beim Öffnen gerade angezeigt wurde),
  **`Page Up`**/**`Page Down`** verschieben ihn gleich um eine ganze
  Bildschirmseite — praktisch, um in einem großen Stapel schnell viel
  Strecke zurückzulegen, wobei am ersten bzw. letzten Bild Schluss ist —
  und **`Return`** öffnet die gerade hervorgehobene Miniaturansicht.
- Drücken Sie **`G`** erneut, oder **`Esc`**, um das Raster ohne Auswahl zu
  verlassen. Das Schließen des Rasters schaltet das Ausblenden von
  Duplikaten **nicht** aus (siehe `D` unten).
- Drücken Sie **`/`**, um nach Dateinamen zu suchen: Oben erscheint eine
  Leiste, und Ihre Eingabe filtert das Raster fortlaufend auf die Namen,
  die sie enthalten. Groß- und Kleinschreibung spielt dabei keine Rolle,
  und die Leiste zeigt, wie viel der Auswahl übrig ist (`3 von 847`). Die
  Rücktaste löscht ein Zeichen, Pfeiltasten, `Page Up`/`Page Down` und
  `Return` wirken auf die Treffer genau wie sonst auf das ganze Raster,
  und **`Esc`** setzt die Suche zurück, sodass wieder alle übrigen Bilder
  erscheinen.
- Drücken Sie **`D`**, um Extra-Kopien derselben Aufnahme auszublenden. Die
  volle Dateiliste bleibt geladen; Einzelstücke bleiben immer sichtbar, und
  jede Gruppe ähnlicher Bilder behält einen Vertreter (die Datei mit der höchsten Auflösung: die meisten Pixel nach EXIF-Ausrichtung; bei gleicher Größe die früheste Datei in der aktuellen Reihenfolge). Die übrigen Mitglieder der Gruppe
  verschwinden aus dem Raster, und verbleibende Zellen, die für zwei oder
  mehr Dateien stehen, zeigen ein kleines Zähler-Abzeichen. `D` wechselt
  **nicht** in eine Galerie nur aus Duplikaten. Drücken Sie **`D`** erneut,
  um jede Miniaturansicht wieder zu zeigen. Wie ähnlich zwei
  Miniaturansichten sein müssen, um als dieselbe Aufnahme zu gelten, stellt
  der Schieberegler **Duplikat-Erkennungsabstand** unter **Datei ->
  Einstellungen…** ein (0–32, Vorgabe 6; niedriger ist strenger, 0 ist ein
  exakter Miniaturansicht-Hash). Zwei Dateien gelten nur dann als Kopien
  derselben Aufnahme, wenn jedes Paar in der Gruppe nahe genug ist — dem
  ersten Bild zu ähneln reicht nicht, wenn die anderen einander nicht
  ebenfalls ähneln. Eine Kette ähnlich aussehender Fotos wird nicht zu
  einer Riesengruppe zusammengefasst.
  Einfarbige Bilder (ohne Detail für den Abgleich) werden nicht als
  Duplikate gruppiert.
  Neu gespeicherte, neu exportierte und verkleinerte Kopien eines Bildes
  werden bei der Vorgabe zuverlässig erkannt; ein deutlich höherer Wert
  findet kaum weitere Kopien, zieht aber zunehmend wirklich verschiedene
  Bilder mit hinein. Zugeschnittene Fassungen erkennt der Abgleich bei
  keiner Einstellung — ein Zuschnitt verschiebt den gesamten Bildinhalt,
  und genau den vergleicht er.
  Eine Änderung am Schieberegler, während Extra-Kopien ausgeblendet sind,
  gruppiert das Raster sofort neu. `/`-Suche und Ausblenden stapeln sich:
  ein Namensfilter bei ausgeblendeten Extra-Kopien zeigt nur die übrigen
  Zellen, deren Namen passen. Solange eine Suche offen ist, ist `d`/`D` ein
  Buchstabe der Eingabe, nicht der Ausblend-Schalter.
- Drücken Sie **`Shift+D`**, um **Duplikate dieser Aufnahme anzuzeigen** —
  alle Kopien der **hervorgehobenen** Aufnahme (im Raster) bzw. der
  **aktuellen** Aufnahme (in der Einzelbildansicht). Das Raster listet nur
  diese Gruppe, einschließlich Extra-Kopien, die `D` ausblenden würde.
- In der Variantenansicht sind die Zähl-Badges ausgeblendet.
- Die Fenstertitelzeile zeigt die hervorgehobene Miniaturansicht als
  `(Position) [BreitexHöhe] vollständiger-Pfad`, z. B.
  `(2/7) [1440x780] /photos/vacation/IMG_0123.jpg`. `[Zusammenführen]`, Sortier- und
  `[Zufällig]`-Präfixe sind in der Variantenansicht ausgeblendet. Pfeiltasten
  und Zeigen mit der Maus bewegen die Hervorhebung und damit den Titel.
  Verlassen der Variantenansicht stellt den Dateinamen-Titel und jene Präfixe
  wieder her.
- Wenn Miniaturansichten noch gehasht werden, erscheint ein Info-Hinweis
  **Die Bilder werden gerade analysiert**; die Gruppe erscheint, sobald das
  Hashen abgeschlossen ist. Eine einzigartige Aufnahme (bereits gehasht,
  keine Kopien) bewirkt nichts.
- **`Esc`** beendet die Duplikat-Anzeige, bevor das Ausblenden
  ausgeschaltet wird. **`G`**/Schließen lassen das Ausblenden an,
  **beenden** aber die Duplikat-Anzeige.
- Eine Variante mit `Return` oder einem Klick zu öffnen zeigt **diese** Datei,
  auch wenn das Ausblenden sonst die Kopie mit der höchsten Auflösung behalten
  würde. Links/Rechts laufen dann nur durch die Gruppe; Home/Ende springen
  weiter zum ersten/letzten sichtbaren Bild der ganzen Menge, danach laufen
  Links/Rechts weiter durch die Gruppe. `Esc` oder `G`
  in dieser Ansicht öffnet wieder das Varianten-Raster; ein weiteres `Esc`
  kehrt zum Raster mit ausgeblendeten Extra-Kopien zurück. `D` und der
  Bilderrahmen-Modus tun nichts, solange Varianten angezeigt werden oder diese
  Schleife aktiv ist.
- Solange die `/`-Suche offen ist, ist **`Shift+D`** keine
  Duplikat-Anzeige (`D` ist ein Buchstabe).
- Im Diaschau-Modus bewirkt **`Shift+D`** nichts, wie **`G`**.
- **Mehrere auf einmal auswählen**, um sie gemeinsam zu bearbeiten:
  **`Cmd/Strg+Klick`** auf eine Miniaturansicht nimmt sie in die Auswahl auf
  (ein erneuter Klick nimmt sie wieder heraus), **`Shift+Klick`** wählt
  alles zwischen ihr und der zuletzt angeklickten aus, **`Leertaste`**
  wählt die gerade hervorgehobene Miniaturansicht, und **`Cmd/Strg+A`**
  wählt alle aus.
  Ziehen Sie ein Rechteck über die Miniaturansichten, um alles auszuwählen, das es berührt; halten Sie dabei Shift oder Cmd/Strg, um zur bestehenden Auswahl hinzuzufügen.
  Ausgewählte Miniaturansichten sind in der Akzentfarbe
  eingefärbt, und die obere Leiste zählt sie (`12 ausgewählt`). Ein
  Klick ohne Ziehen öffnet weiterhin nur ein Bild.
- Sind genau zwei Dateien ausgewählt, drücken Sie **`Cmd/Strg+D`** oder wählen
  **Aktionen -> Ausgewählte Bilder vergleichen**. Unter macOS ist `Cmd+D` die
  native Tastenkombination; die physische **`Ctrl+D`** funktioniert unter macOS ebenfalls.
  Im selben Fenster erscheint
  eine undurchsichtige Vergleichsansicht, in der beide Bilder in feste
  50/50-Bereiche eingepasst sind; die in der aktuellen Rasterreihenfolge
  frühere Datei steht links. Während des Ladens zeigt jede Seite einen eigenen
  Fortschrittsbalken. Durchscheinende Abzeichen in den unteren Ecken benennen
  die Dateien mit ihrem Basisnamen; sind diese gleich, werden beide zum
  kürzesten unterscheidbaren Ordner/Datei-Suffix erweitert. Der Fenstertitel
  folgt derselben Reihenfolge, zum Beispiel
  `Vergleich: links.jpg | rechts.jpg - PicFetch`. Eine separate
  durchscheinende Karte enthält die Schaltfläche **Entkoppeln** oben links;
  sie und die physische Tastenkombination **`Ctrl+L`** bleiben inaktiv, bis
  beide Bilder bereit sind. Oben rechts bleibt eine durchscheinende
  Aktionsleiste sichtbar: **Zurück zur Rasteransicht** ist
  auch beim Laden verfügbar, **Tauschen** wird aktiv, sobald beide Bilder
  bereit sind. Tauschen vertauscht Bilder, Abzeichen und Titel, ohne eine Datei
  erneut zu laden. **Wischen** legt beide Bilder über den vollständigen
  Vergleichsbereich und fügt eine senkrechte Trennlinie hinzu. Die Schaltfläche
  wird erst aktiv, sobald beide Bilder bereit sind. Ziehen Sie die Trennlinie,
  um die Aufteilung zu ändern; Ziehen an anderer Stelle verschiebt weiterhin
  beide Bilder. Solange Wischen aktiv ist, verschieben **`Left`** / **`Right`**
  die Trennlinie um 5 Prozentpunkte, **`Shift+Left`** / **`Shift+Right`** um 1
  Prozentpunkt und **`Home`** / **`End`** setzen sie auf 0 %/100 %.
  **Nebeneinander** kehrt zu festen 50/50-Bereichen zurück. Beim Wechsel der
  Anordnung bleiben Position und Größe beider Fotos, die Kamera und die
  Trennlinienposition erhalten; ein neuer Vergleich beginnt nebeneinander bei
  50 %. Im normal gekoppelten Zustand steuern Zoom und Verschieben eine
  gemeinsame Kamera über den beiden Fotos. Scrollen Sie über einem Bereich
  oder verwenden Sie **`+`** / **`-`**, um diese Kamera zu zoomen. Ziehen in
  einem Vergleichsbereich oder Shift+Scrollen bewegt beide Ansichten um
  dieselbe Bildschirmstrecke. Die Kamerabewegung stoppt, bevor ein Foto die
  Mitte seines Bereichs vollständig passiert. **`0`** rahmt beide Fotos in ihrer
  aktuellen Anordnung mit einer Kamerabewegung ein und behält ihre relativen
  Größen und Versätze bei. **`1`** setzt die Kamera auf ihre 1x-Ausgangsansicht relativ
  zur gespeicherten Anordnung zurück; nach getrenntem Skalieren zeigt
  es nicht beide Fotos mit 100 % der dekodierten Pixelgröße. Verwenden Sie die
  Schaltfläche **Entkoppeln** oben links oder drücken Sie die physische
  Tastenkombination **`Ctrl+L`** (`Ctrl`/`Strg`, auch unter macOS; nicht `Cmd`),
  um zwischen gekoppelten und entkoppelten Ansichten umzuschalten. Der erste
  Klick oder Tastendruck entkoppelt die Bereiche, bis eines der beiden
  Bedienelemente erneut verwendet wird; das Loslassen von Control hat keine
  Wirkung. Nach dem Entkoppeln wechselt die Schaltfläche zu **Koppeln**, und der
  Status **Entkoppelt** erscheint direkt daneben; nach der Auswahl eines
  Bereichs folgt zusätzlich **Links** oder **Rechts**. Ziehen, Scrollen oder Shift+Scrollen
  ändert dann nur den Bereich unter dem Mauszeiger; die unveränderten Tasten
  **`0`**, **`1`**, **`+`** und **`-`** ändern das Foto unter dem Mauszeiger
  beziehungsweise das zuletzt berührte Foto und bewirken nichts, solange noch
  kein Bereich ausgewählt wurde. Dabei passt **`0`** nur dieses Foto in die
  aktuelle Kameraansicht ein und zentriert es; **`1`** zeigt nur dieses Foto mit
  100 % der dekodierten Pixelgröße. Ein Foto lässt sich verschieben, bis einer
  seiner Ränder die Mitte des Bereichs erreicht. Das Umschalten der Kopplung
  bewegt oder skaliert keines der Fotos: Beim erneuten Drücken von **`Ctrl+L`**
  wird die aktuelle Anordnung gekoppelt, danach verändern gekoppelte Befehle nur
  noch die Kamera. Fenstergrößen- und Anordnungsänderungen bewahren die
  Fotoanordnung und Kamera. Tauschen koppelt und verwirft zuerst alle
  Unterschiede anhand des zuletzt berührten Bereichs und vertauscht dann die
  Bilder. Ein neuer Vergleich beginnt immer gekoppelt.
  Rasterquellen bleiben in der vollen dekodierten Auflösung und verwenden ihre
  kanonische EXIF-korrigierte Ausrichtung; eine vorübergehende Drehung in der
  Einzelbildansicht wird nicht in den Vergleich übernommen. SVGs werden für
  ihre effektive Bildschirm-Pixelgröße neu gerendert, sobald sich Zoom,
  Anordnung oder Fenstergröße ändern. RAW-Dateien verwenden dieselbe
  eingebettete JPEG-Vorschau wie die normale Einzelbildansicht. Animierte
  Eingaben bleiben auf ihrem ersten dekodierten Einzelbild eingefroren, solange
  der Vergleich geöffnet ist. Eine begrenzte Übersicht bleibt sichtbar, während schärfere
  Detailkacheln im Hintergrund eintreffen. Verschieben und Zoomen aktualisieren diese stabile
  GPU-Fläche direkt, sodass die Interaktion nicht auf die schärferen Kacheln wartet.
  Volle Wiedergabetreue kann den kombinierten
  dekodierten Speicher beider Quellen beanspruchen, auch wenn der Bild-Cache nur
  eine davon behält. Die vorhandenen Grenzen für kodierte Eingaben und
  Vektor-Raster gelten weiterhin. Kann eine Quelle nicht vollständig geladen
  werden, meldet PicFetch den Fehler und kehrt zum unveränderten Raster zurück;
  die Bereiche werden weder verkleinert noch entfernt, um eine Grenze
  einzuhalten. **Zurück zur
  Rasteransicht** oder **`Esc`** bringt Sie zum unveränderten Raster zurück,
  auch wenn ein Datei- oder Duplikatfilter ausgewählte Bilder gerade ausblendet.
- Der Vergleich ist ein exklusiver Hauptfenster-Modus. Bis Sie zur
  Rasteransicht zurückkehren, deaktiviert oder ignoriert PicFetch normale
  Befehle für Betrachter, Raster, Dateien, Favoriten und Aktionen; Tastatur-
  und Zeigereingaben erreichen die verdeckten Ansichten nicht. Die
  Vergleichs-Werkzeugleiste, **`Esc`**, die Hilfe mit **`F1`** und das normale
  Schließen des Fensters bleiben verfügbar. Versuche, Dateien über den
  Dateidialog, durch Ablegen oder über die Öffnen-mit-Übergabe des
  Betriebssystems zu öffnen, werden verworfen und zeigen
  **Kehren Sie zur Rasteransicht zurück, bevor Sie Dateien öffnen**.
- Mit einer getroffenen Auswahl verschiebt **`Shift+Delete`** alles davon in
  den Papierkorb (nach der üblichen Nachfrage, die die Anzahl nennt statt
  jeder einzelnen Datei), und **`Cmd/Strg+C`** kopiert die Dateien selbst in
  die Zwischenablage — Einfügen im Finder, Explorer oder Ihrem
  Dateimanager erzeugt Kopien davon. Ohne Auswahl wirken beide auf die
  hervorgehobene Miniaturansicht allein, und das Raster bleibt danach
  geöffnet, sodass Sie Ihre Position behalten.
- Weil die Auswahl eine Menge von *Dateien* ist, wählt `Cmd/Strg+A` nach
  einer Einschränkung mit `/` genau die Treffer aus — `/urlaub`,
  `Cmd/Strg+A`, `Shift+Delete` räumt jedes Urlaubsfoto aus einem Ordner mit
  Tausenden heraus. Das anschließende Zurücksetzen der Suche lässt die
  Auswahl unberührt.
- **`Esc`** nimmt jeweils eine Sache zurück: zuerst ein noch nicht
  beendetes Rechteckziehen, dann die Auswahl, dann die Suche, dann die
  Duplikat-Anzeige, dann das Ausblenden von Duplikaten, dann das Raster
  selbst. `G`
  schließt das Raster wie gewohnt, bleibt aber wirkungslos, solange eine
  Auswahl oder eine Suche besteht, damit es keine begonnene Arbeit
  verwirft. `G` und Schließen lassen das Ausblenden an: die Einzelbildansicht
  überspringt weiterhin versteckte Extra-Kopien, bis Sie erneut `D` drücken
  (oder `Esc` durch diese Stufe, solange das Raster offen ist).
- Davon abgesehen wird jede andere Taste ignoriert, solange das Raster
  geöffnet ist — Zoom, `S`/`M`/`P`/`I` bewirken nichts, bis Sie entweder eine
  Miniaturansicht auswählen (Klick oder `Return`) oder mit `G`/`Esc`
  zurückgehen. `D` und `Shift+D` sind die Ausnahmen (außer wenn eine Suche
  offen ist); `Cmd/Strg+D` öffnet außerdem den Vergleich, wenn genau zwei
  Dateien ausgewählt sind. Solange eine Suche geöffnet ist, sind die
  Buchstabentasten Zeichen Ihrer Eingabe — `G` schließt das Raster dann nicht mehr und
  `D`/`Shift+D` schalten weder Ausblenden noch Duplikat-Anzeige um,
  `Esc` schon.
- Die Suche schränkt nur ein, was das Raster zeigt. An der Auswahl selbst
  ändert sie nichts: Nach dem Öffnen eines Bildes blättern die Pfeiltasten
  weiterhin durch alle abgelegten Dateien, und beim nächsten Öffnen
  beginnt das Raster ungefiltert und ohne Auswahl. Das Ausblenden von
  Duplikaten ist anders: es bleibt nach dem Verlassen des Rasters an, und
  Pfeiltasten, `Home`/`End` und der Diaschau-Wechsel überspringen die
  versteckten Extra-Kopien, bis Sie es ausschalten.
- Miniaturansichten werden im Hintergrund erzeugt, sobald sie ins Blickfeld
  scrollen, jeweils einige auf einmal, sodass das Öffnen des Rasters bei
  einem Ordner mit Tausenden von Bildern das Fenster nicht blockiert,
  während alle im Voraus dekodiert werden.
- Das Raster benötigt mindestens ein geladenes Bild und lässt sich nicht mit
  dem Diaschau-Modus kombinieren — das Öffnen des einen schließt das andere.

### Bildmosaike

Wählen Sie bei geöffnetem Raster **Aktionen -> Bildmosaik erstellen...**. Wenn
Sie Miniaturansichten ausdrücklich ausgewählt haben, bilden nur diese Dateien
den Quellenvorrat; andernfalls verwendet PicFetch eine Momentaufnahme aller
Bilder im aktuellen gefilterten Rasterergebnis. Spätere Auswahl-, Filter-,
Navigations-, Umbenennungs- oder Löschvorgänge ändern das Ziel eines bereits
geöffneten Mosaikfensters nicht, und die Erstellung verändert niemals eine
Quelldatei.

Wählen Sie den Zielbildschirm anhand von Name, nativer Pixelauflösung und
Seitenverhältnis. **Bildschirme aktualisieren** aktualisiert diese Liste; wurde
der gewählte Bildschirm entfernt, verlangt PicFetch eine neue Auswahl.
**Erweitert** blendet minimale Bildgröße, Rahmen, Größenvariation,
Überlappung, maximale Drehung und die Schlagschattenoption ein; bei
eingeklappten erweiterten Einstellungen ist der Zielbildschirm die einzige
sichtbare Gestaltungseinstellung. Die Erstellung läuft im Hintergrund. `Esc`
bricht eine laufende Erstellung ab, ein zweites `Esc` schließt das Fenster;
ohne laufende Arbeit schließt `Esc` es direkt. Alle Bedienelemente und Aktionen
sind mit `Tab` und `Shift+Tab` erreichbar; `Enter` oder die Leertaste aktiviert
eine fokussierte Schaltfläche.

Nach der Erstellung kehren Sie mit **Neu beginnen** zur Konfiguration zurück,
verwerfen die aktuelle Vorschau und Statusmeldung und behalten Quellen,
ausgewählten Bildschirm, visuelle Einstellungen und Exportformat bei, damit
Sie einen anderen Bildschirm wählen und dessen Hintergrundbild erstellen
können. **Neu erstellen** behält Quellen, Bildschirm und Einstellungen bei,
erzeugt aber eine neue Anordnung. **Bild speichern** exportiert das exakte
Ergebnis in voller Auflösung als PNG oder JPEG. **Als Hintergrundbild
festlegen** verwendet eine dauerhafte PicFetch-eigene Kopie: Unter Windows und
macOS wird nur der gewählte Bildschirm geändert, während der gewöhnliche
Hintergrundbildbefehl im Hauptfenster global bzw. für alle Bildschirme gilt.
Die verfügbaren GNOME-/KDE-Integrationen unter Linux arbeiten nur global;
deshalb wird ein zielgerichtetes Mosaik vor jeder Änderung des Desktops
abgelehnt. **Bild speichern** bleibt verfügbar.

---

## 9. Diaschau-Modus

Drücken Sie **`P`**, um die aktuelle Bildauswahl in eine Vollbild-Diaschau zu
verwandeln — praktisch, um PicFetch einfach dastehen und durch einen
Ordner voller Fotos wie einen digitalen Bilderrahmen laufen zu lassen.

- Das Fenster wechselt in den **Vollbildmodus**. Das Bild wird
  bildschirmfüllend skaliert, das Seitenverhältnis bleibt erhalten — nie
  gestreckt oder beschnitten, dasselbe Einpassverhalten wie im normalen
  Fenster.
- Alle **10 Sekunden** (standardmäßig) wechselt die Ansicht **automatisch**
  zum nächsten Bild, am Ende beginnt es wieder von vorne, genau wie bei
  manueller Navigation. Jeder Wechsel wird **überblendet** — das
  ausscheidende Bild verblasst, das neue blendet ein — statt des sofortigen
  Wechsels beim normalen Durchblättern. Auch die manuelle Navigation
  (`Left`/`Right`/`Home`/`End`) wird während des Diaschau-Modus auf dieselbe Weise
  überblendet. Sind Duplikate ausgeblendet, überspringen automatischer
  Wechsel und diese Tasten versteckte Extra-Kopien genauso wie das normale
  Durchblättern.
- **`Up`** erhöht das Intervall um eine Sekunde, **`Down`** verringert es (bis
  zu einer Untergrenze von einer Sekunde). Solange der Diaschau-Modus aktiv
  ist, steuern `Up`/`Down` den Timer statt zu navigieren — nutzen Sie
  **`Left`**/**`Right`** (oder `Home`/`End`) zum manuellen Navigieren, das
  weiterhin wie gewohnt funktioniert und den Countdown ab dem neuen Bild neu
  startet.
- **`Shift+P`** schaltet **Zufällige Wiedergabe** ein oder aus: Ist sie
  aktiv, wählt der automatische Wechsel jedes Mal ein zufälliges anderes
  Bild statt des nächsten in der Reihenfolge (nie das gerade angezeigte),
  und die Titelzeile beginnt mit **`[Zufällig]`**. Die manuelle Navigation
  mit `Left`/`Right`/`Home`/`End` bleibt davon unberührt — sie durchläuft die
  Auswahl immer in Reihenfolge. Die Zufällige Wiedergabe verhält sich wie
  eine dauerhafte Einstellung, genau wie der Zusammenführen-Modus:
  `Shift+P` funktioniert schon, bevor Sie den Diaschau-Modus überhaupt
  einschalten, und auch außerhalb davon.
- **Animierte GIFs werden immer zu Ende abgespielt.** Wenn ein
  GIF-Durchlauf länger dauert als das aktuelle Intervall, wartet der
  Diaschau-Modus, bis er mindestens einmal komplett durchgelaufen ist, statt
  ihn mittendrin abzubrechen.
- Ihr gewähltes Intervall und die Einstellung der Zufälligen Wiedergabe
  werden beim nächsten Einschalten des Diaschau-Modus gemerkt — und bleiben
  auch beim nächsten Start von PicFetch erhalten.
- Drücken Sie **`P`** erneut, oder **`Esc`**, um den Diaschau-Modus zu
  verlassen und zum normalen Fenster zurückzukehren. `Esc` verlässt hier nur
  den Diaschau-Modus — es leert nicht auch die geladenen Bilder; drücken Sie
  es danach erneut, um das zu tun.
- Der Diaschau-Modus benötigt mindestens ein geladenes Bild — `P` auf dem
  leeren Ablegebildschirm bewirkt nichts.

---

## 10. Eine Datei löschen

Drücken Sie **`Shift+Delete`**, um die gerade angezeigte Datei in den
Papierkorb des Betriebssystems zu verschieben. Eine Bestätigungskarte
erscheint mit zwei Schaltflächen:

- **Abbrechen** — standardmäßig ausgewählt
- **In den Papierkorb verschieben** (in Rot)

Die gerade ausgewählte Schaltfläche ist umrandet, sodass immer sichtbar ist,
was `Return` auslösen wird, bevor Sie es drücken.

Sie können auf beide Arten antworten:

- **Mit der Maus**: Klicken Sie direkt auf eine der beiden Schaltflächen.
- **Mit der Tastatur**: Drücken Sie **`Right`**, um die Auswahl auf „In den
  Papierkorb verschieben“ zu verschieben (**`Left`** verschiebt sie zurück auf
  „Abbrechen“) —
  der Rahmen bewegt sich mit — und dann **`Return`**, um mit der jeweils
  ausgewählten Option fortzufahren. **`Esc`** bricht sofort ab, egal welche
  Option gerade ausgewählt ist.

Solange die Karte angezeigt wird, wird jede andere Taste ignoriert —
Navigation, Zoom, `S`/`M`/`P`/`I`/`G` bewirken nichts, bis Sie die Nachfrage
auf die eine oder andere Weise beantworten.

Das Löschen der aktuellen Datei entfernt sie aus der Auswahl und zeigt, was
jetzt an ihre Stelle tritt, wobei die Navigation im Kreis läuft, genau wie
sonst; war sie die letzte verbliebene Datei, gelangen Sie zurück zum leeren
Ablegebildschirm. Stellt sich heraus, dass die Datei bereits verschwunden
ist, oder kann sie aus einem anderen Grund nicht gelöscht werden (zum
Beispiel wegen Berechtigungen), erklärt eine Toast-Meldung, was schiefgegangen
ist, und die Datei bleibt in der Auswahl.

Wird `Shift+Delete` gedrückt, während die Rasteransicht angezeigt wird,
fragt es stattdessen nach allem, was dort ausgewählt ist (siehe oben). Die
Karte erscheint über dem Raster, und das Raster bleibt nach Ihrer Antwort
geöffnet. Was das System nicht verschieben kann, bleibt sowohl auf der
Festplatte als auch in der Auswahl, und die Toast-Meldung nennt, wie viele
tatsächlich verschoben wurden.

---

## 11. Tastenkürzel

- **`F1`** — dieses Handbuch öffnen
- **`Cmd`/`Strg+O`** / **`Cmd`/`Strg+Shift+O`** — den System-Dateidialog
  öffnen (dasselbe wie ein Klick in den Ablegebereich; beide
  Tastenkombinationen bewirken dasselbe; Dateien und Ordner unter
  macOS/Linux, nur Dateien unter Windows — siehe oben)
- **`Cmd`/`Strg+1`** bis **`Cmd`/`Strg+9`** — die sortierten Favoriten 1 bis
  9 öffnen; **`Cmd`/`Strg+0`** öffnet Favorit 10 (siehe „Menü“ unten)
- **`Cmd`/`Strg+Shift+F`** — „Favoriten verwalten…“ öffnen, vollständig über
  die Tastatur bedienbar (einschließlich der Rückfrage vor dem Entfernen):
  die Pfeiltasten bewegen einen Rahmen über die Favoriten und über deren
  Schaltflächen „Öffnen“/„Entfernen“, `Return` löst die gerade markierte
  Option aus, `Esc` schließt (siehe „Menü“ unten)
- **`Opt`/`Alt+Shift+F`** — „Aktuelle Liste zu Favoriten hinzufügen…“ öffnen
  (ausgegraut, und dieses Tastenkürzel tut nichts, wenn keine Dateien
  geladen sind; siehe „Menü“ unten)
- **`Right`** / **`Down`** — nächstes Bild
- **`Left`** / **`Up`** — vorheriges Bild
- **`Home`** / **`End`** — erstes / letztes Bild
- **`S`** — Sortierreihenfolge durchschalten: Name -> Aufnahmedatum ->
  Änderungszeitpunkt -> Größe -> unsortiert -> zurück zu Name
- **`M`** — Zusammenführen-Modus ein-/ausschalten (neues Ablegen ergänzt die
  Auswahl, statt sie zu ersetzen); wird in der Titelzeile mit dem Präfix
  **`[Zusammenführen]`** angezeigt
- **`G`** — Rasteransicht ein-/ausschalten (siehe oben); Pfeiltasten bewegen
  die Hervorhebung, `Page Up`/`Page Down` gleich um eine ganze Seite,
  `Return` oder ein Klick öffnet sie, `G`/`Esc` bricht ab. `G` schaltet das
  Ausblenden von Duplikaten nicht aus
- **`V`** — zur normalen Bildansicht zurückkehren (schließt die Rasteransicht
  oder verlässt den Bilderrahmen-Modus). Kein Ein-/Ausschalter. Solange eine
  Rastersuche offen ist, tippt die Taste den Buchstaben `v`
- **`D`** — Extra-Kopien derselben Aufnahme ausblenden (siehe
  „Rasteransicht“); verbleibende Zellen zeigen ein Zähler-Abzeichen.
  Solange eine Rastersuche offen ist, tippt die Taste den Buchstaben `d`
- **`Shift+D`** — Duplikate dieser Aufnahme anzeigen (siehe
  „Rasteransicht“); solange eine Rastersuche offen ist, tippt die Taste den
  Buchstaben `D`
- **`/`** — (nur im Raster) das Raster nach Dateinamen durchsuchen; stapelt
  sich mit dem Ausblenden von Duplikaten. `Esc` setzt die Suche zurück,
  dann die Duplikat-Anzeige, dann das Ausblenden, dann verlässt es das
  Raster
- **`Leertaste`** — (nur im Raster) die hervorgehobene Miniaturansicht zur
  Auswahl hinzufügen oder wieder herausnehmen
- **`Cmd`/`Strg+A`** — (nur im Raster) alle gerade angezeigten
  Miniaturansichten auswählen (bei aktiver Suche nur die Treffer)
- **`Cmd`/`Strg+Klick`** / **`Shift+Klick`** / **Klicken-und-Ziehen** — (nur im Raster) eine Miniaturansicht zur Auswahl hinzufügen / den Bereich auswählen / jede Miniaturansicht auswählen, die das Rechteck berührt (Shift- oder Cmd/Strg+Ziehen fügt hinzu statt zu ersetzen)
- **`+`** / **`-`** — vergrößern / verkleinern (das Fenster skaliert mit dem
  Bild, zwischen Startgröße und dem Maximum in den Einstellungen)
- **`1`** — auf 100 % zoomen; **`0`** — zurück zur Fenstereinpassung und zur
  Standard-Fenstergröße (und setzt die Rotation zurück, siehe unten)
- Scrollen (Mausrad oder Trackpad) — vergrößern/verkleinern, verankert am
  Mauszeiger
- **Shift** + Scrollen — verschieben statt zoomen
- Klicken und ziehen — ein hineingezoomtes Bild verschieben
- **`R`** / **`Shift+R`** — das angezeigte Bild um 90° im/gegen den
  Uhrzeigersinn drehen (nur Ansicht; wird bei `0` oder dem nächsten Bild
  zurückgesetzt)
- **`I`** — Info-Overlay ein-/ausschalten (Dateiname, Position, Abmessungen,
  Dateigröße, Zoomstufe)
- **`E`** — das EXIF-Datenfenster für das aktuelle Bild öffnen
  (Kamerahersteller/-modell, Objektiv, Belichtung, Blende, ISO, Brennweite,
  Aufnahmedatum, Koordinaten); auch über den Link **„EXIF-Daten anzeigen“** im
  Info-Overlay erreichbar. Solange dieses Fenster den Fokus hat, wechseln
  Links/Rechts das Bild.
- **`Cmd`/`Strg+E`** — das aktuelle Bild in eine neue Datei exportieren: eine
  Abfrage fragt nach dem Format (**`Left`**/**`Right`** wählt zwischen PNG und JPEG,
  **`Return`** exportiert, **`Esc`** bricht ab), danach benennen Sie die Datei
  im Speichern-Dialog des Systems (siehe „Menü“ unten). Das einfache `E` oben
  öffnet weiterhin das EXIF-Fenster — nur die Tastenkombination exportiert
- **`Cmd`/`Strg+Shift+E`** — das aktuelle Bild als Hintergrundbild des
  Schreibtischs festlegen (siehe „Menü“ unten)
- **`Cmd`/`Strg+C`** — das aktuelle Bild in die Systemzwischenablage
  kopieren, als Bilddaten, die Sie in eine andere App einfügen können (keine
  Datei). In der Rasteransicht werden stattdessen die ausgewählten *Dateien*
  kopiert, sodass ein Einfügen im Dateimanager Kopien davon erzeugt
- **`Opt`/`Alt+Shift+C`** — Auswahl-kopieren-Modus in der normalen
  Bildansicht. Ziehen Sie ein Rechteck auf dem Bild (es bleibt innerhalb des
  Bildes). Ziehen innerhalb des Rechtecks verschiebt es, Ziehen an einem
  Anfasser ändert die Größe. **In die Zwischenablage kopieren** (oder
  `Return`/`Enter`) kopiert diesen Bildbereich in der Auflösung des Bildes
  als PNG, ohne die Fensteroberfläche. `Esc` verlässt den Modus, ohne die
  Zwischenablage zu ändern. Zoom und Verschieben bleiben möglich.
  Nicht verfügbar, solange Raster oder Diaschau-Modus aktiv sind, ein Dialog
  das Fenster besitzt, oder kein dekodiertes Bild angezeigt wird
- **`Cmd`/`Strg+Shift+C`** — den Dateipfad des aktuellen Bildes in die
  Zwischenablage kopieren
- **`Cmd`/`Strg+R`** — die aktuelle Datei im Dateimanager anzeigen, in ihrem
  Ordner bereits ausgewählt (siehe „Menü“ unten). Ein einfaches `R` dreht
  weiterhin das Bild
- **`Shift+Delete`** — die aktuelle Datei nach Bestätigung in den Papierkorb
  verschieben (siehe „Eine Datei löschen“ oben); in der Rasteransicht alles
  dort Ausgewählte
- **`P`** — Diaschau-Modus ein-/ausschalten (Vollbild-Diaschau mit
  Überblendung zwischen den Bildern, siehe oben)
- **`Shift+P`** — Zufällige Wiedergabe für den automatischen Wechsel im
  Diaschau-Modus ein-/ausschalten; wird in der Titelzeile mit dem Präfix
  **`[Zufällig]`** angezeigt
- **`Up`** / **`Down`** *(im Diaschau-Modus)* — das Auto-Weiterschalt-Intervall
  um eine Sekunde erhöhen/verringern
- **`Esc`** — die aktuellen Bilder leeren und zum anfänglichen
  Ablegebildschirm zurückkehren; beendet die App, wenn nichts geladen ist,
  das geleert werden könnte (im Handbuchfenster schließt es nur das
  Handbuch; im Diaschau-Modus verlässt es zuerst den Diaschau-Modus);
  solange noch ein Scan läuft (ein abgelegter Ordner, oder das Einlesen
  des Ordners einer einzelnen geöffneten Datei), bricht es stattdessen den
  Scan ab (siehe „Ordner einlesen“ unten)

**Zwischenablage unter Linux.** Das Kopieren des Bildes selbst (`Strg+C`)
ruft ein externes Werkzeug auf, da Linux keinen einzigen eingebauten Weg hat,
um Bilddaten in die Zwischenablage zu legen: `xclip`, oder als Fallback
`wl-copy` (aus dem Paket `wl-clipboard`) für eine Wayland-Sitzung ohne
XWayland. Die meisten Distributionen installieren standardmäßig keines von
beiden — installieren Sie eines mit Ihrem Paketmanager, z. B.
`sudo apt install xclip` oder `sudo apt install wl-clipboard` unter
Debian/Ubuntu. Ohne eines der beiden zeigt `Strg+C` eine Fehler-Toast-Meldung,
statt zu kopieren. Das Kopieren des Dateipfads (`Strg+Shift+C`) ist reiner
Text und funktioniert immer, ohne zusätzliches Werkzeug. macOS und Windows
benötigen in beiden Fällen nichts Zusätzliches.

**Dateimanager unter Linux.** „Im Dateimanager anzeigen“ (`Strg+R`) bittet
Ihren Dateimanager über D-Bus, den Ordner mit der bereits ausgewählten Datei
zu öffnen — Nautilus, Dolphin, Nemo, Thunar und PCManFM beantworten das alle.
Auf einem Schreibtisch ohne einen solchen Dateimanager wird auf `xdg-open` für
den Ordner zurückgegriffen, der ihn öffnet, ohne etwas auszuwählen. Ist keines
von beidem verfügbar, erscheint eine Fehler-Toast-Meldung. Unter macOS
(Finder) und Windows (Explorer) wird die Datei immer selbst ausgewählt.

---

## 12. Menü

- **Datei -> Dateien öffnen…** — öffnet den Dateibrowser des Systems, genau
  wie `Cmd/Strg+O`
- **Datei -> Änderungen speichern** (`Cmd/Strg+S`) — schreibt eine mit
  `R`/`Shift+R` vorgenommene Drehung in die Ursprungsdatei zurück, in deren
  eigenem Format. Ausgegraut, solange es keine Drehung zu speichern gibt;
  nicht verfügbar für Animationen und für Formate, die PicFetch lesen,
  aber nicht schreiben kann (WebP, HEIC, ICO, XPM, SVG). Dabei wird die
  Originaldatei ersetzt und neu kodiert. Bei JPEG kopiert PicFetch die
  Metadaten der Originaldatei (EXIF, einschließlich Kamera/Datum/GPS, sowie
  XMP, ICC und IPTC, falls vorhanden) in die neue Datei und setzt das
  Orientierungs-Tag auf 1, weil die Pixel bereits sowohl die Kameraorientierung
  als auch die soeben gespeicherte Drehung enthalten. Ein in EXIF gespeichertes
  JPEG-Vorschaubild wird entfernt, damit es nicht das ungedrehte Foto zeigt
- **Datei -> Bild exportieren** (`Cmd/Strg+E`) — fragt über eine
  tastaturbedienbare Abfrage nach dem gewünschten Format, in
  derselben Form wie die Lösch-Bestätigung: **`Left`**/**`Right`** wählt zwischen
  **PNG** (standardmäßig ausgewählt) und **JPEG**, **`Return`** exportiert,
  **`Esc`** bricht ab, ohne einen Speichern-Dialog zu öffnen. Das gewählte
  Format speichert das Bild dann so, wie es gerade angezeigt wird, Drehung
  eingeschlossen, in eine neue Datei Ihrer Wahl. Anders als „Änderungen
  speichern“ funktioniert das für jedes anzeigbare Bild, auch für WebP- und
  HEIC-Dateien und für ein einzelnes Bild eines animierten GIFs, und die
  Originaldatei bleibt unangetastet. Endet der von Ihnen eingegebene Name
  bereits auf ein Format, das PicFetch schreiben kann, hat dieses Vorrang
  vor dem in der Abfrage gewählten Format. Ein als JPEG exportiertes JPEG
  behält dieselben Metadaten wie „Änderungen speichern“; ein aus einem
  anderen Format exportiertes JPEG oder jeder PNG-Export enthält keine Metadaten
- **Datei -> Dateien schließen** — zurück zum Ablagebereich, ohne das
  Programm zu beenden
- **Datei -> Einstellungen…** — öffnet das Einstellungsfenster, darunter den
  Schieberegler **Duplikat-Erkennungsabstand** (0–32, Vorgabe 6;
  niedriger ist strenger, 0 ist ein exakter Miniaturansicht-Hash), den das
  Ausblenden von Duplikaten (`D`) verwendet, und das Kontrollkästchen
  **Favoriten-Vorschauen auf der Festplatte zwischenspeichern** (standardmäßig
  an) für die unten beschriebene Hintergrund-Erzeugung der
  Favoriten-Vorschauen. Im Tab **Grenzwerte** stehen **Maximale Dateien pro
  Ordner-Scan**, **Maximaler Bildcache (MB)**, **Maximaler Miniaturbild-Cache
  (MB)** und **Maximale Dateigröße (MB)**. **Nach Updates suchen**
  (standardmäßig aus) steht unter Updates. Wenn aktiviert, prüft PicFetch
  höchstens einmal täglich im Hintergrund bei
  GitHub auf eine neuere Version. Diese wird unauffällig heruntergeladen;
  ein von GitHub bereitgestellter SHA-256-Digest wird, falls vorhanden,
  geprüft, ebenso zwingend GitHubs unveränderliche
  Sigstore-Release-Attestierung, bevor das Update bereitgestellt wird. Ein
  bereitgestelltes Update wird beim normalen Beenden ohne Neustart installiert.
  Beim nächsten manuellen Start zeigt PicFetch ein Neuigkeiten-Fenster mit
  den Versionshinweisen.

  **Jetzt prüfen** startet eine einmalige manuelle Prüfung. Sie funktioniert
  auch bei deaktivierter automatischer Prüfung und umgeht die tägliche
  Begrenzung. Zuerst zeigt PicFetch die laufende Suche. Ist die installierte
  Version aktuell, erscheint eine Meldung nur mit **OK**. Bei einer neueren
  Version wird die Versionsnummer angezeigt; ein Fortschrittsbalken mit
  Prozentanzeige erscheint nur, wenn die Archivgröße bekannt ist, andernfalls
  bleibt er unbestimmt. Das Archiv wird mit derselben SHA-256-Prüfung und
  derselben verpflichtenden Sigstore-Attestierungsprüfung wie bei einem
  automatischen Download verifiziert und bereitgestellt. Fehler beim Prüfen,
  Herunterladen,
  Verifizieren, Entpacken oder Bereitstellen werden in einer Meldung nur mit
  **OK** angezeigt.

  Mit **Später** bleibt das Update bereitgestellt; beim normalen Beenden wird
  es weiterhin ohne Neustart installiert. **Update installieren** beendet
  PicFetch, installiert die bereitgestellte Version beim Herunterfahren und
  startet PicFetch neu. Der aktualisierte Start zeigt anschließend die
  Neuigkeiten zu dieser Version.

  Konnte sich PicFetch unter Windows nicht selbst ersetzen – meist, weil der
  Überwachte Ordnerzugriff den Installationsordner schützt –, meldet PicFetch
  das beim nächsten Start, nennt die betroffene Datei und bietet eine
  Schaltfläche zur Download-Seite. Damit Updates wieder automatisch
  übernommen werden, erlauben Sie PicFetch den Zugriff unter
  Windows-Sicherheit -> Viren- & Bedrohungsschutz -> Ransomware-Schutz -> App
  durch überwachten Ordnerzugriff zulassen, oder installieren Sie PicFetch
  außerhalb der geschützten Benutzerordner (Dokumente, Bilder, Musik, Videos,
  Desktop).
- **Favoriten -> Aktuelle Liste zu Favoriten hinzufügen…** (`Opt/Alt+Shift+F`) — speichert die
  gesamte aktuell geöffnete Dateiliste als benannte Sammlung. Favoriten
  bleiben nach einem Neustart von PicFetch erhalten. Gespeichert werden
  Verweise auf die Originaldateien, keine Kopien der Bilder; wird ein
  Original verschoben oder gelöscht, kann es aus dem Favoriten nicht mehr
  geladen werden. Der Dialog ist vollständig über die Tastatur bedienbar:
  beim Öffnen ist bereits das Namensfeld fokussiert, Sie können also sofort
  lostippen; **`Return`** im Feld speichert mit dem gerade eingegebenen
  Namen, **`Down`** bewegt die Tastatur weiter zu einem Rahmen über
  „Abbrechen“/„Hinzufügen“ (steht anfangs auf „Abbrechen“), **`Up`** bewegt
  sie zurück zum Feld, **`Left`**/**`Right`** bewegen dort den Rahmen, und
  **`Esc`** bricht von beiden Stellen aus ab. „Hinzufügen“ bleibt
  ausgegraut, solange der Name ungültig ist — also leer ist oder eines der
  Zeichen `/ \ : * ? " < > |` enthält
- **Favoriten -> _Favoritenname_** — öffnet die gespeicherte Liste mit
  demselben Scan-, Sortier- und Zusammenführen-Verhalten wie „Dateien
  öffnen“. Jeder Eintrag zeigt, wie viele Dateien er enthält, z. B.
  „Urlaub 2024 (128)“; die Einträge sind ohne Beachtung der
  Groß-/Kleinschreibung nach Namen sortiert. Die ersten neun zeigen
  `Cmd`/`Strg+1` bis `Cmd`/`Strg+9`, der zehnte zeigt `Cmd`/`Strg+0`. Wird
  eine Sammlung unter einem bereits vorhandenen Namen gespeichert, fragt
  PicFetch vor dem Ersetzen der gespeicherten Liste nach, und auch diese
  Rückfrage lässt sich vollständig über die Tastatur beantworten:
  **`Left`**/**`Right`** bewegen einen Rahmen zwischen „Abbrechen“ und „Ersetzen“
  — er steht anfangs auf „Abbrechen“, sodass `Return` von sich aus nichts
  ersetzt —, **`Return`** löst die markierte Option aus, **`Esc`** bricht
  ab. Beide Wege, die Rückfrage abzubrechen, öffnen den Dialog zum
  Hinzufügen erneut, mit dem eingegebenen Namen weiterhin im Feld, statt
  ihn erneut eintippen zu lassen
- **Favoriten -> Favoriten verwalten…** (auch `Cmd`/`Strg+Shift+F`) — zeigt
  alle gespeicherten Sammlungen mit ihrer Dateianzahl an und lässt Sie eine
  davon öffnen oder entfernen. Vollständig über die Tastatur bedienbar:
  **`Up`**/**`Down`** bewegen den Rahmen zwischen den Zeilen, **`Left`**/**`Right`**
  bewegen ihn zwischen den Schaltflächen „Öffnen“ und „Entfernen“ der
  jeweiligen Zeile, **`Return`** löst die gerade markierte Option aus, und
  ein Klick führt immer die angeklickte Schaltfläche aus, unabhängig davon,
  wo der Rahmen gerade steht. **`Esc`** schließt den Dialog. Das Entfernen
  fragt vorher nach Bestätigung; auch diese Rückfrage lässt sich vollständig
  über die Tastatur beantworten: **`Left`**/**`Right`** bewegen den Rahmen zwischen
  „Abbrechen“ und „Entfernen“ — er steht anfangs auf „Abbrechen“, sodass
  `Return` von sich aus nichts entfernt —, **`Return`** löst die markierte
  Option aus, **`Esc`** bricht ab. Wird bestätigt, wandert nur der eigene
  Ordner der Sammlung in den Papierkorb; die Originalbilder werden **nicht**
  verschoben oder gelöscht
- **Aktionen -> Sortierreihenfolge** (`S`) — Untermenü mit denselben fünf
  Sortierungen wie in den Einstellungen: Name, Aufnahmedatum,
  Änderungszeitpunkt, Dateigröße, Ablagereihenfolge. Die aktuelle
  Sortierung ist angehakt. Eine Auswahl springt dorthin (schaltet nicht
  durch). `S` schaltet weiterhin durch. Erneutes Wählen der aktuellen
  Sortierung ändert nichts
- **Aktionen -> Duplikate ein-/ausblenden** (`D`) — dasselbe wie `D`:
  blendet Extra-Kopien derselben Aufnahme aus und ist angehakt, solange das
  Ausblenden an ist. Ausgegraut, wenn keine Dateien geladen sind.
  Funktioniert aus dem Menü auch, während eine Rastersuche offen ist
- **Aktionen -> Varianten anzeigen** (`Shift+D`) — zeigt im Raster jede
  Kopie der hervorgehobenen/aktuellen Aufnahme, dasselbe wie `Shift+D`,
  sobald es läuft. Angehakt, solange dieser Browse-Filter an ist.
  Ausgegraut, bis Duplikate ein-/ausblenden an ist **und** die aktuelle
  Datei Duplikate hat, und auch ohne geladene Dateien oder im
  Bilderrahmen-Modus. `Shift+D` funktioniert weiterhin mit ausgeschaltetem
  Ausblenden; dieser Menüpunkt nicht
- **Aktionen -> Ausgewählte Bilder vergleichen** (`Cmd/Strg+D`) — vergleicht
  genau zwei ausdrücklich ausgewählte Rasterdateien in eingepassten
  nebeneinanderliegenden Bereichen. Ausgegraut, außer das Raster ist mit genau
  zwei ausgewählten Dateien geöffnet. In der Vergleichs-Werkzeugleiste
  vertauscht **Tauschen** die bezeichneten Seiten, sobald beide bereit sind.
  **Zurück zur Rasteransicht** oder `Esc` kehrt zum unveränderten Raster zurück
- **Aktionen -> Bildmosaik erstellen...** — öffnet den Mosaikablauf für die
  ausdrückliche Rasterauswahl oder, wenn nichts ausgewählt ist, für jedes Bild
  im aktuellen Rasterergebnis. Außerhalb eines nicht leeren Rasterergebnisses
  ausgegraut. Bedienelemente, Export und der Hintergrundbildumfang je Plattform
  sind oben unter „Bildmosaike“ beschrieben
- **Aktionen -> Bild drehen (im Uhrzeigersinn)** (`R`) — 90° im
  Uhrzeigersinn, nur Ansicht, dasselbe wie `R`. Ausgegraut ohne geladenes
  Bild oder solange das Raster offen ist. `Shift+R` bleibt nur über die
  Tastatur erreichbar
- **Aktionen -> Vergrößern** (`+`) / **Verkleinern** (`-`) — dasselbe wie
  `+`/`-`. Ausgegraut ohne geladenes Bild oder solange das Raster offen ist
- **Aktionen -> Zusammenführen-Modus umschalten** (`M`) — dasselbe wie `M`.
  Angehakt, solange der Modus an ist. Funktioniert auch vor dem Laden von
  Dateien
- **Aktionen -> Info-Overlay ein-/ausblenden** (`I`) — dasselbe wie `I`.
  Angehakt, solange die Overlay-Einstellung an ist. Ausgegraut, solange das
  Raster offen ist
- **Aktionen -> Bild kopieren** (`Cmd/Strg+C`) — die angezeigten Pixel oder
  die Rasterauswahl als Dateien. Ausgegraut, wenn keine Dateien geladen sind
- **Aktionen -> Auswahl kopieren** (`Opt/Alt+Shift+C`) — startet den
  Auswahl-kopieren-Modus, damit Sie einen rechteckigen Bildbereich statt des
  ganzen Bildes kopieren können. Ziehen zeichnet das Rechteck, danach können
  Sie es verschieben oder in der Größe ändern; **In die Zwischenablage
  kopieren** (oder `Return`) kopiert PNG-Pixel in voller Auflösung. `Esc`
  bricht ab, ohne die Zwischenablage zu ändern. Ausgegraut ohne dekodiertes
  Bild, solange Raster oder Diaschau-Modus aktiv sind, oder solange ein
  Dialog das Fenster besitzt
- **Aktionen -> Bildpfad kopieren** (`Cmd/Strg+Shift+C`) — der Pfad der
  aktuellen Datei. Ausgegraut, wenn keine Dateien geladen sind
- **Aktionen -> Im Dateimanager anzeigen** (`Cmd/Strg+R`) — öffnet Finder,
  Explorer oder Ihren Linux-Dateimanager mit der aktuellen Datei ausgewählt,
  damit Sie sie außerhalb von PicFetch umbenennen, verschieben oder teilen
  können. Das Info-Overlay (`I`) enthält denselben Befehl als Link. Immer die
  angezeigte Datei, nie die Rasterauswahl. Ausgegraut, wenn keine Dateien
  geladen sind
- **Aktionen -> Als Hintergrundbild festlegen** (`Cmd/Strg+Shift+E`) —
  macht das angezeigte Bild zum Hintergrundbild des Schreibtischs, genau so,
  wie es gerade aussieht. PicFetch legt dafür eine eigene Kopie im
  Zwischenspeicher an und verweist den Schreibtisch darauf, das
  Hintergrundbild bleibt also erhalten, wenn Sie das Original verschieben,
  umbenennen oder in den Papierkorb legen. Unter Linux wird dafür
  `gsettings` (GNOME, Cinnamon, Budgie, Unity) oder
  `plasma-apply-wallpaperimage` (KDE Plasma ab 5.24) benötigt; ist keines
  von beiden installiert, erscheint ein entsprechender Hinweis. Ausgegraut,
  bis ein Bild geladen ist
- **Aktionen -> Bild in den Papierkorb legen** (`Shift+Delete`/`Shift+Entf`)
  — dasselbe wie `Shift+Delete`: fragt nach, verschiebt dann die aktuelle
  Datei (oder die Rasterauswahl) in den Papierkorb. Ausgegraut, wenn keine
  Dateien geladen sind
- **Fenster -> Bildanzeige** (`V`) — zeigt die normale Bildansicht. Schließt
  die Rasteransicht oder verlässt den Bilderrahmen-Modus, wenn einer aktiv
  ist. Ausgegraut, solange Sie bereits in dieser Ansicht sind. `V` ist kein
  Ein-/Ausschalter. Solange die Rastersuche (`/`) offen ist, tippt die Taste
  den Buchstaben `v` in die Suche. Unter macOS stehen diese Einträge im
  System-Menü Fenster, oberhalb von „Miniaturisieren“
- **Fenster -> EXIF-Daten** (`E`) — öffnet das EXIF-Panel für das gerade
  angezeigte Bild, genau wie der Link **„EXIF-Daten anzeigen“** im
  Info-Overlay. Ausgegraut, solange dieses Panel bereits offen ist, oder wenn
  nichts angezeigt wird
- **Fenster -> Rasteransicht** (`G`) — öffnet die Miniaturübersicht.
  Ausgegraut, solange die Rasteransicht angezeigt wird, der Bilderrahmen-Modus
  aktiv ist, oder keine Dateien geladen sind. Schließen erfolgt über
  Bildanzeige / `V`, `G` oder `Esc` — nicht über diesen Menüpunkt
- **Fenster -> Bilderrahmen-Modus** (`P`) — wechselt in den Vollbild-
  Bilderrahmen-Modus. Ausgegraut, solange er bereits aktiv ist, oder keine
  Dateien geladen sind. Verlassen erfolgt über Bildanzeige / `V`, `P` oder
  `Esc`
- **Fenster -> Hilfe** (`F1`) — öffnet dieses Handbuch, genau wie
  Hilfe -> Handbuch. Ausgegraut, solange das Handbuchfenster bereits offen ist
- **Hilfe -> Handbuch** — öffnet dieses Handbuch, genau wie `F1`

---

## 13. Rückmeldung beim Laden

Das Dekodieren läuft im Hintergrund, sodass das Fenster auch bei sehr großen
Dateien reaktionsfähig bleibt.

- Ein animierter **Fortschrittsbalken** erscheint am oberen Rand des
  Fensters, während ein Bild geladen wird, und verschwindet, sobald es
  fertig ist. Er wird über das Bild gelegt und verschiebt nie etwas auf dem
  Bildschirm.
- Beim allerersten Ablegen ändert sich der Hinweistext zu **„Wird
  geladen…“**.
- Bei späteren Wechseln bleibt das vorherige Bild sichtbar, bis das neue
  bereit ist, sodass es keinen leeren Blitzer zwischen den Bildern gibt.
- Tastendrücke, während ein Bild noch lädt, werden ignoriert. Das
  Gedrückthalten einer Pfeiltaste staut daher keinen Rückstand an
  Dekodierungen für Bilder an, die Sie bereits übersprungen haben.
- Wird ein langsames Bild fertig dekodiert, nachdem Sie bereits weitergegangen
  sind, wird das Ergebnis verworfen — Sie sehen immer das zuletzt ausgewählte
  Bild.

**Ordner einlesen.** Enthält Ihr Ablegen Ordner, durchsucht PicFetch diese
zuerst (und jeden Unterordner), um unterstützte Bilder zu sammeln, bevor
irgendetwas angezeigt wird. Eine **einzelne Bilddatei** zu öffnen macht
einen ähnlichen Scan nur in deren eigenem Ordner (keine Unterordner) und
bleibt auf der geöffneten Datei stehen:

- Ein Spinner erscheint zusammen mit einem laufenden Zähler, z. B.
  **„Scannen… 42 Bilder“**, der aktualisiert wird, sobald weitere Bilder
  gefunden werden. Ein sehr großer Ordner kann einen Moment brauchen,
  selbst wenn nur eine Datei geöffnet wurde.
- Sobald der Scan abgeschlossen ist, verschwindet der Spinner. Ein
  abgelegter Ordner zeigt das erste gefundene Bild; eine einzelne
  geöffnete Datei bleibt die Datei, die Sie geöffnet haben.
- Ablegen von zwei oder mehr losen Dateien (ohne Ordner) überspringt
  diesen Schritt und lädt sofort.
- Drücken Sie jederzeit **`Esc`**, während der Spinner angezeigt wird, um
  den Scan abzubrechen. War dies das allererste Ablegen, gelangen Sie zurück
  zum anfänglichen Ablegebildschirm, genau als wäre nichts abgelegt worden;
  haben Sie in eine bereits geladene Auswahl zusammengeführt, bleiben die
  Bilder, die Sie vor Beginn des Scans hatten, unangetastet.

---

## 14. Meldungen und Fehlerbehandlung

PicFetch zeigt in folgenden Fällen ein Dialogfenster an. Schließen Sie es
mit der Schaltfläche **OK**.

- **Eine nicht unterstützte Datei abgelegt** — *„…“ ist keine unterstützte
  Bilddatei*
- **Mehrere Dateien, keine davon unterstützt** — *Keine der N abgelegten
  Dateien ist ein unterstütztes Bildformat*
- **Datei kann nicht gelesen oder dekodiert werden** — *„…“ konnte nicht
  gelesen werden*
- **Datei dekodiert zu einem Bild der Größe null** — *Ungültige
  Bildabmessungen für „…“*

**Gemischte Ablagen werden stillschweigend behandelt.** Wenn Sie einen
Stapel Fotos zusammen mit ein paar Textdateien ablegen, werden die
unterstützten Bilder angezeigt, und der Rest wird ohne Dialog übersprungen.
Nur ein Ablegen, das **kein** verwendbares Bild enthält, erzeugt einen
Fehler.

**Eine Datei, die sich nicht dekodieren lässt, wird aus der Auswahl
entfernt.** Stellt sich erst beim Navigieren zu einer Datei heraus, dass sie
unlesbar oder beschädigt ist (eine Prüfung anhand der Dateiendung beim
Ablegen kann nicht alles abfangen), wird sie aus der Auswahl entfernt, und
das nächste Bild wird automatisch angezeigt, wobei die Navigation im Kreis
läuft, falls es die letzte Datei war — Sie stehen nie vor einer defekten
Datei, bei der Titelzeile und Positionszähler nicht mehr zu dem passen, was
tatsächlich angezeigt wird. Sie erhalten lediglich eine Toast-Meldung mit dem
Namen der übersprungenen Datei. Erweist sich jede Datei in der Auswahl als
defekt, gelangen Sie zurück zum anfänglichen Ablegebildschirm.

**Abgelegte Ordner werden aufgelöst.** Das Ablegen eines Ordners durchsucht
ihn und jeden Unterordner nach unterstützten Bildern; Sie können eine
beliebige Mischung aus einzelnen Bilddateien und Ordnern auf einmal ablegen.
Siehe „Ordner einlesen“ oben für das, was Sie sehen, während ein Ordner
durchsucht wird.

---

## 15. Sprache

Der Oberflächentext (der Ablege-Hinweis, „Wird geladen…“, das Menü) kann
übersetzt werden. PicFetch wird mit Englisch ausgeliefert und folgt Ihrer
Systemsprache, sofern eine passende Übersetzung verfügbar ist; andernfalls
greift es auf Englisch zurück. Dieses Handbuch folgt derselben Regel: Bei
deutscher Systemsprache öffnet sich diese deutsche Fassung, sonst die
englische.

---

## 16. Beenden

Drücken Sie `Esc` im Bildfenster, wenn nichts geladen ist, oder schließen
Sie es auf die für Ihre Plattform übliche Weise (die rote Schaltfläche unter
macOS, das ✕ unter Windows und Linux). Sind Bilder geladen, leert `Esc` sie
zunächst und kehrt zum anfänglichen Ablegebildschirm zurück — drücken Sie es
erneut (jetzt, da die Auswahl geleert ist), um die App zu beenden. Der
Zusammenführen-Modus, die Sortierreihenfolge, das Diaschau-Intervall samt
Zufälliger Wiedergabe und die Fenstergröße bleiben bis zum nächsten Start
erhalten (siehe die jeweiligen Abschnitte oben); alles andere nicht — Zoom,
Rotation und das zuletzt betrachtete Bild werden zurückgesetzt.

---

## 17. Aktuelle Einschränkungen

Dinge, die PicFetch absichtlich (noch) nicht tut:

- Keine Navigation per Mausrad oder Trackpad-Scroll; das Durchblättern der
  Bilder funktioniert nur über die Tastatur (Pfeiltasten, `Home`/`End`,
  siehe oben)
- Keine echte Pinch-to-Zoom-Trackpad-Geste; Shift+Scrollen ist der
  nächstliegende Ersatz für Zweifinger-Verschieben (siehe „Zoom und
  Verschieben“ oben)
- Keine Zoomsteuerung innerhalb des Diaschau-Modus selbst, und keine
  bildspezifische Zeitsteuerung — jedes Bild erhält dasselbe Intervall
  (animierte GIFs ausgenommen)
- Keine Bildbearbeitung über das Drehen hinaus: kein Zuschneiden, keine
  Farb- oder Belichtungskorrektur, keine Größenänderung. Auswahl kopieren
  legt einen Bereich in die Zwischenablage und ändert die Quelldatei nicht.
  Auf die Festplatte schreiben lassen sich eine Drehung (**Datei ->
  Änderungen speichern**), eine Kopie in einem anderen Format (**Datei ->
  Bild exportieren**) und eine Hintergrundbild-Kopie (**Aktionen -> Als
  Hintergrundbild festlegen**), alle unter „Menü“ oben beschrieben
- Kein RAW-Demosaic und keine PDF-Unterstützung: RAW-Dateien zeigen nur die
  vom Fotoapparat eingebettete JPEG-Vorschau; Auswahl kopieren kopiert
  diese Vorschau, und auf die Festplatte schreiben lassen sich weiterhin
  eine Drehung encodierbarer Formate, ein Export oder eine
  Hintergrundbild-Kopie
- Auswahl kopieren bei SVG nutzt die logische Bildgröße und dieselbe
  Vektor-Raster-Obergrenze wie die Anzeige; die Obergrenze wird nicht
  angehoben
- Ein animiertes Bild bleibt auf dem beim Start von Auswahl kopieren
  sichtbaren Einzelbild stehen und läuft weiter, wenn der Modus endet (die
  genaue Taktphase muss nicht erhalten bleiben)
- Eine sehr große Auswahl kann trotzdem fehlschlagen, wenn der Prozess oder
  die Systemzwischenablage sie nicht aufnehmen kann; das Rechteck bleibt
  zum erneuten Versuch stehen
- Keine Wiedergabesteuerung (Pause, Einzelschritt, Neustart) für animierte
  GIFs
- Keine Offline-Karten: die Ortsansicht im EXIF-Fenster benötigt eine
  Internetverbindung, da sie OpenStreetMap-Kacheln live lädt

---

![Trane wedelt mit dem Schwanz](trane_wags.webp)

## 18. Kurzübersicht

- **Laden** — Bilddateien auf das Fenster ziehen (ersetzt die aktuelle
  Auswahl)
- **Öffnen** — auf den Ablegebereich klicken, oder `Cmd`/`Strg+O` drücken
  (oder `Cmd`/`Strg+Shift+O`, dasselbe), für den System-Dateidialog (Dateien
  und Ordner unter macOS/Linux, nur Dateien unter Windows)
- **Favoriten** — Favoriten -> Aktuelle Liste zu Favoriten hinzufügen…
  (`Opt`/`Alt+Shift+F`) speichert die geöffnete Liste; der Favoritenname
  öffnet sie wieder, `Cmd`/`Strg+1`–`9` öffnet die ersten neun sortierten
  Favoriten und `Cmd`/`Strg+0` den zehnten; „Favoriten verwalten…“
  (`Cmd`/`Strg+Shift+F`) entfernt Sammlungen, ohne ihre Originalbilder
  anzutasten
- **Zusammenführen-Modus** — `M` schaltet ihn ein/aus (auch Aktionen ->
  Zusammenführen-Modus umschalten); solange aktiv, ergänzen Ablagen die
  Auswahl, statt sie zu ersetzen, und die Titelzeile zeigt `[Zusammenführen]`
- **Nächstes / Vorheriges** — `Right` `Down` / `Left` `Up` (läuft im Kreis)
- **Erstes / Letztes** — `Home` / `End`
- **Sortierreihenfolge** — `S` schaltet durch Name -> Aufnahmedatum ->
  Änderungszeitpunkt -> Größe -> unsortiert -> zurück zu Name (Aktionen ->
  Sortierreihenfolge springt direkt zu einer)
- **Rasteransicht** — `G` schaltet ein fensterfüllendes Miniaturraster
  ein/aus; Pfeiltasten bewegen die Hervorhebung, `Page Up`/`Page Down`
  gleich um eine ganze Seite, `Return` oder ein Klick öffnet, `G`/`Esc`/`V`
  (oder Fenster -> Bildanzeige) bricht ohne Auswahl ab
- **Extra-Kopien ausblenden** — `D` blendet Extra-Kopien derselben Aufnahme
  aus (auch Aktionen -> Duplikate ein-/ausblenden; Einzelstücke bleiben
  sichtbar; verbleibende Zellen zeigen ein Zähler-Abzeichen). Pfeiltasten,
  `Home`/`End` und der Diaschau-Wechsel überspringen die versteckten, bis Sie
  erneut `D` drücken. `G`/Schließen lassen das an. In den Einstellungen liegt
  der Abstands-Schieberegler
- **Duplikate anzeigen** — `Shift+D` zeigt alle Kopien der
  hervorgehobenen/aktuellen Aufnahme im Raster (auch Aktionen -> Varianten
  anzeigen), einschließlich Extra-Kopien, die `D` ausblenden würde; `G`/
  Schließen beenden die Anzeige, lassen das Ausblenden aber an. Return/Klick
  behält die gewählte Kopie auf dem Bildschirm und läuft die Gruppe mit
  Links/Rechts durch (auch nach Home/Ende, die weiter über die ganze Menge
  springen); `Esc` oder `G` kehrt zum Varianten-Raster zurück, dann
  `Esc` zum Raster mit ausgeblendeten Extra-Kopien. `D` und der Bilderrahmen
  bleiben während dieser Schleife aus
- **Suche nach Namen** — `/` im Raster filtert es auf die Dateinamen, die
  Ihre Eingabe enthalten; stapelt sich mit dem Ausblenden von Duplikaten.
  `Esc` setzt den Filter zurück, dann die Duplikat-Anzeige, dann das
  Ausblenden, dann das Raster. Filter und Auswahl überleben einander, sodass
  `/`, gefolgt von
  `Cmd`/`Strg+A`, genau auf die Treffer wirkt
- **Zoom** — `+`/`-` vergrößern/verkleinern (auch Aktionen -> Vergrößern/
  Verkleinern; Fenster folgt dem Bild, Minimum ist die Startgröße, Maximum
  kommt aus den Einstellungen), `1` für 100 %, `0` für Fenstereinpassung,
  oder Scrollen zum Zoomen am Mauszeiger; ziehen, oder Shift+Scrollen, zum
  Verschieben, sobald das Bild nicht mehr passt
- **Drehen** — `R`/`Shift+R` dreht um 90° im/gegen den Uhrzeigersinn (Aktionen
  -> Bild drehen (im Uhrzeigersinn) dreht nur in diese Richtung), nur
  Ansicht; `0` setzt es zusammen mit dem Zoom zurück
- **Info-Overlay** — `I` schaltet eine Karte mit Dateiname, Position,
  Abmessungen, Dateigröße und Zoomstufe ein/aus (auch Aktionen -> Info-Overlay
  ein-/ausblenden)
- **EXIF-Datenfenster** — `E`, oder der Link „EXIF-Daten anzeigen“ im
  Info-Overlay, öffnet Kamerahersteller/-modell, Objektiv, Belichtung,
  Blende, ISO, Brennweite, Aufnahmedatum und Koordinaten für das aktuelle
  Bild sowie eine ausklappbare Karte des Aufnahmeorts, wenn das Foto
  GPS-Tags trägt; bei JPEGs steht unter den Tags (oberhalb der Karte)
  **Metadaten entfernen**, das nach Bestätigung identifizierende Tags direkt
  aus der Datei entfernt und fehlt, wenn die Tag-Liste leer ist oder nichts
  mehr zu entfernen ist; solange das Fenster den Fokus hat, wechseln `Left`/`Right` das Bild
- **Diaschau-Modus** — `P` schaltet eine Vollbild-Diaschau mit Überblendung
  zwischen den Bildern ein/aus; `Up`/`Down` stellen das (standardmäßig 10 s)
  Auto-Weiterschalt-Intervall ein, solange sie aktiv ist; `Shift+P` schaltet
  die Zufällige Wiedergabe ein/aus (`[Zufällig]` in der Titelzeile); verlassen
  mit `V`/`P`/`Esc` oder Fenster -> Bildanzeige
- **Kopieren** — `Cmd`/`Strg+C` kopiert das aktuelle Bild (Aktionen -> Bild
  kopieren), `Opt`/`Alt+Shift+C` kopiert einen Bildbereich (Aktionen ->
  Auswahl kopieren), `Cmd`/`Strg+Shift+C` kopiert seinen Dateipfad (Aktionen
  -> Bildpfad kopieren); im Raster kopiert `Cmd`/`Strg+C` die ausgewählten
  Dateien selbst
- **Löschen** — `Shift+Delete`/`Shift+Entf` öffnet eine Bestätigungskarte
  (Aktionen -> Bild in den Papierkorb legen; `Left`/`Right` zum Auswählen, `Return`
  zum Bestätigen, `Esc` zum Abbrechen); verschiebt die Datei in den
  Papierkorb, oder die ganze Auswahl des Rasters
- **Im Raster auswählen** — `Cmd`/`Strg+Klick` oder `Leertaste` für eine,
  `Shift+Klick` für einen Bereich, Klicken-und-Ziehen für jede vom Rechteck
  berührte Miniaturansicht (Shift- oder Cmd/Strg+Ziehen fügt hinzu), `Cmd`/`Strg+A` für alle (bzw. alle
  Suchtreffer); `Esc` setzt die Auswahl zurück
- **Handbuch** — `F1`, oder Hilfe -> Handbuch oder Fenster -> Hilfe
- **Leeren / Beenden** — `Esc` (leert zuerst die geladenen Bilder, beendet
  dann; bricht stattdessen einen noch laufenden Scan ab, falls einer
  läuft)
- **Formate** — JPEG, PNG, GIF (inkl. animiert), WebP, BMP, TIFF, ICO, XPM,
  HEIC/HEIF, AVIF, SVG, Kamera-RAW (eingebettete JPEG-Vorschau)
- **Maximale Fenstergröße** — 1500 × 950
