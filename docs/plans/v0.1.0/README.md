---
type: Plan
title: v0.1.0 土台（スキャフォールド）
description: Phase 1 最初のマイルストーン。テスト仕様本文は書かない。
tags: [plan, scaffold]
timestamp: 2026-09-27T01:20:00Z
---

# v0.1.0 土台（スキャフォールド）

- **状態:** 実装中 → 本 PR で充足予定
- **マイルストーン:** v0.1.0（[roadmap](../../roadmap.md)）
- **テスト仕様:** なし（配置方針のみ。本文は v0.3.0 以降）

## 受け入れ条件

- [x] `go/sscachian` モジュール骨格（driver/memory・firestore スタブ）
- [x] devcontainer: Go 1.26 + Firestore Emulator + `act`
- [x] CI 骨格（path filter / ancestor skip / CI result / docs-only skip）
- [x] `make lint` / `make test` が通る

## やらぬこと

- キャッシュ API の実装
- `docs/tests/` ドメイン本文
- Firestore / メモリ Driver の実ロジック
