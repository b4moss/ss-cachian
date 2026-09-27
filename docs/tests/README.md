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
| layer | `layer/` | v0.5.0 | 未着手 |
| driver-firestore | `driver-firestore/` | v0.5.0 | 未着手 |
| purge | `purge/` | v0.7.0 | 未着手 |

## v0.3.0 で決めた PoC 固定（実装時に揃える）

テスト仕様内の曖昧さを次で固定する。

- Driver `Incr`: 未作成キーは **0+1 → 1**
- current-version 初回作成（値 **1**）は Cache Type 側が **Set** で行う
- アプリ `Set`/`Delete` 成功後は自動 `BumpVersion`。Bump 失敗はエラーを返す
- `GetOrLoad` の書き戻しは **Bump しない**。書き戻し失敗時は Loader 値を返しログする
- データ無し `Delete` は成功扱いし **Bump する**
