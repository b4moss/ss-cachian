---
okf_version: "0.1"
---

# ss-cachian

サーバーサイドのキャッシュ戦略ライブラリ。Phase 1 PoC は **v0.7.0** で完了。本ディレクトリが知識バンドルの正本。

# 入口

* [main](./main.md) - プロダクト仕様ハブ
* [specs](./specs/) - 現行機能の仕様正本
* [憲章（charter）](./charter/) - 開発方針の最上位ルール
* [憲章オーバーライド](./override-charter.md) - 当プロジェクト固有の上書き

# 決定事項

* [コンセプト](./concept.md) - 目的、中心概念、キャッシュの位置づけ
* [振る舞い](./behavior.md) - キー、多層、Version、Purge、エントリメタ
* [Driver と Layer 契約](./drivers.md) - Driver 優先順、Layer 契約、Driver 詳細
* [アプリケーション API](./api.md) - 現行 API と後続任意 API
* [ロードマップ](./roadmap.md) - 実装フェーズとマルチランタイム方針
* [開発・CI/CD](./development.md) - devcontainer、テスト、CI/CD、バッジ

# 未決定

* [未決事項](./open-questions.md) - Phase 2 以降の未決定・未詳細

# 作業用

* [plans](./plans/) - これからやる内容（Phase 2 以降）
* [tests](./tests/) - テスト仕様（ドメイン別）

# アーカイブ

* [アーカイブ](./_archived/) - 壁打ちメモ、完了マイルストーン計画
