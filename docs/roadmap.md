---
type: Roadmap
title: ロードマップ
description: 実装フェーズとマルチランタイム方針。
tags: [roadmap, decided]
timestamp: 2026-09-07T12:00:00Z
---

# ロードマップ

## 方針

- マルチランタイム移植を前提とする（`b4moss/crudian` / `b4moss/cachian` と同様）。
- 先行実装は Go。
- Go では型安全 API を優先する。設定ファイル駆動は後回しでよい。

## Phase 1（PoC）

おおよそ v0.7.0 想定。

- Go で利用可能
- キービルダー
- 複数ストレージ選択
- Driver: インメモリ、Firestore

API 面は [アプリケーション API](./api.md) の「初期 PoC」を対象とする。

## Phase 2（〜 v1.0.0）

- 他 Driver（Valkey 等）
- 他言語ポート（Node.js、PHP）
