# Go 1.27.1 and security dependency refresh

Status: complete

Route: Standard. This changes the module graph, project tooling, contributor
requirements, and release notes without changing application behavior or
package architecture.

Deliverable: PicFetch requires Go 1.27.1, uses Rekor's maintained OpenPGP
implementation, and runs a repository-pinned govulncheck successfully.

## Locked decisions

| Decision | Contract |
|---|---|
| Go baseline | `go.mod` and active setup documentation require Go 1.27.1. |
| OpenPGP remediation | Upgrade Rekor to v1.5.4 and accept only dependency changes required by that patch and module tidying. |
| Scanner | Pin `golang.org/x/vuln` v1.7.0 as a Go tool and invoke it with `go tool govulncheck`. |
| Residual advisory | Keep latest `golang.org/x/crypto` because PicFetch needs unaffected packages; do not suppress GO-2026-5932 or replace Sigstore. |
| Scope | No application code, package layout, architecture record, workflow, or historical plan changes. |

## Tasks

### Task 1 - Refresh the module and security tool graph

Owner: T0 inline

Files: `go.mod`, `go.sum`

Upgrade the Go directive and Rekor, add the pinned govulncheck tool dependency,
and run `go mod tidy`.

Verify: `go mod tidy -diff && go mod verify && go mod why golang.org/x/crypto/openpgp`

### Task 2 - Use the pinned scanner

Owner: T0 inline

Files: `Makefile`

Run govulncheck through `go tool` and remove its redundant global install.

Verify: `make security-govulncheck`

### Task 3 - Align active documentation and land the change

Owner: T0 inline

Files: `README.md`, `.github/CONTRIBUTING.md`, `todos.md`, this plan

Document the Go baseline, pinned scanner workflow, Rekor remediation, and the
accepted module-only advisory.

Verify: `make verify`

## Budget and gate

Zero spawns; at most two review rounds; one full suite. The final security gate
also runs `go tool govulncheck -show verbose ./...` and confirms that
GO-2026-5932 remains module-only with no imported package or reachable symbol.

## Outcome

PicFetch now requires Go 1.27.1. Rekor v1.5.4 removes the unmaintained
`x/crypto/openpgp` package from the dependency path, while govulncheck v1.7.0
is pinned beside goimports and runs through `go tool` without a global binary.

The scanner reports no reachable or imported vulnerabilities. GO-2026-5932
remains visible only at module granularity because unaffected packages from the
latest `x/crypto` module are still required and the advisory has no fixed
version.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 1 | no | Go, Rekor, govulncheck, and the tidied module graph complete. |
| T2 | 0 / 0 | 1 | no | Pinned scanner target and packaging-tool install contract complete. |
| T3 | 0 / 0 | 1 | no | Active setup docs and release notes aligned. |
| gate | - | - | - | yes | `make verify` passed. |

## Verification record

- `go mod tidy -diff` produced no diff, `go mod verify` passed, and `go mod
  why golang.org/x/crypto/openpgp` reported that the main module does not need
  the package.
- `make security-govulncheck` passed with zero reachable vulnerabilities.
- `go tool govulncheck -show verbose ./...` reported zero symbol and package
  vulnerabilities and only GO-2026-5932 at module level (`Fixed in: N/A`).
- `make verify` passed formatting, TUF root validation, vet, build, and the
  Linux/amd64 race suite (`internal/ui` 686.351s; `internal/ui/compare`
  28.099s).
