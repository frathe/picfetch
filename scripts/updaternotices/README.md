# Updater notices

Run from the repository root:

```sh
make check-updater-notices
make generate-updater-notices
go test -race ./scripts/updaternotices
go run ./scripts/updaternotices -artifact path/to/picfetch-windows-amd64.zip
go run ./scripts/updaternotices -artifact path/to/picfetch-microsoft-store.msixbundle
```

The source check runs `go list -mod=readonly -tags=no_emoji -deps -json
./internal/update` with cgo enabled for Darwin/Linux/Windows on amd64/arm64.
It excludes tests and build tools. The reviewed manifest pins both module versions
and the union of selected package paths: an existing module's newly imported
package can introduce a different file license. Replacements require a fresh
source review rather than silently inheriting the original module's notices.
Each module's `source` must be its exact `https://proxy.golang.org` ZIP URL for
the resolved module path and version, using Go's uppercase-letter escaping.
Changing a version requires updating that source URL even when its license
files are byte-identical. Historical `files[].source` supplements retain their
separately reviewed upstream URLs.

After a dependency or import change, review the selected source files and every
applicable root/nested license, NOTICE, copyright file and embedded license
header. Update `manifest.json` only after that review. `files[].sha256` covers
the **entire original file**, even when `start`/`end` select an inclusive,
one-based line range for reproduction. `applies_to` records other selected files
covered by the same header. `repository: true` denotes a retained historical
license with its own exact `source` URL, rather than a file in the module cache.
Then regenerate and review the Markdown diff. The generator shares byte-identical
texts and preserves all content outside its two updater markers.
Inline `#updater-text-...` links in the generated section, including manifest
notes, must target a license heading emitted from the current inventory. Both
checking and regeneration reject missing targets, so a shared-license link
must be reviewed when its last matching source text changes or is removed.

The artifact check compares `LICENSE`, `THIRD-PARTY-NOTICES.md` and `PRIVACY.md`
byte for byte with the checkout. macOS release ZIPs use
`PicFetch.app/Contents/Resources/`; Windows ZIPs, Linux tarballs and MSIX payloads
use the archive root. An MSIX bundle must contain `picfetch-x64.msix` and
`picfetch-arm64.msix`; each is independently checked. Missing, stale, duplicate,
misplaced or non-regular notices fail. Archive inspection reads without extraction
and does not validate code signatures, executable behavior or Windows SDK schemas.

CI checks source coverage. Release checks all six final archives after Windows
signing and before publication; Store packaging checks the final signed bundle
before recording its provenance or uploading it. Artifact mode intentionally
needs only the Go standard library and checkout notices, not a module-cache
license audit on the signing/packaging runner.

The [implementation evidence](../../plans/2026-09-13-updater-notices.md) records
the initial inventory and the inferred Python pathspec ancestry. The retained
CPython license is a contemporaneous reference, not a claim that its exact
release was consumed. No dependency or verifier behavior changed.
