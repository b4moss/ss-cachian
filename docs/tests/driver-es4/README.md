---
type: TestSpec
title: driver-es4 テスト仕様
description: es4 Layer Driver。v0.11.0 導入。v0.12.0 で期限付き書き込み（es4 API）＋ Firestore 日時型 expires_at。
tags: [tests, driver-es4, v0.11.0, v0.12.0, node]
timestamp: 2026-10-28T00:00:00Z
---

# driver-es4

対象:

- Go: `go/sscachian/driver/es4`
- Node: `node/sscachian` の es4 ドライバ
- HTTP Es4 Server アダプタ（埋め込みと並列）

前提:

- [plans/v0.11.0/es4-layer-driver.md](../../plans/v0.11.0/es4-layer-driver.md)（Driver 導入・決定 2026-10-11）
- [plans/v0.12.0/plan.md](../../plans/v0.12.0/plan.md)（期限付き書き込み・Firestore TTL）
- [specs/layer](../../specs/layer/) / [tests 索引](../README.md)
- 上流 es4: キー区切り `:` ＋ `Scan` ＋ **期限付き書き込み**（API 名は仮。es4 計画 PR で確定後に本仕様の表記を置換）

固定セマンティクス:

| 項目 | 決定 |
|------|------|
| DB 所有権 | **A** — Driver が `Open` / `Close` を所有（v0.11） |
| Backend | **D** — Firestore まで全 Backend 公式サポート（CI は memory_only 基本） |
| TTL（v0.12） | es4 の期限付き書き込みを使う。Firestore では日時型 `expires_at`。期限なしはフィールド省略 |
| Entry value | キャッシュ payload の JSON。**value 内に文字列 `expires_at` を二重持ちしない**（推奨。旧形式フォールバックは要決定） |
| Incr 再試行 | Firestore Driver に揃える |
| PurgeExact | Scan なし。キー既知で version data のみ削除 |
| PurgePrefix | es4 `Scan(prefix)` → 各キー `Delete` |
| HTTP アダプタ | 埋め込みと同セマンティクス（期限付き含む） |

config `layers[].driver` 名は `es4`。本スイートは Driver 直接生成でも Config 経由でもよい。

---

### Open / Close（所有権 A）

#### テスト：正常系

- Open 成功後、Set → Get が動く（memory_only でよい）
- Close 後は以降の Layer 操作がエラーになる（または契約どおり利用不可）
- 二重 Close は冪等（エラーなし）

#### テスト: 異常系

- 不正 Options（到達不能 Backend 等）で Open 失敗し、半端な Driver を返さない
- Close 済み Driver への Set/Get はパニックせず error
-（任意）Firestore Options 欠落は Open エラー（Backend D の範囲）

---

### Get / Set（期限付き書き込み・v0.12）

- Set は payload を es4 value として書く。
- TTL>0（または Entry.ExpiresAt）なら **es4 期限付き API**（仮: `SetWithExpiry` / `Set(..., WithExpiry)`）を呼ぶ。
- TTL=0 かつ ExpiresAt ゼロなら通常 Set（期限なし）。
- Get は欠落 / es4 期限切れ不在 / 壊れた value → Miss（`hit=false`）。エラーにしない（壊れた JSON は Miss で固定しキー削除可）。
- 負の TTL → `ErrNegativeTTL`。空キー → `ErrEmptyKey`。

#### テスト：正常系

- Set → Get で同じ `value` が取れる（オブジェクト・数値・文字列）
- TTL 付き Set のあと、期限前は Hit、期限を過去にしたものは Get で Miss
- 期限なし Set のエントリは無期限として Hit し続ける
-（Firestore Emulator / スパイ）TTL 付き Set が期限付き API を呼び、通常 Set は呼ばない
-（Firestore Emulator）TTL 付きドキュメントに日時型フィールド `expires_at` がある
-（Firestore Emulator）期限なしドキュメントに `expires_at` フィールドが **無い**

#### テスト: 異常系

