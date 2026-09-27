---
type: Roadmap
title: ロードマップ
description: 実装フェーズ、マイルストーン、マルチランタイム方針。
tags: [roadmap, decided]
timestamp: 2026-09-27T01:15:00Z
---

# ロードマップ

## 方針

- マルチランタイム移植を前提とする（`b4moss/crudian` / `b4moss/cachian` と同様）。
- 先行実装は Go（**Go 1.26**）。配置は `go/sscachian`（Driver は `go/sscachian/driver/...`）。
- Go では型安全 API を優先する。設定ファイル駆動は後回しでよい。
- **テスト仕様は各マイルストーン実装前に、その版の範囲だけ書く**（一気に全部は書かない）。

開発環境・CI/CD の詳細は [開発・CI/CD](./development.md) を参照。  
PoC 方針の詳細は [plans/v0.7.0](./plans/v0.7.0/) を参照。

## Phase 1（PoC）— 〜 v0.7.0

API 面は [アプリケーション API](./api.md) の「初期 PoC」を対象とする。

| 版 | 内容 | 実装前のテスト仕様 | 実装範囲 |
| --- | --- | --- | --- |
| **v0.1.0** | 土台（スキャフォールド） | なし（`docs/tests/` の配置方針・索引のみ） | devcontainer（Go 1.26 / Firestore Emulator / `act`）、`go/sscachian` 骨格、CI 骨格 |
| ~~v0.2.0~~ | **欠番**（タグも切らない） | — | — |
| **v0.3.0** | コア + インメモリ | `version/`・`driver-memory/` など当該範囲 | Key Builder、Entry ラッパー、current-version / Bump / 初回 Get、インメモリ Driver、`Get` / `Set` / `Delete` / `GetOrLoad`（自動 Bump）。**単一 L1** |
| **v0.5.0** | 多層 + Firestore | `layer/`・`driver-firestore/` など当該範囲 | 多層 Get / 書き戻し、Firestore Driver（Emulator 結合） |
| **v0.7.0** | Purge・PoC 締め | `purge/` など残り | `PurgeExact`（全 version データ削除）、Layer 共通面の揃え、PoC 最小成功条件の充足 |

## Phase 2（拡張）— 〜 v1.0.0

- 他 Driver（Valkey 等）
- 他言語ポート（Node.js、PHP）
- （任意・後続）SWR / SIE、Purge Prefix/Tag、分散 L1 無効化 等

マイルストーンの切り方は Phase 1 と同様、**版ごとにテスト仕様 → 実装**とする。
