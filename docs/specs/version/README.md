---
type: Spec
title: version
description: current-version / Bump / キー組み立て（現行 v0.7.0）。
tags: [specs, version]
timestamp: 2026-09-27T05:00:00Z
---

# version

対象: L1 上の `__version__` とキービルド  
関連: [layer](../layer/) / [tests/version](../../tests/version/)

## キー

ビジネス文脈をキーに含める。構造例:

```text
{app_slug}:cache:{tenant_id}:{query_type}:{version}
```

例: `my-app:cache:0123456:client_list_page1:7`

- Version はキー名の一部として埋め込む。
- 同一論理エントリは、全 Layer で **同じキー名** を使う。
- 論理プレフィックス: KeyBuilder の戻り（末尾に `:{n}` / `:__version__` を付けない）。
- データキー: `{prefix}:{version}`（version ≥ 1）。
- current-version キー: `{prefix}:__version__`（**L1 のみ**）。

```text
{app_slug}:cache:{tenant_id}:{query_type}:__version__
```

## 振る舞い

- 初回参照で `__version__` が無ければ **Set で 1** を作成する（Incr ではない）。デフォルト値 **`1`**。
- `CurrentVersion` / `BumpVersion` / `BuildLatestKey` はこの経路を使う。
- `BumpVersion` は L1 `Incr`（最良努力: read → +1 → write。競合時の上書き負けを許容）。
- `Set` は bump-then-write、`Delete` は delete-then-bump。
- Version bump では Layer を能動クリアしない。旧データキーは TTL または Purge で消える。
- `Set` で L1 への書き込みが失敗しても、既に進んだ version は戻さない。
- 初期の Get は **最新 version のみ** 返す。旧世代の取得・列挙は現行スコープ外。

### Version と TTL

```text
Version = 論理 invalidate
TTL     = 物理 cleanup
```
