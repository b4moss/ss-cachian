---
type: Spec
title: config
description: YAML/JSON からの Cache Type ロード（Registry / LoadTypes）。v0.9.0。
tags: [specs, config, v0.9.0]
timestamp: 2026-09-27T06:40:00Z
---

# config

対象:

- Go: `NewRegistry` / `RegisterLoader` / `RegisterKeyBuilder` / `LoadTypes`（`LoadTypesYAML` / `LoadTypesJSON`）
- Node: `createRegistry` / `registerLoader` / `registerKeyBuilder` / `loadTypes`（`loadTypesYAML` / `loadTypesJSON`）

関連: [plans/v0.9.0](../../plans/v0.9.0/) / [tests/config](../../tests/config/) / [cache-type](../cache-type/)

## 概要

設定ファイルから Cache Type を組み立てる。Loader・カスタム KeyBuilder の実体はコード側で名前登録し、設定は参照のみ。型パラメータ `T` は設定に書かない。コード面の `Define`…`Build` も併用可。

## スキーマ（詳細）

手厚い利用ガイド（フィールド表・TTL・options・拒否例）は
[ルート README（英語）](../../../README.md#config-file-yaml--json) および
[node/sscachian/README.md](../../../node/sscachian/README.md#config-driven-v090) を参照。
雛形は [plans/v0.9.0/sscachian.example.yaml](../../plans/v0.9.0/sscachian.example.yaml)。

- `schema_version`: `"0.9"`（文字列必須）
- `types[]`: `name` / 任意 `key_builder`（省略=`default`）/ 任意 `loader` / 任意 `ttl` / `layers[]`
- `layers[]`: `driver`（`memory` | `firestore`）/ 任意 `ttl` / 任意 `options`
- TTL 文字列: Go `time.ParseDuration` 互換。優先: `layers[i].ttl` → `types[].ttl` → `0`
- firestore `options`: `project_id` / `collection` のみ。未知キーは拒否。Emulator は `FIRESTORE_EMULATOR_HOST`
- memory `options`: 空のみ（未知キー拒否）

## Registry

- 新規時に KeyBuilder `"default"` を登録
- 同名再登録は後勝ち
- 空名 / nil 関数はエラー

## ロード結果

- Go: `map[string]*CacheType[T]`
- Node: `Map<string, CacheType>`
- 意味論（Get/Set/GetOrLoad/Purge）は設定経路でも [cache-type](../cache-type/) / [layer](../layer/) と同一
- ロード後に Registry を変更しても、既に構築した Type の参照は変わらない

## Driver 登録（Go）

`driver/memory` と `driver/firestore` の `init` が `RegisterDriverFactory` する。`LoadTypes` 利用時はこれらのパッケージを import（テストは blank import）すること。
