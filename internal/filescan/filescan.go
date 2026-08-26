// Package filescan gathers displayable image files from a dropped or opened
// set of paths. Images recurses into directories; Siblings lists only the
// parent directory of a single opened file.
//
// It sits beside internal/filesort and internal/imaging rather than under
// internal/ui because it draws nothing and knows about no widget - it walks
// fyne.URI values as plain data, touching only the filesystem, and hands
// back a flat slice plus a truncation flag for a caller (internal/ui's
// handleDrop) to turn into a file set and a UI update. That placement is
// what makes this package's two invariants - the traversal order
// filesort.Order's ByDropOrder mode preserves verbatim, and the
// mime-sniff avoidance imaging.IsSupportedImage's own doc comment warns
// about - provable with plain, fast tests instead of a full viewer, a Fyne
// test app, and drain machinery.
package filescan

import (
	"context"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
)

// DefaultMax caps how many images a single call to Images will gather. It's
// a safety valve for pathological trees (a runaway symlink cycle
// EvalSymlinks doesn't resolve to a repeat, or a genuinely enormous
// archive) - past this, stat-ing and holding URIs would stall the caller's
// scan and bloat its resulting file set well past anything the rest of the
// pipeline is meant to handle. Callers with a smaller, user-configurable
// cap (the viewer's settings.maxScan field, which tests shrink per-viewer
// instead of creating hundreds of thousands of temp files to exercise the
// cap) pass that instead.
const DefaultMax = 200_000

// realPathOf resolves u's filesystem path through any symlinks, falling
// back to the URI's own path if that fails (a broken symlink, or a
// filesystem race between the scan and something else touching the same
// path) so callers always have something to key a visited-set on.
func realPathOf(u fyne.URI) string {
	symlinkPath, err := filepath.EvalSymlinks(u.Path())
	if err != nil {
		return u.Path()
	}
	return symlinkPath
}

// Images walks uris and returns every supported image found, recursing into
// directories. truncated reports that the walk stopped at max rather than
// exhausting the tree, so len(images) == max exactly when truncated is
// true. max is floored at 1 - a 0 (or negative) cap is not "unlimited", it
// stops the walk right after its first image, mirroring SetMaxScan's own
// floor.
//
// progress, if non-nil, is called synchronously on the calling goroutine
// with the running image count, throttled to n == 1, every 10th image, and
// the final call when truncation happens - marshaling that onto a UI
// goroutine (fyne.Do or otherwise) is the caller's job, not this package's.
//
// ctx is checked before considering each candidate URI and again at the top
// of each directory pop; on cancellation, Images returns whatever it has
// gathered so far rather than a fully correct result - a caller that
// discards a superseded scan's result anyway (as internal/ui's handleDrop
// does) only needs the walk to stop touching the filesystem promptly, not
// to finish correctly.
func Images(ctx context.Context, uris []fyne.URI, max int, progress func(n int)) (images []fyne.URI, truncated bool) {
	if max < 1 {
		max = 1
	}

	// dirs is a LIFO stack, not a queue: the pop below always takes the
	// most recently pushed directory. That produces a specific traversal
	// order - dropped URIs first in argument order, then each directory's
	// own children depth-first before its siblings - which is exactly what
	// filesort.Order's ByDropOrder mode ("stupid sort") preserves verbatim.
	// A queue would silently reorder that mode for every user.
	dirs := make([]fyne.URI, 0, len(uris))
	count := 0

	// visitedDirs guards against symlink cycles (e.g. a symlink inside a
	// dropped folder pointing back at one of its own ancestors), which
	// would otherwise send this walk into an unbounded loop: each visit
	// resolves the directory to its real, symlink-free path and only
	// descends into a given real path once. A plain map of dropped URIs
	// wouldn't catch this - a cycle keeps producing new, ever-longer URIs
	// (a/link, a/link/link, ...) that all resolve to the same real
	// directory.
	visitedDirs := make(map[string]bool)
	visitDir := func(u fyne.URI) bool {
		pathOf := realPathOf(u)
		if visitedDirs[pathOf] {
			return false
		}
		visitedDirs[pathOf] = true
		return true
	}

	// seenFiles dedupes images within this one call, keyed the same way as
	// visitedDirs: passing a folder together with one of its own
	// subfolders, or a symlinked file reachable via two different
	// directory paths, would otherwise add the same picture to images
	// twice. This is scoped to a single call, not persisted across calls -
	// a caller that merges results across drops (internal/ui's merge mode)
	// already allows re-adding a file that's already loaded.
	seenFiles := make(map[string]bool)

	process := func(u fyne.URI) {
		if truncated || ctx.Err() != nil {
			return
		}

		// Checked before IsSupportedImage so directories - which have no
		// extension and would otherwise fall through to MimeType()'s
		// open-and-sniff fallback - are recognized via a cheap stat
		// instead of a wasted file open.
		if canList, err := storage.CanList(u); err == nil && canList {
			if visitDir(u) {
				dirs = append(dirs, u)
			}
			return
		}

		if !imaging.IsSupportedImage(u) {
			return
		}

		pathOf := realPathOf(u)
		if seenFiles[pathOf] {
			return
		}
		seenFiles[pathOf] = true

		images = append(images, u)
		count++
		if count >= max {
			truncated = true
		}
		if progress != nil && (count == 1 || count%10 == 0 || truncated) {
			progress(count)
		}
	}

	for _, u := range uris {
		process(u)
	}

	for len(dirs) > 0 && !truncated {
		if ctx.Err() != nil {
			return images, truncated
		}
		d := dirs[len(dirs)-1]
		dirs = dirs[:len(dirs)-1]
		children, err := storage.List(d)
		if err != nil {
			continue
		}
		for _, child := range children {
			process(child)
		}
	}

	return images, truncated
}

// Siblings returns the supported images that share file's parent directory.
// It does not recurse into subdirectories. If file itself is a supported
// image it is always the first entry — the caller's URI, not a possibly
// different URI storage.List produced for the same path — so a caller that
// looks the opened file up by URI.String() still finds it after a sort.
// Directories among the children are skipped. max is floored at 1, the same
// as Images; truncated means the listing stopped at max rather than
// exhausting the directory. On Parent/List failure the result is just file
// when it is a supported image, otherwise empty. ctx is checked before any
// work and before each child; an already-cancelled context returns nil,
// false rather than a partial directory.
func Siblings(ctx context.Context, file fyne.URI, max int, progress func(n int)) (images []fyne.URI, truncated bool) {
	if max < 1 {
		max = 1
	}
	if ctx.Err() != nil {
		return nil, false
	}

	origin := realPathOf(file)
	seen := make(map[string]bool)
	count := 0
	add := func(u fyne.URI) {
		if truncated || ctx.Err() != nil {
			return
		}
		if canList, err := storage.CanList(u); err == nil && canList {
			return
		}
		if !imaging.IsSupportedImage(u) {
			return
		}
		pathOf := realPathOf(u)
		if seen[pathOf] {
			return
		}
		seen[pathOf] = true
		if pathOf == origin {
			u = file
		}
		images = append(images, u)
		count++
		if count >= max {
			truncated = true
		}
		if progress != nil && (count == 1 || count%10 == 0 || truncated) {
			progress(count)
		}
	}

	if imaging.IsSupportedImage(file) {
		add(file)
	}
	if truncated {
		return images, truncated
	}

	parent, err := storage.Parent(file)
	if err != nil {
		return images, truncated
	}
	children, err := storage.List(parent)
	if err != nil {
		return images, truncated
	}
	for _, child := range children {
		add(child)
	}
	return images, truncated
}
