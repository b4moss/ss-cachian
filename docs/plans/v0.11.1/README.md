---
type: Plan
title: v0.11.1 公開消費者向け hotfix
description: npm file: 依存 ENOENT と Go サブディレクトリモジュール／タグ不一致の緊急修正。
tags: [plan, hotfix, v0.11.1]
timestamp: 2026-10-22T07:00:00Z
---

# v0.11.1 — 公開版コンシューマ修正（hotfix）

- **状態:** 実装中（パッチ）。マージ・タグ・npm publish・GitHub Release は人間側
- **マイルストーン:** v0.11.1（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `cursor/hotfix-publish-consumer-0.11.1-*` → **`main`**（緊急修正。憲章 Hotfix）
- **やらないこと:** マージ、タグ作成、npm publish、Release 作成、secret / リポ設定変更

## 目的

1. `@b4moss/ss-cachian` を `npm i` で入れられるようにする（レジストリ `@b4moss/es4@0.10.0`）
2. Go を `go get` → import → build できるようにする（サブディレクトリ module path + タグ）
3. 再発防止の CI（npm pack 空ディレクトリ install / `file:` 検出、Go コンシューマ検証）

## Go 方式の決定

**採用:** サブディレクトリモジュール（`github.com/b4moss/ss-cachian/go/sscachian`）＋ タグ `go/sscachian/v0.11.1`。

**理由:**

- `go.mod` は既に `go/sscachian/` にあり、モノレポ（`node/sscachian` 並置）を維持する
- ルート `v*` タグだけだと、ネストした `go.mod` 配下が module zip から除外され `.go` が届かない（0.11.0 の実害）
- 同一 org の `b4moss/es4`（`packages/go` + `packages/go/v*`）と同じ正しい慣例に揃える
- ルートへ `go.mod` を移す案は import を短く保てるが、ディレクトリ大移動が必要で hotfix 向きでない

npm 用に **同じコミット**へルートタグ `v0.11.1` も打つ（既存 `npm-publish.yml` の `v*` トリガー）。

## 受け入れ

- [x] `package.json` の `@b4moss/es4` がレジストリ版
- [x] `npm pack` → 空 dir install が CI / `make verify-npm-pack` で通る
- [x] Go module path = `.../go/sscachian`、README の `go get` がそれに一致
- [x] `make verify-go-consumer`（replace コンシューマ）が通る
- [ ] 人間が `main` マージ後に `v0.11.1` と `go/sscachian/v0.11.1` を打ち、npm / `go get` を確認
