---
type: Plan
title: Phase 1 PoC（v0.7.0）方針
description: Go PoC 実装前に固めた方針確定。Issue #7 および残件一問一答を転記。
tags: [plan, poc]
timestamp: 2026-09-27T01:00:00Z
---

# Phase 1 PoC（v0.7.0）方針

- **状態:** 方針確定
- **マイルストーン:** v0.7.0（[ロードマップ](../../roadmap.md) Phase 1）
- **関連 Issue:** #7
- **正本:** [behavior](../../behavior.md) / [drivers](../../drivers.md) / [api](../../api.md) / [development](../../development.md)

## 目的

Go でキービルダー・多層・インメモリ + Firestore の PoC を実装できる粒度まで、境界条件と契約を固める。

## やらぬこと（PoC）

- 多言語共通シリアライズ
- 旧 version 列挙・取得
- 分散 L1 無効化通知
- SWR / SIE / negative cache
- Purge Prefix / Tag
- Firestore ネイティブ TTL ポリシー

## 決定サマリ（優先 10 問 + 残件 3 問）

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
| 11 | パッケージ構成 | `go/sscachian` + `go/sscachian/driver/...`（他言語と並列） |
| 12 | テスト仕様 | `docs/tests/` をドメイン別（正常≈3・異常≈3〜5） |
| 13 | devcontainer | Go 1.26 + Firestore Emulator + `act` |

## 次アクション

マイルストーンは [roadmap](../../roadmap.md) に従う。

1. **v0.1.0** 土台（スキャフォールド）— 本リポジトリで充足中（テスト仕様本文は書かない）
2. 以降、各版の実装前に当該範囲の `docs/tests/` を書いてから実装（v0.3.0 → v0.5.0 → v0.7.0）
