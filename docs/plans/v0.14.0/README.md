---
type: Plan
title: v0.14.0 PHP ポート
description: Go/Node と同セマンティクスの PHP パッケージ。旧 v0.13.0 意図スタブを繰り下げ。
tags: [plan, phase2]
timestamp: 2026-10-28T00:00:00Z
---

# v0.14.0 PHP ポート

- **状態:** 意図スタブ（未着手）
- **マイルストーン:** v0.14.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.14.0` → `develop` → `main`
- **繰り下げ理由:** 旧 v0.13.0 枠を Valkey（→ v0.13.0）に空け、PHP をその次へ（PO 決定 1-A・2026-10-28）

## 目的

`php/` 配下に Go/Node パリティの Cache Type API を移植する。

## やらぬこと

- Valkey 固有機能の PHP 先行実装（Valkey は v0.13.0 前提）

----

以上
