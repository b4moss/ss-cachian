# ss-cachian × es4 Layer Driver 計画

| 項目 | 値 |
|------|-----|
| 状態 | **完了**（実装 #70・タグ `v0.11.0` + npm。Issue [#68](https://github.com/b4moss/ss-cachian/issues/68) クローズ。意味論は [specs/driver-es4](../../specs/driver-es4/)） |
| マイルストーン | [v0.11.0](https://github.com/b4moss/ss-cachian/milestone/2)（es4 Driver 完了。SWR 等は後続） |
| Tracking Issue | [#68](https://github.com/b4moss/ss-cachian/issues/68) |
| Plans PR | [#67](https://github.com/b4moss/ss-cachian/pull/67)（base: `develop`・merged） |
| 対象リポジトリ | [b4moss/ss-cachian](https://github.com/b4moss/ss-cachian)（Driver）／依存先 [b4moss/es4](https://github.com/b4moss/es4) |
| 開発方針の正本 | [b4moss/charter](https://github.com/b4moss/charter)（本リポ `docs/charter/` 同梱） |
| override | `docs/override-charter.md` は **現状設定なし** → 憲章どおり |
| 関連計画 | es4: [キーデリミタ `:` 化](https://github.com/b4moss/es4/blob/main/docs/plans/v0.10.0/key-delimiter.md)（Milestone [v0.10.0](https://github.com/b4moss/es4/milestone/1)） |
| 転記元 | Project Context 計画ドラフト（2026-10-10 転記・2026-10-11 決定追記） |

## 決定事項（2026-10-11）

こうきさん一問一答の確定。Issue [#68](https://github.com/b4moss/ss-cachian/issues/68)・Milestone **v0.11.0**。

| # | 項目 | 決定 |
|---|------|------|
| 7 | DB 所有権 | **A** — Driver が `Open` / `Close` を所有 |
| 8 | 第一版 Backend | **D** — Firestore まで全 Backend 公式サポート |
| 9 | Entry JSON | **B** — `{"value": <payload>, "expires_at": "<RFC3339>"}`（`expires_at` 省略＝期限なし） |
| 10 | Incr 再試行 | **A** — 既存 Firestore Driver に揃える |
| 11 | PurgeExact | **A** — Scan なし（キー既知で version data のみ削除） |
| 12 | HTTP Es4 Server アダプタ | **C** — **v0.11.0 に埋め込みと並列で含める** |
| 13 | v0.11 内の順序 | **C** — SWR と並行（別 PR）。Valkey は v0.13 並存（旧 v0.12 枠は [セッション TTL](../v0.12.0/plan.md) に転用・PO 決定済み） |

**前提（上流）:** es4 v0.10.0 でデリミタ `:` ＋ `Scan` が先。TTL は Driver Entry ラップ。PurgePrefix は es4 `Scan` + Delete。

**本ファイルの位置づけ:** 完了注記。現行の振る舞い正本は [`docs/specs/driver-es4/`](../../specs/driver-es4/) · [`docs/tests/driver-es4/`](../../tests/driver-es4/)。

----

## 1. 目的・範囲

### 目的

ss-cachian の **Layer 契約**を満たす **es4 Driver** を追加し、埋め込み `es4.Open` をストレージとして使えるようにする。Valkey Driver（roadmap **v0.13.0**・意図スタブ）と **並存**する別 Backend。HTTP Es4 Server アダプタは **埋め込みと並列で v0.11.0 に含める**（決定 C）。

### 入れる（第一版・承認済み）

| 項目 | 方針 |
|------|------|
| 置き場所 | **ss-cachian 側**に Layer / Driver（依存方向: **ss-cachian → es4**） |
| 接続形態 | **埋め込み open が本命**（プロセス内 `es4.Open` / Node `open`）。**HTTP Server アダプタも同版に並列** |
| TTL | es4 本体に TTL を足さない。**Driver 側で Entry をラップ**して期限を持つ（Entry JSON **B**） |
| Incr | es4 **Tx** 上で read-modify-write。再試行は **Firestore Driver に揃える** |
| PurgePrefix | es4 に薄い **`Scan(prefix)`** を足し、Driver が返却キーへ `Delete` を回す |
| PurgeExact | **Scan なし**。キー既知で version data のみ削除（`IsVersionDataKey`） |
| Backend | **Firestore まで全サポート**（第一版） |
| DB 寿命 | **Driver が Open / Close を所有** |
| 並存 | Valkey Driver 計画を置き換えない（v0.13）。SWR と並行・別 PR |

### 入れない（第一版）

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
| Valkey は roadmap **v0.13.0**・plans は意図スタブ | `docs/roadmap.md`・`docs/plans/v0.13.0/`・`unscheduled/future-apis.md` |
| TTL は Layer `Set` 引数。memory は Entry + `ExpiresAt` の遅延削除 | `driver/memory/memory.go`・`docs/specs/driver-memory/` |
| config `layers[].driver` は `memory` \| `firestore`（未知は拒否） | specs/config・config テスト |

### es4

| 事実 | 根拠 |
|------|------|
| 公開 State: `Set` / `Get` / `Delete` / `Exists` / `Clear`。Tx 別面 | `docs/specs/state/state.md`・`docs/specs/tx/` |
| **Scan(prefix)** あり（es4 v0.10.0） | state specs・es4-server |
| キーは階層 **`:`**（es4 v0.10.0 済み） | `ValidateKey`・[key-delimiter 完了注記](https://github.com/b4moss/es4/blob/main/docs/plans/v0.10.0/key-delimiter.md) |
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
- **Valkey（ss-cachian v0.13.0）:** 並存のまま。本計画は **v0.11.0**（SWR と並行・別 PR）。v0.12.0 は [セッション TTL](../v0.12.0/plan.md)。

----

## 4. API 寄せ方（Layer ↔ es4）

Layer は変えず、Driver が es4 に写像する。

| Layer | es4 側 | 第一版の寄せ方 |
|-------|--------|----------------|
| `Get` | `Get` | 欠落/`ErrNotFound` → miss（`hit=false`）。値は **Entry JSON** を decode。期限切れは論理 miss（必要なら削除） |
| `Set` | `Set` | Entry JSON **B**（`value` + `expires_at` RFC3339）を保存。`ttl` で `expires_at` を埋める（ttl=0／省略＝無期限）。es4 に TTL API は足さない |
| `Delete` | `Delete` | Layer は冪等。es4 の欠落エラーは **握りつぶして成功** |
| `Incr` | `BeginTx` → Get/Set → Commit | 未作成は 0+1→1。非整数 → `ErrNotInteger`。溢位 → `ErrIncrOverflow`。競合再試行は **firestore Driver に揃える** |
| `PurgeExact` | キー既知で version data のみ `Delete`（**Scan なし**） | `IsVersionDataKey`。`__version__` は消さない |
| `PurgePrefix` | `Scan(prefix)` → 各キー `Delete` | 空 prefix → `ErrEmptyKey`。冪等 |

補足:

- Entry JSON 形は **決定 B** で固定。テスト仕様にそのまま書く。
- 論理キーは ss-cachian 側が既に `:` を使う。es4 が `:` 階層規則に揃った後、拒否規則（先頭/末尾/連続 `:`）に触れるキーを出さないことは KeyBuilder / テストの責任。

### 設定・配線（方針）

- config `layers[].driver` に es4 相当名を追加（文字列細部は実装時に既存慣例へ合わせる。候補例: `es4`）。
- `options` は es4 `Options` に写像。**第一版は Firestore まで全 Backend を公式サポート**（決定 D）。
- **`DB` の Open / Close は Driver が所有**（決定 A）。

### HTTP Es4 Server

埋め込み open が本命。**HTTP Server アダプタも v0.11.0 に埋め込みと並列で含める**（決定 C）。

----

## 5. charter 準拠フェーズと Issue / PR タイミング

| # | フェーズ | 誰 | 成果物 | 完了の目安 |
|---|----------|-----|--------|------------|
| **A** | 方針合意 | PO + 設計 | 本ファイル + デリミタ計画の承認。§9 決定済み（2026-10-11） | 完了 |
| **B** | リポジトリ docs（plans） | **main / PO** | 本ファイル転記（PR #67）。決定追記フォロー。roadmap 追記は任意フォロー | plans が仕様詳細まで |
| **C** | テスト仕様 | 設計ドラフト → 実装 Agent がリポへ | `docs/tests/driver-es4/`（仮）+ 既存 layer/purge への追記方針 | FIX |
| **D** | es4 前提（デリミタ + Scan） | **es4 実装 Agent**（別計画） | `:` キー規則 + `Scan(prefix)` | ss-cachian が依存できるタグ／コミット |
| **E** | TDD 実装（**Go 先**） | **ss-cachian 実装 Agent** | Red→Green→Refactor。`go/sscachian/driver/es4`（仮）＋ HTTP アダプタ | Go 受け入れ緑 |
| **F** | TDD 実装（**Node 後**） | 同上 | Go パリティ | Node 受け入れ緑 |
| **G** | マージ後 docs 昇格 | **main / PO** | plans → specs、roadmap 完了反映 | specs が正本 |

### Issue / PR

- **Issue:** 実装トラッキングは GitHub Issue（Milestone `v0.11.0`）。es4 側 Issue（デリミタ／Scan）へ相互リンク。
- **PR:** テスト仕様 FIX 後の TDD のみ。plans/roadmap/specs を実装 PR で書き換えない（charter `doc-rule`）。SWR とは並行・別 PR。
- ブランチ慣例: charter は `feat-*`。Cloud 制約名がある場合は PR 本文で Issue を明示。

### 推奨実装順（コード）

1. es4: キーデリミタ変更（es4 `docs/plans/v0.10.0/key-delimiter.md`）
2. es4: `Scan(prefix)`
3. ss-cachian **Go** es4 Driver（＋ HTTP アダプタ並列可）
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
| PurgeExact | version data のみ削除、`__version__` 残存（Scan なし） |
| PurgePrefix | `Scan` 結果を全削除。prefix 外は残す。空 prefix 拒否 |
| 多層 | memory L1 + es4 L2 の書き戻し（既存 layer テストに Driver を差し込む形が望ましい） |

### テスト実行環境

- CI は **埋め込み memory_only（または同等の軽い State）** を基本とする。
- Firestore まで公式サポートするため、既存 es4 E2E / Emulator 方針に合わせた任意ジョブを追加してよい。

----

## 7. 受け入れ条件（第一版）

1. Go の es4 Driver が Layer 全メソッドを満たし、上記テストが緑。
2. Node が Go と同セマンティクス（パリティ後）。
3. 依存はモジュール／パッケージとして **ss-cachian → es4**。逆依存なし。
4. TTL は Driver Entry ラップのみ。es4 公開 API に TTL が増えていない。
5. PurgePrefix が es4 `Scan` + Delete で実装され、Valkey 計画と並存できる（Driver 名が衝突しない）。
6. 埋め込み open で受け入れ可能。HTTP Server アダプタも同版に含む。
7. charter 順（plans → テスト仕様→TDD）を踏んでいる。

----

## 8. リスク

| リスク | 影響 | 緩和 |
|--------|------|------|
| es4 `Scan` / デリミタが遅れ・仕様変更 | Driver がブロックまたは二重改修 | 本計画は es4 完了をフェーズ D のゲートにする |
| Entry JSON 形の実装ずれ | 言語間・版間で読めない | 決定 B をテスト仕様で固定 |
| es4 `Delete`/`Get` の ErrNotFound 写像漏れ | Layer 冪等・miss 契約破壊 | 異常系テスト必須 |
| Incr の Tx 競合 | 失われた更新 | Firestore Driver 再試行に揃える |
| 大きな Scan + 逐次 Delete | PurgePrefix が重い | 第一版は正しさ優先。バッチ API は入れない |
| v0.11.0 内で SWR と作業競合 | 版の混乱 | 並行・別 PR（決定 C）。Valkey は v0.13.0 |

----

## 9. 決定済み（2026-10-11）

先頭の **決定事項（2026-10-11）** と同内容。記録用の詳細:

1. **SemVer / roadmap 上の版** — **v0.11.0**。**C:** SWR と es4 Driver は並行・別 PR、順序不問。Valkey は v0.13 並存のまま。
2. **config の `driver` 文字列と options キー** — **D:** Firestore まで含む全 Backend を第一版サポート。driver 名文字列の細部は実装時に既存に合わせる。
3. **Entry JSON スキーマ** — **B:** `{"value": <payload>, "expires_at": "<RFC3339>"}`。`expires_at` 省略＝期限なし。
4. **Incr 競合時の再試行** — **A:** 既存 Firestore Driver に揃える。
5. **`DB` / `close` の所有権** — **A:** Driver が Open / Close を所有。
6. **PurgeExact** — **A:** Scan なし。キー既知で version data のみ削除。
7. **HTTP Server アダプタ** — **C:** **v0.11.0 に埋め込みと並列で含める**。

----

## 10. 関連リンク

- Layer 契約: `docs/specs/layer/`
- Purge: `docs/specs/purge/`
- Valkey 意図スタブ: `docs/plans/v0.13.0/`
- セッション TTL / Firestore TTL: `docs/plans/v0.12.0/plan.md`
- es4 State: es4 `docs/specs/state/state.md`
- es4 キーデリミタ計画: [b4moss/es4 `docs/plans/v0.10.0/key-delimiter.md`](https://github.com/b4moss/es4/blob/main/docs/plans/v0.10.0/key-delimiter.md)

----

以上
