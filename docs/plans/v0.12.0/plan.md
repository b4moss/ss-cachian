---
type: Plan
title: v0.12.0 セッション TTL / Firestore TTL（仕様詳細）
description: es4 期限付き書き込み利用、firestore ゼロ値 expires_at 抑止、TTL ポリシー設定の docs。
tags: [plan, phase2, v0.12.0, ttl, es4, firestore]
timestamp: 2026-10-28T15:25:00+09:00
status: 仕様詳細
---

# v0.12.0 — セッション TTL / Firestore TTL

| 項目 | 値 |
|------|-----|
| 状態 | **仕様詳細**（PO 決定反映済み 2026-10-28。**Go 実装先行**・Node は es4 npm `0.11.0` 公開後） |
| マイルストーン | **v0.12.0**（PO 決定。Valkey→v0.13.0 / PHP→v0.14.0） |
| 対象リポジトリ | [b4moss/ss-cachian](https://github.com/b4moss/ss-cachian) |
| 上流依存 | [b4moss/es4](https://github.com/b4moss/es4) **v0.11.0**（期限付き `Set` options。計画 [es4#52](https://github.com/b4moss/es4/pull/52) · `5aa6b41`）。**タグ後に実装開始** |
| 利用側 | jparse（b4m-inner/jp-org-parse）v0.10.0 方針・セッションはサーバー側。TTL ポリシー作成（gcloud/Terraform）は **jparse の TF** |
| override | `docs/override-charter.md` は現状設定なし → 憲章どおり |
| テスト仕様 | [tests/driver-es4](../../tests/driver-es4/) · [tests/driver-firestore](../../tests/driver-firestore/) |
| 転記元 | uploads `session-ttl-common.md`（2026-10-28 PO 決定） |

## PO 決定（こうきさん・2026-10-28 15:20〜15:25 JST）

| # | 項目 | 決定 |
|---|------|------|
| 1 | 版割当 | **A** — 本内容 = **v0.12.0**。Valkey → **v0.13.0**、PHP → **v0.14.0** |
| 2 | es4 API 形 | **es4 側で確定**（[es4#52](https://github.com/b4moss/es4/pull/52) · `5aa6b41`）。`Set` ＋ options。Go `WithExpiresAt`、Node `{ expiresAt }`。**`SetWithExpiry` は使わない**。過去時刻 Set は保存成功・読取不在。Export/Recovery は値のみ。HTTP はクエリ `expires_at`。Redis/Valkey は読取フィルタのみ |
| 3 | Entry JSON 内文字列 `expires_at` | **A** — 廃止＋**旧形式フォールバック**（読み取り） |
| 4 | 期限あり→なし上書き | **A** — フィールド削除で無期限 |
| 5 | Incr と既存期限 | **A** — 期限を削除して無期限 |
| 6 | es4 ピン時期 | **A** — **es4 `v0.11.0` タグ後**に実装開始 |
| 7 | TTL docs 置き場 | **A** — **`docs/specs/driver-firestore/` の運用節**（実装時に昇格） |

----

## 1. 背景と現状の問題

### 構成（jparse 本番想定）

```text
jparse → ss-cachian（es4 ドライバ）→ es4（state_backend=firestore）→ Firestore
```

- セッション・DB クエリキャッシュ・件数を TTL 付きで Firestore に保存する。
- セッションは Firestore 層のみ（メモリ L1 に置かない）。件数・クエリ結果は memory L1 + es4/Firestore L2。
- 期限切れ文書は **Firestore TTL ポリシー**（日時型フィールド）で自動削除する（PO 決定）。アプリ側の掃除 Job は作らない。

### 問題（調査済み・現行 ss-cachian v0.11.0）

| 箇所 | 事実 | 影響 |
|------|------|------|
| es4 Firestore state（〜v0.10.0） | ドキュメントは `{value: string}` のみ。State `Set` は key+value | TTL ポリシーを掛けられない → es4 **v0.11.0** で解消予定 |
| ss-cachian es4 ドライバ | 期限を **value JSON 内の文字列** `expires_at`（RFC3339）に書き、読み取り時に判定するだけ（v0.11.0 決定 B） | Firestore 側に日時型が無い。TTL ポリシー不可 |
| ss-cachian firestore ドライバ | 日時型 `expires_at` を書くが、期限なしのとき **ゼロ値**（Go: `time.Time{}` / Node: `new Date(0)`）を書く。`Incr` も同様 | TTL ポリシーを掛けると期限なし文書が即削除対象になる |

----

## 2. 目的（本マイルストーンで入れるもの）

1. **es4 ドライバ**が es4 の `Set`＋`WithExpiresAt` / `{ expiresAt }` を使い、Firestore 上で **日時型 `expires_at`** が付くようにする。
2. **firestore ドライバ**が期限なしのとき **ゼロ値の `expires_at` を書かない**（フィールド省略。上書き・`Incr` ではフィールド削除）。
3. **`docs/specs/driver-firestore/` の運用節**に、Firestore TTL ポリシーで使う設定（フィールド名・コレクション・注意）を書く。ポリシー作成自体（gcloud/TF）は利用側。

Go / Node パリティ必須。

----

## 3. やらぬこと

- アプリ側の掃除 Job / バッチ削除
- TTL ポリシーの gcloud / Terraform 作成（jparse TF 側）
- Valkey Driver（→ [v0.13.0](../v0.13.0/)）
- PHP ポート（→ [v0.14.0](../v0.14.0/)）
- es4 本体の実装（上流リポ。本リポは **消費側**。開始は es4 `v0.11.0` タグ後）
- SWR / SIE / negative（v0.11.0 意図スタブの残り。本版では触らない）

----

## 4. 上流依存（es4 v0.11.0・確定）

参照: [b4moss/es4#52](https://github.com/b4moss/es4/pull/52)（`5aa6b41`）。

| API | 意味 |
|-----|------|
| Go `Set(ctx, key, value, opts ...SetOption)` ＋ `WithExpiresAt(t)` | 期限付き Set。Firestore ではドキュメントに日時型 `expires_at` を書く |
| Node `set(key, value, { expiresAt?: Date })` | 同上 |
| 期限なし `Set` / `set`（options なし） | 既存どおり。Firestore では **`expires_at` を省略**。上書き時はフィールド削除 |
| Get / Exists / Scan | 期限切れは **不在**。**過去時刻の Set は保存成功**、以降の読取は不在 |
| HTTP | クエリ `expires_at`（RFC3339） |
| Export / Recovery | **値のみ**（期限メタを含めない） |
| Redis / Valkey | ネイティブ EXPIRE なし。**読み取り時フィルタのみ** |
| Memory / SQLite / Firestore | 同一公開契約。Firestore は日時型 `expires_at` |

**`SetWithExpiry` は使わない。**

実装開始条件: es4 **`v0.11.0` がタグ付け**されたあと、その版にピンして ss-cachian 実装 PR を開始する。

----

## 5. ss-cachian 方針

### 5.1 es4 ドライバ

| 項目 | 方針（PO 決定済み） |
|------|---------------------|
| 書き込み | Layer `Set(ttl>0)` / `Entry.ExpiresAt` 指定時は es4 `Set(..., WithExpiresAt(t))` / `set(..., { expiresAt })`。値本体はキャッシュ payload（JSON）。**Firestore の日時型 `expires_at` は es4 が書く** |
| 期限なし | options なしの `Set`。Firestore 上で `expires_at` は付かない（es4 が上書き時に削除） |
| 読み取り | es4 が期限切れを不在返す前提。ドライバ側でも防御的に Miss してよい |
| Entry JSON | value 内文字列 `expires_at` は **書かない（廃止）**。読み取りでは **旧形式フォールバック**（`{"value", "expires_at":"<RFC3339>"}` を解釈し、期限切れは Miss） |
| Backend 面 | `Backend.Set` を options 対応（または同等）。HTTP アダプタは上流クエリ `expires_at` に合わせる |
| Incr | Tx 上の read-modify-write。新規は期限なし。**既存に期限があるキーへの Incr は期限を削除して無期限**（決定 5-A） |
| 依存 | **es4 `v0.11.0` タグ後**にピン。memory_only CI で期限セマンティクスを検証 |

### 5.2 firestore ドライバ

| 項目 | 方針（PO 決定済み） |
|------|---------------------|
| 期限あり Set | 現行どおり日時型 `expires_at = now+ttl`（または Entry.ExpiresAt）を書く |
| 期限なし Set | **`expires_at` をドキュメントに含めない**（ゼロ値・`Date(0)`・null を書かない） |
| 上書き | 期限あり→なしに変えたとき、既存の `expires_at` を **削除**する（`FieldValue.delete()` 相当）（決定 4-A） |
| Incr | ゼロ値を書かない。**既存 `expires_at` があるキーへの Incr はフィールドを削除して無期限**（決定 5-A） |
| Get | フィールド欠落 = 期限なし。過去日時 = 論理 Miss（削除は必須としない。TTL ポリシーに任せる） |
| ドキュメント ID | 現行どおりキー文字列そのまま（変更しない） |

### 5.3 docs（TTL ポリシー設定）

実装時に **`docs/specs/driver-firestore/` の運用節**へ書く（決定 7-A）。ルート README からは短いリンク可。es4 経路は「es4 の `state_firestore_collection` + フィールド `expires_at`」と明記し、es4 docs へ相互リンク。

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

```bash
gcloud firestore fields ttls update expires_at \
  --collection-group=sscachian \
  --enable-ttl
```

**es4 ドライバ経由（jparse 本番経路）**

| 項目 | 値 |
|------|-----|
| Collection | es4 Options の `state_firestore_collection`（利用側が決める） |
| TTL フィールド | es4 が書く **`expires_at`**（日時型） |
| ポリシー単位 | その collection group |

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
| es4 ドライバが value JSON 内 `expires_at` 文字列を書かなくなる | 旧形式は **読み取りフォールバック**で互換（決定 3-A） | **minor** |
| firestore ドライバがゼロ値を書かない／上書き・Incr でフィールド削除 | ストレージ形が変わる。TTL ポリシー利用時は必須 | minor |
| 最小 es4 版を **v0.11.0** へ引き上げ | 利用者は es4 を更新する必要 | minor |

Go `VERSION` = `0.12.0`。`package.json`（Node）は **es4 npm `@b4moss/es4@0.11.0` 公開後の後続**で `0.12.0` に揃える（PO 決定 2026-10-28 16:11 JST: Go 先行）。タグ `v0.12.0` は公開手順時。

----

## 7. 作業順（実装時・TDD）

1. ~~計画・テスト仕様・PO 決定反映~~（本 PR）
2. **es4 `v0.11.0` タグ**を待つ（決定 6-A）
3. firestore ドライバ: ゼロ値抑止 + 上書き／Incr 時フィールド削除（**先に Go**）— テスト先行
4. es4 ドライバ: `WithExpiresAt` 対応。Entry JSON 廃止＋旧形式フォールバック。Backend / HTTP 追随（**先に Go**）
5. docs: TTL ポリシー設定を **`docs/specs/driver-firestore/` 運用節**へ反映
6. es4 Go `v0.11.0` 依存ピン・CI（memory_only 必須、Firestore Emulator はフィールド有無のスモーク）
7. Go `VERSION` = `0.12.0` → develop →（指示後）main → タグ
8. **後続:** es4 npm `@b4moss/es4@0.11.0` 公開後に Node パリティ（`{ expiresAt }`・firestore 同修正・`package.json` 0.12.0）

実装 Agent は charter どおり **コード + 当該 `docs/tests/`** のみ。plans / roadmap / specs の昇格は main 側。

----

## 8. 受け入れ条件

- [ ] es4 ドライバが `WithExpiresAt` / `{ expiresAt }` を使い、Firestore backend で日時型 `expires_at` が付く
- [ ] 期限なし書き込みで Firestore 上に `expires_at` が無い（es4 経路・firestore ドライバ直接の両方）
- [ ] firestore ドライバがゼロ値 `expires_at` を書かない。期限あり→なし上書き／Incr でフィールドが消える
- [ ] Get は期限切れを Miss。旧 Entry JSON（文字列 `expires_at`）もフォールバックで解釈できる
- [ ] Go パリティ（本実装）。**Node は es4 npm 公開後**（PO 2026-10-28 16:11）
- [ ] `docs/specs/driver-firestore/` 運用節に TTL ポリシー用のフィールド名・collection・注意がある
- [ ] 依存 es4 Go ≥ `v0.11.0`。Go `VERSION` = `0.12.0`（npm パッケージ版は後続）

----

## 9. 版割当（PO 決定 1-A）

| 版 | 変更前 | 確定 |
|----|--------|------|
| v0.11.0 | es4 Layer Driver ほか（実装済み・VERSION 0.11.0） | 変更なし |
| **v0.12.0** | Valkey Driver（意図スタブのみ） | **本 TTL 作業** |
| **v0.13.0** | PHP ポート（意図スタブのみ） | **Valkey** |
| **v0.14.0** | （なし） | **PHP** |
| v1.0.0 | Phase 2 締め | 変更なし |

----

## 10. 関連リンク

- 共通前提: uploads `session-ttl-common.md`（2026-10-28）
- 上流 es4 計画: [es4#52](https://github.com/b4moss/es4/pull/52)（`5aa6b41`）・es4 `docs/plans/v0.11.0/`
- 現行 es4 ドライバ計画（Entry JSON B）: [v0.11.0/es4-layer-driver.md](../v0.11.0/es4-layer-driver.md) — 本版で TTL を上流 API 利用へ更新
- Firestore TTL（Google）: [Manage data retention with TTL policies](https://firebase.google.com/docs/firestore/ttl)
- specs（実装後更新）: [driver-firestore](../../specs/driver-firestore/)（運用節）· driver-es4（未作成なら本版で新設）

----

以上
