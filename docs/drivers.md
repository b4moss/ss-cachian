---
type: Concept
title: Driver と Layer 契約
description: Driver 実装順と Layer 共通契約の決定事項。
tags: [drivers, decided]
timestamp: 2026-09-07T12:00:00Z
---

# Driver と Layer 契約

## Driver 実装順

1. インメモリ
2. Firestore
3. Redis / Valkey
4. 以降: MongoDB、Object Storage、Filesystem 等

## Layer 共通契約

共通面は薄く保つ。

- 共通: Get / Set / Delete + 任意 TTL
- TTL 値は Layer ごとに異なってよい
- 原子性・Prefix・Tag・INCR などは Driver 固有 API
- 高度機能を全 Driver でエミュレートして揃えない
- 差し込み口は揃えても、レイテンシや TTL 意味まで同一保証しない

## Valkey 固有機能

Rate Limit / Lock / Stampede など Valkey 文脈の機能は、

- Cache Type が Valkey を選んだときだけ有効
- ライブラリ本体は実装せず、Valkey Driver が操作するモジュールへ渡すだけ
