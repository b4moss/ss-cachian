# @b4moss/ss-cachian (Node.js)

TypeScript / JavaScript port of [ss-cachian](https://github.com/b4moss/ss-cachian).

Semantics match the Go package at `go/sscachian` (**v0.9.0**): versioned keys, multilayer
write-back, Exact Purge, Memory / Firestore drivers, and **config-driven** `loadTypes`.
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

## 設定ファイル駆動（v0.9.0）

`define`…`build` と同等の Cache Type を、**YAML（正）または同等 JSON** から組めます。
Loader / カスタム KeyBuilder の**実体はコードで登録**し、設定には**登録名だけ**を書きます。
値の型 `T` は設定に書きません（TypeScript 側で付けます）。

完全な雛形: [`docs/plans/v0.9.0/sscachian.example.yaml`](../../docs/plans/v0.9.0/sscachian.example.yaml)

### 起動コード

```ts
import { createRegistry, loadTypes } from "@b4moss/ss-cachian";

const reg = createRegistry();

// 設定の loader / key_builder 名に対応する実関数を登録
reg.registerLoader("load_client_list", async (kc) => {
  // DB 等から取得
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

YAML 文字列や JSON だけ渡す場合:

```ts
import { loadTypesYAML, loadTypesJSON } from "@b4moss/ss-cachian";

const types = await loadTypesYAML(reg, yamlText);
const types2 = await loadTypesJSON(reg, jsonText);
```

Go も同スキーマです（`NewRegistry` / `LoadTypes`）。Go では `driver/memory` と `driver/firestore` を import してファクトリを登録してください。

### 最小の YAML

```yaml
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default   # 省略時も default
    layers:
      - driver: memory
        ttl: 5m
```

### フィールド一覧

#### ルート

| キー | 必須 | 説明 |
| --- | --- | --- |
| `schema_version` | はい | 文字列 `"0.9"` のみ受理（クォート推奨。数値 `0.9` は拒否） |
| `types` | はい | Cache Type の配列。空は不可。`name` の重複不可 |

#### `types[]`

| キー | 必須 | 説明 |
| --- | --- | --- |
| `name` | はい | ロード結果のキー（`types.get(name)`） |
| `key_builder` | いいえ | Registry 上の名前。省略時 `"default"` |
| `loader` | いいえ | Registry 上の名前。未指定なら `getOrLoad` は `ErrNoLoader` |
| `ttl` | いいえ | **全 Layer 共通 TTL**（`withLayerTTL` / `withPolicy` 相当） |
| `layers` | はい | 1 本以上。上から L1, L2, … |

#### `layers[]`

| キー | 必須 | 説明 |
| --- | --- | --- |
| `driver` | はい | v0.9.0 は `memory` または `firestore`。未知は拒否 |
| `ttl` | いいえ | この Layer だけ上書き |
| `options` | いいえ | Driver 固有。文字列値のみ |

### TTL の書き方と優先順位

期間は **Go `time.ParseDuration` 互換の文字列**です（数値の秒などは不可）。

例: `"300ms"` / `"1s"` / `"5m"` / `"1h"` / `"24h"` / `"1h30m"`

優先順位（Layer ごと）:

1. その Layer の `layers[i].ttl` があればそれ
2. なければ Type の `types[].ttl`
3. どちらもなければ `0`（無期限）

```yaml
# 例: L1 は 1分、L2 は type 共通の 30分
- name: session_blob
  ttl: 30m
  layers:
    - driver: memory
      ttl: 1m
    - driver: firestore
```

負の TTL（例: `-1s`）やパース不能な文字列（例: `abc`）はロード時に拒否します。

### key_builder / loader（名前参照）

| 設定値 | 意味 |
| --- | --- |
| `key_builder: default` または省略 | 組み込み。`{appSlug}:cache:{tenantId}:{queryType}` |
| `key_builder: report_v2` | 事前に `registerKeyBuilder("report_v2", fn)` が必要 |
| `loader: load_client_list` | 事前に `registerLoader("load_client_list", fn)` が必要 |
| `loader` なし | Miss 時の `getOrLoad` は `ErrNoLoader`（`get` / `set` は可） |

同名の再登録は**後勝ち**です。ロード済みの Cache Type は、その後 Registry を変えても影響を受けません（構築時にスナップショット）。

### Driver と options

#### `memory`

- `options` は付けない（空または省略）。未知キーは拒否。

```yaml
layers:
  - driver: memory
    ttl: 5m
```

#### `firestore`

許可キーは次のみ（いずれも省略可。省略時は Driver 既定）:

| options キー | 既定 |
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

**Emulator は設定に書かない。** `FIRESTORE_EMULATOR_HOST`（例: `127.0.0.1:8080`）を環境変数で渡します。
`emulator_host` などの未知 options はロード拒否です。

認証情報・秘密鍵を YAML に埋め込まないでください。

### 多層の例（L1 memory + L2 Firestore + Loader）

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

意味論はコード組み立てと同じです（上位 hit で下位スキップ、書き戻し、Exact Purge など）。詳細は [docs/specs](../../docs/specs/)。

### JSON でも可

拡張子 `.json`、または `loadTypesJSON` で同じツリーを渡せます。

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

### よくある拒否理由

- `schema_version` 欠落・`"0.8"`・数値 `0.9`
- `types` 空 / `name` 空・重複 / `layers` 空
- 未知 `driver`（例: `valkey` — 後続マイルストーン）
- 未登録の `key_builder` / `loader` 名
- TTL が文字列でない、または不正・負
- `options` の未知キー、または非文字列値

### やらないこと（v0.9.0）

- Valkey / PHP / ブラウザ向けバンドル
- SWR・SIE・PurgePrefix/Tag（後続版）
- 設定ファイルへの秘密情報の埋め込み

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
