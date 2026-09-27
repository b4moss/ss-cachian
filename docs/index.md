---
okf_version: "0.1"
---

# ss-cachian

サーバーサイドのキャッシュ戦略ライブラリ構想。実装未着手。本ディレクトリが知識バンドルの正本。

# 入口

* [main](./main.md) - プロダクト仕様ハブ
* [憲章（charter）](./charter/) - 開発方針の最上位ルール（`charter` リモートから取り込み）
* [憲章オーバーライド](./override-charter.md) - 当プロジェクト固有の上書き

# 決定事項

* [コンセプト](./concept.md) - 目的、中心概念、キャッシュの位置づけ
* [振る舞い](./behavior.md) - キー、多層、Version、Purge、エントリメタ
* [Driver と Layer 契約](./drivers.md) - Driver 優先順、Layer 契約、PoC Driver 詳細
* [アプリケーション API](./api.md) - 初期 PoC 必須 API と後続任意 API
* [ロードマップ](./roadmap.md) - 実装フェーズとマルチランタイム方針
* [開発・CI/CD](./development.md) - devcontainer、テスト、CI/CD、バッジ

# 未決定

* [未決事項](./open-questions.md) - 後続フェーズの未決定・未詳細

# 作業用

* [plans](./plans/) - これからやる内容（[v0.7.0 PoC 方針](./plans/v0.7.0/)）
* [tests](./tests/) - テスト仕様（PoC はドメイン別）
* [specs](./specs/) - 現行機能の仕様正本（実装後）

# アーカイブ

* [アーカイブ](./_archived/) - 壁打ちメモ等の旧稿
