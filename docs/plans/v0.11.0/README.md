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
3. **es4 Layer Driver**（埋め込み open・Entry TTL ラップ・PurgePrefix は es4 `Scan` 依存・HTTP アダプタ同版並列）— 詳細は [es4-layer-driver.md](./es4-layer-driver.md)（決定 2026-10-11 追記済み）

## やらぬこと

- Valkey（→ v0.13.0）、PHP（→ v0.14.0）、セッション TTL / Firestore TTL（→ [v0.12.0](../v0.12.0/plan.md)）、Purge Prefix/Tag（v0.10.0）、設定ファイル（v0.9.0）
- es4 本体のキーデリミタ変更 / `Scan`（[b4moss/es4 v0.10.0](https://github.com/b4moss/es4/milestone/1) 側）

## 受け入れ条件（概要）

- [ ] テスト仕様と specs 更新
- [ ] Go / Node パリティ
- [ ] `VERSION` / package = `0.11.0`
