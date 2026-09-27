---
type: Roadmap
title: ロードマップ
description: 実装フェーズとマルチランタイム方針。
tags: [roadmap, decided]
timestamp: 2026-09-27T01:00:00Z
---

# ロードマップ

## 方針

- マルチランタイム移植を前提とする（`b4moss/crudian` / `b4moss/cachian` と同様）。
- 先行実装は Go（**Go 1.26**）。配置は `go/sscachian`（Driver は `go/sscachian/driver/...`）。
- Go では型安全 API を優先する。設定ファイル駆動は後回しでよい。

開発環境・CI/CD の詳細は [開発・CI/CD](./development.md) を参照。

## Phase 1（PoC）

おおよそ v0.7.0 想定。方針は [plans/v0.7.0](./plans/v0.7.0/) を参照。

- Go で利用可能（`go/sscachian`）
- キービルダー
- 複数ストレージ選択
- Driver: インメモリ、Firestore
- devcontainer: Go 1.26 + Firestore Emulator + `act`

API 面は [アプリケーション API](./api.md) の「初期 PoC」を対象とする。

## Phase 2（〜 v1.0.0）

- 他 Driver（Valkey 等）
- 他言語ポート（Node.js、PHP）
