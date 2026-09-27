---
type: Spec
title: driver-firestore
description: Firestore Layer（現行 v0.7.0・Emulator 対応）。
tags: [specs, driver-firestore]
timestamp: 2026-09-27T03:05:00Z
---

# driver-firestore

対象: `go/sscachian/driver/firestore`  
関連: [drivers](../../drivers.md) / [tests/driver-firestore](../../tests/driver-firestore/)

- 1 キャッシュキー = 1 ドキュメント。ドキュメント ID はキー文字列をそのまま使う（PoC）。
- フィールド: `value`（JSON バイト）、`created_at`、`expires_at`。Get 時に JSON を `any` へ復元。
- TTL は読み時判定（ネイティブ TTL ポリシーは使わない）。期限切れは論理 Miss（削除は必須でない）。
- `Incr` はトランザクション。未作成は 1。競合（Aborted）はクライアント側で再試行する。
- `PurgeExact` はコレクション走査し、`IsVersionDataKey` に合う ID だけ削除。
- `FIRESTORE_EMULATOR_HOST` 設定時に Emulator へ接続（CI / `make test` で起動）。
