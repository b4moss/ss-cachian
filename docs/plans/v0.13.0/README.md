---
type: Plan
title: v0.13.0 Valkey Driver
description: Redis/Valkey Layer と固有機能委譲（RateLimit / Lock 等）。旧 v0.12.0 意図スタブを繰り下げ。
tags: [plan, phase2]
timestamp: 2026-10-28T00:00:00Z
---

# v0.13.0 Valkey Driver

- **状態:** 意図スタブ（未着手）
- **マイルストーン:** v0.13.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.13.0` → `develop` → `main`
- **繰り下げ理由:** 旧 v0.12.0 枠を [セッション TTL / Firestore TTL](../v0.12.0/plan.md) に充てたため（PO 決定 1-A・2026-10-28）

## 目的

Valkey（Redis 互換）Driver を追加する。Rate Limit / Lock / Stampede 等は Valkey 選択時のみ委譲。

## やらぬこと

- PHP ポート、他 Object Storage Driver

----

以上
