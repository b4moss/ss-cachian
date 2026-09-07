# Directory Update Log

## 2026-09-07

* **Update**: 旧 version 列挙・取得 API は後続任意とし、PoC では作らない。
* **Update**: シリアライズ形式は Go PoC では未決定とし、最初の他言語ポート時に決める。
* **Update**: current-version を L1・データキー同系の `__version__` サフィックスに決定。
* **Update**: SWR / SIE / negative cache は PoC 対象外とし、後続実装メモを `open-questions.md` に記載。
* **Update**: 構想ドキュメントを OKF v0.1 に再構成。決定事項と未決事項を分離。
* **Update**: `note-refs1.md` / `note-refs2.md` を `_archived/` へ移動。
* **Creation**: `behavior.md` / `drivers.md` / `roadmap.md` / `open-questions.md` を追加。
* **Deprecation**: `note.md` / `key-builder.md` の内容を決定事項ドキュメントへ吸収して削除。
