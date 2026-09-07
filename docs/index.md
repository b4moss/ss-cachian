# ss-cachian 構想まとめ

サーバーサイドキャッシュ戦略ライブラリ **ss-cachian**（Server Side Cachian）の構想知識。実装コードは未着手で、本ディレクトリのメモ群が現時点の正本に近い。

本ファイルは (1) 構想の統合解釈、(2) 確定した前提、(3) 各メモへの索引、をまとめた入口。

---

# 統合解釈

## このライブラリが何者か

ss-cachian は **単一キャッシュストアの薄い抽象**ではない。

目指すものは次の一文に収束する。

> アプリケーションが **Cache Type** としてキャッシュ戦略を宣言し、ライブラリがその戦略を **複数ストレージ（Layer）にまたがって実行する**。

ストレージ中心（「Redis ライブラリ」）ではなく、戦略中心（「Cache Strategy Library」）である。既存の `eko/gocache` や Keyv はストア抽象・チェーン・Loadable などで近いが、本構想の差別化は **Cache Type をアプリケーションの設計単位にする**点にある。

## 動機

- DB を本質的なアクセスに限定し、守ること。
- セッション確認のような高頻度 Read、クエリ結果の再読を DB に載せ続けないこと。
- Memory / Redis・Valkey / Firestore / Object Storage など複数メモリストレージ種を、Adapter（Driver）化し、同一アプリ内で階層利用すること。

キャッシュは「正しいデータの保存場所」ではなく、**PostgreSQL 等を正本とした読み取り高速化の一時層**として扱う。

## 中心概念

| 概念 | 意味 |
| --- | --- |
| Cache Type | 商品・セッション・記事など、種類ごとの戦略定義単位 |
| Key Builder | ビジネス文脈（tenant / locale / query_type / version 等）を含むキー生成 |
| Layer / Driver | 「どこに保存するか」。階層数は固定しない。共通契約は薄い |
| Policy | TTL、（将来）SWR / SIE 等 |
| Loader | Cache Miss 時の取得（Read-through） |
| Invalidation | 第1級は Version。Purge API で多彩な明示削除も提供 |
| Entry メタ | 当面は `created_at` / `expires_at`。以後拡張 |

利用側は、保存先・階層構成・キー生成・Miss 時の取得を意識せず、概ね次で済むことを理想とする。

- `Get` / `GetOrLoad`
- `Set`
- `Delete` / `Purge`
- Version bump（論理 invalidate）

## 多層キャッシュ

典型フロー:

```text
Cache Type
  → Key generation
  → L1 … hit なら返却
  → L2 … miss 伝播
  → L3 …
  → Loader（DB / API）
  → 必要に応じて上位 Layer へ書き戻し
```

例:

- テナント共通レスポンス: L1 Memory → L2 Firestore → DB
- セッション: L1 Memory → L2 共有ストア（Firestore または Valkey）
- 大きめの成果物: Memory → Object Storage（高速キャッシュというより再計算回避）

上位 hit 時は下位を見ない。

Read-through（Loader 付き）を基本にしつつ、Cache-aside / 明示 Set も許容する。

## 多層の書き戻し / 無効化（確定方針）

### Populate（書き戻し）

下位 hit または Loader 成功時、**上位 Layer へ無条件で書き戻す**。

例: L1 → L2 → Loader で Loader 成功なら L2 と L1 の両方へ Set。

- TTL は **Layer ごとに別値**を持てる（仕様として正しい）。
- 例: L1=30s、L2=1h など。Policy は Layer 単位で保持する。

### Version bump 時

能動的な Layer クリアはしない。

- version 入りデータキーは **全 Layer で同一キー名**。
- bump 後、Get が見るのは新 version のキーだけ。
- 旧 version のエントリは各 Layer の TTL で自然消滅する。

### 初期の Get 方針（最新 version のみ）

初期実装では **常に最新 version のエントリだけを返す**。

- 旧 version を意図的に読まない・返さない。
- 利用者から見えるキャッシュは常に現在版のみ。
- 旧キーの存在は許容し、掃除は TTL 任せ。

「最新 version 番号」を知る手段は、初期は次のいずれかで足りる。

- current-version を表す 1 キーを持つ（O(1) で参照）
- アプリ側が version を渡す／単一世代運用にする

