---
type: Plan
title: v0.9.0 設定ファイル駆動
description: YAML/JSON から Cache Type（Layers / TTL / KeyBuilder 参照 / Policy）を組み立てる。Go / Node パリティ。
tags: [plan, phase2, config]
timestamp: 2026-09-27T06:28:00Z
---

# v0.9.0 設定ファイル駆動

- **状態:** 完了（タグ `v0.9.0` + npm）
- **マイルストーン:** v0.9.0（[roadmap](../../roadmap.md) Phase 2）
- **作業ブランチ:** `dev-v0.9.0` → `develop` → `main`
- **設定雛形（スキーマ固定案）:** [sscachian.example.yaml](./sscachian.example.yaml)
- **意味論正本（実装後）:** [specs](../../specs/) に `config` ドメインを追加する想定

## 目的

`Define` / `WithLayers` / `WithKeyBuilder` / `WithLayerTTL(s)` / `WithLoader` で組み立てている Cache Type を、**YAML（正）または同等 JSON** からロードできるようにする。Go / Node で同等セマンティクス。Loader・カスタム KeyBuilder の実体は引き続きコード登録とし、設定は **登録名の参照のみ**。

## やらぬこと

- Purge 拡張（Prefix/Tag）・便利 API（Has/Remember 等）→ v0.10.0
- SWR / SIE / negative / 分散 L1 → v0.11.0
- Valkey Driver → v0.12.0
- PHP ポート → v0.13.0
- ブラウザ向けバンドル
- 設定ファイルへの認証情報・秘密鍵の埋め込み
- ランタイム間共通シリアライズ形式の変更（現行どおり）
- コード面 Builder API の破壊的変更（設定ロードは追加面）

## スキーマ方針（雛形で固定）

| 項目 | 決定 |
| --- | --- |
| フォーマット | YAML 正。同等 JSON 可（同じ論理ツリー） |
| `schema_version` | `"0.9"`（破壊的変更時に上げる） |
| 複数 Type | 1 ファイルに `types:` 配列 |
| `types[].name` | 必須。ロード結果のマップキー |
| `key_builder` | 登録名。省略時は組み込み `"default"`（現行 `DefaultKeyBuilder`） |
| `loader` | 任意。登録名のみ。未指定なら `GetOrLoad` は `ErrNoLoader`（現行どおり） |
| `layers[].driver` | 必須。v0.9.0 は `memory` / `firestore` のみ |
| `layers[].ttl` | 任意。省略 = `0`（無期限） |
| `types[].ttl` | 任意。**全 Layer 共通 TTL**（`WithLayerTTL` / `WithPolicy` 相当）。`layers[].ttl` がある Layer はそちら優先 |
| TTL 文字列 | Go `time.ParseDuration` 互換（`300ms` / `1s` / `5m` / `1h` / `24h`）。Node も同一解釈 |
| `layers[].options` | Driver 固有。memory は空想定。firestore は `project_id` / `collection`（既定は現行 Driver と同じ） |
| Emulator | **設定に書かない**。`FIRESTORE_EMULATOR_HOST` 環境変数を正とする |
| 型パラメータ `T` | 設定に書かない。Go generics / TS ジェネリクスでロード時に付ける |

### TTL 優先順位（確定）

1. `layers[i].ttl` があればその値
2. なければ `types[].ttl`（あれば）
3. どちらもなければ `0`

負 TTL・不正 duration 文字列はロード拒否（Build 時の負 TTL 拒否と同型）。

### エラー方針（概要）

ロード時に拒否する（呼び出し側で Recover しない想定）:

- 未対応 / 不正 `schema_version`
- `types` 空、`name` 重複・空
- `layers` 空、未知 `driver`
- 未登録の `key_builder` / `loader` 名
- 不正 TTL 文字列、負 TTL
- firestore / memory `options` の未知キーは **拒否**（[tests/config](../../tests/config/) で固定）

## API 面（追加）

コード面の `Define` 系は残す。設定駆動は並列の入口。

### Go（案）

```go
reg := sscachian.NewRegistry()
reg.RegisterLoader("load_client_list", myLoader)
reg.RegisterKeyBuilder("report_v2", myKB)

types, err := sscachian.LoadTypes[MyT](ctx, reg, "sscachian.yaml")
// types["client_list"] → *CacheType[MyT]
```

- `Registry`: Loader / KeyBuilder の名前→関数。組み込み `"default"` は Registry 作成時に登録済みでよい
- `LoadTypes`: パスまたは `io.Reader`。YAML/JSON は拡張子または内容で判定
- Driver 生成はライブラリ内（`memory.New` / `firestore.New`）。アプリは Options を YAML に書くだけ

### Node（案）

```ts
const reg = createRegistry();
reg.registerLoader("load_client_list", myLoader);
reg.registerKeyBuilder("report_v2", myKB);

const types = await loadTypes(reg, "./sscachian.yaml");
const cache = types.get("client_list");
```

- 非同期（Firestore 生成が async）。Go の `context` 相当は任意の `AbortSignal`（既存方針）
- `loadTypes` は `Map<string, CacheType>`（[tests/config](../../tests/config/) で固定）

## 作業順（TDD）

1. **テスト仕様** — 済: [docs/tests/config](../../tests/config/)  
2. **スキーマ解析**（純関数寄り）  
   - YAML/JSON → 内部 Config AST → 検証  
   - Go: 標準/`gopkg.in/yaml.v3` 等（依存は最小）  
   - Node: 既存依存を優先。無ければ軽量 YAML パーサを追加  
3. **Registry + LoadTypes**  
   - 登録名解決 → 既存 `Define` Builder 相当で `CacheType` 構築  
   - memory / firestore ファクトリ  
4. **結合**  
   - 雛形 YAML をフィクスチャに、Get/Set/GetOrLoad/Purge がコード組み立てと同等であること  
   - Firestore は Emulator（`FIRESTORE_EMULATOR_HOST`）  
5. **版上げ・文書**  
   - Go `VERSION` = `0.9.0`、Node `package.json` = `0.9.0`  
   - 実装後に `docs/specs/config/` を正本化。本計画はアーカイブ候補  
6. **昇格**  
   - `dev-v0.9.0` → `develop` → `main` → タグ `v0.9.0` → npm Trusted Publisher

## 受け入れ条件

- [x] `docs/tests/config` 本文あり（Go / Node 共通 + ランタイム差分明記）
- [x] 雛形 [sscachian.example.yaml](./sscachian.example.yaml) をフィクスチャとしてロードできる（同等の最小 YAML で検証）
- [x] Go: `LoadTypes` + Registry で Cache Type が動き、既存テスト Green
- [x] Node: `loadTypes` + registry で同等、`npm run lint` / `npm test` Green
- [x] TTL 優先順位・Emulator 環境変数方針がテストで固定されている
- [x] コード面 Builder のみの既存利用が壊れていない
- [x] `VERSION` / `@b4moss/ss-cachian` = `0.9.0`
- [ ] タグ `v0.9.0` で npm 公開（main 昇格後）

## 実装時の注意

- 設定ロードは **Build 相当の組み立て**であり、キャッシュ意味論（Version / write-back / Exact Purge）は変えない
- `WithPolicy` は `WithLayerTTL` の別名のまま。設定の `types[].ttl` がそれに対応
- Firestore クライアントの Close ライフサイクルは実装時に最小判断（破棄 API が無ければドキュメントのみ。本マイルストーンのテスト必須とはしない）
- PHP / Valkey 向けの設定キーは予約しない（未知 driver は拒否）
- Registry 同名再登録は **後勝ち**（[tests/config](../../tests/config/)）
