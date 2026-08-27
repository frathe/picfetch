## What's Changed

### Bugfix

- Reject zip/tar entries that are not `filepath.IsLocal` during update extract (GitHub CodeQL go/zipslip).

### Internal

- Drop deprecated `tar.TypeRegA` in update extract (still accept the historic NUL regular-file typeflag).
- Install CI's Linux GUI packages in CodeQL so `internal/winpos/linux.go` can compile.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.8...v0.2.9
