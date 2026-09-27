---
type: Spec
title: cache-type
description: CacheType の定義・構築と読み書き API（現行 v0.7.0）。
tags: [specs, cache-type]
timestamp: 2026-09-27T05:00:00Z
---

# cache-type

対象: `go/sscachian` の `Define` / `Builder` / `CacheType[T]`  
関連: [version](../version/) / [layer](../layer/) / [tests/cache-type](../../tests/cache-type/)

## 定義・構築

- `Define[T]`（`name` 付き）で Builder を開始する。既定 KeyBuilder は `DefaultKeyBuilder`（`{app}:cache:{tenant}:{query}`）。
- `WithLayers`（1 本以上必須）、`WithKeyBuilder`、`WithLayerTTL` / `WithPolicy`（全 Layer 同一 TTL）、`WithLayerTTLs`（Layer ごと。不足分は 0、負は Build 拒否）、`WithLoader`、`Build`。
- Layer 未設定 / KeyBuilder nil / 負 TTL は Build エラー。
- `KeyContext`（AppSlug / TenantID / QueryType）。

## 読み書き

- `Get`: 最新 version キー。戻りは `value` のみ（Entry は内部）。Miss は ok=false。
- `GetOrLoad`: Get 相当のあと Miss なら Loader。書き戻しは **Bump しない**。
- `Set`: **先に L1 で BumpVersion**し、その新 version キーへ全 Layer に書く。L1 Set 失敗はエラー（既に進んだ version は戻さない）。L2+ Set 失敗はログして全体成功。
- `Delete`: 最新キーを全 Layer から削除（冪等）したあと L1 で Bump。`__version__` は消さない。

## Entry メタ（初期）

保存時はラッパー必須:

```text
{ value, created_at, expires_at }
```

- アプリ向け `Get` は `value` だけ返す。
- 現行ではこれ以外のメタを設計しない。SWR / SIE / negative cache 用フィールドは後続（[plans/unscheduled](../../plans/unscheduled/)）。

## 付帯

- `Name` / `Layers` / `LayerTTL` はデバッグ・テスト用。
- `BuildKey` / `BuildLatestKey` / `VersionKey` / `CurrentVersion` / `BumpVersion` — 詳細は [version](../version/)。
- `Purge` — 詳細は [purge](../purge/)。
- 型不一致は `ErrTypeMismatch`。文脈不正は `ErrInvalidContext`。

## 最小成功条件（充足済み）

Cache Type を定義し、最新 version で Get/Set でき、自動／明示の `BumpVersion` と Exact Purge ができること。

後続任意 API の候補は [plans/unscheduled](../../plans/unscheduled/)。
