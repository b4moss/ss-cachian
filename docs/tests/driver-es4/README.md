---
type: TestSpec
title: driver-es4 テスト仕様
description: es4 Layer Driver（埋め込み open・Entry TTL ラップ・HTTP アダプタ並列）。v0.11.0。正常≈3 / 異常≈3〜5。
tags: [tests, driver-es4, v0.11.0, node]
timestamp: 2026-10-11T00:00:00Z
---

# driver-es4

対象:

- Go: `go/sscachian/driver/es4`
- Node: `node/sscachian/src/driver/es4.ts`
- HTTP Es4 Server アダプタ（同版・埋め込みと並列。決定 C）

前提:

- [specs/driver-es4](../../specs/driver-es4/)（正本）· [plans/v0.11.0/es4-layer-driver.md](../../plans/v0.11.0/es4-layer-driver.md)（完了注記）
- [specs/layer](../../specs/layer/) / [tests 索引](../README.md)
- 上流 es4 v0.10.0: キー区切り `:` ＋ `Scan(prefix) []string`（ページングなし・Tx 内可）

固定セマンティクス（決定）:

| 項目 | 決定 |
|------|------|
| DB 所有権 | **A** — Driver が `Open` / `Close` を所有 |
| Backend | **D** — Firestore まで全 Backend 公式サポート（CI は memory_only 基本） |
| Entry JSON | **B** — `{"value": <payload>, "expires_at": "<RFC3339>"}`。`expires_at` 省略＝期限なし |
| Incr 再試行 | **A** — 既存 Firestore Driver に揃える |
| PurgeExact | **A** — **Scan なし**。キー既知で version data のみ削除 |
| PurgePrefix | es4 `Scan(prefix)` → 各キー `Delete` |
| HTTP アダプタ | **C** — v0.11.0 に埋め込みと並列 |

Entry は es4 値（JSON）として上記 B 形で保存する。es4 本体に TTL API は足さない。  
`created_at` は Entry JSON に **含めない**（決定 B）。必要なら実装メモのみ。

config `layers[].driver` 名は **`es4`**。本スイートは Driver 直接生成でも Config 経由でもよい。

---

### Open / Close（所有権 A）

- Driver 生成時に `es4.Open`（または Node `open`）を行い、`Close` で DB を閉じる。
- 呼び出し側が DB を注入する形（OpenWith 相当）は第一版の必須にしない。

#### テスト：正常系

- Open 成功後、Set → Get が動く（memory_only でよい）
- Close 後は以降の Layer 操作がエラーになる（または契約どおり利用不可）
- 二重 Close は冪等（エラーなし）

#### テスト: 異常系

- 不正 Options（到達不能 Backend 等）で Open 失敗し、半端な Driver を返さない
- Close 済み Driver への Set/Get はパニックせず error
-（任意）Firestore Options 欠落は Open エラー（Backend D の範囲）

---

### Entry JSON（決定 B）・Get / Set

- Set は `value` と、TTL>0 なら `expires_at`（RFC3339）を JSON 化して es4 に書く。
- TTL=0 / 無期限は `expires_at` を **省略**する。
- Get は Entry JSON を decode。`expires_at` が過去なら論理 Miss（必要なら Delete）。
- 欠落 / es4 `ErrNotFound` → Miss（`hit=false`）。エラーにしない。

#### テスト：正常系

- Set → Get で同じ `value` が取れる（オブジェクト・数値・文字列）
- TTL 付き Set のあと、期限前は Hit、過去 `expires_at` を書けば Get で Miss
- `expires_at` 省略のエントリは無期限として Hit し続ける

#### テスト: 異常系

- 空キーは `ErrEmptyKey`
- 負の TTL は `ErrNegativeTTL`
- 壊れた Entry JSON（フィールド欠落・型不正）でもパニックせず Miss または error（**Miss で固定**しキー削除可）
- es4 が拒否する不正キー（先頭/末尾 `:`・連続 `::`）は error

---

### Delete

- Layer は冪等。es4 の欠落エラーは握りつぶして成功。

#### テスト：正常系

- 存在するキーを Delete すると以降 Get は Miss
- 存在しないキーの Delete はエラーにならない
- Delete 後に Set し直すと再び Hit

#### テスト: 異常系

- 空キーはエラー
- Close 後 Delete は error
- 他キーは影響を受けない

---

### Incr（再試行は Firestore 揃え・決定 A）

