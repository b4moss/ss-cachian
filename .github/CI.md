# CI policy (ss-cachian)

Scaffold aligned with `b4moss/crudian` and [development.md](../docs/development.md).

## Goals

- Lint + unit/integration on PRs (and push) for touched Go packages.
- Path filter: only schedule jobs for changed areas.
- Ancestor skip: same package + workflow identity already green → short-circuit.
- Docs-only changes: Go jobs skipped; aggregate gate still succeeds.
- No product E2E in CI (PoC).

## Required status

Use the aggregate check **CI result** as the required status when branch protection is enabled.

## Local smoke

```bash
make act
```

`act` is wiring smoke only; gate logic correctness is covered later by `.github/tests/` when added.
