package main

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

// goldenBlock is renderDocumentTypes(bareExtensions())'s exact output as of
// this test's writing. It is the anti-drift guard for the whole package: if
// internal/imaging.SupportedExtensions() gains or loses a format, or
// contentTypeUTIs changes, this test fails until the golden text below is
// updated to match - the same way any golden test works.
const goldenBlock = `	<key>CFBundleDocumentTypes</key>
	<array>
		<dict>
			<key>CFBundleTypeName</key>
			<string>Image</string>
			<key>CFBundleTypeRole</key>
			<string>Viewer</string>
			<key>LSHandlerRank</key>
			<string>Alternate</string>
			<key>CFBundleTypeExtensions</key>
			<array>
				<string>jpg</string>
				<string>jpeg</string>
				<string>jpe</string>
				<string>jfif</string>
				<string>png</string>
				<string>gif</string>
				<string>webp</string>
				<string>bmp</string>
				<string>tif</string>
				<string>tiff</string>
				<string>ico</string>
				<string>xpm</string>
				<string>heic</string>
				<string>heif</string>
				<string>avif</string>
				<string>svg</string>
				<string>cr2</string>
				<string>cr3</string>
				<string>nef</string>
				<string>nrw</string>
				<string>arw</string>
				<string>dng</string>
				<string>orf</string>
				<string>rw2</string>
				<string>raf</string>
				<string>pef</string>
				<string>srw</string>
				<string>raw</string>
			</array>
			<key>LSItemContentTypes</key>
			<array>
				<string>public.jpeg</string>
				<string>public.png</string>
				<string>com.compuserve.gif</string>
				<string>public.webp</string>
				<string>org.webmproject.webp</string>
				<string>com.microsoft.bmp</string>
				<string>public.tiff</string>
				<string>com.microsoft.ico</string>
				<string>public.heic</string>
				<string>public.heif</string>
				<string>public.avif</string>
				<string>public.svg-image</string>
				<string>public.camera-raw-image</string>
				<string>com.canon.cr2-raw-image</string>
				<string>com.canon.cr3-raw-image</string>
				<string>com.nikon.raw-image</string>
				<string>com.nikon.nrw-raw-image</string>
				<string>com.sony.arw-raw-image</string>
				<string>com.adobe.raw-image</string>
				<string>com.olympus.raw-image</string>
				<string>com.panasonic.rw2-raw-image</string>
				<string>com.fuji.raw-image</string>
				<string>com.pentax.raw-image</string>
				<string>com.samsung.raw-image</string>
			</array>
		</dict>
		<dict>
			<key>CFBundleTypeName</key>
			<string>Folder</string>
			<key>CFBundleTypeRole</key>
			<string>Viewer</string>
			<key>LSHandlerRank</key>
			<string>Alternate</string>
			<key>LSItemContentTypes</key>
			<array>
				<string>public.folder</string>
			</array>
		</dict>
	</array>`

func TestRenderDocumentTypes_Golden(t *testing.T) {
	got := renderDocumentTypes(bareExtensions())
	if got != goldenBlock {
		t.Errorf("renderDocumentTypes(bareExtensions()) does not match the golden block\n--- got ---\n%s\n--- want ---\n%s", got, goldenBlock)
	}
}

// TestRenderDocumentTypes_EveryExtensionEmitted is the anti-drift guard
// this whole port exists for: every extension internal/imaging accepts must
// end up in CFBundleTypeExtensions, or a format the app can open silently
// wouldn't offer PicFetch in Finder's "Open With".
func TestRenderDocumentTypes_EveryExtensionEmitted(t *testing.T) {
	block := renderDocumentTypes(bareExtensions())

	for _, ext := range imaging.SupportedExtensions() {
		bare := strings.TrimPrefix(ext, ".")
		want := "<string>" + bare + "</string>"
		if !strings.Contains(block, want) {
			t.Errorf("rendered block is missing %q for imaging.SupportedExtensions() entry %q", want, ext)
		}
	}
}

// TestContentTypeUTIs_ExtensionsAreAllSupported guards the table the other
// direction: every extension contentTypeUTIs references should still be one
// internal/imaging actually decodes, so the table doesn't silently reference
// a format that was removed.
func TestContentTypeUTIs_ExtensionsAreAllSupported(t *testing.T) {
	supported := make(map[string]bool)
	for _, ext := range imaging.SupportedExtensions() {
		supported[strings.TrimPrefix(ext, ".")] = true
	}

	for _, m := range contentTypeUTIs {
		if !supported[m.ext] {
			t.Errorf("contentTypeUTIs references extension %q, which imaging.SupportedExtensions() no longer lists", m.ext)
		}
	}
}

