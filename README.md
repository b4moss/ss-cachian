# ss-cachian

[![CI](https://github.com/b4moss/ss-cachian/actions/workflows/ci.yml/badge.svg)](https://github.com/b4moss/ss-cachian/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/@b4moss/ss-cachian)](https://www.npmjs.com/package/@b4moss/ss-cachian)
[![Release](https://img.shields.io/github/v/release/b4moss/ss-cachian)](https://github.com/b4moss/ss-cachian/releases)
[![License](https://img.shields.io/github/license/b4moss/ss-cachian)](https://github.com/b4moss/ss-cachian/blob/main/LICENSE)

Server-side **cache strategy** library: declare a **Cache Type**, run it across one or more storage **Layers**.

Not a thin key/value wrapper. Applications define *how* caching works (keys, TTLs, multilayer write-back, versioned invalidation, Exact Purge); the library executes that strategy.

- [日本語版 README](./README-ja.md)

Current line: **v0.9.0** (Go `VERSION` and `@b4moss/ss-cachian` stay in lockstep for this release).

## What you get

| Capability | Notes |
|------------|--------|
| **Cache Type API** | `Define` / `define` → layers, key builder, TTL(s), optional loader → `Get` / `Set` / `Delete` / `GetOrLoad` / `Purge` |
| **Versioned keys** | Current-version on L1; mutations bump; `GetOrLoad` fill does **not** bump |
| **Multilayer** | L1 → L2 → … exploration; write-back on lower hit / loader success |
| **Exact Purge** | Delete all version data keys for a logical prefix; leave `__version__` alone |
| **Drivers** | **memory**, **Firestore** (honors `FIRESTORE_EMULATOR_HOST`) |
| **Config-driven (v0.9.0)** | YAML (canonical) or equivalent JSON → `LoadTypes` / `loadTypes` |

Out of scope today: SWR / SIE / negative cache, PurgePrefix/Tag, Valkey, PHP, browser bundles. See [roadmap](./docs/roadmap.md).

## Choose a language port

| Port | Package | Language hub |
|------|---------|--------------|
| **Go** | `github.com/b4moss/ss-cachian` | [`go/sscachian/README.md`](./go/sscachian/README.md) |
| **TypeScript** | `@b4moss/ss-cachian` | [`node/sscachian/README.md`](./node/sscachian/README.md) |
| **PHP** | — | Planned later — [roadmap](./docs/roadmap.md) |

This root README is the **shared host workflow** (concepts, config file, pitfalls, docs map). Install commands, import paths, and API surface details live in the language hubs.

---

## Prerequisites

| | Go | Node |
|--|----|------|
| Runtime | Go **1.26+** | Node.js **≥ 20** |
| Package | `github.com/b4moss/ss-cachian` (`VERSION=0.9.0`) | `@b4moss/ss-cachian@0.9.0` |
| Firestore (optional) | Emulator or GCP project | Same (`@google-cloud/firestore`) |

You do **not** need this monorepo cloned to consume published packages. Clone for contributing, Emulator CI, or editing docs.

---

## Host usage (step by step)

### Step 1 — Pick a port and install

```bash
# Go
go get github.com/b4moss/ss-cachian@v0.9.0

# Node
npm install @b4moss/ss-cachian
```

Hubs: [Go](./go/sscachian/README.md) · [Node](./node/sscachian/README.md)

### Step 2 — Decide code-built vs config-built Cache Types

| Path | When to use |
|------|-------------|
| **Code** `Define` / `define` | Few types, fully dynamic wiring, tests |
| **Config** YAML/JSON + Registry | Many types, ops-friendly TTLs/layers, shared schema across runtimes |

Both paths produce the same runtime semantics ([specs](./docs/specs/)).

### Step 3 — Register loaders / custom key builders (if used)

Config only stores **names**. Bind functions at process start:

```text
Registry
  ├─ key_builder "default"     (built-in)
  ├─ key_builder "report_v2"   ← your RegisterKeyBuilder
  └─ loader "load_client_list" ← your RegisterLoader
```

Missing names → load fails. Re-register is last-wins; types already loaded keep their snapshot.

### Step 4 — Write layers and TTLs

- Order in `layers` is L1, L2, … (exploration and write-back follow that order).
- TTL string format: Go `time.ParseDuration` (`5m`, `1h`, `300ms`, …). Not bare numbers.
- Precedence per layer: `layers[i].ttl` → type-level `ttl` → `0` (no expiry).

### Step 5 — Wire Firestore (optional)

- Config `options`: `project_id`, `collection` only (unknown keys rejected).
- **Do not** put emulator host in YAML. Set `FIRESTORE_EMULATOR_HOST` (e.g. `127.0.0.1:8080`).
- Never embed secrets in the config file.

### Step 6 — Call the Cache Type

Conceptually:

```text
Get(kc)           → latest version key across layers; value only
Set(kc, value)    → bump L1 version, write all layers
Delete(kc)        → delete latest on all layers, then bump
GetOrLoad(kc)     → Get; on miss call loader; write-back all layers; no bump
Purge(kc)         → Exact purge of version data keys; __version__ untouched
```

`kc` carries business context (`AppSlug` / `appSlug`, tenant, query type) for the key builder.

### Step 7 — Verify before release

```bash
make lint test          # Go (+ Firestore Emulator)
make lint-node test-node
```

---

## Config file (YAML / JSON)

Canonical format is **YAML**; JSON with the same tree is accepted (`.json` or `LoadTypesJSON` / `loadTypesJSON`).

Full example: [`docs/plans/v0.9.0/sscachian.example.yaml`](./docs/plans/v0.9.0/sscachian.example.yaml)  
Formal notes: [`docs/specs/config/`](./docs/specs/config/)  
Node-oriented narrative: [`node/sscachian/README.md`](./node/sscachian/README.md#config-driven-v090)

### Minimal file

```yaml
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default   # optional; default if omitted
    layers:
      - driver: memory
        ttl: 5m
```

### Field reference

#### Root

| Key | Required | Description |
|-----|----------|-------------|
| `schema_version` | yes | String `"0.9"` only (quote it; numeric `0.9` is rejected) |
| `types` | yes | Non-empty array; `name` must be unique |

#### `types[]`

| Key | Required | Description |
|-----|----------|-------------|
| `name` | yes | Map / `Map` key after load |
| `key_builder` | no | Registry name; default `"default"` → `{app}:cache:{tenant}:{query}` |
| `loader` | no | Registry name; omit → `GetOrLoad` returns `ErrNoLoader` on miss |
| `ttl` | no | Same TTL for every layer (overridden per layer if set) |
| `layers` | yes | One or more; index 0 is L1 |

#### `layers[]`

| Key | Required | Description |
|-----|----------|-------------|
| `driver` | yes | `memory` \| `firestore` (v0.9.0) |
| `ttl` | no | Layer override |
| `options` | no | Driver-specific string map |

### TTL precedence

1. `layers[i].ttl` if present  
2. else `types[].ttl` if present  
3. else `0` (no expiry)

Negative or unparsable TTL → load error.

### Drivers and options

**memory** — no options (unknown keys rejected).

**firestore**

| Option | Default |
|--------|---------|
| `project_id` | `ss-cachian-dev` |
| `collection` | `sscachian` |

Emulator: environment only (`FIRESTORE_EMULATOR_HOST`).

### Multilayer + loader example

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

### Load from code

**Go** (blank-import drivers so factories register):

```go
import (
  "context"

  "github.com/b4moss/ss-cachian"
  _ "github.com/b4moss/ss-cachian/driver/firestore"
  _ "github.com/b4moss/ss-cachian/driver/memory"
)

reg := sscachian.NewRegistry()
_ = reg.RegisterLoader("load_client_list", myLoader)
types, err := sscachian.LoadTypes[MyT](ctx, reg, "sscachian.yaml")
cache := types["client_list"]
```

**Node**:

```ts
import { createRegistry, loadTypes } from "@b4moss/ss-cachian";

const reg = createRegistry();
reg.registerLoader("load_client_list", async () => ({ items: [] }));
const types = await loadTypes(reg, "./sscachian.yaml");
const cache = types.get("client_list");
```

### Common config rejections

| Symptom | Likely cause |
|---------|----------------|
| Unsupported `schema_version` | Missing, `"0.8"`, or unquoted number |
| Empty / duplicate `name` | Fix `types[].name` |
| Unknown `driver` | Only `memory` / `firestore` in v0.9.0 |
| Unknown `key_builder` / `loader` | Register before `LoadTypes` |
| Invalid `ttl` | Must be duration string; negatives rejected |
| Unknown `options` key | e.g. `emulator_host` in YAML — use env instead |

---

## Common pitfalls

| Pitfall | Fix |
|---------|-----|
| Expecting config to embed loader source | Config holds names only; bind in Registry |
| Putting Emulator host in YAML | Use `FIRESTORE_EMULATOR_HOST` |
| Assuming `GetOrLoad` bumps version | Only `Set` / `Delete` bump; fill write-back does not |
| Forgetting Go driver imports | Blank-import `driver/memory` and `driver/firestore` before `LoadTypes` |
| Sharing one Firestore collection across tests | Use unique `collection` options per test |
| Treating cache as source of truth | Cache is a temporary read accelerator; DB remains authoritative |

---

## Develop in this repo

```bash
make lint test           # Go + Emulator
make lint-node test-node
make act                 # local CI smoke (Docker)
```

Devcontainer / Emulator: [`.devcontainer/`](./.devcontainer/) · [`docker/`](./docker/) · [`.github/CI.md`](./.github/CI.md)

Publish: tags `v*` on `main` → npm Trusted Publisher (OIDC). Go modules consume `v0.9.0` from the Git tag.

---

## Docs map

| Doc | Contents |
|-----|----------|
| [`docs/README.md`](./docs/README.md) | Product pillar (purpose, scope, tech policy) |
| [`docs/roadmap.md`](./docs/roadmap.md) | Milestones |
| [`docs/specs/`](./docs/specs/) | Current behavior (domain-oriented) |
| [`docs/specs/config/`](./docs/specs/config/) | Config load semantics |
| [`docs/plans/`](./docs/plans/) | Upcoming work |
| [`docs/plans/v0.9.0/sscachian.example.yaml`](./docs/plans/v0.9.0/sscachian.example.yaml) | Config fixture |
| [`docs/tests/`](./docs/tests/) | TDD acceptance specs |
| [`docs/charter/`](./docs/charter/) | Charter + OKF |
| [`.github/CI.md`](./.github/CI.md) | CI/CD / publish |
| [`go/sscachian/README.md`](./go/sscachian/README.md) | Go language hub |
| [`node/sscachian/README.md`](./node/sscachian/README.md) | Node language hub |

### Charter import

Charter body is not edited in-tree; pull from `b4moss/charter`:

```bash
git remote add charter https://github.com/b4moss/charter.git   # once
git fetch charter main
git checkout charter/main -- docs/charter
```

## License

MIT © Bicycle for Mind LLC. — see [`LICENSE`](./LICENSE).
