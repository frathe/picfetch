## What's Changed

### Bugfix

- Spiral: avoid an identifier rejected by Intel's Windows GLSL compiler.
  v1.1.0 compiles, v1.1.1 fails on `active`, and the corrected shader compiles
  on the same Iris Xe driver. Shader tests, Docker build checks/full vet and
  the complete Linux Spiral race package pass. See
  [the diagnosis and verification limits](docs/spiral-windows-2026-09-13.md).

### Internal

- Microsoft Store `release` mode is implemented: reconcile the prior release and
  conditionally submit the frozen current release under one approval, retaining
  standalone `reconcile` and `submit`. CI and documentation are consolidated onto
  the Spiral bugfix branch. Windows tooling tests/vet, native Linux/amd64 tooling
  race checks and the repository vet/build gate pass. PR #23 CI also passes the
  full Linux race suite, Windows tests and both macOS native guards on `3011ea5`;
  an approved live Store run remains. See the
  [implementation plan](plans/2026-09-13-store-combined-release.md).

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.1.1...v1.1.2
