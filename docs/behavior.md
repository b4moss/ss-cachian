---
type: Concept
title: 振る舞い
description: キー設計、多層キャッシュ、Version、Purge、エントリメタの決定事項。
tags: [behavior, decided]
timestamp: 2026-09-27T03:05:00Z
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

### current-version

ライブラリが現在版番号を表す 1 キーを持ち、`Get` / `BumpVersion` がそれを参照・更新する。

- 置き場: Cache Type の **L1**
- キー名: データキーと同系で、末尾を version 用にする

```text
{app_slug}:cache:{tenant_id}:{query_type}:__version__
```

例:

```text
my-app:cache:0123456:client_list_page1:__version__
```

#### 初回 Get（未作成時）

- `__version__` が無いとき、デフォルト値 **`1`** とみなす。
- そのタイミングで L1 に `__version__` を **作成する**（デフォルト値で書き込み）。
- 以降の lookup は作成済みの current-version を使う。

## 多層キャッシュ

```text
L1 → miss → L2 → … → Loader → 上位へ書き戻し
```

- 上位 hit 時は下位を見ない。
- 下位 hit または Loader 成功時、上位 Layer へ **無条件で書き戻す**。
- TTL は **Layer ごとに別値** を持てる。
- L1 / L2 / … は Cache Type が定義した順序上の名前である。
- L1 はインメモリであることが多いが、必須ではない。

### 書き戻し途中失敗

- 読み取り（下位 hit または Loader）が成功していれば、その **値は呼び出し側へ返す**。
- 上位 Layer への書き戻しが一部失敗しても、成功した Layer の結果は維持する。
- 失敗はログして無視する（読み取り成功を優先）。

## Version（第1級 invalidate）

```text
Version = 論理 invalidate
TTL     = 物理 cleanup
```

- `Set` は **先に** L1 で `BumpVersion` し、その新 version キーへ書く。`Delete` は最新キー削除の **あと** に `BumpVersion` する。
- 明示の `BumpVersion` も提供する。
- `BumpVersion` は **最良努力**（read → +1 → write）。競合時の上書き負けを許容し、最終的に番号が進んでいれば十分とする。
- Version bump 時に Layer を能動クリアしない。旧キーは TTL で消える（Purge しない限り）。
- `Set` で L1 への書き込みが失敗しても、既に進んだ version は戻さない。
- 初期の Get は **最新 version のみ** 返す。旧世代の取得・列挙は初期スコープ外。

## Purge

- Exact / Prefix / Tag など、多彩な明示パージを提供する。
- 日常の invalidate は Version、明示削除・運用は Purge。
- Purge のデフォルト対象は **全 Layer**。
- `Purge(ctx, kc, layerIdx ...int)` で Layer インデックスを渡し、単一または複数に絞れる（未指定は全 Layer）。
- Purge は **データキーのみ** 削除し、`__version__` は進めない・消さない。

### Exact Purge（現行）

- 指定論理キーに紐づく **全 version のデータキー** を、対象 Layer から削除する。
- `__version__` は残す。

## エントリメタ（初期）

保存時はラッパー必須:

```text
{ value, created_at, expires_at }
```

- アプリ向け `Get` は `value` だけ返す。
- 現行ではこれ以外のメタを設計しない。SWR / SIE / negative cache 用フィールドは後続 Phase で追加する。
- 用語と追加忘れ防止のメモは [未決事項](./open-questions.md) を参照。
