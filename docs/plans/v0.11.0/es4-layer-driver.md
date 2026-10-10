# ss-cachian × es4 Layer Driver 計画

| 項目 | 値 |
|------|-----|
| 状態 | 方針合意ドラフト（こうきさん承認済み方針を反映。細部の未決は §9） |
| マイルストーン | [v0.11.0](https://github.com/b4moss/ss-cachian/milestone/2)（次 minor・現行リリース v0.10.0 の次。同版に [SWR 意図スタブ](./README.md) あり） |
| 対象リポジトリ | [b4moss/ss-cachian](https://github.com/b4moss/ss-cachian)（Driver）／依存先 [b4moss/es4](https://github.com/b4moss/es4) |
| 開発方針の正本 | [b4moss/charter](https://github.com/b4moss/charter)（本リポ `docs/charter/` 同梱） |
| override | `docs/override-charter.md` は **現状設定なし** → 憲章どおり |
| 関連計画 | es4: [キーデリミタ `:` 化](https://github.com/b4moss/es4/blob/main/docs/plans/v0.10.0/key-delimiter.md)（Milestone [v0.10.0](https://github.com/b4moss/es4/milestone/1)） |
| 転記元 | Project Context 計画ドラフト（2026-10-10 転記） |

**本ファイルの位置づけ:** ss-cachian リポジトリ内 `docs/plans/` の正本（v0.11.0）。実装コードは別途。charter どおり実装 Agent は本ファイルを編集しない。

----

## 1. 目的・範囲

### 目的

ss-cachian の **Layer 契約**を満たす **es4 Driver** を追加し、埋め込み `es4.Open` をストレージとして使えるようにする。Valkey Driver（roadmap **v0.12.0**・意図スタブ）と **並存**する別 Backend。HTTP Es4 Server 経由は第一選択にしない。

### 入れる（第一版・承認済み）

| 項目 | 方針 |
|------|------|
| 置き場所 | **ss-cachian 側**に Layer / Driver（依存方向: **ss-cachian → es4**） |
| 接続形態 | **埋め込み open が本命**（プロセス内 `es4.Open` / Node `open`） |
| TTL | es4 本体に TTL を足さない。**Driver 側で Entry をラップ**して期限を持つ（memory Driver と同型） |
| Incr | es4 **Tx** 上で read-modify-write |
| PurgePrefix | es4 に薄い **`Scan(prefix)`** を足し、Driver が返却キーへ `Delete` を回す |
| PurgeExact | Layer 契約どおり。実装は既存 memory/firestore と同セマンティクス（`IsVersionDataKey`）に寄せる |
| 並存 | Valkey Driver 計画を置き換えない |

### 入れない（第一版）

- HTTP Es4 Server を必須経路にする設計（任意の将来拡張は未決）
- es4 本体へのネイティブ TTL / 期限フィールド
- Valkey 固有委譲（RateLimit / Lock / Stampede）を es4 Driver に載せる
- PHP ポート

----

## 2. リポジトリ現状（根拠）

### ss-cachian

| 事実 | 根拠 |
|------|------|
| 現行線 **v0.10.0**。Drivers は **memory** / **firestore** のみ | ルート README・`docs/specs/driver-*` |
| Layer 契約: `Get` / `Set` / `Delete` / `Incr` / `PurgeExact` / `PurgePrefix` | `docs/specs/layer/README.md`・Go `types.go`・Node `types.ts` |
| 既定 KeyBuilder は **`:` 区切り**（`{app}:cache:{tenant}:{query}`） | Node `defaultKeyBuilder` / specs |
| Valkey は roadmap **v0.12.0**・plans は意図スタブ | `docs/roadmap.md`・`docs/plans/v0.12.0/`・`unscheduled/future-apis.md` |
| TTL は Layer `Set` 引数。memory は Entry + `ExpiresAt` の遅延削除 | `driver/memory/memory.go`・`docs/specs/driver-memory/` |
| config `layers[].driver` は `memory` \| `firestore`（未知は拒否） | specs/config・config テスト |

### es4

| 事実 | 根拠 |
|------|------|
| 公開 State: `Set` / `Get` / `Delete` / `Exists` / `Clear`。Tx 別面 | `docs/specs/state/state.md`・`docs/specs/tx/` |
| **Scan / LIST は持たない**（転記時点） | state specs・es4-server |
| キーは階層 **`/`**（転記時点）。**`:` 化は es4 v0.10.0 計画** | `ValidateKey`・[key-delimiter 計画](https://github.com/b4moss/es4/blob/main/docs/plans/v0.10.0/key-delimiter.md) |
| 値は JSON（`json.RawMessage`）。欠落 Get/Delete → `ErrNotFound` | state specs |
| Firestore Doc ID = path-percent-encode | state specs |
| 埋め込みライブラリ + 任意 HTTP Es4 Server | ルート README |

### charter

憲章順: **wishlist → plans →（版）tests → TDD → マージ後 plans→specs**（`doc-rule.md`）。実装 Agent はコード + 当該 `docs/tests/`（+ 任意 `docs-agent-note/`）のみ。`plans` / `roadmap` / `wishlist` / `specs` は触らない。`v0.n.0` は破壊的変更許容（`versioning-rule.md`）。

----

## 3. 依存関係

```text
[es4] キー `/` → `:` （破壊的・es4 v0.10.0）
   └─► [es4] 薄い Scan(prefix) 追加
         └─► [ss-cachian] es4 Layer Driver（本計画・v0.11.0）
```

- **ハード依存:** PurgePrefix 実装前に es4 側 `Scan(prefix)` が必要（承認済み）。
- **推奨依存:** Driver 実装・契約テストのキー語彙は、es4 が **`:` デリミタ**になった後（または同一 es4 マイルストーン内でデリミタ確定後）に合わせる。
- **Valkey（ss-cachian v0.12.0）:** 並存。本計画は次 minor **v0.11.0** に載せる（SWR 意図スタブと同マイルストーン。実装順序は Issue／PO で調整可）。

----

## 4. API 寄せ方（Layer ↔ es4）

Layer は変えず、Driver が es4 に写像する。

| Layer | es4 側 | 第一版の寄せ方 |
|-------|--------|----------------|
| `Get` | `Get` | 欠落/`ErrNotFound` → miss（`hit=false`）。値は **Entry JSON** を decode。期限切れは論理 miss（必要なら削除） |
| `Set` | `Set` | Entry（value + created/expires）を JSON 化して保存。`ttl` で `ExpiresAt` を埋める（ttl=0 は無期限）。es4 に TTL API は足さない |
| `Delete` | `Delete` | Layer は冪等。es4 の欠落エラーは **握りつぶして成功** |
| `Incr` | `BeginTx` → Get/Set → Commit | 未作成は 0+1→1。非整数 → `ErrNotInteger`。溢位 → `ErrIncrOverflow`。競合時の再試行回数は **未決**（firestore は最大 8・要確認） |
| `PurgeExact` | `Scan` + フィルタ + `Delete` またはキー列挙方針 | `IsVersionDataKey` で version data のみ。`__version__` は消さない |
| `PurgePrefix` | `Scan(prefix)` → 各キー `Delete` | 空 prefix → `ErrEmptyKey`。冪等 |

補足:

- es4 値は JSON 必須のため、Entry ラップの JSON 形（フィールド名・時刻フォーマット）は **テスト仕様で固定**する（実装前に書く。ここでの最終スキーマは未決）。
- 論理キーは ss-cachian 側が既に `:` を使う。es4 が `:` 階層規則に揃った後、拒否規則（先頭/末尾/連続 `:`）に触れるキーを出さないことは KeyBuilder / テストの責任。

### 設定・配線（方針）

- config `layers[].driver` に es4 相当名を追加する想定（正確な文字列は **未決**。候補例: `es4`）。
- `options` は es4 `Options` に写像可能な最小セット（例: `memory_only` / `state_path` / redis・firestore 系）。**第一版でどの Backend 組み合わせを公式サポートするかは未決**（最低限 `memory_only` は受け入れる想定だが、確定ではない）。
- プロセスが Layer 用に `es4.Open` した `DB` を閉じるタイミング（CacheType 寿命との関係）は **未決**。

### HTTP Es4 Server

第一選択にしない。埋め込み open のみを第一版の受け入れ対象とする。Server 経由アダプタは wishlist / 別計画扱い。

----

## 5. charter 準拠フェーズと Issue / PR タイミング

| # | フェーズ | 誰 | 成果物 | 完了の目安 |
|---|----------|-----|--------|------------|
| **A** | 方針合意 | PO + 設計 | 本ファイル + デリミタ計画の承認。§9 未決の回答 | 実装 Agent を動かしてよい、と言うまでコード禁止 |
| **B** | リポジトリ docs（plans） | **main / PO** | 本ファイル転記（本 PR）。roadmap 追記は任意フォロー | plans が仕様詳細まで |
| **C** | テスト仕様 | 設計ドラフト → 実装 Agent がリポへ | `docs/tests/driver-es4/`（仮）+ 既存 layer/purge への追記方針 | FIX |
| **D** | es4 前提（デリミタ + Scan） | **es4 実装 Agent**（別計画） | `:` キー規則 + `Scan(prefix)` | ss-cachian が依存できるタグ／コミット |
| **E** | TDD 実装（**Go 先**） | **ss-cachian 実装 Agent** | Red→Green→Refactor。`go/sscachian/driver/es4`（仮） | Go 受け入れ緑 |
| **F** | TDD 実装（**Node 後**） | 同上 | Go パリティ | Node 受け入れ緑 |
| **G** | マージ後 docs 昇格 | **main / PO** | plans → specs、roadmap 完了反映 | specs が正本 |

### Issue / PR

- **Issue:** 実装トラッキングは GitHub Issue（Milestone `v0.11.0`）。es4 側 Issue（デリミタ／Scan）へ相互リンク。
- **PR:** テスト仕様 FIX 後の TDD のみ。plans/roadmap/specs を実装 PR で書き換えない（charter `doc-rule`）。
- ブランチ慣例: charter は `feat-*`。Cloud 制約名がある場合は PR 本文で Issue を明示。

### 推奨実装順（コード）

1. es4: キーデリミタ変更（es4 `docs/plans/v0.10.0/key-delimiter.md`）
2. es4: `Scan(prefix)`
3. ss-cachian **Go** es4 Driver
4. ss-cachian **Node** パリティ
5. （任意・別枠）config 例・ドキュメント pillar 更新は PO

----

## 6. テスト方針

charter `tdd.md`: 正常系〜3 / 異常系 3〜5 を目安。既存 `docs/tests/driver-memory` / `driver-firestore` / `purge` と同型の **driver-es4** 節を新設する想定。

### Driver 契約（Go 先・Node 後で同セマンティクス）

| 領域 | 観点（例） |
|------|------------|
| CRUD | Set→Get hit、Delete 冪等、空キー `ErrEmptyKey`、負 TTL `ErrNegativeTTL` |
| TTL | 期限前 hit・期限後論理 miss（Entry ラップ） |
| Incr | 未作成→1、整数インクリメント、非整数、溢位、Tx 原子性（並行は可能な範囲で） |
| PurgeExact | version data のみ削除、`__version__` 残存 |
| PurgePrefix | `Scan` 結果を全削除。prefix 外は残す。空 prefix 拒否 |
| 多層 | memory L1 + es4 L2 の書き戻し（既存 layer テストに Driver を差し込む形が望ましい） |

### テスト実行環境

- 第一版の CI は **埋め込み memory_only（または同等の軽い State）** を推奨（Emulator / 外部 Valkey 必須にしない）。
- Firestore / Redis を State に使う経路を公式サポートするなら、既存 es4 E2E / Emulator 方針に合わせた任意ジョブ（範囲は未決）。

----

## 7. 受け入れ条件（第一版）

1. Go の es4 Driver が Layer 全メソッドを満たし、上記テストが緑。
2. Node が Go と同セマンティクス（パリティ後）。
3. 依存はモジュール／パッケージとして **ss-cachian → es4**。逆依存なし。
4. TTL は Driver Entry ラップのみ。es4 公開 API に TTL が増えていない。
5. PurgePrefix が es4 `Scan` + Delete で実装され、Valkey 計画と並存できる（Driver 名が衝突しない）。
6. 埋め込み open で受け入れ可能。HTTP Server 必須ではない。
7. charter 順（plans → テスト仕様→TDD）を踏んでいる。

----

## 8. リスク

| リスク | 影響 | 緩和 |
|--------|------|------|
| es4 `Scan` / デリミタが遅れ・仕様変更 | Driver がブロックまたは二重改修 | 本計画は es4 完了をフェーズ D のゲートにする |
| Entry JSON 形が曖昧 | 言語間・版間で読めない | テスト仕様でスキーマ固定 |
| es4 `Delete`/`Get` の ErrNotFound 写像漏れ | Layer 冪等・miss 契約破壊 | 異常系テスト必須 |
| Incr の Tx 競合 | 失われた更新 | 再試行方針を A で決める |
| 大きな Scan + 逐次 Delete | PurgePrefix が重い | 第一版は正しさ優先。バッチ API は入れない |
| v0.11.0 内で SWR と作業競合 | 版の混乱 | Issue で順序を明示。Valkey は v0.12.0 のまま |

----

## 9. こうきさん確認が残る点（未決）

1. **v0.11.0 内の実装順序** — SWR 意図スタブとの前後・併記の詳細。
2. **config の `driver` 文字列と options キー** — 第一版でサポートする es4 Backend（memory_only のみか、sqlite/redis/firestore までか）。
3. **Entry JSON スキーマ** — フィールド名・時刻（RFC3339 等）・`value` の型の持ち方。
4. **Incr 競合時の再試行** — 回数・バックオフを firestore に揃えるか。
5. **`DB` / `close` の所有権** — Driver が Open するか、呼び出し側注入か（`OpenWith` 相当）。
6. **PurgeExact を Scan 前提にするか** — Exact だけ別実装（キーパターン既知）にするか。承認済み大枠は PurgePrefix→Scan；Exact の実装詳細は未決。
7. **HTTP Server アダプタを backlog に残すか** — 残すなら wishlist 一言のみ、など。

----

## 10. 関連リンク

- Layer 契約: `docs/specs/layer/`
- Purge: `docs/specs/purge/`
- Valkey 意図スタブ: `docs/plans/v0.12.0/`
- es4 State: es4 `docs/specs/state/state.md`
- es4 キーデリミタ計画: [b4moss/es4 `docs/plans/v0.10.0/key-delimiter.md`](https://github.com/b4moss/es4/blob/main/docs/plans/v0.10.0/key-delimiter.md)

----

以上
