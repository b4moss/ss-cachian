---
type: Roadmap
title: ロードマップ
description: 実装フェーズ、マイルストーン、マルチランタイム方針。
tags: [roadmap, decided]
timestamp: 2026-09-27T04:23:00Z
---

# ロードマップ

## 方針

- マルチランタイム移植を前提とする（`b4moss/crudian` / `b4moss/cachian` と同様）。
- 先行実装は Go（**Go 1.26**）。配置は `go/sscachian`（Driver は `go/sscachian/driver/...`）。
- Go では型安全 API を優先する。設定ファイル駆動は後回しでよい。
- **テスト仕様は各マイルストーン実装前に、その版の範囲だけ書く**。

開発環境・CI/CD は [開発・CI/CD](./development.md)。現行仕様は [specs](./specs/)。

## Phase 1（PoC）— 〜 v0.7.0 — **完了**

タグ: `v0.3.0` / `v0.5.0` / `v0.7.0`（`main`）。完了計画の履歴は [_archived/plans](./_archived/plans/)。

| 版 | 内容 | 結果 |
| --- | --- | --- |
| **v0.1.0** | 土台（スキャフォールド） | 完了（タグなし） |
| ~~v0.2.0~~ | 欠番 | — |
| **v0.3.0** | コア + インメモリ | 完了（タグあり） |
| **v0.5.0** | 多層 + Firestore | 完了（タグあり） |
| **v0.7.0** | Exact Purge・PoC 締め | 完了（タグ + GitHub Release） |

## Phase 2（拡張）— 〜 v1.0.0

| 版 | 内容 | 状態 |
| --- | --- | --- |
| **v0.8.0** | Node.js ポート（Go v0.7.0 パリティ・`@b4moss/ss-cachian`） | 計画・テスト仕様（[plans/v0.8.0](./plans/v0.8.0/)） |
| （後続） | 他 Driver（Valkey 等）、PHP ポート、（任意）SWR / SIE、Purge Prefix/Tag、分散 L1 等 | 未割当 |

マイルストーンの切り方は Phase 1 と同様、**版ごとにテスト仕様 → 実装**とする。作業単位は [plans](./plans/)。
