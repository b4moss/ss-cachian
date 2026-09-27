---
type: TestSpec
title: purge テスト仕様
description: Exact / Prefix / Tag Purge。Go v0.7.0 Exact・v0.10.0 Prefix/Tag。Node 同パリティ。正常≈3 / 異常≈3〜5。
tags: [tests, purge, v0.7.0, v0.8.0, v0.10.0, node]
timestamp: 2026-09-27T07:57:34Z
---

# purge

対象:

- Go: `go/sscachian` CacheType の `Purge` / `PurgeExact`（別名）/ `PurgePrefix` / `PurgeTag`、Layer `PurgeExact` / `PurgePrefix`
- Node: `node/sscachian` 同 API（`purge` / `purgeExact` / `purgePrefix` / `purgeTag`、Layer `purgeExact` / `purgePrefix`）

前提: [specs/purge](../../specs/purge/) / [specs/layer](../../specs/layer/) / [plans/v0.10.0](../../plans/v0.10.0/) / [tests 索引（Node 差分）](../README.md)

固定セマンティクス:

- アプリ `Purge` / `PurgeExact`（別名）は Exact（論理キーに紐づく **全 version のデータキー** を削除）
- Layer `PurgeExact(ctx|signal, logicalPrefix)` — 削除対象は `{logicalPrefix}:{n}`（n≥1 の数字サフィックスのみ）
- `{logicalPrefix}:__version__` は Exact では **残す**。Bump もしない（`CurrentVersion` 不変）
- Layer `PurgePrefix` — キー ID が `prefix` で **始まるすべて**を削除（`__version__` や非数字サフィックスも含む。Exact より広い）
- `PurgeTag` — L1 タグ索引から論理プレフィックス集合を解決し、各々に Exact Purge。索引エントリを消す
- デフォルト対象は **全 Layer**。Layer インデックス絞り込み可（未指定 = 全 Layer）。失敗方針は Delete と同型: **L1 失敗はエラー**、L2+ はログして続行。L1 を対象外にしたときは、対象 Layer の失敗をエラー
- `Delete`（最新キー削除 + Bump）とは別。Purge 系は物理掃除であり version を進めない（Prefix で `__version__` を消した場合を除き Bump もしない）
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
- L1 の `PurgeExact` が失敗したらエラーを返す（**L1 失敗で即エラー。L1→Ln 順**）

---

### Purge（Layer 絞り込み）

- Layer インデックスで単一または複数 Layer に限定できる。
- 指定されなかった Layer のデータキーは残る。
- 不正インデックス（負・範囲外）はエラー。

#### テスト：正常系

- L2 のみ指定で L2 のデータは消え、L1 のデータは残る
- 引数なし（全 Layer）で L1・L2 ともデータが消える
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
- 空の logicalPrefix（KeyBuilder が空文字を返した場合）は Driver 側でエラーまたは no-op とし、他キーを巻き込まない（**Layer は空 prefix を ErrEmptyKey**）

---

### PurgeExact（アプリ別名）

- `PurgeExact(ctx, kc, layerIdx ...int)` は現行 `Purge` と **同一意味・同一失敗方針**（非破壊の命名明確化）。
- 既存の `Purge` 呼び出しは挙動を変えない。

#### テスト：正常系

- `PurgeExact` 後、Exact と同条件でデータキーは Miss・`__version__` は残り・`CurrentVersion` 不変
- `Purge` と `PurgeExact` を同じ状態に対して呼んでも結果が一致する（どちらも冪等）
- Layer 絞り込み引数も `Purge` と同じく効く

#### テスト: 異常系

- 文脈不正は `Purge` と同じエラー
- 範囲外 `layerIdx` は `ErrInvalidLayerIndex` でどの Layer も変更しない
- L1 `PurgeExact` 失敗時はエラー（`Purge` と同型）

---

### PurgePrefix（アプリ・全 Layer / 絞り込み）

