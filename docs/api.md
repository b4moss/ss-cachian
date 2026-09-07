---
type: API
title: アプリケーション API
description: アプリ向け API 面。初期 PoC 必須と後続任意に分けた決定リスト。
tags: [api, decided]
timestamp: 2026-09-07T12:00:00Z
---

# アプリケーション API

アプリケーションへ提供する API 面。Go 先行のメソッド名イメージ。他言語は同セマンティクスを踏襲する。  
名前の最終形は実装時に調整してよい。

振る舞い前提は [振る舞い](./behavior.md) を参照。

## 初期 PoC で提供する

### 定義・構築

- `Define` / `DefineCacheType`
- `WithLayers`
- `WithKeyBuilder`
- `WithPolicy` / `WithLayerTTL`
- `WithLoader`
- `Build`

### 読み書き

- `Get`（最新 version のみ）
- `GetOrLoad`
- `Set`
- `Delete`

### Version

- `CurrentVersion`
- `BumpVersion`

### Purge

- `Purge`（Exact ベースを含む）
- Layer 配列による対象指定（未指定時は全 Layer）

### キー

- `BuildKey`
- `BuildLatestKey`

### PoC の最小成功条件

Cache Type を定義し、最新 version で Get/Set でき、`BumpVersion` と簡易 Purge ができること。

## 後続で任意追加する

初期必須にはしない。

### 読み書き・エントリ

- `Has` / `Exists`
- `GetEntry`
- `GetLatest`
- `Remember` / `RememberForever`
- `Forget`

### Purge 拡張

- `PurgeExact`
- `PurgePrefix`
- `PurgeTag`
- Purge 用の専用オプション API

### Policy・高度挙動

- Stale-While-Revalidate 関連
- Stale-If-Error 関連
- Negative cache 関連
- Entry メタ拡張フィールドへのアクセス

### Valkey 選択時のみ（委譲）

- `WithValkeyOptions`
- `RateLimit`
- `Lock` / `TryLock`
- `ProtectStampede`

### 設定・運用

- 設定ファイルからの Cache Type ロード
- 旧 version の明示列挙・取得
- 分散 L1 無効化通知
