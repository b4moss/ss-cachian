---
type: Plan
title: v0.3.0 コア + インメモリ
description: Key Builder・Entry・Version・単一 L1（メモリ）・Get/Set/Delete/GetOrLoad。v0.2.0 は欠番。
tags: [plan, poc]
timestamp: 2026-09-27T01:35:00Z
---

# v0.3.0 コア + インメモリ

- **状態:** 仕様詳細（実装前）
- **マイルストーン:** v0.3.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.3.0` → PR → `develop`
- **v0.2.0:** 欠番（タグも切らない）

## 目的

単一 Layer（インメモリ L1）で Cache Type を定義し、最新 version の Get/Set/Delete/GetOrLoad と Version bump が動くこと。

## やらぬこと（この版）

- 多層読み取り・書き戻し（→ v0.5.0）
- Firestore Driver（→ v0.5.0）
- Purge / PurgeExact（→ v0.7.0）
- SWR / SIE / negative cache、他言語

## 作業順（TDD）

憲章どおり **テスト仕様 → 失敗するテスト → 実装**。

### 1. テスト仕様（実装前）

`docs/tests/` に当該範囲のみ書く（正常≈3 / 異常≈3〜5）。

| ドメイン | パス | 対象 |
| --- | --- | --- |
| version | `docs/tests/version/` | 初回 Get で `__version__=1` 作成、Bump 最良努力、Set/Delete 自動 Bump、CurrentVersion |
| driver-memory | `docs/tests/driver-memory/` | Get/Set/Delete、遅延 TTL、ゴルーチン安全、Incr |
| （アプリ面） | `docs/tests/version/` または `cache-type/` に併記可 | Define/Build、KeyBuilder、Get/Set/Delete/GetOrLoad、BuildKey/BuildLatestKey |

`layer/`・`purge/`・`driver-firestore/` は **書かない**（後続マイルストーン）。

### 2. 実装範囲

配置: [`go/sscachian/`](../../go/sscachian/)（公開） / [`go/sscachian/driver/memory/`](../../go/sscachian/driver/memory/)

| 要素 | 内容（決定済み仕様） |
| --- | --- |
| Entry | `{ value, created_at, expires_at }`。アプリ `Get` は value のみ |
| Layer 面（メモリが実装） | Get / Set / Delete / Incr。PurgeExact はスタブまたは未実装でよい（v0.7.0） |
| インメモリ | sync.Map 相当、プロセスローカル、Get 時遅延削除、`any` 保持 |
| Key | `{app}:cache:{tenant}:{query}:{version}` / `__version__` |
| Version | 初回 Get で L1 に `1` 作成。Bump は read→+1→write。Set/Delete 後自動 Bump |
| Cache Type API | Define / WithLayers / WithKeyBuilder / WithPolicy・TTL / WithLoader / Build |
| 読み書き | Get / GetOrLoad / Set / Delete / CurrentVersion / BumpVersion / BuildKey / BuildLatestKey |
| 多層 | **この版は L1 のみ**（WithLayers は 1 本前提でよい。複数 Layer の連鎖は v0.5.0） |

### 3. 受け入れ条件

- [ ] 上記テスト仕様が `docs/tests/` にある
- [ ] 単体結合テスト（正常・異常）が Green
- [ ] メモリ L1 のみで Get/Set/Delete/GetOrLoad + 自動/明示 Bump が動く
- [ ] `VERSION` = `0.3.0`（Git タグは develop マージ方針に従い、このブランチ作業中は必須としない）
- [ ] `make lint test` 通過

### 4. ドキュメント更新（実装と並行〜完了時）

- [roadmap.md](../../roadmap.md) に v0.2.0 欠番を明記
- [docs/tests/README.md](../../tests/README.md) 索引を更新
- [docs/log.md](../../log.md)
- 実装後、必要なら骨格を `docs/specs/` へ（機能が「現行に存在する」段階で）

## 依存・前提

- v0.1.0 スキャフォールド済み（`dev-v0.3.0` 先端）
- 振る舞いは [behavior](../../behavior.md) / [drivers](../../drivers.md) / [api](../../api.md)
