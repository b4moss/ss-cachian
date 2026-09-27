---
type: Plan
title: v0.12.0 Valkey Driver
description: Redis/Valkey Layer と固有機能委譲（RateLimit / Lock 等）。
tags: [plan, phase2]
timestamp: 2026-09-27T05:56:00Z
---

# v0.12.0 Valkey Driver

- **状態:** 意図スタブ（未着手）
- **マイルストーン:** v0.12.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.12.0` → `develop` → `main`

## 目的

Valkey（Redis 互換）Driver を追加する。Rate Limit / Lock / Stampede 等は Valkey 選択時のみ委譲。

## やらぬこと

- PHP ポート、他 Object Storage Driver
