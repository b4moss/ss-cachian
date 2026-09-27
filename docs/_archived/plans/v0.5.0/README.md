---
type: Plan
title: v0.5.0 多層 + Firestore
description: 多層読み書き戻しと Firestore Driver（Emulator）。Purge は v0.7.0。
tags: [plan, poc]
timestamp: 2026-09-27T02:10:00Z
---

# v0.5.0 多層 + Firestore

- **状態:** アーカイブ（完了マイルストーン履歴。現行正本は [specs](../../../specs/)）
- **マイルストーン:** v0.5.0（[roadmap](../../../roadmap.md)）
- **作業ブランチ:** `dev-v0.5.0` → `develop` → `main`

## 目的

L1→L2→…→Loader の多層 Get / 書き戻しと、Firestore Driver（Emulator 結合）を提供する。

## やらぬこと

- Purge / PurgeExact（→ v0.7.0）
- Firestore ネイティブ TTL ポリシー

## 作業順（TDD）

1. テスト仕様: [layer](../../../tests/layer/) / [driver-firestore](../../../tests/driver-firestore/)
2. CacheType 多層化 + Layer TTL
3. Firestore Driver + Emulator 結合
4. CI に Emulator を組み込み

## 受け入れ条件

- [x] `docs/tests/layer` / `driver-firestore` 本文あり
- [x] 多層 Get 書き戻しが動く
- [x] Firestore Emulator 結合が Green
- [x] 既存 v0.3.0 テストが Green
- [x] `VERSION` = `0.5.0`（タグは main マージ時）
- [x] `make lint test` 通過
