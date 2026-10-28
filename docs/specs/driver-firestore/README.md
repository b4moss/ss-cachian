---
type: Spec
title: driver-firestore
description: Firestore Layer（v0.12.0・Emulator 対応・TTL ポリシー向け expires_at）。
tags: [specs, driver-firestore, v0.12.0]
timestamp: 2026-10-28T16:20:00+09:00
---

# driver-firestore

対象: `go/sscachian/driver/firestore`（`Store` / `New` / `Options` / `Close`）  
関連: [layer](../layer/) / [tests/driver-firestore](../../tests/driver-firestore/) / [plans/v0.12.0/plan.md](../../plans/v0.12.0/plan.md)

## 構築

- `New(ctx, Options{ProjectID, Collection})`。`FIRESTORE_EMULATOR_HOST` 設定時に Emulator へ接続（CI / `make test` で起動）。
- 既定: `ProjectID = "ss-cachian-dev"`、`Collection = "sscachian"`。
- `Close()` でクライアントを閉じる。

## ドキュメント

- 1 キャッシュキー = 1 ドキュメント。**ドキュメント ID はキー文字列をそのまま使う**（URL エンコードやサニタイズは現行では行わない）。
- フィールド: `value`（JSON バイト）、`created_at`、**期限があるときだけ** `expires_at`（日時型 / Timestamp）。
- Get 時に JSON を `any` へ復元。`expires_at` 欠落 = 期限なし。過去日時 = **論理 Miss**（ドキュメント削除は必須でない）。
- 期限なし Set は **`expires_at` を書かない**（ゼロ値禁止）。期限あり→なし上書きはドキュメント置換でフィールドを消す。
- `Incr` もゼロ値を書かず、既存 `expires_at` があれば削除して無期限化する。
- nil value の Set は JSON `null` として保存可。JSON 化できない値はエラー。

## 操作

- `Incr` はトランザクション。未作成は 0+1→1。競合（Aborted）は最大 8 回クライアント側で再試行。
- 整数以外 → `ErrNotInteger`。`MaxInt64` → `ErrIncrOverflow`。
- `PurgeExact` はコレクション全走査し、`IsVersionDataKey` に合う ID だけ削除。
- `PurgePrefix` はコレクション全走査し、ドキュメント ID が prefix で始まるものをすべて削除。
- 空キー / 空 prefix → `ErrEmptyKey`。負 TTL → `ErrNegativeTTL`。
- `ClearCollection` はテスト専用。

## 運用（Firestore TTL ポリシー）

利用側（例: jparse Terraform）が TTL ポリシーを作成する。本ライブラリはポリシー作成 API を持たない。

### 共通

| 項目 | 値 |
|------|-----|
| TTL フィールド | **`expires_at`**（日時型） |
| 期限なし | フィールド欠落（ゼロ値・`0001-01-01` / Unix epoch は書かない） |
| 削除タイミング | 期限後すぐではない（通常最大約 24h）。読み取り側でも期限切れを Miss する |

### firestore ドライバ直接利用

| 項目 | 値 |
|------|-----|
| Collection（既定） | `sscachian`（`Options.Collection` で変更可） |
| ポリシー単位 | collection group = 上記 collection ID |

```bash
gcloud firestore fields ttls update expires_at \
  --collection-group=sscachian \
  --enable-ttl
```

### es4 ドライバ経由

| 項目 | 値 |
|------|-----|
| Collection | es4 Options の `state_firestore_collection`（利用側設定） |
| TTL フィールド | es4 が書く **`expires_at`**（日時型） |

```bash
gcloud firestore fields ttls update expires_at \
  --collection-group=<state_firestore_collection> \
  --enable-ttl
```

### 移行注意

- 既存ドキュメントにゼロ値 `expires_at` が残っていると、ポリシー有効化後に削除対象になる。バックフィルでフィールド削除するか、期限なしで再書き込みする。
- Emulator では TTL ポリシー挙動を本番相当に再現できない。単体はフィールドの有無と読み取り Miss で担保する。

他言語間の共通シリアライズ形式は未決定（[plans/unscheduled](../../plans/unscheduled/)）。

----

以上
