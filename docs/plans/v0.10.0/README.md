---
type: Plan
title: v0.10.0 Purge 拡張と便利 API
description: PurgePrefix/Tag と Has/GetEntry/Remember 系を同一マイルストーンで実装する。
tags: [plan, phase2]
timestamp: 2026-09-27T05:56:00Z
---

# v0.10.0 Purge 拡張 ＋ 便利読み書き API

- **状態:** 意図スタブ（未着手）
- **マイルストーン:** v0.10.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.10.0` → `develop` → `main`

## 目的

1. Exact に加え `PurgePrefix` / `PurgeTag`（必要ならアプリ `PurgeExact` 整理）
2. `Has` / `Exists` / `GetEntry` / `Remember` / `RememberForever` / `Forget` など薄い便利面

## やらぬこと

- SWR / SIE / negative、分散 L1、Valkey、PHP、設定ファイル（v0.9.0）

## 受け入れ条件（概要）

- [ ] テスト仕様（purge 拡張 ＋ cache-type 便利 API）
- [ ] Go / Node パリティ
- [ ] `VERSION` / package = `0.10.0`
