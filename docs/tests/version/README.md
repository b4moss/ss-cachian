---
type: TestSpec
title: version テスト仕様
description: current-version / Bump / 初回 Get。Go v0.3.0 導入・Node v0.8.0 再適用。正常≈3 / 異常≈3〜5。
tags: [tests, version, v0.3.0, v0.8.0, node]
timestamp: 2026-09-27T05:30:00Z
---

# version

対象:

- Go: Cache Type の Version まわり（L1 = インメモリ前提）
- Node: 同 API（`currentVersion` / `bumpVersion` 等。公開名は実装で Go と同型に揃える）

前提: [specs/version](../../specs/version/) / [tests 索引（Node 差分）](../README.md)  

キー例: `{app}:cache:{tenant}:{query}:__version__`  
初回: `__version__` 未作成なら **1** を L1 に作成する。導入マイルストーン: v0.3.0。

---

### CurrentVersion

- L1 の `__version__` を読んで現在版を返す。
- 未作成なら 1 を書き込んでから 1 を返す（初回 Get と同じ初期化方針に揃える）。

#### テスト：正常系

- 未初期化スコープで CurrentVersion すると 1 が返り、L1 に `__version__` が作られる
- 既存が 7 のとき CurrentVersion は 7 を返す（書き換えない）
- BumpVersion 後の CurrentVersion は進んだ値を返す

#### テスト: 異常系

- KeyBuilder / 文脈が不正で version キーを組めないときエラーになる
- L1 Get が失敗したときエラーが伝播する
- L1 上の `__version__` が整数として解釈できないときエラーになる

---

### BumpVersion

- L1 の current-version を最良努力で +1 する（read → +1 → write）。
- 競合時の上書き負けを許容する。
- Layer のデータキーは消さない。

#### テスト：正常系

- 1 の状態で Bump すると 2 になり CurrentVersion も 2
- 連続 Bump で単調非減少に進む
- Bump 前後で旧 version 付きデータキーが残存する（能動クリアしない）

#### テスト: 異常系

- `__version__` 未作成のとき、Bump は 1 を作成してから 2 にする、または 1→Incr 相当で 2 にする（実装は一方に固定し、結果として版が進む）
- L1 書き込み失敗時はエラーを返す（最良努力でも自プロセスの I/O 失敗は隠さない）
- キー構築失敗時はエラーになり、既存 `__version__` を壊さない

---

### ensureCurrentVersion（初回 Get 時の内部初期化）

- `Get` / `GetOrLoad` / `CurrentVersion` など「最新版が必要」な経路の冒頭で呼ばれる想定。
- `__version__` が無ければ L1 に **1** を Set する。
- 既にあれば触らない。

#### テスト：正常系

- 未作成時に呼ばれると `__version__=1` が作られ、戻りは 1
- 既存 5 のときは 5 のまま（Set し直さない）
- 複数の並行呼び出しから同時に呼んでも最終的に有効な正の整数が残る（競合負け許容。Go: ゴルーチン / Node: `Promise.all`）

#### テスト: 異常系

- L1 Set 失敗時はエラーになり、呼び出し側 Get も失敗する
- キー構築失敗時はエラー
- 既存値が壊れているときはエラー（勝手に 1 にリセットしない）
