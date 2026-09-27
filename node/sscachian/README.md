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

Tags `v*` on `main` trigger `.github/workflows/npm-publish.yml` (`npm publish`).
Version is taken from the tag (`v0.8.0` → `0.8.0`). Requires repo secret `NPM_TOKEN` (npm Granular Access Token).
