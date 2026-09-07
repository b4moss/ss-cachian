---
type: OpenQuestions
title: 未決事項
description: 未決定または未詳細の設計論点。
tags: [open]
timestamp: 2026-09-07T12:00:00Z
---

# 未決事項

決定済みの振る舞いは [振る舞い](./behavior.md) を参照。ここには未決定・未詳細のみを残す。

## current-version の具体化

初期 Get は最新 version のみ返す、は決定済み。

未詳細:

- current-version を 1 キーで持つ場合のキー規約
- そのキーをどの Layer に置くか
- アプリが version を渡す方式との優先関係

## エントリメタの拡張

初期メタは `created_at` / `expires_at`。

未詳細:

- SWR / SIE / negative cache 用フィールドの追加タイミングと形

## マルチランタイム

セマンティクスの移植前提は決定済み。

未詳細:

- シリアライズ形式
- ランタイム間で揃える保証の範囲

## その他（後回し）

- 旧 version の明示列挙・取得 API の要否詳細
- 分散環境向け L1 無効化通知の要否
