---
type: Spec
title: purge
description: Exact Purge（現行 v0.7.0）。
tags: [specs, purge]
timestamp: 2026-09-27T03:05:00Z
---

# purge

対象: `CacheType.Purge` と Layer `PurgeExact`  
関連: [behavior](../../behavior.md) / [tests/purge](../../tests/purge/)

## セマンティクス

- Exact: 論理プレフィックスに紐づく **全 version データキー** `{prefix}:{n}` を削除する。
- `{prefix}:__version__` は残す。Bump しない（`CurrentVersion` 不変）。
- `Purge(ctx, kc, layerIdx ...int)`。省略時は全 Layer。インデックスは 0 始まり（範囲外は `ErrInvalidLayerIndex`、変更なし）。
- 失敗方針: 対象に L1 を含むとき L1 失敗はエラー・以降スキップ。L2+ 失敗はログして続行。L1 を含まない絞り込みでは対象 Layer の失敗をエラー。

## Layer

- `PurgeExact(ctx, logicalPrefix)`。空 prefix は `ErrEmptyKey`。数字サフィックスのみ削除（`IsVersionDataKey`）。冪等。
