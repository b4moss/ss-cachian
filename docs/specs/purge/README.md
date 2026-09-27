---
type: Spec
title: purge
description: Exact Purge（現行 v0.7.0）。
tags: [specs, purge]
timestamp: 2026-09-27T05:00:00Z
---

# purge

対象: `CacheType.Purge` と Layer `PurgeExact`  
関連: [version](../version/) / [tests/purge](../../tests/purge/)

## 方針

- Exact / Prefix / Tag など、多彩な明示パージを提供する（現行は Exact のみ）。
- 日常の invalidate は Version、明示削除・運用は Purge。
- Purge のデフォルト対象は **全 Layer**。
- `Purge(ctx, kc, layerIdx ...int)` で Layer インデックスを渡し、単一または複数に絞れる（未指定は全 Layer）。
- Purge は **データキーのみ** 削除し、`__version__` は進めない・消さない。

## Exact Purge（現行）

- Exact: 論理プレフィックスに紐づく **全 version データキー** `{prefix}:{n}` を削除する。
- `{prefix}:__version__` は残す。Bump しない（`CurrentVersion` 不変）。
- インデックスは 0 始まり（範囲外は `ErrInvalidLayerIndex`、変更なし）。
- 失敗方針: 対象に L1 を含むとき L1 失敗はエラー・以降スキップ。L2+ 失敗はログして続行。L1 を含まない絞り込みでは対象 Layer の失敗をエラー。

## Layer

- `PurgeExact(ctx, logicalPrefix)`。空 prefix は `ErrEmptyKey`。数字サフィックスのみ削除（`IsVersionDataKey`）。冪等。

Prefix / Tag 等の拡張は [plans/unscheduled](../../plans/unscheduled/)。
