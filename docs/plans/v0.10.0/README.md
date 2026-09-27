---
type: Plan
title: v0.10.0 Purge 拡張と便利 API
description: PurgePrefix/Tag と Has/GetEntry/Remember 系を同一マイルストーンで実装する。
tags: [plan, phase2, v0.10.0]
timestamp: 2026-09-27T07:57:34Z
---

# v0.10.0 Purge 拡張 ＋ 便利読み書き API

- **状態:** テスト仕様済み（実装未着手）
- **マイルストーン:** v0.10.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.10.0` → `develop` → `main`

## 目的

1. Exact に加え `PurgePrefix` / `PurgeTag`（アプリ `PurgeExact` 別名）
2. `Has` / `Exists` / `GetEntry` / `Remember` / `RememberForever` / `Forget` など薄い便利面

## やらぬこと

- SWR / SIE / negative、分散 L1、Valkey、PHP、設定ファイル（v0.9.0）
- README バッジ / Scorecard / Codecov（本マイルストーンの製品実装とは別）

## テスト仕様

- [tests/purge](../../tests/purge/) — Exact 既存 + PurgeExact 別名 / PurgePrefix / Tags / PurgeTag
- [tests/cache-type](../../tests/cache-type/) — Has/Exists / GetEntry / Forget / Remember
- [tests/driver-memory](../../tests/driver-memory/)・[tests/driver-firestore](../../tests/driver-firestore/) — Layer `PurgePrefix`

## 受け入れ条件（概要）

- [x] テスト仕様（purge 拡張 ＋ cache-type 便利 API）
- [ ] Go / Node パリティ（実装）
- [ ] `VERSION` / package = `0.10.0`
