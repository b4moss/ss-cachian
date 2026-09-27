---
type: API
title: アプリケーション API
description: アプリ向け API 面。現行（v0.7.0）と後続任意に分けた決定リスト。
tags: [api, decided]
timestamp: 2026-09-27T03:05:00Z
---

# アプリケーション API

アプリケーションへ提供する API 面。Go 実装（`go/sscachian`）を正とする。他言語は同セマンティクスを踏襲する。

振る舞い前提は [振る舞い](./behavior.md)。ドメイン仕様は [specs](./specs/)。

## 現行（v0.7.0）

### 定義・構築

- `Define[T](name)`（既定 KeyBuilder 付き）
- `WithLayers` / `WithKeyBuilder` / `WithPolicy` / `WithLayerTTL` / `WithLayerTTLs` / `WithLoader` / `Build`
- `KeyContext`（AppSlug / TenantID / QueryType）、`DefaultKeyBuilder`

### 読み書き

- `Get`（最新 version。戻りは `value`）
- `GetOrLoad`（Miss 時 Loader。書き戻しは Bump しない）
- `Set`（**Bump してから**新 version キーへ書く）
- `Delete`（最新キー削除のあと Bump）

### Version・キー

- `CurrentVersion` / `BumpVersion`
- `BuildKey` / `BuildLatestKey` / `VersionKey`

### Purge

- `Purge(ctx, kc, layerIdx ...int)`（Exact: 全 version データ削除。`__version__` 非接触。未指定は全 Layer）

### Layer 契約（Driver）

- `Get` / `Set` / `Delete` / `Incr` / `PurgeExact`
- ヘルパ: `IsVersionDataKey`

### 最小成功条件（充足済み）

Cache Type を定義し、最新 version で Get/Set でき、自動／明示の `BumpVersion` と Exact Purge ができること。

## 後続で任意追加する

### 読み書き・エントリ

- `Has` / `Exists` / `GetEntry` / `GetLatest` / `Remember` / `RememberForever` / `Forget`

### Purge 拡張

- `PurgeExact`（アプリ API としての別名・オプション整理）
- `PurgePrefix` / `PurgeTag` / Purge 専用オプション API

### Policy・高度挙動

- SWR / SIE / negative cache、Entry メタ拡張

### Valkey 選択時のみ（委譲）

- `WithValkeyOptions` / `RateLimit` / `Lock` / `TryLock` / `ProtectStampede`

### 設定・運用

- 設定ファイルからの Cache Type ロード
- 旧 version の明示列挙・取得
- 分散 L1 無効化通知
