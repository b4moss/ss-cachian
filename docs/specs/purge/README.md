---
type: Spec
title: purge
description: Exact / Prefix / Tag Purge（現行 v0.10.0）。
tags: [specs, purge, v0.10.0]
timestamp: 2026-09-27T08:20:00Z
---

# purge

対象: `CacheType.Purge` / `PurgeExact` / `PurgePrefix` / `PurgeTag` と Layer `PurgeExact` / `PurgePrefix`  
関連: [version](../version/) / [layer](../layer/) / [tests/purge](../../tests/purge/)

## 方針

- 日常の invalidate は Version、明示削除・運用は Purge。
- Purge 系のデフォルト対象は **全 Layer**。`layerIdx ...int`（未指定は全 Layer、重複除去・昇順）。
- Exact / Tag は **データキーのみ** 削除し、`__version__` は進めない・消さない（Prefix は先頭一致ですべて消す）。
- 失敗方針: 対象に L1 を含むとき L1 失敗はエラー・以降スキップ。L2+ 失敗はログして続行。L1 を含まない絞り込みでは対象 Layer の失敗をエラー。

## Exact Purge

- アプリ: `Purge(ctx, kc, layerIdx ...int)`。`PurgeExact` は同一意味の別名。
- Exact: 論理プレフィックスに紐づく **全 version データキー** `{prefix}:{n}`（n は 1 桁以上の数字）を削除する。
- `{prefix}:__version__` は残す。Bump しない。
- Layer: `PurgeExact(ctx, logicalPrefix)`。空 prefix は `ErrEmptyKey`。`IsVersionDataKey` で数字サフィックスのみ。冪等。

## PurgePrefix

- アプリ: `PurgePrefix(ctx, prefix string, layerIdx ...int)` — 生文字列（KeyContext 経由ではない）。
- Layer: `PurgePrefix(ctx, prefix)` — ID が `prefix` で始まる **すべて**を削除（`__version__`・非数字サフィックス含む）。
- 空 prefix → `ErrEmptyKey`。冪等。

## Tags + PurgeTag

- 書き込みオプション: Go `WithTags` / Builder `WithDefaultTags`。Node `{ tags }` / `withDefaultTags`。空文字タグは無視。既定＋呼び出しは和集合。
- L1 索引キー: `__sscachian_tag__:{tag}` → 論理プレフィックス集合（Entry 値）。L1 データ書き込み成功後に更新。
- `PurgeTag(ctx, tag, layerIdx ...int)`: 索引から prefix を取り各々 Exact Purge → 索引削除。未登録は成功。空タグ → `ErrEmptyKey`。
- タグは versioned データキーのレイアウトを変えない。
