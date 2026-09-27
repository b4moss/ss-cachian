# @b4moss/ss-cachian (Node.js)

TypeScript / JavaScript port of [ss-cachian](https://github.com/b4moss/ss-cachian).

Semantics match the Go package at `go/sscachian` (**v0.7.0** PoC): versioned keys, multilayer
write-back, Exact Purge, Memory and Firestore drivers. Specs: [docs/specs](../../docs/specs/).

## Install

```bash
npm install @b4moss/ss-cachian
```

## Quick start

```ts
import { define, newMemoryStore, defaultKeyBuilder } from "@b4moss/ss-cachian";

const cache = define<string>("users")
  .withLayers(newMemoryStore())
  .withKeyBuilder(defaultKeyBuilder)
  .build();

const kc = { appSlug: "app", tenantId: "t1", queryType: "user" };
await cache.set(kc, "alice");
const got = await cache.get(kc); // { ok: true, value: "alice" }
```

Firestore (Emulator / GCP):

```ts
import { define, newFirestoreStore, newMemoryStore } from "@b4moss/ss-cachian";

const fs = await newFirestoreStore({ projectId: "my-project" });
const cache = define("users").withLayers(newMemoryStore(), fs).build();
```

## Develop

```bash
cd node/sscachian
npm ci
npm run lint
npm test
# Firestore tests need FIRESTORE_EMULATOR_HOST (CI starts the emulator)
```

## Publish

Tags `v*` on `main` (or Actions → **npm publish** → Run workflow) publish via
**npm Trusted Publisher (OIDC)**. See [.github/CI.md](../../.github/CI.md).
