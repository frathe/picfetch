package main

import (
	"fmt"
	"strings"

	"github.com/frathe/picfetch/internal/imaging"
)

// utiMapping pairs one supported image extension (lowercase, no leading
// dot) with its Uniform Type Identifier. Order here is the order
// contentTypeList renders LSItemContentTypes in - grouped by format family
// (common raster, then vector, then camera RAW) rather than
// imaging.SupportedExtensions()'s own order, since that reads better in a
// hand-maintained Info.plist entry than an extension-alphabetical one
// would.
//
// contentTypeUTIs is hand-maintained, not derived, because Go has no UTI
// database and no single UTI covers every camera-RAW vendor. An extension
// with no stable public UTI (xpm) simply has no entry here; it still
// matches via CFBundleTypeExtensions, which macOS falls back to for files
// with no attached type metadata. public.webp (macOS 11+) and the older
// org.webmproject.webp are both kept for webp, so older files carrying only
// the pre-public.webp type metadata still match.
type utiMapping struct {
	ext string
	uti string
}

var contentTypeUTIs = []utiMapping{
	{"jpg", "public.jpeg"},
	{"jpeg", "public.jpeg"},
	{"jpe", "public.jpeg"},
	{"jfif", "public.jpeg"},
	{"png", "public.png"},
	{"gif", "com.compuserve.gif"},
	{"webp", "public.webp"},
	{"webp", "org.webmproject.webp"},
	{"bmp", "com.microsoft.bmp"},
	{"tif", "public.tiff"},
	{"tiff", "public.tiff"},
	{"ico", "com.microsoft.ico"},
	{"heic", "public.heic"},
	{"heif", "public.heif"},
	{"avif", "public.avif"},
	{"svg", "public.svg-image"},
	{"raw", "public.camera-raw-image"},
	{"cr2", "com.canon.cr2-raw-image"},
	{"cr3", "com.canon.cr3-raw-image"},
	{"nef", "com.nikon.raw-image"},
	{"nrw", "com.nikon.nrw-raw-image"},
	{"arw", "com.sony.arw-raw-image"},
	{"dng", "com.adobe.raw-image"},
	{"orf", "com.olympus.raw-image"},
	{"rw2", "com.panasonic.rw2-raw-image"},
	{"raf", "com.fuji.raw-image"},
	{"pef", "com.pentax.raw-image"},
	{"srw", "com.samsung.raw-image"},
}

// contentTypeList flattens contentTypeUTIs into the LSItemContentTypes
// array: each UTI once, in first-seen order, since several extensions
// (jpg/jpeg/jpe/jfif, tif/tiff) share one UTI.
func contentTypeList() []string {
	seen := make(map[string]bool, len(contentTypeUTIs))
	out := make([]string, 0, len(contentTypeUTIs))

	for _, m := range contentTypeUTIs {
		if seen[m.uti] {
			continue
		}
		seen[m.uti] = true
		out = append(out, m.uti)
	}

	return out
}

// bareExtensions is imaging.SupportedExtensions() with the leading dot
// stripped from each entry - the form CFBundleTypeExtensions needs.
func bareExtensions() []string {
	exts := imaging.SupportedExtensions()
	out := make([]string, len(exts))

	for i, e := range exts {
		out[i] = strings.TrimPrefix(e, ".")
	}

	return out
}

// documentType is one CFBundleDocumentTypes <dict> entry. extensions is nil
// for a type with no CFBundleTypeExtensions key (Folder: directories have
// no extension).
type documentType struct {
	name         string
	extensions   []string
	contentTypes []string
}

// dictLines renders one documentType as a <dict>...</dict> block, indented
// so the <dict> tag itself sits at indent and its keys one tab deeper.
func dictLines(indent string, dt documentType) []string {
	inner := indent + "\t"

	lines := []string{
		indent + "<dict>",
		inner + "<key>CFBundleTypeName</key>",
		inner + "<string>" + dt.name + "</string>",
		inner + "<key>CFBundleTypeRole</key>",
		inner + "<string>Viewer</string>",
		inner + "<key>LSHandlerRank</key>",
		inner + "<string>Alternate</string>",
	}

	if dt.extensions != nil {
		lines = append(lines, stringArrayLines(inner, "CFBundleTypeExtensions", dt.extensions)...)
	}
	lines = append(lines, stringArrayLines(inner, "LSItemContentTypes", dt.contentTypes)...)
	lines = append(lines, indent+"</dict>")

	return lines
}

// stringArrayLines renders one <key>/<array> pair of <string> values,
// indented at indent.
func stringArrayLines(indent, key string, values []string) []string {
	lines := []string{
		indent + "<key>" + key + "</key>",
		indent + "<array>",
	}

	for _, v := range values {
		lines = append(lines, indent+"\t<string>"+v+"</string>")
	}

	lines = append(lines, indent+"</array>")

	return lines
}

// renderDocumentTypes renders the full CFBundleDocumentTypes block -
// <key>CFBundleDocumentTypes</key> plus its <array> of two <dict> entries,
// Image and Folder - tab-indented to match the depth of a key sitting
// directly inside Info.plist's top-level <dict> (see insertDocumentTypes).
// extensions is normally bareExtensions(); the caller passes it explicitly
// so tests can render a golden block from a small, fixed list.
//
// The Folder entry needs no Go-side handling beyond this declaration:
// handleDrop already scans directories recursively, so a folder dropped on
// the Dock icon works once macOS knows PicFetch can open one.
func renderDocumentTypes(extensions []string) string {
	lines := []string{
		"\t<key>CFBundleDocumentTypes</key>",
		"\t<array>",
	}

	lines = append(lines, dictLines("\t\t", documentType{
		name:         "Image",
		extensions:   extensions,
		contentTypes: contentTypeList(),
	})...)
	lines = append(lines, dictLines("\t\t", documentType{
		name:         "Folder",
		contentTypes: []string{"public.folder"},
	})...)

	lines = append(lines, "\t</array>")

	return strings.Join(lines, "\n")
}

// plistAnchor is the opening tag insertDocumentTypes looks for; the first
// <dict> after it is Info.plist's top-level dictionary.
const plistAnchor = `<plist version="1.0">`

// insertDocumentTypes inserts renderDocumentTypes's block right after the
// first <dict> following plistAnchor in plist - a targeted text insertion,
// not a parse-and-re-emit and not a plutil shell-out, so fyne's own
// Info.plist output survives byte-for-byte apart from this one insertion.
// It errors instead of guessing if the anchor is missing or
// CFBundleDocumentTypes is already present, so a future fyne template
// change fails make package-mac loudly rather than silently producing a
// broken bundle.
func insertDocumentTypes(plist string, extensions []string) (string, error) {
	if strings.Contains(plist, "<key>CFBundleDocumentTypes</key>") {
		// Capitalised because the string starts with the file's proper name,
		// which is the one case the lowercase-error-string rule allows.
		//goland:noinspection GoErrorStringFormat
		return "", fmt.Errorf("Info.plist already has a CFBundleDocumentTypes entry")
	}

	anchorAt := strings.Index(plist, plistAnchor)
	if anchorAt == -1 {
		return "", fmt.Errorf("could not find %q in Info.plist", plistAnchor)
	}

	dictAt := strings.Index(plist[anchorAt:], "<dict>")
	if dictAt == -1 {
		return "", fmt.Errorf("could not find <dict> after %q in Info.plist", plistAnchor)
	}

	insertAt := anchorAt + dictAt + len("<dict>")
	block := "\n" + renderDocumentTypes(extensions)

	return plist[:insertAt] + block + plist[insertAt:], nil
}