- 空キーは `ErrEmptyKey`
- 負の TTL は `ErrNegativeTTL`
- 壊れた value JSON でもパニックせず Miss（キー削除可）
- es4 が拒否する不正キー（先頭/末尾 `:`・連続 `::`）は error
- Close 後 Set/Get は error

---

### 旧 Entry JSON（v0.11 形式）の読み取り（要決定次第）

v0.11 は `{"value": ..., "expires_at": "<RFC3339>"}` を es4 value に書いた。v0.12 推奨は二重持ち廃止。

#### テスト：正常系（フォールバック採用時のみ）

- 旧形式（文字列 `expires_at` 付き）を Get でき、期限前は Hit・期限切れは Miss
- 新形式（期限は es4 メタのみ）も同様

#### テスト: 異常系

- 新旧混在でパニックしない
-（フォールバック非採用時）旧形式は Miss または error で固定しテスト

---

### Delete

#### テスト：正常系

- 存在するキーを Delete すると以降 Get は Miss
- 存在しないキーの Delete はエラーにならない
- Delete 後に Set し直すと再び Hit

#### テスト: 異常系

- 空キーはエラー
- Close 後 Delete は error
- 他キーは影響を受けない

---

### Incr

- es4 Tx 上で read-modify-write。未作成は 0+1→1。
- 競合時の再試行は Firestore Driver と同型。
- 新規カウンタは **期限なし**（期限付き API を使わない）。
- 既存キーに期限がある場合の Incr 後の期限（維持 / 削除）は計画の要決定に従い固定する。

#### テスト：正常系

- 未作成 Incr → 1、再 Incr → 2
- 既存整数 n へ Incr → n+1
- 並行 Incr の最終値が呼び出し回数と一致する（memory_only または Emulator）
- 新規 Incr 後、Firestore 上に `expires_at` が無い（Emulator / スパイ）

#### テスト: 異常系

- 空キーはエラー
- 整数以外 → `ErrNotInteger`
- 溢位 → `ErrIncrOverflow`
-（任意）Tx 競合を起こしても再試行後に正しい最終値、または契約どおりの error

---

### PurgeExact（Scan なし）

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` があるとき PurgeExact 後、数字キーだけ Miss で `__version__` は Hit
- 別プレフィックス `other:1` は影響を受けない
- 対象キーが無い状態でもエラーにならない

#### テスト: 異常系

- 空の logicalPrefix は `ErrEmptyKey`
- PurgeExact 実装が Scan を呼ばないこと（モック／スパイまたは契約テスト）
- `__version__` だけがあるときも残る

---

### PurgePrefix（Scan + Delete）

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` / `prefix:extra` があるとき PurgePrefix(`prefix`) 後はすべて Miss
- 別先頭 `other:1` は残る
- 対象キーが無い状態でもエラーにならない

#### テスト: 異常系

- 空 prefix は `ErrEmptyKey`
- Scan 失敗は error として返す
- 先頭一致のみ（途中一致では消さない）

---

### Backend 範囲

#### テスト：正常系

- memory_only で Layer 契約（Get/Set/Delete/Incr/Purge*・期限付き）が緑
-（任意ジョブ）firestore Emulator で日時型 `expires_at` の有無スモークが通る

#### テスト: 異常系

- 未知・到達不能 Backend 設定は Open エラー
- firestore 必須キー欠落は Open エラー

---

### HTTP Es4 Server アダプタ

#### テスト：正常系

- アダプタ経由で Set → Get が埋め込みと同セマンティクス（memory_only Server で可）
- TTL 付き Set が Server 経路でも期限切れ Miss になる（上流が対応している前提）
- PurgePrefix が期待どおり消える
- Close／シャットダウン後は操作が error

#### テスト: 異常系

- Server 未起動／接続失敗は Open または初回操作で error
- 空キー・負 TTL は埋め込みと同じ sentinel
- Ready 前相当の 503 は **error で固定**

---

### 多層（推奨）

#### テスト：正常系

- L1 Miss → L2 Hit → 書き戻し後 L1 Hit
- L2 への PurgeExact / PurgePrefix が L1 に残したキーと矛盾しない

#### テスト: 異常系

- L2 失敗時の失敗方針は既存 Layer 契約に従う

----

以上
