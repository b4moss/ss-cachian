---
type: TestSpec
title: driver-firestore テスト仕様
description: Firestore Layer（Emulator）。Go v0.5.0 導入・Node v0.8.0・PurgePrefix v0.10.0・v0.12.0 でゼロ値 expires_at 抑止。
tags: [tests, driver-firestore, v0.5.0, v0.8.0, v0.10.0, v0.12.0, node]
timestamp: 2026-10-28T00:00:00Z
---

# driver-firestore

対象:

- Go: `go/sscachian/driver/firestore`（`New` 既定: ProjectID `ss-cachian-dev` / Collection `sscachian`）
- Node: `node/sscachian` の Firestore ドライバ（`@google-cloud/firestore`）

前提: [specs/driver-firestore](../../specs/driver-firestore/) / [plans/v0.12.0/plan.md](../../plans/v0.12.0/plan.md) / [specs/layer](../../specs/layer/) / [tests 索引（Node 差分）](../README.md)  
実行: Firestore Emulator（`FIRESTORE_EMULATOR_HOST`）。Go は既存 `test-go`、Node は `test-node` で起動する。

ドキュメント:

- 1 キャッシュキー = 1 ドキュメント
- フィールド: `value`（JSON）、`created_at`、**期限があるときだけ** `expires_at`（日時型）
- Get 時に `expires_at` 判定（論理 Miss）。利用側が Firestore TTL ポリシーを掛ける前提（ポリシー作成はライブラリ外。設定は docs）
- **ゼロ値（`0001-01-01` / Unix epoch）の `expires_at` を書かない**（v0.12.0）
- ドキュメント ID: キー文字列をそのまま使う（URL エンコード／サニタイズは現行では行わない）

固定セマンティクス:

- `Incr` 未作成 → **0+1 → 1**（トランザクション。Aborted はリトライ）
- Driver は JSON 往復し、Get 時はデコードして Entry.Value に載せる
- null value の Set は JSON `null` として保存可
- 期限なし Set / Incr は `expires_at` フィールドを **省略**する。期限あり→なし上書きでは既存フィールドを **削除**する

---

### Get

- ドキュメントを読み Entry を返す。
- `expires_at` が過去なら Miss（**論理 Miss とし、削除は必須としない**）。
- `expires_at` 欠落 = 期限なし。

#### テスト：正常系

- Set したキーで Hit し、value / created_at / expires_at が復元できる
- 期限切れドキュメントは Miss になる
- 未存在キーは Miss（エラーではない）
- `expires_at` 欠落のドキュメントは無期限として Hit

#### テスト: 異常系

- 空キーは ErrEmptyKey
- `value` が壊れた JSON でもパニック／未処理例外にせずエラーまたは Miss
- Emulator 未接続などクライアントエラーは error として返す

---

### Set

- ドキュメントを作成／上書きする。
- TTL>0 なら `expires_at = now+ttl`（日時型）。TTL=0 かつ Entry.ExpiresAt 指定があればそれを使う。
- TTL=0 かつ ExpiresAt ゼロなら **`expires_at` を書かない**（ゼロ値・null 禁止）。
- 期限ありドキュメントを期限なしで上書きしたとき、`expires_at` フィールドが **消える**（`FieldValue.delete()` 相当）。
- `value` は JSON エンコードして保存する。

#### テスト：正常系

- 新規 Set → Get で同じ論理値が取れる（文字列・数値・構造体相当）
- 同一キー再 Set で上書きされる
- TTL 付き Set のあと、期限前は Hit、過去 `expires_at` を書けば Get で Miss
- 期限なし Set 後、Emulator 上のドキュメントに `expires_at` キーが存在しない
- 期限なし Set 後、`expires_at` がゼロ日時（epoch / year 1）になっていない
- TTL 付き Set → 期限なし Set のあと、ドキュメントから `expires_at` が消えている

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
- 書き込み時に **ゼロ値 `expires_at` を付けない**（フィールド省略）。既存に `expires_at` がある場合の扱い（削除して無期限化 / 維持）は計画の要決定に従い固定する。

#### テスト：正常系

- 未作成 Incr → 1、再 Incr → 2
- 既存 10 へ Incr → 11
- 並行 Incr の最終値が呼び出し回数と一致する（Emulator 上）
- 未作成 Incr 後、ドキュメントに `expires_at` が無い（ゼロ値も無い）

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

---

### PurgePrefix

- `PurgePrefix(..., prefix)` はコレクション内のドキュメント ID が `prefix` で **始まるすべて**を削除する（`__version__`・非数字サフィックス含む）。
- Exact より広い。対象が無くても成功（冪等）。Emulator 上で検証。空 prefix は `ErrEmptyKey`。

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` / `prefix:extra` があるとき PurgePrefix(`prefix`) 後はすべて Miss
- 別先頭 `other:1` は残る
- 対象ドキュメントが無い状態でもエラーにならない

#### テスト: 異常系

- 空 prefix は ErrEmptyKey
- クライアントエラーは error として返す
- 先頭一致のみ（途中一致では消さない）

----

以上
