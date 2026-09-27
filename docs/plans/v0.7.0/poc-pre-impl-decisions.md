---
type: Plan
title: Phase 1 PoC（v0.7.0）方針
description: Go PoC 実装前に固めた方針確定。Issue #7 の優先一問一答を転記。
tags: [plan, poc]
timestamp: 2026-09-27T00:00:00Z
---

# Phase 1 PoC（v0.7.0）方針

- **状態:** 方針確定
- **マイルストーン:** v0.7.0（[ロードマップ](../../roadmap.md) Phase 1）
- **関連 Issue:** #7
- **正本の振る舞い:** [behavior](../../behavior.md) / [drivers](../../drivers.md) / [api](../../api.md)

## 目的

Go でキービルダー・多層・インメモリ + Firestore の PoC を実装できる粒度まで、境界条件と契約を固める。

## やらぬこと（PoC）

- 多言語共通シリアライズ
- 旧 version 列挙・取得
- 分散 L1 無効化通知
- SWR / SIE / negative cache
- Purge Prefix / Tag
- Firestore ネイティブ TTL ポリシー

## 決定サマリ（優先 10 問）

| # | 論点 | 決定 |
| --- | --- | --- |
| 1 | 初回 Get | `__version__` 未作成時はデフォルト `1` で作成する |
| 2 | BumpVersion 競合 | 最良努力（read → +1 → write）。上書き負け許容 |
| 3 | Set / Delete / Purge | Set/Delete は自動 Bump。Purge はデータのみ（`__version__` 非接触） |
| 4 | Layer 書き戻し失敗 | 読み取り成功優先。失敗はログして無視 |
| 5 | Entry 形 | `{ value, created_at, expires_at }` 必須。Get は value のみ |
| 6 | Go 値保持 | メモリ `any` / Firestore JSON / 境界は generics |
| 7 | Layer 契約 | Get / Set / Delete / Incr / PurgeExact |
| 8 | インメモリ | sync.Map 相当・遅延削除・ゴルーチン安全 |
| 9 | Firestore | 1 キー 1 doc・読み時 TTL 判定 |
| 10 | Purge Exact | 全 version データ削除。`__version__` は残す |

## 次アクション

1. Go パッケージ構成を決める
2. `docs/tests/` に PoC テスト仕様を書く（TDD）
3. devcontainer 最小構成を用意する
4. Phase 1 実装（Red → Green → Refactor）
