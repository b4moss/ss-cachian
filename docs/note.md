---
type: Note
title: note
role: note
---
# note

POの簡単なメモです

- [コンセプト](./concept.md)
- [キービルダー](./key-builder.md)

## 開発方針

- `b4moss/crudian`や`b4moss/cachian`のように、マルチランタイムへのポートを前提とする
- ただし、今回は先行実装を Go から始める

### driver開発順序

- インメモリ
- Firestore
- Redis / Valkey
- MongoDB

## 開発フェーズ

### Phase 1 PoC

v0.7.0ぐらいか

- Goで利用可能
- キービルダーができる
- ドライバーと複数ストレージが選べる
- 以下のドライバーが利用できる
  - インメモリ
  - Firestore

### Phase 2

Until v1.0.0

- 他のドライバー
- 他の言語へのポート
  - Node.js
  - PHP
---

以上
