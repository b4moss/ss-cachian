---
type: Spec
title: layer
description: 多層 Get / 書き戻し / 全 Layer Set・Delete（現行 v0.7.0）。
tags: [specs, layer]
timestamp: 2026-09-27T03:05:00Z
---

# layer

対象: 複数 Layer を束ねた CacheType の連鎖  
関連: [behavior](../../behavior.md) / [tests/layer](../../tests/layer/)

## 探索と書き戻し

- `Get` / `GetOrLoad` は L1 → Ln。上位 hit で下位を見ない。
- 下位 hit または Loader 成功時、それより上（または全 Layer）へ **無条件書き戻し**。失敗はログして値は返す。
- current-version は L1 のみ。データキーは全 Layer で同一論理キー。

## Set / Delete

- データ操作は設定された **全 Layer**。Bump / `__version__` は L1。
- Layer TTL は `WithLayerTTLs`（未指定 Layer は 0＝無期限）。
- L1 失敗はエラー。L2+ 失敗はログして全体成功（Delete の L1 失敗時は Bump しない）。
