---
type: Spec
title: driver-es4
description: es4 Layer Driver（埋め込み open・Entry TTL ラップ・HTTP アダプタ）。v0.11.0。
tags: [specs, driver-es4, v0.11.0]
timestamp: 2026-10-12T05:20:00Z
---

# driver-es4

現行プロダクトに存在する **es4** Layer Driver の仕様正本。  
コード: `go/sscachian/driver/es4` · `node/sscachian/src/driver/es4.ts`。  
検証手順: [tests/driver-es4](../../tests/driver-es4/)。完了注記: [plans/v0.11.0/es4-layer-driver.md](../../plans/v0.11.0/es4-layer-driver.md)。

## 概要

ss-cachian の Layer 契約を、埋め込み `es4.Open`（または HTTP Es4 Server）で満たす。Driver が DB の **Open / Close を所有**する。依存: es4 **v0.10.0**（キー区切り `:` ＋ `Scan(prefix)`）。

## 接続形態

| 形態 | 起動 | config `options` |
|------|------|------------------|
| 埋め込み | `es4.Open` / Node `open` | `memory_only` または `state_backend` / `state_path` / redis・firestore 系 |
| HTTP | Es4 Server へ REST | `base_url` のみ（他キー拒否） |

未指定時の埋め込み既定は **memory_only**（CI 向け）。

## Entry JSON（決定 B）

es4 値として次を保存する（`created_at` なし）:

```json
{ "value": <payload>, "expires_at": "<RFC3339>" }
```

- `expires_at` **省略**＝無期限
- Get 時に過去 `expires_at` → 論理 Miss（削除してよい）
- 壊れた Entry JSON → Miss（パニックしない）

## Layer 契約

| 操作 | 振る舞い |
|------|----------|
| `Get` / `Set` / `Delete` | Entry JSON 経由。空キー `ErrEmptyKey`、負 TTL `ErrNegativeTTL` |
| `Incr` | es4 Tx 上の read-modify-write。再試行は Firestore Driver に揃える |
| `PurgeExact` | **Scan なし**。キー既知で version data のみ削除（`IsVersionDataKey`） |
| `PurgePrefix` | es4 `Scan(prefix)` → 各キー `Delete` |
| `Close` | 所有 DB を閉じる。冪等 |

## Backend

第一版は **Firestore まで全 Backend 公式サポート**（CI 既定は memory_only）。Valkey 固有委譲は対象外（v0.12）。

## 関連

- Layer: [specs/layer](../layer/)
- Config: [specs/config](../config/)
- 上流 es4: [b4moss/es4 specs/state](https://github.com/b4moss/es4/blob/main/docs/specs/state/state.md)