- `PurgePrefix(ctx, prefix string, layerIdx ...int)` — 生の文字列 prefix（KeyContext 経由ではない）。対象 Layer へ Layer `PurgePrefix` を呼ぶ。
- Exact と違い、`{prefix}…` で始まる **すべてのキー**（`__version__`・非 version データ含む）を消す。
- 空 `prefix` → `ErrEmptyKey`（どの Layer も触らない）。
- Layer 選択・失敗方針は `Purge` と同型。

#### テスト：正常系

- 共通先頭を共有する複数論理キー（例: `app:cache:t:q1:1` と `app:cache:t:q2:1`）を `app:cache:t:` の Prefix でまとめて消せる
- Exact では残る `{logical}:__version__` も、その論理が prefix 配下なら Prefix では消える
- 別先頭のキー（例: `app:cache:other:…`）は残る。`CurrentVersion` は Prefix 自体では進めない（消えた version キーは以降の Get で再初期化されうる）
- L2 のみ指定で L2 だけ消え、L1 は残る。引数なしで全 Layer が消える

#### テスト: 異常系

- 空 prefix は `ErrEmptyKey` でどの Layer も変更しない
- 対象キーが無くても成功（冪等）
- 範囲外 `layerIdx` はエラーで変更なし。L1 `PurgePrefix` 失敗はエラー。L2 のみ失敗（L1 成功）はログして全体成功

---

### Tags 書き込み（索引）

- Set / Remember / GetOrLoad 充填時にオプションでタグを付けられる。
  - Go: `Set(ctx, kc, v, WithTags("product:456", …))` 等。Builder `WithDefaultTags(...)` で型全体の既定タグ
  - Node: `set(kc, v, { tags: [...] })` / Builder `withDefaultTags`
- 既定タグと呼び出しタグは **和集合**（重複除去）で索引に載せる。
- 成功した mutate / fill のあと、L1 に `tag → 論理プレフィックス集合` の逆引きを更新する（memory: プロセス内 map、Firestore L1: 予約キー `__sscachian_tag__:{tag}` 配下）。
- タグは versioned データキーの文字列レイアウトを変えない。タグ無し書き込みは索引を触らない。

#### テスト：正常系

- `WithTags("product:456")` 付き Set 後、同タグで PurgeTag するとその論理キーの version データが Exact 相当で消える
- `WithDefaultTags` のみでも索引に載り、PurgeTag で消える
- 既定 + 呼び出しタグの和で複数タグに同一プレフィックスが載る
- GetOrLoad / Remember の Miss 充填でもタグ指定があれば索引に載る（Bump しない充填）

#### テスト: 異常系

- 空文字タグは無視またはエラー（**無視で固定**。空のみのときは索引を更新しない）
- L1 へのデータ書き込みが失敗したとき、タグ索引も更新しない
- タグ無し Set は既存タグ索引を消さない・増やさない

---

### PurgeTag

- `PurgeTag(ctx, tag string, layerIdx ...int)` — L1 索引から論理プレフィックス一覧を取り、各々に Exact Purge（選択 Layer）。その後タグ索引エントリを削除。
- 存在しないタグ / 空集合は成功（冪等）。空タグ文字列は `ErrEmptyKey`（または no-op 成功。**ErrEmptyKey で固定**）。
- Exact と同じく各プレフィックスの `__version__` は残す。Bump しない。
- Layer 選択・失敗方針は `Purge` と同型（索引の読み取り・削除は L1）。

#### テスト：正常系

- 同一タグを付けた複数論理キーを一度の PurgeTag でまとめて Exact 掃除できる
- PurgeTag 後、当該タグの索引は消え、再 PurgeTag しても成功（冪等）。別タグの索引・データは残る
- PurgeTag 前後で各論理キーの `CurrentVersion` は変わらない（`__version__` 残存）
- Layer 絞り込み（例: L2 のみ）ではデータは選択 Layer のみ消え、索引削除は L1 で行う

#### テスト: 異常系

- 未登録タグでもエラーにならない
- 空タグは `ErrEmptyKey`
- 範囲外 `layerIdx` はエラー。L1 Exact 失敗時はエラーで、可能なら索引は消さない（途中失敗で索引だけ消えないこと）
