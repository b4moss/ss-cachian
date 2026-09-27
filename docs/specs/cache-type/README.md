---
type: Spec
title: cache-type
description: CacheType の定義・構築と読み書き API（現行 v0.7.0）。
tags: [specs, cache-type]
timestamp: 2026-09-27T03:05:00Z
---

# cache-type

対象: `go/sscachian` の `Define` / `Builder` / `CacheType[T]`  
関連: [api](../../api.md) / [tests/cache-type](../../tests/cache-type/)

## 定義・構築

- `Define[T](name)` で Builder を開始する。既定 KeyBuilder は `DefaultKeyBuilder`（`{app}:cache:{tenant}:{query}`）。
- `WithLayers`（1 本以上必須）、`WithKeyBuilder`、`WithLayerTTL` / `WithPolicy`（全 Layer 同一 TTL）、`WithLayerTTLs`（Layer ごと。不足分は 0、負は Build 拒否）、`WithLoader`、`Build`。
- Layer 未設定 / KeyBuilder nil / 負 TTL は Build エラー。

## 読み書き

- `Get`: 最新 version キー。戻りは `value` のみ（Entry は内部）。Miss は ok=false。
- `GetOrLoad`: Get 相当のあと Miss なら Loader。書き戻しは **Bump しない**。
- `Set`: **先に L1 で BumpVersion**し、その新 version キーへ全 Layer に書く。L1 Set 失敗はエラー（既に進んだ version は戻さない）。L2+ Set 失敗はログして全体成功。
- `Delete`: 最新キーを全 Layer から削除（冪等）したあと L1 で Bump。`__version__` は消さない。

## 付帯

- `Name` / `Layers` / `LayerTTL` はデバッグ・テスト用。
- 型不一致は `ErrTypeMismatch`。文脈不正は `ErrInvalidContext`。
