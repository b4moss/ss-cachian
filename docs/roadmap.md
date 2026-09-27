---
type: Roadmap
title: ロードマップ
description: 実装フェーズ、マイルストーン、マルチランタイム方針。
tags: [roadmap, decided]
timestamp: 2026-09-27T06:05:00Z
---

# ロードマップ

## 方針

- マルチランタイム移植を前提とする（`b4moss/crudian` / `b4moss/cachian` と同様）。
- 先行実装は Go（**Go 1.26**）。配置は `go/sscachian`（モジュール `github.com/b4moss/ss-cachian`。Driver は `go/sscachian/driver/...`）。
- Node（`node/sscachian` / `@b4moss/ss-cachian@0.8.0`）は Go v0.7.0 パリティ実装済み。
- Go では型安全 API を優先する。設定ファイル駆動は **v0.9.0** で着手。
- **テスト仕様は各マイルストーン実装前に、その版の範囲だけ書く**。

開発環境・技術方針は [README.md](./README.md)（pillar）。CI/CD は [.github/CI.md](../.github/CI.md)。現行仕様は [specs](./specs/)。

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
| **v0.8.0** | Node.js ポート（Go v0.7.0 パリティ・`@b4moss/ss-cachian`） | 完了（タグ + npm） |
| **v0.9.0** | 設定ファイル駆動（Cache Type の YAML/JSON ロード） | 実装済み（タグ待ち・[plans/v0.9.0](./plans/v0.9.0/)） |
| **v0.10.0** | Purge 拡張（Prefix/Tag）＋ 便利読み書き API（Has/GetEntry/Remember 等） | 未着手（[plans/v0.10.0](./plans/v0.10.0/)） |
| **v0.11.0** | SWR / SIE / negative cache ＋ 運用系（旧 version 列挙、分散 L1 無効化） | 未着手（[plans/v0.11.0](./plans/v0.11.0/)） |
| **v0.12.0** | Valkey Driver（固有機能委譲含む） | 未着手（[plans/v0.12.0](./plans/v0.12.0/)） |
| **v0.13.0** | PHP ポート | 未着手（[plans/v0.13.0](./plans/v0.13.0/)） |
| **v1.0.0** | Phase 2 締め（安定化・破壊的変更の凍結） | 未着手（[plans/v1.0.0](./plans/v1.0.0/)） |

マイルストーンの切り方は Phase 1 と同様、**版ごとにテスト仕様 → 実装**とする。作業単位は [plans](./plans/)。
