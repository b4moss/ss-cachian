# Directory Update Log

## 2026-09-27

* **Update**: v0.7.0 実装（Exact Purge / Layer.PurgeExact / VERSION=0.7.0）。
* **Creation**: v0.7.0 向けテスト仕様（`docs/tests/purge`）と driver PurgeExact 節・実装計画。
* **Release**: v0.5.0 を `main` にマージし、タグ `v0.5.0` を付与（GitHub Release は作成しない）。
* **Update**: v0.5.0 実装（多層 CacheType、Firestore Driver、CI Emulator）。`VERSION=0.5.0`。
* **Creation**: v0.5.0 向けテスト仕様（`docs/tests/layer` / `driver-firestore`）と `docs/plans/v0.5.0/`。
* **Release**: v0.3.0 を `main` にマージし、タグ `v0.3.0` を付与（GitHub Release は作成しない）。
* **Update**: v0.3.0 実装（インメモリ Layer、Cache Type API、Version）。`make lint test` Green。
* **Creation**: v0.3.0 向けテスト仕様（`docs/tests/driver-memory` / `version` / `cache-type`）。
* **Creation**: `docs/plans/v0.3.0/` — コア + インメモリ方針。v0.2.0 は欠番（タグなし）。
* **Update**: `roadmap.md` に v0.2.0 欠番を明記。
* **Creation**: v0.1.0 スキャフォールド（`go/sscachian`、devcontainer、CI、Makefile）。
* **Update**: `roadmap.md` に Phase 1 マイルストーン（v0.1.0 / v0.3.0 / v0.5.0 / v0.7.0）を転記。テスト仕様は各版実装前に範囲限定で書く方針を明記。
* **Update**: PoC 残件（パッケージ構成・`docs/tests` 分割・devcontainer）を決定し `development.md` へ反映。devcontainer は Go 1.26 + Firestore Emulator + `act`。
* **Creation**: `docs/tests/README.md` にドメイン別テスト仕様の索引を追加。
* **Update**: Issue #7 の優先一問一答（10 件）を `behavior.md` / `drivers.md` / `api.md` へ反映。
* **Creation**: `docs/plans/v0.7.0/` に PoC 方針確定文書を追加。
* **Update**: `open-questions.md` から PoC 着手前残件を削除（方針確定へ移管）。

## 2026-09-20

* **Update**: charter の取り込みをサブモジュールからリモート `charter`（`b4moss/charter:docs`）による `docs/charter` 直接取り込みへ変更。

## 2026-09-07

* **Creation**: `b4moss/charter`（`docs` ブランチ）を `external/charter` サブモジュールとして追加し、`docs/charter` へ取り込み。
* **Creation**: `docs/main.md` / `docs/override-charter.md` / `docs/plans` / `docs/specs` を追加。
* **Creation**: `development.md` に開発環境・単体結合テスト・CI/CD・バッジ方針を追加。Go は 1.26。
* **Update**: 分散 L1 無効化通知は後続任意とし、PoC では作らない。
* **Update**: 旧 version 列挙・取得 API は後続任意とし、PoC では作らない。
* **Update**: シリアライズ形式は Go PoC では未決定とし、最初の他言語ポート時に決める。
* **Update**: current-version を L1・データキー同系の `__version__` サフィックスに決定。
* **Update**: SWR / SIE / negative cache は PoC 対象外とし、後続実装メモを `open-questions.md` に記載。
* **Update**: 構想ドキュメントを OKF v0.1 に再構成。決定事項と未決事項を分離。
* **Update**: `note-refs1.md` / `note-refs2.md` を `_archived/` へ移動。
* **Creation**: `behavior.md` / `drivers.md` / `roadmap.md` / `open-questions.md` を追加。
* **Deprecation**: `note.md` / `key-builder.md` の内容を決定事項ドキュメントへ吸収して削除。
