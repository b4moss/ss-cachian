# CI policy (ss-cachian)

Scaffold aligned with `b4moss/crudian` and [development.md](../docs/development.md).

## Goals

- Lint + unit/integration on PRs (and push) for touched Go / Node packages.
- Path filter: only schedule jobs for changed areas (`go/**`, `node/**`).
- Ancestor skip (Go): same package + workflow identity already green → short-circuit.
- Docs-only changes: language jobs skipped; aggregate gate still succeeds.
- No product E2E in CI.

## npm publish

Separate workflow: [npm-publish.yml](./workflows/npm-publish.yml) on tag `v*`.
Requires `NPM_TOKEN` secret. See [development.md](../docs/development.md) CD (npm 暫定).

## Required status

Use the aggregate check **CI result** as the required status when branch protection is enabled.

## Local smoke

```bash
make act
```

`act` is wiring smoke only; gate logic correctness is covered later by `.github/tests/` when added.
