---
type: Spec
title: cache-type
description: CacheType の定義・構築と読み書き API（現行 v0.7.0）。
tags: [specs, cache-type]
timestamp: 2026-09-27T05:30:00Z
---

# cache-type

対象: `go/sscachian`（モジュール `github.com/b4moss/ss-cachian`）の `Define` / `Builder` / `CacheType[T]`  
関連: [version](../version/) / [layer](../layer/) / [tests/cache-type](../../tests/cache-type/)

## 定義・構築

- `Define[T]`（`name` 付き）で Builder を開始する。既定 KeyBuilder は `DefaultKeyBuilder`（`{app}:cache:{tenant}:{query}`。空フィールドは `ErrInvalidContext`）。
- `WithLayers`（1 本以上必須）、`WithKeyBuilder`、`WithLayerTTL` / `WithPolicy`（全 Layer 同一 TTL）、`WithLayerTTLs`（Layer ごと。不足分は 0、余分は無視、負は `ErrNegativeTTL`）、`WithLoader`、`Build`。
- Layer 未設定 → `ErrNoLayer`。KeyBuilder nil → `ErrNoKeyBuilder`。
- `KeyContext`（AppSlug / TenantID / QueryType）。

## 読み書き

- `Get`: 最新 version キーを L1→Ln。戻りは `value` のみ（Entry は内部）。Miss は ok=false。型不一致は `ErrTypeMismatch`。
- `GetOrLoad`: Get 相当のあと Miss なら Loader。成功時は **全 Layer へ書き戻し**（Bump しない）。Loader 未設定は `ErrNoLoader`。書き戻し失敗はログして値は返す。
- `Set`: **先に L1 で BumpVersion**し、その新 version キーへ書く。L1 Set 失敗はエラー（既に進んだ version は戻さない）。L2+ Set 失敗はログして全体成功。
- `Delete`: 最新キーを全 Layer から削除（冪等）したあと L1 で Bump。`__version__` は消さない。L1 Delete 失敗時は Bump しない。

## Entry メタ（初期）

保存時はラッパー必須:

```text
{ value, created_at, expires_at }
```

- アプリ向け `Get` は `value` だけ返す。Layer `Set` が `created_at` 未設定なら now、TTL>0 なら `expires_at = now+ttl` を付与する。
- 現行ではこれ以外のメタを設計しない。SWR / SIE / negative cache は後続（[plans/unscheduled](../../plans/unscheduled/)）。

## 付帯 API

- `BuildKey`（`version < 1` → `ErrInvalidVersion`）/ `BuildLatestKey` / `VersionKey` / `CurrentVersion` / `BumpVersion` — [version](../version/)
- `Purge` — [purge](../purge/)
- `Name` / `Layers` / `LayerTTL` — デバッグ・テスト用

## 公開エラー（パッケージ）

| エラー | 主な発生箇所 |
| --- | --- |
| `ErrNoLayer` / `ErrNoKeyBuilder` / `ErrNegativeTTL` | Build |
| `ErrInvalidContext` | DefaultKeyBuilder / キー構築 |
| `ErrInvalidVersion` | BuildKey |
| `ErrTypeMismatch` | Get の型アサーション |
| `ErrNoLoader` | GetOrLoad（Loader なし） |
| `ErrCorruptVersion` | `__version__` が整数として解釈不能 |
| `ErrInvalidLayerIndex` | Purge の layerIdx |
| `ErrEmptyKey` / `ErrNegativeTTL` / `ErrNotInteger` / `ErrIncrOverflow` | Layer Driver |

## 最小成功条件（充足済み）

Cache Type を定義し、最新 version で Get/Set でき、自動／明示の `BumpVersion` と Exact Purge ができること。

後続任意 API の候補は [plans/unscheduled](../../plans/unscheduled/)。
