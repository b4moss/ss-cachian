---
type: Concept
title: Driver と Layer 契約
description: Driver 実装順と Layer 共通契約、現行 Driver 詳細の決定事項。
tags: [drivers, decided]
timestamp: 2026-09-27T03:05:00Z
---

# Driver と Layer 契約

## Driver 実装順

1. インメモリ（実装済み）
2. Firestore（実装済み）
3. Redis / Valkey（Phase 2）
4. 以降: MongoDB、Object Storage、Filesystem 等

## Layer 共通契約（現行）

- `Get` / `Set` / `Delete`
- `Incr`（current-version 更新用）
- `PurgeExact`（論理キーに紐づく全 version データキーの削除）
- TTL 値は Layer ごとに異なってよい（`Set` 時に渡す）

補足:

- Prefix / Tag など高度な Purge は後続。共通面には入れない。
- 差し込み口は揃えても、レイテンシや TTL 意味まで同一保証しない。
- Valkey 固有の Rate Limit / Lock 等は従来どおり Driver 固有（委譲）。

ドメイン仕様: [driver-memory](./specs/driver-memory/) / [driver-firestore](./specs/driver-firestore/)。

## Go の値表現

- アプリ境界は generics で型安全にする。
- **インメモリ:** 値は `any` として保持し、取り出し時に型アサーションする。
- **Firestore:** `value` は JSON バイト列として保存する。Get 時に `any` へデコードする。
- 他言語間の共通シリアライズ形式は未決定（[未決事項](./open-questions.md)）。

## Driver 詳細

### インメモリ

- プロセスローカル（プロセスをまたがない）。
- `sync.Mutex` + `map` でゴルーチン安全を保証する。
- TTL はエントリの `expires_at` を読み、Get 時に遅延削除する。
- バックグラウンド掃除ゴルーチンは持たない。

### Firestore

- 1 キャッシュキー = 1 ドキュメント（ドキュメント ID = キー文字列）。
- フィールド: `value`（JSON）、`created_at`、`expires_at`。
- TTL は読み取り時に `expires_at` を判定する。ネイティブ TTL ポリシーは使わない。
- `FIRESTORE_EMULATOR_HOST` で Emulator 接続。`Incr` はトランザクション（Aborted 時は再試行）。

## Valkey 固有機能

Rate Limit / Lock / Stampede など Valkey 文脈の機能は、

- Cache Type が Valkey を選んだときだけ有効
- ライブラリ本体は実装せず、Valkey Driver が操作するモジュールへ渡すだけ
