---
type: Hub
title: ss-cachian（pillar）
description: プロダクト目的・スコープ・技術方針の pillar 正本（旧 main.md）。
tags: [hub, pillar]
timestamp: 2026-09-27T05:56:00Z
---

# ss-cachian

サーバーサイドのキャッシュ戦略ライブラリ。  
**Phase 1 PoC 完了（v0.7.0）** — コード正本は `go/sscachian`（モジュール `github.com/b4moss/ss-cachian`、`VERSION=0.7.0`）。  
**Phase 2 v0.8.0 完了** — Node（`node/sscachian` / `@b4moss/ss-cachian@0.8.0`）は Go v0.7.0 パリティ実装済み（CacheType / Memory / Firestore / Exact Purge）。

OKF の版索引は [index.md](./index.md)（`okf_version` のみ）。本文の pillar 正本は本ファイル。

## 目的

ss-cachian は、単一キャッシュストアの薄い抽象ではない。

アプリケーションが **Cache Type** としてキャッシュ戦略を宣言し、ライブラリがその戦略を複数ストレージ（Layer）にまたがって実行する。

狙いは、DB を本質的なアクセスに限定して守ることである。キャッシュは正本ではなく、読み取り高速化のための一時層とする。

## スコープ

- やること: Cache Type による多層キャッシュ、Version 第1級 invalidate、Exact Purge、Driver（memory / Firestore ほか）
- やらぬこと（現行）: SWR / SIE / negative cache、Purge Prefix/Tag、分散 L1 無効化、設定ファイル駆動（後続は [plans](./plans/)）

### 中心概念

| 概念 | 意味 |
| --- | --- |
| Cache Type | キャッシュ対象の種類ごとの戦略定義単位 |
| Key Builder | ビジネス文脈を含むキー生成 |
| Layer / Driver | 保存先。階層数は固定しない |
| Policy | TTL など。Layer 単位で保持可能 |
| Loader | Cache Miss 時のデータ取得（Read-through） |
| Invalidation | 第1級は Version。Purge API で明示削除も提供 |
| Entry メタ | 初期は `created_at` / `expires_at` |

### 利用上の前提

- Read-through（Loader 付き）と Cache-aside（明示 Get/Set）の両方を許容する。
- キャッシュ可否は更新頻度ではなく、許容できる staleness で判断する。

## 技術方針

- マルチランタイム移植を前提に、言語ごとにトップレベルディレクトリを並べる。
- 先行実装は **Go 1.26**（`go/sscachian`、Driver は `go/sscachian/driver/...`）。import: `github.com/b4moss/ss-cachian`。
- Node 公開面は `node/sscachian`（npm: `@b4moss/ss-cachian@0.8.0`）。Go v0.7.0 パリティ実装済み。
- 開発は **devcontainer**（Go 1.26 / Firestore Emulator / `act`）。
- テストは単体結合（正常系・異常系）。仕様は [tests](./tests/)、方針は [憲章 TDD](./charter/tdd.md)。
- CI/CD・バッジの詳細は [.github/CI.md](../.github/CI.md)。

```text
go/sscachian/           # Go 公開モジュール
go/sscachian/driver/    # Driver 実装（memory, firestore, …）
node/sscachian/         # @b4moss/ss-cachian（v0.8.0 実装済み）
docs/                   # 知識バンドル正本（OKF v0.1）
```

### 憲章の取り込み

憲章本体は編集せず、`b4moss/charter` の **`main` の `docs/charter/`**（OKF v0.1）を取り込む。  
プロジェクト固有の上書きは [override-charter.md](./override-charter.md)。

```bash
git remote add charter https://github.com/b4moss/charter.git  # 未追加時のみ
git fetch charter main
git checkout charter/main -- docs/charter
```

## 索引

- [roadmap](./roadmap.md) — マイルストーン
- [specs](./specs/) — 現行仕様（ドメイン別）
- [plans](./plans/) — これからやる内容
- [tests](./tests/) — テスト仕様（specs と同じドメイン切り）
- [憲章](./charter/) — 開発ルール
- [OKF v0.1](./charter/okf/) — 本バンドルの版定義
- [override-charter](./override-charter.md) — 憲章オーバーライド
- [_archived](./_archived/) — 歴史資料
