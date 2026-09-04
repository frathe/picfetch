# Multilingual Website Generation — Implementation Map

Status: resolved

The acceptance contract is the public Makefile workflow. Tests invoke commands and
inspect generated pages; they do not depend on private generator structure.

| Ticket | Slice | Depends on | Verification |
| --- | --- | --- | --- |
| 01 | English regular page from `website.md` | — | `go test ./sitecontract -run 'TestMakeBuild.*EnglishRegular|TestInvalidSource'` |
| 02 | English AMP from the shared model | 01 | `go test ./sitecontract -run 'TestMakeBuild.*EnglishAMP|TestEnglishVariants'` and `make validate-amp` |
| 03 | Hash-keyed DeepL cache and atomic refresh | 01 | `go test ./sitecontract -run 'TestTranslate|TestBuildRejects.*Translation'` |
| 04 | German regular page and language behavior | 03 | `go test ./sitecontract -run 'TestGermanRegular|TestLanguageSelection'` |
| 05 | German AMP and the four-route metadata graph | 02, 04 | `go test ./sitecontract -run 'TestFourPage|TestMetadata|TestSelectors|TestLinks'` and `make validate-amp` |
| 06 | Offline stale-output, validation, and publishing guards | 05 | `make check` |

## Task graph and ownership

```text
01 English regular ─┬─> 02 English AMP ─────────┐
                    └─> 03 DeepL cache ─> 04 German regular ─┤
                                                            v
                                                    05 German AMP/matrix
                                                            v
                                                    06 workflow guards
```

- Lead: specification, architecture, all red/green slices, review, fixes, and final gate.
- Read-only scouts: preservation inventories and local tooling reconnaissance only.
- Generated files: `docs/index.html`, `docs/amp/index.html`,
  `docs/de/index.html`, and `docs/de/amp/index.html`.

## Final gate

```sh
gofmt -l .
go vet ./...
go build ./...
make check
git diff --check
```

The first production `make translate` additionally requires a maintainer-owned
`DEEPL_API_KEY`. Routine build and check commands remain offline and credential-free.

## Completion

The maintainer authorized the production DeepL refresh. The current cache contains
all 68 German translation units, all four deployment pages were regenerated, and
the full offline check passes. Contract coverage includes transactional failures,
unsafe HTML and paths, exact-route and stale-output detection, metadata/link
relationships, both AMP documents, language behavior, and blocked browser storage.
The final visual smoke covered all four routes at desktop and narrow widths in both
light and dark modes.
