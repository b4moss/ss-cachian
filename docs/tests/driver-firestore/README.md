---
type: TestSpec
title: driver-firestore テスト仕様
description: Firestore Layer（Emulator）。Go v0.5.0 導入・Node v0.8.0 再適用。正常≈3 / 異常≈3〜5。
tags: [tests, driver-firestore, v0.5.0, v0.8.0, node]
timestamp: 2026-09-27T04:23:00Z
---

# driver-firestore

対象:

- Go: `go/sscachian/driver/firestore`
- Node: `node/sscachian` の Firestore ドライバ（`@google-cloud/firestore`）

前提: [drivers](../../drivers.md) / [behavior](../../behavior.md) / [tests 索引（Node 差分）](../README.md)  
実行: Firestore Emulator（`FIRESTORE_EMULATOR_HOST`）。Go は既存 `test-go`、Node は `test-node` で起動する。

ドキュメント:

- 1 キャッシュキー = 1 ドキュメント
- フィールド: `value`（JSON）、`created_at`、`expires_at`
- TTL は Get 時に `expires_at` 判定（ネイティブ TTL ポリシーは使わない）
- ドキュメント ID: キー文字列をそのまま使う（使用不可文字があれば URL セーフにエンコードして固定）

固定セマンティクス:

- `Incr` 未作成 → **0+1 → 1**（トランザクション。Aborted はリトライ）
- Driver は JSON 往復し、Get 時はデコードして Entry.Value に載せる
- null value の Set は JSON `null` として保存可

---

### Get

- ドキュメントを読み Entry を返す。
- `expires_at` が過去なら Miss（**論理 Miss とし、削除は必須としない**）。

#### テスト：正常系

- Set したキーで Hit し、value / created_at / expires_at が復元できる
- 期限切れドキュメントは Miss になる
- 未存在キーは Miss（エラーではない）

#### テスト: 異常系

- 空キーは ErrEmptyKey
- `value` が壊れた JSON でもパニック／未処理例外にせずエラーまたは Miss
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
- JSON 化できない値はエラー（Node: BigInt や循環参照など）

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
- 値は `value` に JSON 数値として保持。

#### テスト：正常系

- 未作成 Incr → 1、再 Incr → 2
- 既存 10 へ Incr → 11
- 並行 Incr の最終値が呼び出し回数と一致する（Emulator 上）

#### テスト: 異常系

- 空キーはエラー
- 整数以外が格納されているキーへ Incr すると ErrNotInteger
- Number.MAX_SAFE_INTEGER / MaxInt64 付近は ErrIncrOverflow（実装の上限に合わせて固定）

---

### PurgeExact

- `PurgeExact(..., logicalPrefix)` はコレクション内のドキュメント ID が `{logicalPrefix}:{n}`（n≥1 の数字）のものだけ削除する。
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
