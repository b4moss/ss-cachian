---
type: Concept
title: コンセプト
description: ss-cachian の目的と中心概念。
tags: [concept, decided]
timestamp: 2026-09-07T12:00:00Z
---

# コンセプト

## 目的

ss-cachian は、単一キャッシュストアの薄い抽象ではない。

アプリケーションが **Cache Type** としてキャッシュ戦略を宣言し、ライブラリがその戦略を複数ストレージ（Layer）にまたがって実行する。

狙いは、DB を本質的なアクセスに限定して守ることである。キャッシュは正本ではなく、読み取り高速化のための一時層とする。

## 中心概念

| 概念 | 意味 |
| --- | --- |
| Cache Type | キャッシュ対象の種類ごとの戦略定義単位 |
| Key Builder | ビジネス文脈を含むキー生成 |
| Layer / Driver | 保存先。階層数は固定しない |
| Policy | TTL など。Layer 単位で保持可能 |
| Loader | Cache Miss 時のデータ取得（Read-through） |
| Invalidation | 第1級は Version。Purge API で明示削除も提供 |
| Entry メタ | 初期は `created_at` / `expires_at` |

## 利用上の前提

- Read-through（Loader 付き）と Cache-aside（明示 Get/Set）の両方を許容する。
- キャッシュ可否は更新頻度ではなく、許容できる staleness で判断する。
