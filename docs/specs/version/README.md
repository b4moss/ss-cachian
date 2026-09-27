---
type: Spec
title: version
description: current-version / Bump / キー組み立て（現行 v0.7.0）。
tags: [specs, version]
timestamp: 2026-09-27T03:05:00Z
---

# version

対象: L1 上の `__version__` とキービルド  
関連: [behavior](../../behavior.md) / [tests/version](../../tests/version/)

## キー

- 論理プレフィックス: KeyBuilder の戻り（末尾に `:{n}` / `:__version__` を付けない）。
- データキー: `{prefix}:{version}`（version ≥ 1）。
- current-version キー: `{prefix}:__version__`（**L1 のみ**）。

## 振る舞い

- 初回参照で `__version__` が無ければ **Set で 1** を作成する（Incr ではない）。
- `CurrentVersion` / `BumpVersion` / `BuildLatestKey` はこの経路を使う。
- `BumpVersion` は L1 `Incr`（最良努力）。`Set` は bump-then-write、`Delete` は delete-then-bump。
- Version bump では Layer を能動クリアしない。旧データキーは TTL または Purge で消える。
