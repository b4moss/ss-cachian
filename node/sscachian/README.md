# @b4moss/ss-cachian (Node.js)

TypeScript / JavaScript port of [ss-cachian](https://github.com/b4moss/ss-cachian).

> **Status:** package scaffold for npm release path. Cache Type API is not implemented yet.
> Go PoC is complete at `go/sscachian` (**v0.7.0**). Specs: [docs/specs](../../docs/specs/).

## Install (after first publish)

```bash
npm install @b4moss/ss-cachian
```

## Develop

```bash
cd node/sscachian
npm ci
npm run lint
npm test
npm run build
```

## Publish

Tags `v*` on `main` (or Actions → **npm publish** → Run workflow) publish via
**npm Trusted Publisher (OIDC)**. See [development.md](../../docs/development.md).