旧世代の列挙や「任意 version の取得」は初期スコープ外。

### Purge の到達範囲

- **デフォルト: 定義されている全 Layer** に対して purge する。
- 引数 / オプションで Layer の配列を渡し、**指定した単一または複数 Layer だけ**に絞れる。

### L1 の意味

- 階層の基本は **L1 から始まる**（Cache Type が定義する順序）。
- **L1 = インメモリとは限らない**。多いパターンではあるが、Cache Type の設計次第。
- したがって「L2 が常に共有ストア」「Memory 以外が共有」といった固定対応にはしない。

## キー設計

単なる文字列 ID ではなく、アプリのコンテキストをキーに含める。

構造例（[キービルダー](./key-builder.md)）:

```text
{app_slug}:cache:{tenant_id}:{query_type}:{version}
```

例:

```text
my-app:cache:0123456:client_list_page1:7
```

同一論理エントリは、L1 / L2 / … のいずれでも **同じキー名**を使う。

```text
L1  my-app:cache:0123456:client_list_page1:7
L2  my-app:cache:0123456:client_list_page1:7
```

マルチテナント衝突回避、一覧ページ、設定・マスタなど業務キーを自然に表現できることを重視する。

Version は **固定のキー名パターンの一部として埋め込む**（専用の巨大な Version Store 抽象は置かない）。  
現在版番号の参照用に current-version 1 キーを置くのは、初期方針として許容する。

## Invalidation（確定方針）

- **第1級: Version**
  - Mutation 後は該当スコープの version を進める（論理 invalidate）。
  - Get は常に最新 version のみ（初期方針）。
  - 旧 version のキーは参照されなくなり、TTL で物理 cleanup される。
  - 大量キーの個別 DELETE を避ける（一覧・ページネーション向けに特に有効）。
- **Purge API: 多彩な手法**
  - Exact / Prefix / Tag など、明示パージの引き出しを提供する。
  - デフォルトは全 Layer。Layer 配列指定で対象を絞れる。
  - 日常のアプリ設計は Version、運用・例外・強制削除は Purge、という二段構え。

役割分担の覚え方:

```text
Version = 論理的な invalidate
TTL     = 物理的な cleanup
Purge   = 明示的・多様な削除手段
```

## Layer 契約（確定方針）

共通面は **薄く保つ（方針 A）**。

- 共通: Get / Set / Delete + 任意 TTL 程度
- TTL 値は Layer ごとに異なる値を指定可能
- 原子性・Prefix・Tag・INCR などは Driver 固有 API
- 高度機能を全 Driver でエミュレートして揃えることはしない

Object Storage なども差し込み口は同じだが、レイテンシや TTL の意味まで同一保証はしない。

L1 / L2 / L3 は **Cache Type が並べた順序上の名前**であり、Driver 種別の固定マッピングではない。

## エントリメタ（確定方針・初期）

Layer に載せる値の封筒は、当面次のみ。

- `created_at`
- `expires_at`

SWR / SIE / negative cache 等は開発が進んでから拡張する。

## Valkey 固有機能（確定方針）

Rate Limit / Lock / Stampede など Valkey 文脈の機能は、

- Cache Type が Valkey を選んだときだけ有効なオプション
- 当ライブラリ本体は実装せず、**Valkey Driver が操作するモジュールへ渡すだけ**

とする。

## 業務判断（いつキャッシュするか）

キャッシュ可否は更新頻度そのものではなく、**staleness（古さ）をどこまで許容できるか**で決める。

有力例: ユーザー情報、権限、設定、マスタ、カテゴリ、外部 API、重い集計、ダッシュボード。  
慎重 / 避ける例: 決済状態、リアルタイム在庫、古さを許容できない承認・残高、細かい個人 CRUD。

一覧は page=1 を優先し、page=2 以降はアクセス頻度と DB 負荷を見て判断する。

基本運用パターン:

```text
Cache Aside（または Read-through）
  + 長めの TTL
  + Mutation 時の version increment
  + 旧 version は参照せず TTL で自然消滅
```

## Driver / 実装優先（確定方針）

Driver 開発・PoC 優先順:

1. インメモリ
2. Firestore
3. Redis / Valkey
4. （以降）MongoDB、Object Storage、Filesystem 等

補足:

