---
type: Spec
title: driver-es4
description: es4 Layer（埋め込み open・HTTP アダプタ）。v0.12.0 で WithExpiresAt と旧 Entry JSON フォールバック。
tags: [specs, driver-es4, v0.11.0, v0.12.0]
timestamp: 2026-10-28T16:20:00+09:00
---

# driver-es4

対象: `go/sscachian/driver/es4`（`Store` / `New` / `NewHTTP` / `NewWithBackend`）  
関連: [layer](../layer/) / [tests/driver-es4](../../tests/driver-es4/) / [plans/v0.12.0/plan.md](../../plans/v0.12.0/plan.md)  
依存: es4 Go **`v0.11.0`**（`WithExpiresAt`）。Node ドライバは es4 npm 公開後に追随。

## 構築

- 埋め込み: `New(ctx, Options{...})` — Driver が `es4.Open` / `Close` を所有。未指定時は memory_only。
- HTTP: `NewHTTP(ctx, HTTPOptions{BaseURL})` — Es4 Server。キーは `/v1/keys/...`。期限付き PUT はクエリ `expires_at`（RFC3339）。
- テスト注入: `NewWithBackend(Backend)`。

## 値と期限（v0.12.0）

- **書き込み:** es4 value はキャッシュ payload の JSON のみ。期限は `Set(..., WithExpiresAt(t))`（HTTP は `?expires_at=`）。value 内に文字列 `expires_at` は **書かない**。
- **期限なし:** options なしの Set。Firestore 上の `expires_at` フィールドは付かない（es4 が上書き時に削除）。
- **読み取り:** es4 が期限切れを不在返す。加えて **旧形式フォールバック** — `{"value":...,"expires_at":"<RFC3339>"}`（キー ⊆ `{value,expires_at}`）を解釈し、期限切れは Miss。
- 負 TTL → `ErrNegativeTTL`。空キー → `ErrEmptyKey`。

## 操作

- `Incr`: Tx 上で read-modify-write。未作成 0+1→1。**既存期限は削除して無期限**。再試行は Firestore Driver に揃える。
- `PurgeExact`: Scan なし。論理規則から version data キーのみ削除。
- `PurgePrefix`: es4 `Scan(prefix)` → Delete。

## TTL ポリシー

es4 `state_backend=firestore` 利用時の collection / フィールドは [driver-firestore 運用節](../driver-firestore/) を参照。

----

以上
