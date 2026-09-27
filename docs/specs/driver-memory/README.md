---
type: Spec
title: driver-memory
description: インメモリ Layer（現行 v0.7.0）。
tags: [specs, driver-memory]
timestamp: 2026-09-27T03:05:00Z
---

# driver-memory

対象: `go/sscachian/driver/memory`  
関連: [drivers](../../drivers.md) / [tests/driver-memory](../../tests/driver-memory/)

- プロセスローカル。`sync.Mutex` + `map` でゴルーチン安全。
- Entry: `{ value any, created_at, expires_at }`。TTL は Get 時に `expires_at` を見て遅延削除。
- `Get` / `Set` / `Delete`（冪等）/ `Incr`（未作成は 0+1→1）/ `PurgeExact`。
- 空キーは `ErrEmptyKey`。負 TTL は `ErrNegativeTTL`。
