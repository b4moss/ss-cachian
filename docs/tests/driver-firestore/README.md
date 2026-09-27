---
type: TestSpec
title: driver-firestore テスト仕様
description: Firestore Layer（v0.5.0・Emulator）。正常≈3 / 異常≈3〜5。
tags: [tests, driver-firestore, v0.5.0]
timestamp: 2026-09-27T02:10:00Z
---

# driver-firestore

対象: `go/sscachian/driver/firestore`  
前提: [drivers](../../drivers.md) / [behavior](../../behavior.md)  
実行: Firestore Emulator（`FIRESTORE_EMULATOR_HOST`）。CI / compose で起動する。

ドキュメント:

- 1 キャッシュキー = 1 ドキュメント
- フィールド: `value`（JSON バイト）、`created_at`、`expires_at`
- TTL は Get 時に `expires_at` 判定（ネイティブ TTL ポリシーは使わない）
- ドキュメント ID: キー文字列をそのまま使う（使用不可文字があれば URL セーフにエンコードして固定）

PoC 固定:

- `Incr` 未作成 → **0+1 → 1**（トランザクション）
- `value` は JSON。アプリ型 `T` への復元は CacheType 側。Driver は `[]byte` またはデコード済み `any` を Entry.Value に載せる（**PoC: Entry.Value はデコード後の `any`（map/slice/スカラー）。CacheType が JSON 経由で T に合わせる場合は Driver が `json.RawMessage`/`[]byte` を返し CacheType が Unmarshal — 実装は「Driver が JSON 往復し、Get 時は `json.Unmarshal` して `any`」で固定**）
- nil value の Set は空 JSON `null` として保存可

---

### Get

- ドキュメントを読み Entry を返す。
- `expires_at` が過去なら Miss（ドキュメント削除はしてもよい／しなくてもよい。**PoC: 論理 Miss とし、削除は必須としない**）。

#### テスト：正常系

- Set したキーで Hit し、value / created_at / expires_at が復元できる
- 期限切れドキュメントは Miss になる
- 未存在キーは Miss（エラーではない）

#### テスト: 異常系

- 空キーは ErrEmptyKey
- `value` が壊れた JSON でもパニックせずエラーまたは Miss
- Emulator 未接続などクライアントエラーは error として返す

---

### Set

- ドキュメントを作成／上書きする。
- TTL>0 なら `expires_at = now+ttl`。TTL=0 かつ Entry.ExpiresAt 指定があればそれを使う。
- `value` は JSON エンコードして保存する。

#### テスト：正常系

- 新規 Set → Get で同じ論理値が取れる（文字列・数値・構造体相当）
- 同一キー再 Set で上書きされる
- TTL 付き Set のあと、期限前は Hit、過去 `expires_at` を書けば Get で Miss

#### テスト: 異常系

- 空キーはエラー
- 負の TTL は ErrNegativeTTL
- JSON 化できない値（chan 等）はエラー

---

### Delete

- ドキュメントを削除する。無くても成功（冪等）。

#### テスト：正常系

- 存在するキーを Delete すると以降 Get は Miss
- 存在しないキーの Delete はエラーにならない
- Delete 後に Set し直すと再び Hit する

#### テスト: 異常系

- 空キーはエラー
- クライアントエラーは error
- 他キーのドキュメントは影響を受けない

---

### Incr

- トランザクションで整数カウンタを +1。
- 未作成は 1。既存 n は n+1。
- 値は Entry 互換で格納（整数フィールドまたは value JSON 内の整数。**PoC: `value` に JSON 数値として保持**）。

#### テスト：正常系

- 未作成 Incr → 1、再 Incr → 2
- 既存 10 へ Incr → 11
- 並行 Incr の最終値が呼び出し回数と一致する（Emulator 上）

#### テスト: 異常系

- 空キーはエラー
- 整数以外が格納されているキーへ Incr すると ErrNotInteger
- MaxInt64 付近は ErrIncrOverflow

---

### PurgeExact（v0.7.0）

- `PurgeExact(ctx, logicalPrefix)` はコレクション内のドキュメント ID が `{logicalPrefix}:{n}`（n≥1 の数字）のものだけ削除する。
- `{logicalPrefix}:__version__` および別プレフィックスは残す。
- 対象が無くても成功（冪等）。Emulator 上で検証する。

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` があるとき PurgeExact 後、数字キーだけ Miss で `__version__` は Hit
- 別プレフィックス `other:1` は影響を受けない
- 対象ドキュメントが無い状態でもエラーにならない

#### テスト: 異常系

- 空の logicalPrefix は ErrEmptyKey
- クライアントエラーは error として返す
- `__version__` ドキュメントだけがあるときも残る
