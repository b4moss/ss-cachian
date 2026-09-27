---
type: TestSpec
title: config テスト仕様
description: YAML/JSON からの Cache Type ロード（Registry / LoadTypes）。Go・Node v0.9.0 導入。正常≈3 / 異常≈3〜5。
tags: [tests, config, v0.9.0, node]
timestamp: 2026-09-27T06:32:00Z
---

# config

対象:

- Go: `sscachian.NewRegistry` / `RegisterLoader` / `RegisterKeyBuilder` / `LoadTypes`
- Node: `createRegistry` / `registerLoader` / `registerKeyBuilder` / `loadTypes`

前提: [plans/v0.9.0](../../plans/v0.9.0/)（スキーマ・TTL 優先順位） / [tests 索引（Node 差分）](../README.md)  
実装後の正本は [specs](../../specs/) に `config` を追加する想定。キャッシュ意味論自体は [cache-type](../cache-type/) / [layer](../layer/) を変えない。

固定セマンティクス:

- フォーマットは YAML 正。同等 JSON 可（同じ論理ツリー）
- `schema_version` は `"0.9"` のみ受理（未対応・不正はロード拒否）
- `key_builder` / `loader` は **登録名の参照**。実体は Registry にコード登録
- `key_builder` 省略時は組み込み `"default"`（現行 DefaultKeyBuilder）
- `loader` 省略時は Loader なし（GetOrLoad は ErrNoLoader。現行どおり）
- Driver は v0.9.0 で `memory` / `firestore` のみ。未知 driver は拒否
- TTL 文字列は Go `time.ParseDuration` 互換。Node も同一解釈
- TTL 優先: `layers[i].ttl` → `types[].ttl` → `0`
- Firestore Emulator は設定に書かない。`FIRESTORE_EMULATOR_HOST` を正とする
- firestore `options` の未知キーは **拒否**
- 型パラメータ `T` は設定に書かない（ロード API 側で付ける）
- Node の `loadTypes` 戻りは **`Map<string, CacheType>`**（`.get(name)`）
- Go の `LoadTypes` 戻りは **`map[string]*CacheType[T]`**

フィクスチャ: [sscachian.example.yaml](../../plans/v0.9.0/sscachian.example.yaml)（必要ならテスト用に loader / key_builder を Registry へ登録してロード）

---

### Registry

- Loader / KeyBuilder を名前で登録する。
- 新規 Registry には組み込み `"default"` KeyBuilder が登録済み。
- 同名の再登録は後勝ち、または拒否（**後勝ちで固定**）。

#### テスト：正常系

- `RegisterLoader` した名前を設定の `loader` から解決できる
- `RegisterKeyBuilder` した名前を設定の `key_builder` から解決できる
- 未カスタムの Registry でも `key_builder: default`（または省略）でロードできる

#### テスト: 異常系

- 空文字の名前での登録はエラーになる
- `nil` / 未定義の関数での登録はエラーになる
- （任意）Loader 未登録名は LoadTypes 側で拒否する（本節ではなく LoadTypes 異常系）

---

### LoadTypes / loadTypes（解析・組み立て）

- 設定ファイル（または Reader / 文字列）を読み、検証し、各 `types[].name` をキーに Cache Type を組み立てる。
- YAML / JSON のどちらでも、同じ論理内容なら同じ Type 集合になる。
- コード面の `Define`…`Build` と同等の Layer / TTL / KeyBuilder / Loader が付く。

#### テスト：正常系

- 最短（memory 1 層 + `key_builder: default` + layer TTL）をロードし、Set / Get できる
- 同等内容の JSON をロードした結果が YAML とキー集合・動作で一致する
- 1 ファイルに複数 Type があるとき、各 `name` で取り出せ、互いに独立する

#### テスト: 異常系

- `schema_version` 欠落・空・`"0.8"` など未対応値はエラー（どの Type も作らない）
- `types` が空、または `name` が空 / 重複のときエラー
- `layers` が空、または未知 `driver`（例: `valkey`）のときエラー
- 設定で参照した `key_builder` / `loader` 名が Registry に無いときエラー
- 不正 TTL 文字列（例: `abc`）または負 TTL（例: `-1s`）はエラー

---

### TTL 解決

- Layer ごとの実効 TTL を設定から決める。
- 優先順位: 当該 `layers[i].ttl` → `types[].ttl` → `0`。
- 実効 TTL は Set 後の Entry `expires_at`（または同等の観測）で検証する。

#### テスト：正常系

- layer TTL のみ指定 → その Layer の `expires_at` が指定 duration に基づく
- type TTL のみ（layers に ttl なし）→ 全 Layer に同じ TTL が付く（WithLayerTTL 相当）
- type TTL と layer TTL が混在 → layer 指定がある Layer は layer 側、無い Layer は type 側

#### テスト: 異常系

- 負の TTL 文字列はロード拒否（Build 時の負 TTL 拒否と同型）
- パース不能な TTL はロード拒否
- TTL キーの型が文字列でない（例: 生の数値秒だけ、など契約外）場合は拒否（実装が文字列のみ受理と固定）

---

### Driver オプション（firestore / memory）

- `layers[].options` は Driver 固有。
- memory は options 空（または省略）を想定。
- firestore は `project_id` / `collection`（省略時は現行 Driver 既定: `ss-cachian-dev` / `sscachian`）。
- Emulator 接続は環境変数のみ。設定に `emulator` 等を書いても受理しない（未知キー拒否に含む）。

#### テスト：正常系

- firestore options で `collection` を指定してロードし、Emulator 上で Set / Get がその collection に載る
- `project_id` / `collection` 省略時は Driver 既定値で生成される
- memory 層のみの Type は options なしでロード・動作する

#### テスト: 異常系

- firestore `options` に未知キー（例: `emulator_host`）があるときロード拒否
- memory に非空の未知 options があるときロード拒否（厳密に揃える）
- `FIRESTORE_EMULATOR_HOST` 未設定かつ実 Firestore に届かない環境では、クライアントエラーとして失敗しうる（パニックにしない）。Emulator 必須の結合は CI 前提

---

### ロード後の Cache Type 振る舞い（結合）

- 設定から得た Type は、同等のコード組み立て Type と同じ読み書き意味論を持つ。
- 詳細ケースは [cache-type](../cache-type/) / [layer](../layer/) / [purge](../purge/) に委譲し、本節は設定経路の代表のみ。

#### テスト：正常系

- loader 参照付き Type で GetOrLoad: Miss 時に登録 Loader が呼ばれ、値が返り、Bump しない
- 多層（memory + firestore）を雛形どおりロードし、L1 Miss・L2 Hit で書き戻しが起きる（Emulator）
- カスタム `key_builder` 参照の Type で BuildKey / Set が登録関数のキー規則に従う

#### テスト: 異常系

- loader 未指定 Type で GetOrLoad Miss → ErrNoLoader
- 文脈不正はコード組み立て時と同じくエラー（設定経路固有の差を作らない）
- ロード成功後に Registry を変更しても、既に作った Type の Loader / KeyBuilder は変わらない（スナップショット）
