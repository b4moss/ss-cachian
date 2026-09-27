---
状態: 意図スタブ
マイルストーン: unscheduled
---

# 後続任意 API・Driver 候補

現行 API の正本は [specs/cache-type](../../specs/cache-type/) ほか。ここは **未実装の候補**のみ。

## 読み書き・エントリ

- `Has` / `Exists` / `GetEntry` / `GetLatest` / `Remember` / `RememberForever` / `Forget`

## Purge 拡張

- `PurgeExact`（アプリ API としての別名・オプション整理）
- `PurgePrefix` / `PurgeTag` / Purge 専用オプション API

## Policy・高度挙動

- SWR / SIE / negative cache、Entry メタ拡張（詳細は [open-questions.md](./open-questions.md)）

## Valkey 選択時のみ（委譲）

- `WithValkeyOptions` / `RateLimit` / `Lock` / `TryLock` / `ProtectStampede`
- Cache Type が Valkey を選んだときだけ有効。ライブラリ本体は実装せず、Valkey Driver が操作するモジュールへ渡すだけ。

## 設定・運用

- 設定ファイルからの Cache Type ロード
- 旧 version の明示列挙・取得
- 分散 L1 無効化通知

## Driver 実装順（候補）

1. インメモリ — **実装済み**（[driver-memory](../../specs/driver-memory/)）
2. Firestore — **実装済み**（[driver-firestore](../../specs/driver-firestore/)）
3. Redis / Valkey（Phase 2）
4. 以降: MongoDB、Object Storage、Filesystem 等
