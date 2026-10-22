---
type: Plan
title: v0.11.0 es4 Layer Driver（＋ SWR 残件）
description: es4 Layer Driver を v0.11.0 で完了。SWR/SIE/negative と運用系は未実装で後続マイルストーンへ。
tags: [plan, phase2, v0.11.0]
timestamp: 2026-10-12T05:20:00Z
---

# v0.11.0 — es4 Layer Driver（完了）／ SWR 等は後続

- **状態:** es4 Layer Driver **完了**（タグ `v0.11.0` + npm `@b4moss/ss-cachian@0.11.0`。Issue [#68](https://github.com/b4moss/ss-cachian/issues/68) クローズ）。SWR / SIE / negative / 運用系は **未着手のまま後続版へ振替**。公開 install 不具合は [v0.11.1](../v0.11.1/) で修正。
- **マイルストーン:** v0.11.0（[roadmap](../../roadmap.md)）
- **意味論正本:** [specs/driver-es4](../../specs/driver-es4/) · [specs/layer](../../specs/layer/) · [specs/config](../../specs/config/)
- **テスト仕様:** [tests/driver-es4](../../tests/driver-es4/)
- **詳細計画（完了注記）:** [es4-layer-driver.md](./es4-layer-driver.md)

## 本版で完了したもの

1. ~~**es4 Layer Driver**~~ — 埋め込み open・Entry TTL ラップ・PurgePrefix via `Scan`・HTTP アダプタ並列

## 後続へ持ち越すもの（旧 v0.11.0 意図スタブ）

1. SWR / SIE / negative cache（Entry メタ拡張）
2. 旧 version の明示列挙・取得、分散 L1 無効化通知

## 受け入れ条件（es4 Driver）

- [x] テスト仕様と specs（driver-es4）
- [x] Go / Node パリティ
- [x] `VERSION` / package = `0.11.0`（タグ + npm）
