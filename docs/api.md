---
type: Note
title: アプリケーション API（仮定）
description: 構想・対話から導いたアプリ向け API 面。初期 PoC 必須とそれ以降任意に分けた仮定リスト。
role: note
---

# アプリケーション API（仮定）

ここまでの構想・確定方針から見た、アプリケーション側へ提供すべき API の仮定リスト。  
実装前のたたき台であり、名前・粒度は今後変わりうる。

Go 先行を想定したメソッド名イメージ。他言語ポート時は同セマンティクスを踏襲する。

関連方針の入口: [index.md](./index.md)

---

## 初期 PoC で提供するもの

Phase 1（Go / キービルダー / Memory + Firestore）で揃える想定。

### 定義・構築

- `Define` / `DefineCacheType`
- `WithLayers`
- `WithKeyBuilder`
- `WithPolicy` / `WithLayerTTL`
- `WithLoader`
- `Build`

### 読み書き（Cache Type インスタンス）

- `Get`（常に最新 version のみ）
- `GetOrLoad`
- `Set`
- `Delete`

### Version（第1級 invalidate）

- `CurrentVersion`
- `BumpVersion`

### Purge

- `Purge`（Exact ベースを含む）
- Layer 指定オプション（配列で単一 / 複数 Layer。未指定時は全 Layer）

### キー

- `BuildKey`
- `BuildLatestKey`（current-version を解決してキー組み立て。`Get` 内部利用でも可）

### PoC での振る舞い前提

- version 入りデータキーは全 Layer で同一キー名
- Miss 後の書き戻しは上位 Layer へ無条件
- TTL は Layer 単位
- Version bump 時の能動クリアはしない（旧キーは TTL で消滅）
- Get は最新 version のみ返す

---

## それ以降（任意・後続）

PoC 後、必要に応じて追加する想定。初期必須にはしない。

### 読み書き・エントリ

- `Has` / `Exists`
- `GetEntry`（値 + `created_at` / `expires_at`）
- `GetLatest`（`Get` の明示的別名）
- `Remember` / `RememberForever`（`GetOrLoad` 系の糖衣）
- `Forget`（`Delete` または Exact Purge の別名）

### Purge の拡張

- `PurgeExact`
- `PurgePrefix`
- `PurgeTag`
- `PurgeOptions` / `WithLayers([...])`（専用 API として切り出す場合）

### Policy・高度キャッシュ挙動

- Stale-While-Revalidate 関連 API
- Stale-If-Error 関連 API
- Negative cache 関連 API
- Entry メタの拡張フィールドへのアクセス

### Valkey 選択時のみ（本体は委譲）

Cache Type が Valkey を選んだときだけ有効。ライブラリ本体は Valkey Driver 先モジュールへ渡す。

- `WithValkeyOptions`
- `RateLimit`
- `Lock` / `TryLock`
- `ProtectStampede`

### 設定・運用

- YAML / 設定ファイルからの Cache Type ロード
- 旧 version の明示列挙・取得
- 分散 L1 無効化通知（pub-sub 等）

---

## メモ

- 名前は仮定。特に `Define` 系と `Purge` 系は実装時にパッケージ構成へ合わせて調整してよい。
- PoC では「Cache Type を定義し、最新 version で Get/Set し、BumpVersion と簡易 Purge ができる」が最小の成功条件。

---

以上
