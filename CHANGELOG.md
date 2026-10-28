# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.11.1] - 2026-10-22

### Fixed

- **npm**: `@b4moss/ss-cachian@0.11.0` could not be installed (`ENOENT` on
  `file:vendor/b4moss-es4-0.10.0.tgz`). Public dependencies now resolve
  `@b4moss/es4@0.10.0` from the npm registry. The temporary `vendor/` tarball
  pin was removed.
- **Go**: `go get github.com/b4moss/ss-cachian@v0.11.0` succeeded but the module
  zip contained no `.go` sources (nested `go/sscachian/go.mod` is excluded from
  a root-tagged module). The module path is now
  `github.com/b4moss/ss-cachian/go/sscachian`, released with tag
  `go/sscachian/v0.11.1` (same pattern as `b4moss/es4` → `packages/go/v*`).

### Changed

- Go import paths: `github.com/b4moss/ss-cachian/...` →
  `github.com/b4moss/ss-cachian/go/sscachian/...`.
- CI: `npm pack` → empty-dir install + reject `file:`/`link:`/`workspace:`
  dependency specs; Go consumer replace-based import/build check.

### Release notes (operators)

Do **not** use root tag alone for Go consumers. Cut **both**:

1. `v0.11.1` on `main` — npm Trusted Publisher (`npm-publish.yml`)
2. `go/sscachian/v0.11.1` on the same commit — Go module proxy

Consumer install:

```bash
npm install @b4moss/ss-cachian@0.11.1
go get github.com/b4moss/ss-cachian/go/sscachian@v0.11.1
```

## [0.11.0] - 2026-10-12

### Added

- es4 Layer Driver (embedded open + HTTP adapter) with Go / Node parity.

### Known issues (superseded by 0.11.1)

- npm package depended on a non-packaged `file:vendor/...` tarball.
- Go module path / tag layout did not match the `go/sscachian/` subdirectory.
