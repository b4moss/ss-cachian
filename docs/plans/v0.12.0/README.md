---
type: Plan
title: v0.12.0 セッション TTL / Firestore TTL
description: es4 期限付き書き込みの利用・firestore ゼロ値 expires_at 抑止・TTL ポリシー docs。
tags: [plan, phase2, v0.12.0, ttl]
timestamp: 2026-10-28T00:00:00Z
---

# v0.12.0 セッション TTL / Firestore TTL

- **状態:** 仕様詳細（本 PR・未実装）
- **マイルストーン:** v0.12.0（[roadmap](../../roadmap.md)）
- **詳細計画:** [plan.md](./plan.md)
- **テスト仕様:** [tests/driver-es4](../../tests/driver-es4/) · [tests/driver-firestore](../../tests/driver-firestore/)
- **作業ブランチ（実装時）:** `dev-v0.12.0` → `develop` → `main`

## 一言

jparse → ss-cachian → es4 → Firestore 経路で、Firestore の日時型 `expires_at` と TTL ポリシーが使えるようにする。コードは本計画 PR では変えない。

## 版割当メモ

旧 v0.12.0（Valkey）・旧 v0.13.0（PHP）は意図スタブのみだったため、**本マイルストーンを v0.12.0 に充て**、Valkey → [v0.13.0](../v0.13.0/)、PHP → [v0.14.0](../v0.14.0/) へ繰り下げた（PO 要決定）。

----

以上
