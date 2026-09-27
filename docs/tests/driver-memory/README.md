---
type: TestSpec
title: driver-memory テスト仕様
description: インメモリ Layer（v0.3.0）。正常≈3 / 異常≈3〜5。
tags: [tests, driver-memory, v0.3.0]
timestamp: 2026-09-27T01:40:00Z
---

# driver-memory

対象: `go/sscachian/driver/memory`  
前提: [drivers](../../drivers.md) / [behavior](../../behavior.md)  
Entry 形: `{ value, created_at, expires_at }`。値は `any`。TTL は Get 時に `expires_at` を見て遅延削除。

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
- 破損・型不正な内部値があってもパニックせずエラーまたは Miss になる
- 並行 Get 中に期限切れ判定してもデータ競合でパニックしない

---

### Set

- キーに Entry を保存する。`created_at` / `expires_at` を付与または引数 TTL から算出する。
- 同一キーは上書きする。
- ゴルーチン安全である。

#### テスト：正常系

- 新規キーに Set すると直後の Get で Hit する
- 同一キーへ再 Set すると新しい `value` に置き換わる
- TTL 付き Set のあと、期限前は Hit、人工的に時刻を進める（または過去の `expires_at` を書く）と Get で Miss になる

#### テスト: 異常系

- 空キーで Set するとエラーになる
- `value` が nil でもエントリとして保存できる、または明示エラーになる（実装で一方に決め、仕様と一致させる）
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

- 整数カウンタキーを原子的に +1（または delta）する。current-version 用。
- キー未作成時は初期値から開始する（version 用の初期は上位が `1` を書く前提だが、Driver 単体では「未作成→指定初期値または 0/1」を契約で固定する。**PoC: 未作成時は 1 を書いてから +1 せず「セットして返す」、または「0 から +1 して 1」**。アプリの current-version 初回は Cache Type 側が `1` を Set する経路と、Bump の Incr 経路を分けてよい）。
- **Driver 契約（v0.3.0）:** 未作成キーに Incr した場合、結果の値は **1**（0+1）とする。Cache Type の「初回 Get で `__version__=1` を作成」は Incr ではなく Set で行う。

#### テスト：正常系

- 未作成キーへ Incr すると値が 1 になり、再 Incr で 2 になる
- 既存の整数値 n へ Incr すると n+1 になる
- 並行に複数 Incr しても最終値が呼び出し回数ぶん増える（最良努力のアプリ Bump とは別。Driver Incr は内部で競合に耐える）

#### テスト: 異常系

- 空キーで Incr するとエラーになる
- 整数以外が格納されているキーへ Incr するとエラーになる
- オーバーフローしうる極大値付近ではエラーまたはラップを契約どおりに扱う（PoC は error でよい）
