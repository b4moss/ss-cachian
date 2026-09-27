---
type: Plan
title: v0.11.0 SWR/SIE/negative と運用系
description: Entry メタ拡張（SWR/SIE/negative）と旧 version 列挙・分散 L1 無効化。
tags: [plan, phase2]
timestamp: 2026-09-27T05:56:00Z
---

# v0.11.0 SWR / SIE / negative ＋ 運用系

- **状態:** 意図スタブ（未着手）
- **マイルストーン:** v0.11.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.11.0` → `develop` → `main`

## 目的

1. SWR / SIE / negative cache（Entry メタ拡張）
2. 旧 version の明示列挙・取得、分散 L1 無効化通知

## やらぬこと

- Valkey、PHP、Purge Prefix/Tag（v0.10.0）、設定ファイル（v0.9.0）

## 受け入れ条件（概要）

- [ ] テスト仕様と specs 更新
- [ ] Go / Node パリティ
- [ ] `VERSION` / package = `0.11.0`