func TestRenderDocumentTypes_DeclaresFolderType(t *testing.T) {
	block := renderDocumentTypes(bareExtensions())

	folderDict := "\t\t<dict>\n" +
		"\t\t\t<key>CFBundleTypeName</key>\n" +
		"\t\t\t<string>Folder</string>\n" +
		"\t\t\t<key>CFBundleTypeRole</key>\n" +
		"\t\t\t<string>Viewer</string>\n" +
		"\t\t\t<key>LSHandlerRank</key>\n" +
		"\t\t\t<string>Alternate</string>\n" +
		"\t\t\t<key>LSItemContentTypes</key>\n" +
		"\t\t\t<array>\n" +
		"\t\t\t\t<string>public.folder</string>\n" +
		"\t\t\t</array>\n" +
		"\t\t</dict>"

	if !strings.Contains(block, folderDict) {
		t.Errorf("rendered block does not declare the Folder document type as expected:\n%s", block)
	}
}

func TestContentTypeList_DedupesPreservingOrder(t *testing.T) {
	got := contentTypeList()

	seen := make(map[string]bool, len(got))
	for _, uti := range got {
		if seen[uti] {
			t.Fatalf("contentTypeList() repeats %q", uti)
		}
		seen[uti] = true
	}

	if !seen["public.jpeg"] {
		t.Error(`contentTypeList() is missing "public.jpeg"`)
	}
	if !seen["public.webp"] || !seen["org.webmproject.webp"] {
		t.Error("contentTypeList() must keep both public.webp and org.webmproject.webp")
	}
}

// fyneTemplateFixture mirrors fyne package -os darwin's real Info.plist
// output (observed at bin/PicFetch.app/Contents/Info.plist, a local
// gitignored build artifact) before any CFBundleDocumentTypes patch has
// been applied to it - tab-indented exactly as fyne emits it.
const fyneTemplateFixture = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>picfetch</string>
	<key>CFBundleIconFile</key>
	<string>icon.icns</string>
	<key>CFBundleIdentifier</key>
	<string>io.github.frathe.picfetch</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>PicFetch</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>0.2.9</string>
	<key>CFBundleSupportedPlatforms</key>
	<array>
		<string>MacOSX</string>
	</array>
	<key>CFBundleVersion</key>
	<string>427</string>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.</string>
	<key>LSMinimumSystemVersion</key>
	<string>10.11</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSSupportsAutomaticGraphicsSwitching</key>
	<true/>
</dict>
</plist>
`

func TestInsertDocumentTypes_RoundTripsValidXMLInsideTopLevelDict(t *testing.T) {
	out, err := insertDocumentTypes(fyneTemplateFixture, bareExtensions())
	if err != nil {
		t.Fatalf("insertDocumentTypes: %v", err)
	}

	dec := xml.NewDecoder(strings.NewReader(out))
	for {
		if _, err := dec.Token(); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("output is not well-formed XML: %v", err)
		}
	}

	firstDict := strings.Index(out, "<dict>")
	lastCloseDict := strings.LastIndex(out, "</dict>")
	inserted := strings.Index(out, "<key>CFBundleDocumentTypes</key>")

	if firstDict == -1 || lastCloseDict == -1 || inserted == -1 {
		t.Fatalf("missing expected markers: firstDict=%d lastCloseDict=%d inserted=%d", firstDict, lastCloseDict, inserted)
	}
	if inserted < firstDict || inserted > lastCloseDict {
		t.Fatalf("CFBundleDocumentTypes at byte %d is not inside the top-level <dict> (spans %d..%d)", inserted, firstDict, lastCloseDict)
	}

	// The block must land immediately after the top-level <dict>, not
	// somewhere later inside it. renderDocumentTypes's first rendered line
	// is itself tab-indented ("\t<key>..."), so the inserted "\n" is
	// followed by that leading tab before the <key> text starts.
	wantAt := firstDict + len("<dict>") + len("\n\t")
	if inserted != wantAt {
		t.Errorf("CFBundleDocumentTypes starts at byte %d, want %d (immediately after the first <dict>)", inserted, wantAt)
	}
}

func TestInsertDocumentTypes_ErrorsWhenAnchorMissing(t *testing.T) {
	_, err := insertDocumentTypes("<plist version=\"0.9\"><dict></dict></plist>", bareExtensions())
	if err == nil {
		t.Fatal("expected an error for a plist with no <plist version=\"1.0\"> anchor, got nil")
	}
}

func TestInsertDocumentTypes_ErrorsWhenDictMissingAfterAnchor(t *testing.T) {
	_, err := insertDocumentTypes(`<plist version="1.0"></plist>`, bareExtensions())
	if err == nil {
		t.Fatal("expected an error for a plist with no <dict> after the anchor, got nil")
	}
}

func TestInsertDocumentTypes_ErrorsWhenAlreadyPresent(t *testing.T) {
	first, err := insertDocumentTypes(fyneTemplateFixture, bareExtensions())
	if err != nil {
		t.Fatalf("first insertDocumentTypes: %v", err)
	}

	if _, err := insertDocumentTypes(first, bareExtensions()); err == nil {
		t.Fatal("expected an error inserting into a plist that already has CFBundleDocumentTypes, got nil")
	}
}
