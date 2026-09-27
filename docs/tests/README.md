# docs/tests

テスト仕様書の索引（TDD の入力）。形式は [憲章の TDD 方針](../charter/tdd.md) に従う。  
エンジニアリング方針の要約は [開発・CI/CD](../development.md) を参照。

## PoC の分割（決定済み）

ドメイン別に置く。各ドメインファイルに正常系おおよそ 3、異常系おおよそ 3〜5。

| ドメイン | パス（予定） | 概要 |
| --- | --- | --- |
| version | `version/` | current-version・Bump・初回 Get |
| layer | `layer/` | 多層読み取り・書き戻し・失敗時 |
| purge | `purge/` | Purge Exact（全 version データ削除） |
| driver-memory | `driver-memory/` | インメモリ Driver |
| driver-firestore | `driver-firestore/` | Firestore Driver（Emulator） |

本文のテスト仕様は実装着手前に追記する（現状は配置方針のみ）。
