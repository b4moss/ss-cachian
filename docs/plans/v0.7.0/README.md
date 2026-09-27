---
type: Plan
title: v0.7.0 Purge・PoC 締め
description: Exact Purge（Layer.PurgeExact + アプリ Purge）と Phase 1 PoC 最小成功条件。
tags: [plan, poc]
timestamp: 2026-09-27T02:37:00Z
---

# v0.7.0 Purge・PoC 締め

- **状態:** 完了（`main` / タグ `v0.7.0` / GitHub Release）
- **マイルストーン:** v0.7.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.7.0` → `develop` → `main`
- **方針正本:** [poc-pre-impl-decisions.md](./poc-pre-impl-decisions.md)

## 目的

Exact Purge（全 version データ削除・`__version__` 非接触）を実装し、Layer 共通面を揃え、PoC 最小成功条件を満たす。

## やらぬこと

- PurgePrefix / PurgeTag / アプリ API としての `PurgeExact` 別名
- 旧 version 列挙・取得、SWR/SIE、分散 L1 無効化
- Firestore ネイティブ TTL ポリシー

## 作業順（TDD）

1. テスト仕様: [purge](../../tests/purge/)（+ driver-memory / driver-firestore の PurgeExact 節）
2. Layer `PurgeExact` + memory / Firestore 実装
3. CacheType `Purge`（Layer 絞り込み・失敗方針は Delete と同型）
4. `VERSION=0.7.0` / `make lint test`

## 受け入れ条件

- [x] `docs/tests/purge` 本文あり
- [x] memory + firestore で Exact Purge Green
- [x] 多層・既存テスト Green / `make lint test`
- [x] `VERSION` = `0.7.0`（タグは main マージ時）
- [x] PoC 最小成功条件（Define + Get/Set + Bump + Exact Purge）を満たす
