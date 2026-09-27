---
type: Spec
title: cache-type
description: CacheType の定義・構築と読み書き API（現行 v0.10.0）。
tags: [specs, cache-type, v0.10.0]
timestamp: 2026-09-27T08:20:00Z
---

# cache-type

対象: `go/sscachian`（モジュール `github.com/b4moss/ss-cachian`）の `Define` / `Builder` / `CacheType[T]`（Node: `define` / `Builder` / `CacheType`）  
関連: [version](../version/) / [layer](../layer/) / [purge](../purge/) / [tests/cache-type](../../tests/cache-type/)

## 定義・構築

- `Define[T]`（`name` 付き）で Builder を開始する。既定 KeyBuilder は `DefaultKeyBuilder`（`{app}:cache:{tenant}:{query}`。空フィールドは `ErrInvalidContext`）。
- `WithLayers`（1 本以上必須）、`WithKeyBuilder`、`WithLayerTTL` / `WithPolicy`、`WithLayerTTLs`、`WithLoader`、`WithDefaultTags`、`Build`。
- Layer 未設定 → `ErrNoLayer`。KeyBuilder nil → `ErrNoKeyBuilder`。負 TTL → `ErrNegativeTTL`。
- `KeyContext`（AppSlug / TenantID / QueryType）。

## 読み書き

- `Get`: 最新 version キーを L1→Ln。戻りは `value` のみ。Miss は ok=false。型不一致は `ErrTypeMismatch`。
- `GetEntry`: Hit 時に value + `created_at` / `expires_at`（公開 `CacheEntry`）。Miss は ok=false。
- `Has` / `Exists`: Get 相当のあと ok（Exists はエイリアス）。
- `GetOrLoad`: Miss なら Loader → 全 Layer 書き戻し（Bump なし）。任意 `WithTags`。Loader 未設定は `ErrNoLoader`。
- `Remember(ctx, kc, ttl, loader)` / `RememberForever`: Hit はキャッシュ。Miss は loader → 全 Layer に引数 TTL で書き戻し（Bump なし）。負 TTL → `ErrNegativeTTL`。Forever = TTL 0。
- `Set`: **先に L1 で BumpVersion**し書く。任意 `WithTags`。L1 成功後にタグ索引更新。
- `Delete` / `Forget`: 最新キー削除 + Bump（Forget はエイリアス）。

## Entry メタ

```text
{ value, created_at, expires_at }
```

- `Get` は value のみ。`GetEntry` はメタも返す。SWR / SIE / negative は後続。

## Purge

- `Purge` / `PurgeExact` / `PurgePrefix` / `PurgeTag` — [purge](../purge/)

## 付帯 API

- `BuildKey` / `BuildLatestKey` / `VersionKey` / `CurrentVersion` / `BumpVersion` — [version](../version/)
- `Name` / `Layers` / `LayerTTL` — デバッグ・テスト用

## 公開エラー（パッケージ）

| エラー | 主な発生箇所 |
| --- | --- |
| `ErrNoLayer` / `ErrNoKeyBuilder` / `ErrNegativeTTL` | Build / Remember |
| `ErrInvalidContext` | DefaultKeyBuilder / キー構築 |
| `ErrInvalidVersion` | BuildKey |
| `ErrTypeMismatch` | Get / GetEntry |
| `ErrNoLoader` | GetOrLoad / Remember（Loader なし） |
| `ErrCorruptVersion` | `__version__` 解釈不能 |
| `ErrInvalidLayerIndex` | Purge 系 layerIdx |
| `ErrEmptyKey` | Layer / PurgePrefix / PurgeTag |
| `ErrNotInteger` / `ErrIncrOverflow` | Layer Driver |
| `ErrInvalidConfig` | LoadTypes（v0.9.0） |
