---
type: Plan
title: v0.12.0 セッション TTL / Firestore TTL（仕様詳細）
description: es4 期限付き書き込み利用、firestore ゼロ値 expires_at 抑止、TTL ポリシー設定の docs。
tags: [plan, phase2, v0.12.0, ttl, es4, firestore]
timestamp: 2026-10-28T00:00:00Z
status: 仕様詳細
---

# v0.12.0 — セッション TTL / Firestore TTL

| 項目 | 値 |
|------|-----|
| 状態 | **仕様詳細**（計画 PR・コード未変更） |
| マイルストーン | **v0.12.0**（提案。旧 Valkey 枠を本内容に充て、Valkey→v0.13.0 / PHP→v0.14.0） |
| 対象リポジトリ | [b4moss/ss-cachian](https://github.com/b4moss/ss-cachian) |
| 上流依存 | [b4moss/es4](https://github.com/b4moss/es4) の期限付き書き込み（**並行計画中。API 名は仮置き**。es4 側計画 PR で確定したら本 plan を追随更新） |
| 利用側 | jparse（b4m-inner/jp-org-parse）v0.10.0 方針・セッションはサーバー側。TTL ポリシー作成（gcloud/Terraform）は **jparse の TF** |
| override | `docs/override-charter.md` は現状設定なし → 憲章どおり |
| テスト仕様 | [tests/driver-es4](../../tests/driver-es4/) · [tests/driver-firestore](../../tests/driver-firestore/) |
| 転記元 | uploads `session-ttl-common.md`（2026-10-28 PO 決定） |

----

## 1. 背景と現状の問題

### 構成（jparse 本番想定）

```text
jparse → ss-cachian（es4 ドライバ）→ es4（state_backend=firestore）→ Firestore
```

- セッション・DB クエリキャッシュ・件数を TTL 付きで Firestore に保存する。
- セッションは Firestore 層のみ（メモリ L1 に置かない）。件数・クエリ結果は memory L1 + es4/Firestore L2。
- 期限切れ文書は **Firestore TTL ポリシー**（日時型フィールド）で自動削除する（PO 決定）。アプリ側の掃除 Job は作らない。

### 問題（調査済み・現行 v0.11.0）

| 箇所 | 事実 | 影響 |
|------|------|------|
| es4 Firestore state | ドキュメントは `{value: string}` のみ。State `Set` は key+value。日時型期限フィールドがない | TTL ポリシーを掛けられない |
| ss-cachian es4 ドライバ | 期限を **value JSON 内の文字列** `expires_at`（RFC3339）に書き、読み取り時に判定するだけ（v0.11.0 決定 B） | Firestore 側に日時型が無い。TTL ポリシー不可 |
| ss-cachian firestore ドライバ | 日時型 `expires_at` を書くが、期限なしのとき **ゼロ値**（Go: `time.Time{}` / Node: `new Date(0)`）を書く。`Incr` も同様 | TTL ポリシーを掛けると期限なし文書が即削除対象になる |

----

## 2. 目的（本マイルストーンで入れるもの）

1. **es4 ドライバ**が es4 の期限付き書き込み（仮 API）を使い、Firestore 上で **日時型 `expires_at`** が付くようにする。
2. **firestore ドライバ**が期限なしのとき **ゼロ値の `expires_at` を書かない**（フィールド省略。`Incr` 含む）。
3. ライブラリ **docs** に、Firestore TTL ポリシーで使う設定（フィールド名・コレクション・注意）を書く。ポリシー作成自体（gcloud/TF）は利用側。

Go / Node パリティ必須。

----

## 3. やらぬこと

- アプリ側の掃除 Job / バッチ削除
- TTL ポリシーの gcloud / Terraform 作成（jparse TF 側）
- Valkey Driver（→ [v0.13.0](../v0.13.0/)）
- PHP ポート（→ [v0.14.0](../v0.14.0/)）
- es4 本体の実装（上流リポの計画・実装に委ねる。本リポは **消費側**）
- SWR / SIE / negative（v0.11.0 意図スタブの残り。本版では触らない）

----

## 4. 上流依存（es4・API 名は仮置き）

es4 側で期限付き書き込みを追加する計画が **並行**している。ss-cachian は次を前提にする。

| 仮置き名 | 意味 |
|----------|------|
| `SetWithExpiry(ctx, key, value, expiresAt)` | 期限付き Set。Firestore ではドキュメントに日時型 `expires_at` を書く |
| または `Set(ctx, key, value, opts... WithExpiry(t))` | オプション付き Set（同等） |
| 期限なし Set | 既存どおり。Firestore では **`expires_at` フィールドを省略**（ゼロ値・null を書かない） |
| Get | 期限切れ（`expires_at` が過去）は **不在扱い**（`ErrNotFound` 相当）。TTL 削除は最大約 24h 遅れるため、読み取り側でも必須 |

確定後の作業:

1. es4 計画 PR の API 名・セマンティクスに本 plan / tests を追随更新する。
2. ss-cachian の es4 依存をその版にピンする。

他バックエンドでの意味（es4 計画に含める想定。ss-cachian は Layer 契約を壊さない範囲で追従）:

| Backend | 期待（仮） |
|---------|------------|
| Memory / SQLite | 読み取り時に期限切れを不在扱い（必要なら内部メタ） |
| Redis / Valkey | `EXPIRE` 等のネイティブ期限と整合 |
| Firestore | 日時型 `expires_at` + 読み取り時不在扱い |
| Tx / Scan / Recovery | 期限メタの写像・Export/Replace での扱いを es4 計画で固定。ss-cachian は Tx 上の Incr が期限付き API と矛盾しないことだけ確認 |

----

## 5. ss-cachian 方針

### 5.1 es4 ドライバ

| 項目 | 方針 |
|------|------|
| 書き込み | Layer `Set(ttl>0)` / `Entry.ExpiresAt` 指定時は es4 の期限付き書き込みを呼ぶ。値本体はキャッシュ payload（JSON）。**Firestore の日時型 `expires_at` は es4 が書く** |
| 期限なし | es4 の通常 Set。Firestore 上で `expires_at` フィールドは付かない |
| 読み取り | es4 が期限切れを不在返す前提。ドライバ側でも防御的に Miss してよい |
| Entry JSON（現行決定 B） | value 内文字列 `expires_at` への二重持ちは **やめる**（推奨）。移行期に両方書く案は要決定 |
| Backend 面 | `Backend.Set` を期限付きに拡張、または別メソッド。HTTP アダプタも同セマンティクス |
| Incr | Tx 上の read-modify-write。カウンタは原則期限なし（フィールド省略）。既存キーに期限がある場合の扱いを要決定 |
| 依存 | 期限付き API が入った es4 版へピン。memory_only CI は期限セマンティクスをモック／memory backend で検証 |

### 5.2 firestore ドライバ

| 項目 | 方針 |
|------|------|
| 期限あり Set | 現行どおり日時型 `expires_at = now+ttl`（または Entry.ExpiresAt）を書く |
| 期限なし Set | **`expires_at` をドキュメントに含めない**（ゼロ値・`Date(0)`・null を書かない） |
| 上書き | 期限あり→なしに変えたとき、既存の `expires_at` を **削除**する（`FieldValue.delete()` 相当）。書かないだけでは旧値が残る |
| Incr | 現行 Node は `expires_at: new Date(0)` を書いてしまう。Go はゼロ値を struct 経由で書く。いずれも **省略**に直す。既存 `expires_at` があるキーへの Incr は要決定（削除して無期限化 / 維持） |
| Get | フィールド欠落 = 期限なし。過去日時 = 論理 Miss（削除は必須としない。TTL ポリシーに任せる） |
| ドキュメント ID | 現行どおりキー文字列そのまま（変更しない） |

### 5.3 docs（TTL ポリシー設定）

実装時に次を **ライブラリ docs** へ書く（正本候補: `docs/specs/driver-firestore/` に運用節、および必要ならルート README の短いリンク。es4 経路は「es4 の collection + フィールド `expires_at`」と明記し、es4 docs へも相互リンク）。

#### 書く内容（草案・実装時に specs へ昇格）

**共通**

- TTL フィールド名: **`expires_at`**（日時型 / Firestore Timestamp）
- フィールド欠落または非日時型 = そのドキュメントは TTL 対象外（期限なし）
- ゼロ値（`0001-01-01` / Unix epoch）を書かないこと（本マイルストーンのコード修正で担保）
- 期限切れ削除は即時ではない（通常最大約 24 時間）。アプリ／ライブラリは **読み取り時にも期限切れを Miss** する
- ポリシー作成・適用は利用側（例: jparse Terraform）。本ライブラリは作成手順のコマンド例だけ示す

**firestore ドライバ直接利用時**

| 項目 | 値 |
|------|-----|
| Collection（既定） | `sscachian`（`Options.Collection` / 設定で変更可） |
| TTL フィールド | `expires_at` |
| ポリシー単位 | collection group = 上記 collection ID |

例（gcloud・参考）:

```bash
gcloud firestore fields ttls update expires_at \
  --collection-group=sscachian \
  --enable-ttl
```

**es4 ドライバ経由（jparse 本番経路）**

| 項目 | 値 |
|------|-----|
| Collection | es4 Options の `state_firestore_collection`（利用側が決める。ss-cachian 既定はない） |
| TTL フィールド | es4 が書く **`expires_at`**（日時型） |
| ポリシー単位 | その collection group |

例:

```bash
gcloud firestore fields ttls update expires_at \
  --collection-group=<state_firestore_collection> \
  --enable-ttl
```

**注意**

- 既存データにゼロ値 `expires_at` が残っていると、ポリシー有効化後に削除対象になる。移行手順（バックフィルでフィールド削除、または再書き込み）を docs に一文で書く。
- Emulator では TTL ポリシー挙動を本番相当に再現できない前提。単体は「フィールドの有無」と「読み取り Miss」で担保する。

----

## 6. 破壊的変更・互換

| 変更 | 影響 | SemVer |
|------|------|--------|
| es4 ドライバが value JSON 内 `expires_at` 文字列を書かなくなる（推奨案） | 旧形式のみのデータを読む互換が要るなら移行期サポート | **minor**（v0.n.0 で破壊許容。読み側フォールバックは要決定） |
| firestore ドライバがゼロ値を書かない／上書き時にフィールド削除 | ストレージ形が変わる。TTL ポリシー利用時は必須の修正 | minor（バグ修正に近いがセマンティクス明示） |
| 最小 es4 版の引き上げ | 利用者は es4 を更新する必要 | minor |

`VERSION` / `package.json` = `0.12.0`。タグ `v0.12.0`。

----

## 7. 作業順（実装時・TDD）

1. ~~計画・テスト仕様~~（本 PR）
2. 上流 es4 の期限付き API がマージ・版付けされたことを確認し、仮名を本 plan / tests で確定名に置換
3. firestore ドライバ: ゼロ値抑止 + 上書き時フィールド削除（Go / Node）— テスト先行
4. es4 ドライバ: Backend / HTTP を期限付き API 対応。Entry JSON 方針に従い Set/Get/Incr を更新（Go / Node）
5. docs: TTL ポリシー設定節を specs（+ README 短い案内）へ反映
6. es4 依存ピン・CI（memory_only 必須、Firestore Emulator はフィールド有無のスモーク）
7. `VERSION` / npm = `0.12.0` → develop → main → タグ

実装 Agent は charter どおり **コード + 当該 `docs/tests/`** のみ。plans / roadmap / specs の昇格は main 側。

----

## 8. 受け入れ条件

- [ ] es4 ドライバが期限付き書き込みを使い、Firestore backend で日時型 `expires_at` が付く（Emulator でフィールド型を確認できる範囲）
- [ ] 期限なし書き込みで Firestore 上に `expires_at` が無い（es4 経路・firestore ドライバ直接の両方）
- [ ] firestore ドライバがゼロ値 `expires_at` を書かない。期限あり→なし上書きでフィールドが消える
- [ ] Get は期限切れを Miss（TTL 削除待ちでも論理 Miss）
- [ ] Go / Node パリティ。既存 Layer / Purge / config 回帰なし
- [ ] docs に TTL ポリシー用のフィールド名・collection・注意が書かれている
- [ ] `VERSION` / `@b4moss/ss-cachian` = `0.12.0`

----

## 9. 版割当の根拠（提案）

| 版 | 現行（変更前） | 本提案 |
|----|----------------|--------|
| v0.11.0 | es4 Layer Driver ほか（実装済み・VERSION 0.11.0） | 変更なし |
| **v0.12.0** | Valkey Driver（意図スタブのみ） | **本 TTL 作業** |
| **v0.13.0** | PHP ポート（意図スタブのみ） | **Valkey**（旧 0.12 から繰り下げ） |
| **v0.14.0** | （なし） | **PHP**（旧 0.13 から繰り下げ） |
| v1.0.0 | Phase 2 締め | 変更なし |

**推奨理由:** jparse 本番経路（Firestore TTL）が Valkey / PHP より先に必要。両スタブは未着手のため番号繰り下げコストが低い。代替案（本内容を v0.14.0 にし Valkey/PHP 番号を維持）は要決定。

----

## 10. 関連リンク

- 共通前提: uploads `session-ttl-common.md`（2026-10-28）
- 現行 es4 ドライバ計画（Entry JSON B）: [v0.11.0/es4-layer-driver.md](../v0.11.0/es4-layer-driver.md) — 本版で TTL 部分を上流 API 利用へ更新
- Firestore TTL（Google）: [Manage data retention with TTL policies](https://firebase.google.com/docs/firestore/ttl)
- specs（実装後更新）: [driver-firestore](../../specs/driver-firestore/) · driver-es4（未作成なら本版で新設）

----

以上