- es4 Tx 上で read-modify-write。未作成は 0+1→1。
- 競合（Aborted 等）時の再試行回数・方針は **既存 Firestore Driver と同一**。
- 非整数 → `ErrNotInteger`。溢位 → `ErrIncrOverflow`。

#### テスト：正常系

- 未作成 Incr → 1、再 Incr → 2
- 既存整数 n へ Incr → n+1
- 並行 Incr の最終値が呼び出し回数と一致する（memory_only または Emulator）

#### テスト: 異常系

- 空キーはエラー
- 整数以外が格納されているキーへ Incr → `ErrNotInteger`
- 溢位付近 → `ErrIncrOverflow`
-（任意）Tx 競合を起こしても再試行後に正しい最終値、または契約どおりの error（Firestore Driver と同型）

---

### PurgeExact（Scan なし・決定 A）

- `PurgeExact(..., logicalPrefix)` は `{logicalPrefix}:{n}`（n≥1 の数字）だけを削除する。
- **es4 `Scan` を使わない**。キーは論理規則から既知（memory/firestore と同型: `IsVersionDataKey`）。
- `{logicalPrefix}:__version__` および別プレフィックスは残す。対象無しでも成功（冪等）。

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` があるとき PurgeExact 後、数字キーだけ Miss で `__version__` は Hit
- 別プレフィックス `other:1` は影響を受けない
- 対象キーが無い状態でもエラーにならない

#### テスト: 異常系

- 空の logicalPrefix は `ErrEmptyKey`
- PurgeExact 実装が Scan を呼ばないこと（ユニットでモック／スパイ、またはコードレビュー＋契約テストで担保）
- `__version__` だけがあるときも残る

---

### PurgePrefix（Scan + Delete）

- `PurgePrefix(..., prefix)` は es4 `Scan(prefix)` の結果キーへ `Delete` を回す。
- `__version__`・非数字サフィックスも含む（Exact より広い）。
- 空 prefix → `ErrEmptyKey`。対象無しでも成功（冪等）。

#### テスト：正常系

- `prefix:1` / `prefix:2` / `prefix:__version__` / `prefix:extra` があるとき PurgePrefix(`prefix`) 後はすべて Miss
- 別先頭 `other:1` は残る
- 対象キーが無い状態でもエラーにならない

#### テスト: 異常系

- 空 prefix は `ErrEmptyKey`
- Scan 失敗は error として返す（部分削除の有無は実装で固定しテスト）
- 先頭一致のみ（途中一致では消さない）

---

### Backend 範囲（決定 D）

- 第一版は memory / sqlite / redis|valkey / firestore を Options 経由で公式サポート。
- CI 既定は **memory_only**（軽い）。Firestore 等は Emulator／任意ジョブ。

#### テスト：正常系

- memory_only で Layer 契約（上記 Get/Set/Delete/Incr/Purge*）が緑
-（任意ジョブ）firestore Emulator で同セマンティクスのスモークが通る

#### テスト: 異常系

- 未知・到達不能 Backend 設定は Open エラー
- firestore 必須キー欠落は Open エラー

---

### HTTP Es4 Server アダプタ（決定 C・同版）

埋め込み open と並列で、HTTP Es4 Server 経由の Layer／アダプタを v0.11.0 に含める。  
キー写像は上流 es4 決定 A（`/v1/keys/a:b:c`）。詳細エンドポイント形は実装時に plans／本節へ追随。

#### テスト：正常系

- アダプタ経由で Set → Get が埋め込みと同セマンティクス（memory_only Server で可）
- PurgePrefix が Server 背後の State でも期待どおり消える（Scan 利用）
- Close／シャットダウン後は操作が error

#### テスト: 異常系

- Server 未起動／接続失敗は Open または初回操作で error
- 空キー・負 TTL は埋め込みと同じ sentinel
- Ready 前相当の 503 をアダプタがどう写すかを固定しテスト（error またはリトライ。**error で固定**推奨）

---

### 多層（任意だが推奨）

- memory L1 + es4 L2 で既存 layer テストに Driver を差し込める。

#### テスト：正常系

- L1 Miss → L2 Hit → 書き戻し後 L1 Hit
- L2 への PurgeExact / PurgePrefix が L1 に残したキーと矛盾しない（アプリ Purge 契約は [purge](../purge/)）

#### テスト: 異常系

- L2 失敗時の失敗方針は既存 Layer 契約（L1 成功・L2 ログ等）に従う

----

以上
