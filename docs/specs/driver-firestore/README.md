---
type: Spec
title: driver-firestore
description: Firestore Layer（現行 v0.7.0・Emulator 対応）。
tags: [specs, driver-firestore]
timestamp: 2026-09-27T05:30:00Z
---

# driver-firestore

対象: `go/sscachian/driver/firestore`（`Store` / `New` / `Options` / `Close`）  
関連: [layer](../layer/) / [tests/driver-firestore](../../tests/driver-firestore/)

## 構築

- `New(ctx, Options{ProjectID, Collection})`。`FIRESTORE_EMULATOR_HOST` 設定時に Emulator へ接続（CI / `make test` で起動）。
- 既定: `ProjectID = "ss-cachian-dev"`、`Collection = "sscachian"`。
- `Close()` でクライアントを閉じる。

## ドキュメント

- 1 キャッシュキー = 1 ドキュメント。**ドキュメント ID はキー文字列をそのまま使う**（URL エンコードやサニタイズは現行では行わない）。
- フィールド: `value`（JSON バイト）、`created_at`、`expires_at`。Get 時に JSON を `any` へ復元。
- TTL は読み時判定（ネイティブ TTL ポリシーは使わない）。期限切れは **論理 Miss**（ドキュメント削除は必須でない）。
- nil value の Set は JSON `null` として保存可。JSON 化できない値はエラー。

## 操作

- `Incr` はトランザクション。未作成は 0+1→1。競合（Aborted）は最大 8 回クライアント側で再試行。
- 整数以外 → `ErrNotInteger`。`MaxInt64` → `ErrIncrOverflow`。
- `PurgeExact` はコレクション全走査し、`IsVersionDataKey` に合う ID だけ削除。
- 空キー / 空 prefix → `ErrEmptyKey`。負 TTL → `ErrNegativeTTL`。
- `ClearCollection` はテスト専用。

他言語間の共通シリアライズ形式は未決定（[plans/unscheduled](../../plans/unscheduled/)）。
