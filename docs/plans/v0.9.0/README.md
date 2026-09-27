---
type: Plan
title: v0.9.0 設定ファイル駆動
description: YAML/JSON から Cache Type（Layers / TTL / KeyBuilder 参照 / Policy）を組み立てる。
tags: [plan, phase2]
timestamp: 2026-09-27T05:56:00Z
---

# v0.9.0 設定ファイル駆動

- **状態:** 意図スタブ（未着手）
- **マイルストーン:** v0.9.0（[roadmap](../../roadmap.md)）
- **作業ブランチ:** `dev-v0.9.0` → `develop` → `main`

## 目的

設定ファイルから Cache Type をロードできるようにする（Go / Node）。Loader はコード登録のままを既定とする。

## 設定雛形（レビュー用）

- [sscachian.example.yaml](./sscachian.example.yaml) — スキーマ案と具体例（未実装・仕様未固定）

主な方針メモ:

- YAML 正、同等 JSON 可
- `layers[].driver` + 任意 `ttl` / `options`
- `key_builder` / `loader` は **登録名の参照**（実体はコード）
- 型パラメータ `T` は設定に書かない

## やらぬこと

- Purge 拡張、便利 API、SWR、Valkey、PHP
- ブラウザ向けバンドル

## 受け入れ条件（概要）

- [ ] 設定スキーマとテスト仕様
- [ ] Go / Node で同等セマンティクスのロード
- [ ] `VERSION` / package = `0.9.0`
