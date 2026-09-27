---
type: Plan
title: v0.8.0 Node.js ポート（Go パリティ）
description: Go v0.7.0 と同意味の Cache Type API を TypeScript（@b4moss/ss-cachian）へ移植する。
tags: [plan, phase2, node]
timestamp: 2026-09-27T04:23:00Z
---

# v0.8.0 Node.js ポート（Go パリティ）

- **状態:** 進行中（テスト仕様）
- **マイルストーン:** v0.8.0（[roadmap](../../roadmap.md) Phase 2）
- **作業ブランチ:** `dev-v0.8.0` → `develop` → `main`
- **成果物:** `@b4moss/ss-cachian@0.8.0`（[node/sscachian](../../../node/sscachian/)）
- **意味論正本:** [specs](../../specs/)（Go と同一。言語差はテスト仕様のランタイム差分のみ）

## 目的

Go PoC（v0.7.0）と同等の Define / 多層 / Version / Exact Purge / Memory / Firestore を Node に移植し、npm から利用可能にする。

## やらぬこと

- Valkey / PHP / SWR・SIE / PurgePrefix・PurgeTag / 分散 L1
- Go 実装の仕様変更
- CJS デュアルパッケージ、ブラウザ向けバンドル

## 作業順（TDD）

1. テスト仕様: 既存ドメインを **Go / Node 共通**として適用（[tests](../../tests/)）。Node 差分（async / AbortSignal 等）を各 README に明記
2. Layer / `define` / `CacheType` / Memory
3. 多層 write-back + Exact Purge
4. Firestore ドライバ + CI `test-node` に Emulator
5. `package.json` = `0.8.0` / README / タグ `v0.8.0` → Trusted Publisher

## 受け入れ条件

- [ ] `docs/tests` 各ドメインが Node（v0.8.0）対象として記載されている
- [ ] Node テストが cache-type / version / layer / purge / memory / firestore で Green
- [ ] `npm run lint` / `npm test`（CI `test-node` 含む）Green
- [ ] `@b4moss/ss-cachian` version = `0.8.0`
- [ ] タグ `v0.8.0` で npm Trusted Publisher 公開可能
