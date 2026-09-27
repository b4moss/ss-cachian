---
type: Spec
title: layer
description: Layer 契約と多層 Get / 書き戻し / 全 Layer Set・Delete（現行 v0.7.0）。
tags: [specs, layer]
timestamp: 2026-09-27T05:00:00Z
---

# layer

対象: Layer 共通契約、および複数 Layer を束ねた CacheType の連鎖  
関連: [cache-type](../cache-type/) / [tests/layer](../../tests/layer/)

## Layer 共通契約（現行）

- `Get` / `Set` / `Delete`
- `Incr`（current-version 更新用）
- `PurgeExact`（論理キーに紐づく全 version データキーの削除）
- ヘルパ: `IsVersionDataKey`
- TTL 値は Layer ごとに異なってよい（`Set` 時に渡す）

補足:

- Prefix / Tag など高度な Purge は後続。共通面には入れない。
- 差し込み口は揃えても、レイテンシや TTL 意味まで同一保証しない。
- Valkey 固有の Rate Limit / Lock 等は Driver 固有（委譲）。後続は [plans/unscheduled](../../plans/unscheduled/)。

## 探索と書き戻し

```text
L1 → miss → L2 → … → Loader → 上位へ書き戻し
```

- `Get` / `GetOrLoad` は L1 → Ln。上位 hit で下位を見ない。
- 下位 hit または Loader 成功時、それより上（または全 Layer）へ **無条件書き戻し**。
- 読み取りが成功していれば、書き戻しの一部失敗でも **値は呼び出し側へ返す**。失敗はログして無視。
- current-version は L1 のみ。データキーは全 Layer で同一論理キー。
- L1 / L2 / … は Cache Type が定義した順序上の名前。L1 はインメモリであることが多いが必須ではない。

## Set / Delete

- データ操作は設定された **全 Layer**。Bump / `__version__` は L1。
- Layer TTL は `WithLayerTTLs`（未指定 Layer は 0＝無期限）。
- L1 失敗はエラー。L2+ 失敗はログして全体成功（Delete の L1 失敗時は Bump しない）。
