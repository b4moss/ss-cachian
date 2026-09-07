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

---

以上
