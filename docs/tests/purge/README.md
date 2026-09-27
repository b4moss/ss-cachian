---
type: TestSpec
title: purge テスト仕様
description: Exact Purge（全 version データ削除・`__version__` 非接触）（v0.7.0）。正常≈3 / 異常≈3〜5。
tags: [tests, purge, v0.7.0]
timestamp: 2026-09-27T02:37:00Z
---

# purge

対象: `go/sscachian` CacheType の `Purge`、および Layer `PurgeExact` のアプリ連携  
前提: [behavior](../../behavior.md) / [api](../../api.md) / [drivers](../../drivers.md)

PoC 固定（v0.7.0）:

- アプリ API 名は `Purge`。PoC の意味は Exact（論理キーに紐づく **全 version のデータキー** を削除）
- Layer メソッド名は `PurgeExact(ctx, logicalPrefix)`
- 削除対象は `{logicalPrefix}:{n}`（n≥1 の数字サフィックスのみ）
- `{logicalPrefix}:__version__` は **残す**。Bump もしない（`CurrentVersion` 不変）
- デフォルト対象は **全 Layer**。`Purge(ctx, kc, layerIdx ...int)` でインデックス絞り込み（未指定 = 全 Layer）
- 失敗方針は Delete と同型: **L1 失敗はエラー**、L2+ はログして続行。L1 を対象外にしたときは、対象 Layer の失敗をエラー
- `Delete`（最新キー削除 + Bump）とは別。Purge は物理データ掃除であり version を進めない

---

### Purge（アプリ・全 Layer）

- `logicalPrefix` を KeyBuilder から解決し、対象 Layer へ `PurgeExact` を呼ぶ。
- 複数回 Set / Bump してできた古い version データもまとめて消える。
- 最新キーの Get は Miss になる（version は残っているので、その後の Set は次の bump 後キーへ書く）。

#### テスト：正常系

- version 1…N にデータがある状態で Purge すると、各 `{prefix}:{i}` は全 Layer で Miss になり、`{prefix}:__version__` は L1 に残る
- Purge 前後で `CurrentVersion` が変わらない
- Purge 後に別論理キー（別 KeyContext）のデータは残る

#### テスト: 異常系

- データキーが無い（初回も未 Set）でも Purge は成功（冪等）し、`__version__` があれば残る／無ければ作らない
- 文脈不正でキーが組めないときはエラー（どの Layer も触らない）
- L1 の `PurgeExact` が失敗したらエラーを返し、L2 以降は呼ばない（または L1 失敗時点で全体エラー。**PoC: L1 失敗で即エラー。既に触った下位は無い前提で L1→Ln 順**）

---

### Purge（Layer 絞り込み）

- `layerIdx` で単一または複数 Layer に限定できる。
- 指定されなかった Layer のデータキーは残る。
- 不正インデックス（負・範囲外）はエラー。

#### テスト：正常系

- `Purge(ctx, kc, 1)`（L2 のみ）で L2 のデータは消え、L1 のデータは残る
- `Purge(ctx, kc)`（引数なし）で L1・L2 ともデータが消える
- 絞り込み対象に L1 を含まないときも `__version__` は触らない（L1 に残る）

#### テスト: 異常系

- 範囲外インデックスはエラーで、どの Layer も変更しない
- L1 を対象に含めず L2 のみ指定し、L2 `PurgeExact` が失敗したらエラーを返す
- 重複インデックスがあっても結果は同じ（冪等。二重実行しても害なし）

---

### Purge vs Delete / Version

- `Delete` は最新キー削除 + Bump。`Purge` は全 version データ削除・Bump なし。
- Version bump だけでは旧キーは残る（TTL 待ち）。Purge で物理削除する。

#### テスト：正常系

- Set → Bump/Set を繰り返し旧キーが残っている状態で Purge すると旧・新とも Miss
- Purge 後に `BumpVersion` して Set すると、新しい version キーに書け、Get できる
- `Delete` 後に残った旧 version キーも、続く `Purge` で消える

#### テスト: 異常系

- L2 のみ `PurgeExact` 失敗（L1 は成功）のとき、全体は成功とし失敗はログ（Delete の L2 失敗と同型）
- Purge 中に `__version__` キーが誤って消えない（メモリ／Firestore いずれでも検証）
- 空の logicalPrefix（KeyBuilder が空文字を返した場合）は Driver 側でエラーまたは no-op とし、他キーを巻き込まない（**PoC: Layer は空 prefix を ErrEmptyKey**）