- Firestore は小規模 SaaS・Cloud Run・常時起動 Redis を置きたくない構成向き。
- Valkey は高頻度共有、Atomic、Lock、Stampede 向き。Firestore の後に載せる。
- Object Storage は「計算済み成果物の保存」として Layer 候補に残すが、高速 L2 とは性質が異なる。

## 開発方針・フェーズ

- `b4moss/crudian` や `b4moss/cachian` と同様、**マルチランタイム移植を前提**にする。
- 先行実装は **Go**。
- Phase 1（PoC / おおよそ v0.7.0 想定）:
  - Go で利用可能
  - キービルダー
  - 複数ストレージ選択
  - Driver: インメモリ + Firestore
- Phase 2（〜 v1.0.0）:
  - 他 Driver（Valkey 等）
  - 他言語ポート（Node.js、PHP）

Go ではまず型安全 API を優先し、YAML 等の設定ファイル駆動は後回しでよい。

## まだ開いている設計論点

- シリアライズとマルチランタイム間のセマンティクス同一性
- Entry メタの拡張タイミング（SWR / SIE / negative 等）
- current-version 1 キーを置く場合の、具体的なキー規約と配置 Layer（初期は最新 version のみ返す方針で先行可）

## 既存ライブラリとの関係

- Node.js: Keyv はストレージ抽象。Version / ドメイン invalidate は自前層が自然。
- Go: `eko/gocache` は Chain / Loadable / Tag などで参考価値が高い。ただし本ライブラリは「ストア組合せ」ではなく「Cache Type としての戦略定義」を前面に出す。

---

# ドキュメント索引

## 方針・概要

* [コンセプト](./concept.md) - サーバーサイドキャッシュの責務、Driver / 階層 / キャッシュタイプ定義の骨子
* [note](./note.md) - PO メモ。開発方針、Driver 順序、Phase 1 / 2

## 設計メモ

* [アプリケーション API（仮定）](./api.md) - 初期 PoC 必須とそれ以降任意に分けたアプリ向け API リスト
* [キービルダー](./key-builder.md) - アプリ・テナント・クエリ種別・version を含むキー構造の種
* [壁打ち参考その1](./note-refs1.md) - Cache Type 中心の API・多層・Policy・Invalidation・MVP の詳細壁打ち
* [壁打ちメモその2](./note-refs2.md) - 業務 Web アプリでのキャッシュ判断、Versioned Cache、TTL 役割分担

## 本ファイル

* [index（本ファイル）](./index.md) - 構想の統合解釈と確定方針の入口

---

# ソースメモとの対応

| 解釈上の主張 | 主な根拠 |
| --- | --- |
| DB を守り、戦略を宣言する | [concept.md](./concept.md) |
| Cache Type / Layer / Policy / Loader | [note-refs1.md](./note-refs1.md) |
| Version = 論理 invalidate、TTL = cleanup | [note-refs2.md](./note-refs2.md)、[key-builder.md](./key-builder.md) |
| Go 先行・マルチランタイム前提 | [note.md](./note.md) |
| Driver 優先: Memory → Firestore → Valkey | [note.md](./note.md) および対話で確定 |
| Invalidation 第1級は Version、Purge は多彩 | 対話で確定 |
| Layer 共通契約は薄い（A） | 対話で確定 |
| Version は固定キー名に埋め込み（全 Layer 同一キー） | 対話で確定 |
| Entry メタ初期は created_at / expires_at | 対話で確定 |
| Valkey 固有機能は Driver 先モジュールへ委譲 | 対話で確定 |
| Populate は上位へ無条件書き戻し、TTL は Layer 単位 | 対話で確定 |
| Version bump 時は能動クリアせず TTL 任せ | 対話で確定 |
| Get は初期は最新 version のみ返す | 対話で確定 |
| Purge デフォルト全 Layer、配列で絞り込み可 | 対話で確定 |
| L1 は順序上の先頭であり Memory 固定ではない | 対話で確定 |
| アプリ向け API 面（PoC / 任意の切り分け） | [api.md](./api.md)（仮定） |

壁打ちメモは生成 AI との整理結果を含むため、個別文は未確定案が混ざる。衝突時は **本 index の「確定方針」と [note.md](./note.md) / [concept.md](./concept.md)** を優先して読む。
