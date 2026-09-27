# docs/tests

テスト仕様書の索引（TDD の入力）。形式は [憲章の TDD 方針](../charter/tdd.md) に従う。  
エンジニアリング方針の要約は [README.md](../README.md)（pillar）と [.github/CI.md](../../.github/CI.md) を参照。

## 分割

ドメイン別に置く。各ロジックに正常系おおよそ 3、異常系おおよそ 3〜5。  
振る舞いの正本は [specs](../specs/)。本ディレクトリは検証観点の正本。  
表の「導入版」は **そのスイートを最初に書いたマイルストーン**であり、現行プロダクト版を限定するものではない。

意味論は **Go / Node 共通**。実装パスだけが異なる。v0.8.0 は既存ドメインを Node 向けに再適用する（ケース追加はランタイム差分のみ）。

| ドメイン | パス | 導入版 | Go テスト | Node |
| --- | --- | --- | --- | --- |
| driver-memory | [driver-memory/](./driver-memory/) | v0.3.0 | `driver/memory/*_test.go` | v0.8.0 適用 |
| version | [version/](./version/) | v0.3.0 | `cache_test.go`（Version 節） | v0.8.0 適用 |
| cache-type | [cache-type/](./cache-type/) | v0.3.0 | `cache_test.go`（単一 L1） | v0.8.0 適用 |
| layer | [layer/](./layer/) | v0.5.0 | `layer_test.go` | v0.8.0 適用 |
| driver-firestore | [driver-firestore/](./driver-firestore/) | v0.5.0 | `driver/firestore/*_test.go` | v0.8.0 適用 |
| purge | [purge/](./purge/) | v0.7.0 | `purge_test.go` + driver PurgeExact | v0.8.0 適用 |
| config | [config/](./config/) | v0.9.0 | `config*_test.go`（予定） | v0.9.0 適用 |

## ランタイム差分（Node）

- 対象パッケージ: `node/sscachian`（npm: `@b4moss/ss-cachian`）
- I/O はすべて `async`（`Promise`）。Go の `context.Context` 相当は任意の `AbortSignal`
- 並行: ゴルーチンの代わりに `Promise.all` 等の並行呼び出しで検証する
- Miss 表現: 「値なし」が分かる契約に固定（例: `{ ok: false }` または `undefined`。実装で一方に固定し Go の ok=false と同型）
- テストランナー: `node --test`（ビルド後 `dist/**/*.test.js`）
- Firestore: `@google-cloud/firestore` + `FIRESTORE_EMULATOR_HOST`（CI `test-node` で Emulator 起動）
- config（v0.9.0）: `loadTypes` は `async`、戻りは `Map`。YAML/JSON パースと Firestore Layer 生成を含む

## 現行で固定しているセマンティクス（要約）

- Driver `Incr`: 未作成キーは **0+1 → 1**。current-version 初回は Cache Type が **Set で 1**
- `Set` は **Bump してから**書く。`Delete` は削除後 Bump。`GetOrLoad` 書き戻しは Bump しない（**全 Layer**）
- 多層: 上位 hit で下位スキップ。書き戻し失敗はログ無視。Bump / `__version__` は L1 のみ
- Layer TTL は `WithLayerTTLs`（不足分 0）。負 TTL は Build 拒否
- Firestore: 1キー1doc、読み時 TTL、ドキュメント ID = キー文字列（エンコードなし）。既定 collection `sscachian`
- `Purge` = Exact。`layerIdx ...int`（重複除去）。`__version__` 非接触。失敗方針は Delete と同型
