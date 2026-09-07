---
type: Concept
title: 振る舞い
description: キー設計、多層キャッシュ、Version、Purge、エントリメタの決定事項。
tags: [behavior, decided]
timestamp: 2026-09-07T12:00:00Z
---

# 振る舞い

## キー

ビジネス文脈をキーに含める。構造例:

```text
{app_slug}:cache:{tenant_id}:{query_type}:{version}
```

例:

```text
my-app:cache:0123456:client_list_page1:7
```

- Version はキー名の一部として埋め込む。
- 同一論理エントリは、全 Layer で **同じキー名** を使う。

## 多層キャッシュ

```text
L1 → miss → L2 → … → Loader → 上位へ書き戻し
```

- 上位 hit 時は下位を見ない。
- 下位 hit または Loader 成功時、上位 Layer へ **無条件で書き戻す**。
- TTL は **Layer ごとに別値** を持てる。
- L1 / L2 / … は Cache Type が定義した順序上の名前である。
- L1 はインメモリであることが多いが、必須ではない。

## Version（第1級 invalidate）

```text
Version = 論理 invalidate
TTL     = 物理 cleanup
```

- Mutation 後は該当スコープの version を進める。
- Version bump 時に Layer を能動クリアしない。旧キーは TTL で消える。
- 初期の Get は **最新 version のみ** 返す。旧世代の取得・列挙は初期スコープ外。

## Purge

- Exact / Prefix / Tag など、多彩な明示パージを提供する。
- 日常の invalidate は Version、明示削除・運用は Purge。
- Purge のデフォルト対象は **全 Layer**。
- 引数またはオプションで Layer 配列を渡し、単一または複数 Layer に絞れる。

## エントリメタ（初期）

- `created_at`
- `expires_at`

これ以外は後続で拡張する。
