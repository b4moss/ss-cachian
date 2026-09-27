# docs/tests

テスト仕様書の索引（TDD の入力）。形式は [憲章の TDD 方針](../charter/tdd.md) に従う。  
エンジニアリング方針の要約は [開発・CI/CD](../development.md) を参照。

## PoC の分割

ドメイン別に置く。各ロジックに正常系おおよそ 3、異常系おおよそ 3〜5。

| ドメイン | パス | 版 | 状態 |
| --- | --- | --- | --- |
| driver-memory | [driver-memory/](./driver-memory/) | v0.3.0 | 本文あり |
| version | [version/](./version/) | v0.3.0 | 本文あり |
| cache-type | [cache-type/](./cache-type/) | v0.3.0 | 本文あり |
| layer | [layer/](./layer/) | v0.5.0 | 本文あり |
| driver-firestore | [driver-firestore/](./driver-firestore/) | v0.5.0 | 本文あり |
| purge | `purge/` | v0.7.0 | 未着手 |

## v0.3.0 で決めた PoC 固定

- Driver `Incr`: 未作成キーは **0+1 → 1**
- current-version 初回作成（値 **1**）は Cache Type 側が **Set** で行う
- アプリ `Set`/`Delete` 成功後は自動 `BumpVersion`。Bump 失敗はエラーを返す
- `GetOrLoad` の書き戻しは **Bump しない**。書き戻し失敗時は Loader 値を返しログする
- データ無し `Delete` は成功扱いし **Bump する**

## v0.5.0 で決めた PoC 固定

- 多層: 上位 hit で下位スキップ。下位 hit / Loader 成功で上位へ無条件書き戻し。失敗はログ無視
- `Set`/`Delete` は全 Layer に同一キー。Bump / `__version__` は L1 のみ
- Layer TTL は `WithLayerTTLs`（不足分は 0）。負の TTL は Build 時拒否
- Firestore: 1キー1doc、`value` JSON、読み時 TTL。Incr はトランザクション。期限切れは論理 Miss（削除は必須でない）
- Firestore ドキュメント ID はキー文字列（必要ならエンコードを実装で固定）
