---
type: TestSpec
title: layer テスト仕様
description: 多層 Get / 書き戻し / Set・Delete 全 Layer。Go v0.5.0 導入・Node v0.8.0 再適用。正常≈3 / 異常≈3〜5。
tags: [tests, layer, v0.5.0, v0.8.0, node]
timestamp: 2026-09-27T05:30:00Z
---

# layer

対象:

- Go: `go/sscachian` CacheType の多層振る舞い
- Node: `node/sscachian` CacheType の多層振る舞い

前提: [specs/layer](../../specs/layer/) / [specs/cache-type](../../specs/cache-type/) / [tests 索引（Node 差分）](../README.md)  

current-version は **L1 のみ**。データキーは全 Layer で同一。

固定セマンティクス:

- 上位 hit 時は下位を見ない
- 下位 hit または Loader 成功時、それより **上の Layer へ無条件書き戻し**
- 書き戻し失敗はログして無視（値は返す）
- Layer ごと TTL（`WithLayerTTLs` 等）。未指定は 0（無期限）
- アプリ `Set` / `Delete` は **全 Layer** に同一キーを適用。Bump は L1 のみ
- `GetOrLoad` の充填書き戻しは **Bump しない**

---

### Get（多層）

- 最新 version キーを L1 → L2 → … の順に探す。
- ある Layer で Hit したら、それより上の Layer へ Entry を書き戻してから `value` を返す。
- 全 Layer Miss なら ok=false（Loader は呼ばない。GetOrLoad とは別）。

#### テスト：正常系

- L1 Hit のとき L2 以降を読まず、値が返る
- L1 Miss・L2 Hit のとき値が返り、直後の L1 Get でも Hit する（書き戻し済み）
- L1・L2 とも Miss のとき ok=false で、いずれの Layer にも書き込まれない

#### テスト: 異常系

- L2 Get がエラーのとき、探索を止めエラーを返す（それまでに得た上位 Miss はそのまま）
- L2 Hit 後の L1 書き戻しが失敗しても、呼び出しには値を返しエラーにしない
- 文脈不正でキーが組めないときはエラー（どの Layer も触らない）

---

### GetOrLoad（多層）

- Get 相当の多層探索で Hit なら Loader を呼ばない。
- 全 Miss なら Loader を呼び、成功したら **全 Layer（L1…Ln）へ書き戻し**（Bump しない）。
- 書き戻し一部失敗でも Loader 値は返す。

#### テスト：正常系

- L1 Hit では Loader 呼び出し回数 0
- 全 Miss で Loader が 1 回呼ばれ、返り値が Get でも取れる（少なくとも L1 に残る）
- Loader 成功後 CurrentVersion は変わらない（Bump しない）

#### テスト: 異常系

- Loader 未設定で全 Miss のとき ErrNoLoader
- Loader エラー時はそのエラーが伝播し、どの Layer にも書かない
- 書き戻しが全 Layer で失敗しても Loader 値は返り、version は進まない

---

### Set（多層）

- bump 後の最新キーへ書く（v0.3.0 セマンティクス継承）。
- 書き込み先は **設定された全 Layer**（各 Layer の TTL を使う）。
- Bump / current-version は L1 のみ。

#### テスト：正常系

- Set 後、L1 と L2 の両方で同じ最新キーが Hit する
- Set 後 CurrentVersion が +1 される（L1）
- Layer TTL が異なるとき、各 Layer の `expires_at` がそれぞれの TTL に基づく

#### テスト: 異常系

- L1 への Set が失敗したらエラーとし、**Bump 後 L1 Set 失敗はエラー。既に進んだ version は戻さない**
- L2 Set だけ失敗した場合はログして無視し、全体は成功（L1 に書けていればよい）
- 文脈不正ではどの Layer にも書かず Bump もしない

---

### Delete（多層）

- 最新キーを **全 Layer** から Delete（冪等）。
- その後 L1 で BumpVersion。
- `__version__` は消さない（L1）。

#### テスト：正常系

- Delete 後、L1・L2 とも最新キーは Miss
- Delete 後 CurrentVersion が進む
- `__version__` キーは L1 に残る

#### テスト: 異常系

- データ無しでも全 Layer Delete は成功扱いで Bump する
- L1 Delete 失敗時は Bump しない
- L2 Delete だけ失敗しても L1 成功なら Bump し、全体は成功（失敗はログ）

---

### WithLayerTTLs / Build

- Layer 数と TTL 配列を対応付ける（短すぎる TTL は残り 0、長すぎる分は無視、で固定）。

#### テスト：正常系

- Layer 2 本 + TTL 2 値で Build できる
- TTL 未指定（単一 WithLayerTTL 後方互換）でも Build でき、全 Layer に同じ TTL が付く
- TTL 配列が Layer より短いとき、足りない分は 0

#### テスト: 異常系

- Layer 0 本の Build はエラー（既存）
- 負の TTL を含む指定は Build 時に拒否
