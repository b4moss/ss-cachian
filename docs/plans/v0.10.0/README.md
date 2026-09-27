---
type: Plan
title: v0.10.0 Purge 拡張と便利 API
description: PurgePrefix/Tag と Has/GetEntry/Remember 系を同一マイルストーンで実装する。Go / Node パリティ。
tags: [plan, phase2, v0.10.0]
timestamp: 2026-09-27T08:05:00Z
---

# v0.10.0 Purge 拡張 ＋ 便利読み書き API

- **状態:** 実装中（`dev-v0.10.0` 向け）
- **マイルストーン:** v0.10.0（[roadmap](../../roadmap.md) Phase 2）
- **作業ブランチ:** `dev-v0.10.0` → `develop` → `main`
- **テスト仕様:** [tests/purge](../../tests/purge/) / [tests/cache-type](../../tests/cache-type/) / driver PurgePrefix 節
- **意味論正本（実装後）:** [specs/purge](../../specs/purge/) / [specs/cache-type](../../specs/cache-type/) / [specs/layer](../../specs/layer/)

## 目的

1. Exact に加え `PurgePrefix` / `PurgeTag`、アプリ `PurgeExact` 別名（非破壊）
2. `Has` / `Exists` / `GetEntry` / `Remember` / `RememberForever` / `Forget` の薄い便利面
3. Go / Node 同等セマンティクス

## やらぬこと

- SWR / SIE / negative、分散 L1、Valkey、PHP
- 設定スキーマ変更（v0.9.0 のまま）
- README バッジ / Scorecard / Codecov（製品実装とは別レーン）
- versioned データキーのレイアウト変更

## セマンティクス（固定）

### 便利 API

| API | 振る舞い |
| --- | --- |
| `Has` / `Exists` | `Get` 相当のあと ok（Exists はエイリアス） |
| `GetEntry` | Latest Hit で value + `created_at` / `expires_at`。Miss は ok=false |
| `Forget` | `Delete` エイリアス（Bump 含む） |
| `Remember(ctx, kc, ttl, loader)` | Hit → キャッシュ。Miss → loader → **全 Layer** 書き戻し（**Bump なし**）。TTL は引数を全 Layer に適用。負 TTL は拒否 |
| `RememberForever` | `Remember` with TTL `0` |

YAML 変更なし。タグ付き書き込みは下記 Tags。

### PurgePrefix

- Layer `PurgePrefix(ctx, prefix)`: ID が `prefix` で始まる **すべて**削除（`__version__`・非数字サフィックス含む）。空 → `ErrEmptyKey`。冪等
- CacheType `PurgePrefix(ctx, prefix, layerIdx ...int)`: Layer 選択・失敗方針は現行 `Purge` と同型
- アプリ `Purge` は Exact のまま。`PurgeExact` = `Purge` 別名

### Tags + PurgeTag

- 書き込みオプション: Go `WithTags(...)` / Builder `WithDefaultTags`。Node `{ tags }` / `withDefaultTags`
- 既定 + 呼び出しは和集合（空文字タグは無視）
- L1 逆引き: キー `__sscachian_tag__:{tag}` → 論理プレフィックス集合（Entry 値）。Set / Remember / GetOrLoad 充填が L1 成功したとき更新
- `PurgeTag(ctx, tag, layerIdx ...int)`: 索引から prefix を取り各々 Exact Purge → 索引削除。未登録は成功。空タグ → `ErrEmptyKey`

## 作業順（TDD）

1. ~~テスト仕様~~ — 済
2. Layer `PurgePrefix`（memory + firestore）
3. CacheType: 便利 API → `PurgePrefix` / `PurgeExact` → タグ索引 + `PurgeTag`
4. specs 更新（purge / cache-type / layer）
5. `VERSION` / `package.json` = `0.10.0`
6. 昇格: `dev-v0.10.0` → `develop` → `main` → タグ / npm（承認後）

## 受け入れ条件

- [x] テスト仕様（purge 拡張 ＋ cache-type 便利 API）
- [ ] Go / Node Green・パリティ
- [ ] Exact Purge / config LoadTypes 回帰なし
- [ ] `VERSION` / package = `0.10.0`（タグ・npm は main 昇格後）
