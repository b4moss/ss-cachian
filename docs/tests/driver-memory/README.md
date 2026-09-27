---
type: TestSpec
title: driver-memory テスト仕様
description: インメモリ Layer。Go v0.3.0 導入・Node v0.8.0・PurgePrefix v0.10.0。正常≈3 / 異常≈3〜5。
tags: [tests, driver-memory, v0.3.0, v0.8.0, v0.10.0, node]
timestamp: 2026-09-27T07:57:34Z
---

# driver-memory

対象:

- Go: `go/sscachian/driver/memory`
- Node: `node/sscachian` の memory ドライバ（例: `driver/memory`）

前提: [specs/driver-memory](../../specs/driver-memory/) / [specs/layer](../../specs/layer/) / [tests 索引（Node 差分）](../README.md)  
Entry 形: `{ value, created_at, expires_at }`。値は任意。TTL は Get 時に `expires_at` を見て遅延削除。  
Node: 単一プロセス内の同期 Map + 排他（mutex 相当）。公開メソッドは `async` で揃えてよい。

---

### Get

- キーでエントリを取得する。
- `expires_at` が過去なら削除して Miss とする。
- Hit 時は Entry 全体（または Driver 契約どおりの戻り）を返す。

#### テスト：正常系

- 存在する未期限切れキーで Hit し、保存した `value` / `created_at` / `expires_at` が取れる
- 期限切れキーで Miss となり、ストアからそのキーが消える
- 未存在キーで Miss（エラーではなく不在）となる

#### テスト: 異常系

- 空キーを渡すとエラー（または契約どおりの無効引数扱い）になる
- 破損・型不正な内部値（`Entry` 以外）があってもパニック／未処理例外にせず **Miss** になり、キーは削除される
- 並行 Get 中に期限切れ判定してもデータ競合でパニックしない

---

### Set

- キーに Entry を保存する。`created_at` / `expires_at` を付与または引数 TTL から算出する。
- 同一キーは上書きする。
- 並行安全である。

#### テスト：正常系

- 新規キーに Set すると直後の Get で Hit する
- 同一キーへ再 Set すると新しい `value` に置き換わる
- TTL 付き Set のあと、期限前は Hit、人工的に時刻を進める（または過去の `expires_at` を書く）と Get で Miss になる

#### テスト: 異常系

- 空キーで Set するとエラーになる
- `value` が null/undefined/nil でもエントリとして保存できる（許容）
- 負の TTL や不正な時刻指定はエラーになる

---

### Delete

- 指定キーのエントリを削除する。
- 存在しなくても成功扱い（冪等）とする。

#### テスト：正常系

- 存在するキーを Delete すると以降 Get は Miss になる
- 存在しないキーを Delete してもエラーにならない
- Delete 後に同じキーへ Set し直すと再び Hit する

#### テスト: 異常系

- 空キーで Delete するとエラーになる
- 並行で Set と Delete してもパニックしない
- Delete 対象以外のキーは影響を受けない

---

### Incr

- 整数カウンタキーを原子的に +1 する。current-version 用。
- **Driver 契約:** 未作成キーに Incr した場合、結果の値は **1**（0+1）とする。Cache Type の「初回 Get で `__version__=1` を作成」は Incr ではなく Set で行う。

#### テスト：正常系

- 未作成キーへ Incr すると値が 1 になり、再 Incr で 2 になる
- 既存の整数値 n へ Incr すると n+1 になる
- 並行に複数 Incr しても最終値が呼び出し回数ぶん増える（Driver Incr は内部で競合に耐える）

#### テスト: 異常系

- 空キーで Incr するとエラーになる
- 整数以外が格納されているキーへ Incr するとエラーになる
- オーバーフローしうる極大値付近ではエラーまたはラップを契約どおりに扱う（error でよい）

---

### PurgeExact

- `PurgeExact(..., logicalPrefix)` は `{logicalPrefix}:{n}`（n が 1 以上の数字）のエントリだけ削除する。
- `{logicalPrefix}:__version__` および別プレフィックスのキーは残す。
- 対象が無くても成功（冪等）。排他下で map を走査する。

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` があるとき PurgeExact 後、数字キーだけ Miss で `__version__` は Hit
- 別プレフィックス `other:1` は影響を受けない
- 対象キーが無い状態でもエラーにならない

#### テスト: 異常系

- 空の logicalPrefix は ErrEmptyKey
- 並行に Set と PurgeExact してもパニックしない
- `prefix:__version__` だけがあるときも `__version__` は残る

---

### PurgePrefix

- `PurgePrefix(..., prefix)` はキー（ドキュメント ID）が `prefix` で **始まるすべて**を削除する（数字サフィックス以外・`__version__` も含む）。
- Exact（`IsVersionDataKey`）より広い。対象が無くても成功（冪等）。空 prefix は `ErrEmptyKey`。

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` / `prefix:extra` があるとき PurgePrefix(`prefix`) 後はすべて Miss
- 別先頭 `other:1` は残る。より短い共通先頭（例: `pre`）でも `prefix:…` は消える
- 対象キーが無い状態でもエラーにならない

#### テスト: 異常系

- 空 prefix は ErrEmptyKey
- 並行に Set と PurgePrefix してもパニックしない
-（任意）`prefix` が他キーの途中一致だけでは消さない（**先頭一致のみ**）
