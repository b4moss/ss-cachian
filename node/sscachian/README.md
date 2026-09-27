# @b4moss/ss-cachian (Node.js)

TypeScript / JavaScript port of [ss-cachian](https://github.com/b4moss/ss-cachian).

Semantics match the Go package at `go/sscachian` (**v0.10.0**): versioned keys, multilayer
write-back, Exact/Prefix/Tag Purge, convenience APIs, Memory / Firestore drivers, and
**config-driven** `loadTypes`.
Specs: [docs/specs](../../docs/specs/). Config schema notes: [docs/specs/config](../../docs/specs/config/).

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

---

## Config-driven (v0.9.0)

Build the same Cache Types as `define`…`build` from **YAML (canonical) or equivalent JSON**.
Loader / custom KeyBuilder **implementations stay in code**; the file only stores **registered names**.
Do not put type parameter `T` in the file (attach it in TypeScript).

Shared host guide (English): [root README — Config file](../../README.md#config-file-yaml--json)  
Full fixture: [`docs/plans/v0.9.0/sscachian.example.yaml`](../../docs/plans/v0.9.0/sscachian.example.yaml)

### Bootstrapping

```ts
import { createRegistry, loadTypes } from "@b4moss/ss-cachian";

const reg = createRegistry();

// Bind functions to the names referenced in YAML
reg.registerLoader("load_client_list", async (kc) => {
  return { items: [] };
});
reg.registerKeyBuilder("report_v2", (kc) => {
  return `report:${kc.tenantId}:${kc.queryType}`;
});

const types = await loadTypes(reg, "./sscachian.yaml");
// Map<string, CacheType>
const cache = types.get("client_list");
if (!cache) throw new Error("missing type");

const kc = { appSlug: "app", tenantId: "t1", queryType: "client_list_page1" };
const value = await cache.getOrLoad(kc);
```

YAML / JSON buffers:

```ts
import { loadTypesYAML, loadTypesJSON } from "@b4moss/ss-cachian";

const types = await loadTypesYAML(reg, yamlText);
const types2 = await loadTypesJSON(reg, jsonText);
```

Go uses the same schema (`NewRegistry` / `LoadTypes`). Blank-import `driver/memory` and `driver/firestore` first.

### Minimal YAML

```yaml
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default   # optional; defaults to "default"
    layers:
      - driver: memory
        ttl: 5m
```

### Field reference

#### Root

| Key | Required | Description |
| --- | --- | --- |
| `schema_version` | yes | String `"0.9"` only (quote it; numeric `0.9` is rejected) |
| `types` | yes | Non-empty array; duplicate `name` rejected |

#### `types[]`

| Key | Required | Description |
| --- | --- | --- |
| `name` | yes | Key in the loaded `Map` (`types.get(name)`) |
| `key_builder` | no | Registry name; omit → `"default"` |
| `loader` | no | Registry name; omit → `getOrLoad` yields `ErrNoLoader` on miss |
| `ttl` | no | Same TTL for all layers (`withLayerTTL` / `withPolicy`) |
| `layers` | yes | One or more; top entry is L1 |

#### `layers[]`

| Key | Required | Description |
| --- | --- | --- |
| `driver` | yes | `memory` or `firestore` in v0.9.0; unknown rejected |
| `ttl` | no | Per-layer override |
| `options` | no | Driver-specific; string values only |

### TTL format and precedence

Durations are **Go `time.ParseDuration` strings** (not bare second numbers).

Examples: `"300ms"` / `"1s"` / `"5m"` / `"1h"` / `"24h"` / `"1h30m"`

Per layer:

1. `layers[i].ttl` if set  
2. else `types[].ttl` if set  
3. else `0` (no expiry)

```yaml
# L1 = 1m, L2 = type-level 30m
- name: session_blob
  ttl: 30m
  layers:
    - driver: memory
      ttl: 1m
    - driver: firestore
```

Negative TTL (e.g. `-1s`) or unparsable strings (e.g. `abc`) fail the load.

### `key_builder` / `loader` (name references)

| Config | Meaning |
| --- | --- |
| `key_builder: default` or omitted | Built-in `{appSlug}:cache:{tenantId}:{queryType}` |
| `key_builder: report_v2` | Requires `registerKeyBuilder("report_v2", fn)` |
| `loader: load_client_list` | Requires `registerLoader("load_client_list", fn)` |
| no `loader` | Miss on `getOrLoad` → `ErrNoLoader` (`get` / `set` still work) |

Same-name re-register is **last-wins**. Already-built Cache Types keep their snapshot if you mutate the Registry later.

### Drivers and options

#### `memory`

No `options` (omit or empty). Unknown keys rejected.

```yaml
layers:
  - driver: memory
    ttl: 5m
```

#### `firestore`

Allowed keys only (both optional; Driver defaults apply when omitted):

| options key | Default |
| --- | --- |
| `project_id` | `ss-cachian-dev` |
| `collection` | `sscachian` |

```yaml
layers:
  - driver: memory
    ttl: 1m
  - driver: firestore
    ttl: 1h
    options:
      project_id: my-gcp-project
      collection: sscachian_app
```

**Do not put the Emulator in the file.** Set `FIRESTORE_EMULATOR_HOST` (e.g. `127.0.0.1:8080`).
Unknown options such as `emulator_host` are rejected. Do not embed credentials in YAML.

### Multilayer example (L1 memory + L2 Firestore + loader)

```yaml
schema_version: "0.9"
types:
  - name: client_list
    key_builder: default
    loader: load_client_list
    layers:
      - driver: memory
        ttl: 1m
      - driver: firestore
        ttl: 1h
        options:
          project_id: ss-cachian-dev
          collection: sscachian
```

Semantics match code-built types (upper hit skips lower layers, write-back, Exact Purge, …). See [docs/specs](../../docs/specs/).

### JSON is fine too

```json
{
  "schema_version": "0.9",
  "types": [
    {
      "name": "user_profile",
      "key_builder": "default",
      "layers": [{ "driver": "memory", "ttl": "5m" }]
    }
  ]
}
```

### Common rejection reasons

- Missing / `"0.8"` / numeric `schema_version`
- Empty `types`, empty/duplicate `name`, empty `layers`
- Unknown `driver` (e.g. `valkey` — later milestone)
- Unregistered `key_builder` / `loader`
- Non-string or invalid/negative `ttl`
- Unknown `options` keys or non-string option values

### Out of scope (v0.10.0)

- Valkey / PHP / browser bundles
- SWR · SIE (later)
- Secrets embedded in the config file

---

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
