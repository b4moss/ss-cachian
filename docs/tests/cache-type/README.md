---
type: TestSpec
title: cache-type テスト仕様
description: Define/Build・Key・読み書き API。Go v0.3.0 導入・Node v0.8.0 再適用。正常≈3 / 異常≈3〜5。
tags: [tests, cache-type, v0.3.0, v0.8.0, node]
timestamp: 2026-09-27T05:30:00Z
---

# cache-type

対象:

- Go: `go/sscachian` 公開 API
- Node: `node/sscachian` 公開 API（`define` / `Builder` / `CacheType`）

前提: [specs/cache-type](../../specs/cache-type/) / [specs/version](../../specs/version/) / [tests 索引（Node 差分）](../README.md)

意味論は Go / Node 同一。Node はすべて `async`。本スイートは **単一インメモリ L1** で検証。多層は [layer](../layer/)、Purge は [purge](../purge/)、Firestore は [driver-firestore](../driver-firestore/) を参照。

---

### Build / Define

- Cache Type を Define し、Layer・KeyBuilder・Policy/TTL・Loader をオプションで付与して Build する。
- 本スイートは Layer 1 本（memory）を前提とする。多層ケースは [layer](../layer/) ドメイン。

#### テスト：正常系

- Define + WithLayers(memory) + WithKeyBuilder + Build で利用可能な Cache Type が得られる
- WithLayerTTL / WithPolicy で TTL を付けると Set したエントリに `expires_at` が反映される
- WithLoader を付けた Type で GetOrLoad が Loader を呼べる

#### テスト: 異常系

- Layer 未設定のまま Build するとエラーになる
- KeyBuilder 未設定のまま Build するとエラーになる
- Layer を 0 本にした Build はエラーになる

---

### BuildKey / BuildLatestKey

- BuildKey は文脈と version からデータキー文字列を作る。
- BuildLatestKey は current-version を解決してからデータキーを作る（必要なら `__version__` を初期化）。

#### テスト：正常系

- 固定文脈 + version=7 で期待どおりのキー文字列になる（`{app}:cache:{tenant}:{query}:7` 形）
- BuildLatestKey は CurrentVersion と同じ版番号をキー末尾に使う
- `__version__` キーはデータキーと同系で末尾が `__version__` になる

#### テスト: 異常系

- 必須文脈フィールド欠落でエラーになる
- version に 0 以下を渡した BuildKey はエラー（または契約した最小値未満はエラー）
- KeyBuilder がエラーを返したらそのまま伝播する

---

### Get

- 最新 version のデータキーだけを L1 から読む（単一 Layer 前提。多層は layer）。
- 戻りは `value` のみ（Entry メタは露出しない）。
- 必要なら先に current-version を初期化（未作成→1）。

#### テスト：正常系

- Set 済みの最新キーを Get すると同じ値が返る
- 初回 Get（version 未作成・データなし）では `__version__=1` が作られ、データは Miss
- Miss 時は「値なし」が分かり、パニック／未処理例外にしない（ok=false または専用エラー。実装で一方に固定）

#### テスト: 異常系

- L1 Get 失敗時はエラーを返す
- 保存型と取り出し型が不一致のときエラーになる（Go: generics / 型アサーション。Node: 実行時の型ガードまたはデコード失敗）
- 文脈不正でキーが組めないときエラーになる

---

### Set

- **先に L1 で BumpVersion**し、その新 version キーへ値を書く（Entry ラッパーは内部）。

#### テスト：正常系

- Set 後、Bump 後の版番号キーに書かれ、CurrentVersion が +1 されている
- Get で書いた値が取れる
- TTL 付き Type では `expires_at` が設定され、期限前 Get で値が取れる

#### テスト: 異常系

- Bump 失敗時はエラーで、データキーへは書かない
- Bump 成功後に L1 Set が失敗したらエラーを返す（**既に進んだ version は戻さない**）
- 文脈不正では Set も Bump もしない

---

### Delete

- 最新 version のデータキーを削除する。
- **成功後に自動 BumpVersion** する。
- `__version__` 自体は消さない。

#### テスト：正常系

- 存在する最新キーを Delete すると以降その版への Get は Miss
- Delete 成功後 CurrentVersion が進む
- `__version__` キーは Delete 後も残る

#### テスト: 異常系

- データが無くても Delete は成功扱いで Bump する（**PoC: 冪等 Delete 成功 + Bump する**）
- L1 Delete 失敗時は Bump しない
- 文脈不正ではエラー

---

### GetOrLoad

- Get して Hit ならその値を返す。
- Miss なら Loader を呼び、成功したら書き戻して値を返す。
- **現行:** 書き戻し先は設定された **全 Layer**（[layer](../layer/)）。本スイートは単一 L1 構成のため L1 のみ観測する。
- **GetOrLoad の書き戻しは Bump しない**（読み取り充填のみ。Mutation の Set/Delete だけ自動 Bump）。

#### テスト：正常系

- Hit 時は Loader を呼ばない
- Miss 時は Loader が一度呼ばれ、返り値が Get でも取れる
- Loader が返した値が呼び出し元に返る

#### テスト: 異常系

- Loader 未設定で Miss のときエラーになる
- Loader がエラーを返したらそのエラーが伝播し、L1 に書かない
- 書き戻し Set が失敗しても、**読み取り成功優先**（Loader の値は返す。失敗はログ）。version は進めない
