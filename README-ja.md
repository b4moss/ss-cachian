# ss-cachian

[![CI](https://github.com/b4moss/ss-cachian/actions/workflows/ci.yml/badge.svg)](https://github.com/b4moss/ss-cachian/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/@b4moss/ss-cachian)](https://www.npmjs.com/package/@b4moss/ss-cachian)
[![Release](https://img.shields.io/github/v/release/b4moss/ss-cachian)](https://github.com/b4moss/ss-cachian/releases)
[![License](https://img.shields.io/github/license/b4moss/ss-cachian)](https://github.com/b4moss/ss-cachian/blob/main/LICENSE)

サーバーサイドの **キャッシュ戦略** ライブラリです。アプリケーションが **Cache Type** を宣言し、ライブラリが複数の保存先（**Layer**）にまたがって実行します。

単一ストアの薄い抽象ではありません。キー生成・TTL・多層書き戻し・Version 無効化・Exact Purge といった「どうキャッシュするか」を型として持ちます。

- [English README (default)](./README.md)

現行版: **v0.10.0**（Go `VERSION` と `@b4moss/ss-cachian` を揃えています）。

## できること

| 機能 | 内容 |
|------|------|
| **Cache Type API** | `Define` / `define` → Layer・KeyBuilder・TTL・任意 Loader → Get / Set / Delete / GetOrLoad / Purge |
| **Version キー** | current-version は L1。Mutation で Bump。`GetOrLoad` の充填は Bump しない |
| **多層** | L1→L2→…。下位 hit / Loader 成功時は上位へ書き戻し |
| **Exact Purge** | 論理プレフィックスの version データキーを削除。`__version__` は残す |
| **Driver** | **memory** / **Firestore**（`FIRESTORE_EMULATOR_HOST` 対応） |
| **設定駆動（v0.9.0）** | YAML（正）または同等 JSON → `LoadTypes` / `loadTypes` |

現行スコープ外: SWR / SIE / negative cache、Valkey、PHP、ブラウザ向け。→ [roadmap](./docs/roadmap.md)

## 言語ポート

| Port | Package | Language hub |
|------|---------|--------------|
| **Go** | `github.com/b4moss/ss-cachian` | [`go/sscachian/README.md`](./go/sscachian/README.md) |
| **TypeScript** | `@b4moss/ss-cachian` | [`node/sscachian/README.md`](./node/sscachian/README.md) |

ルート README（英語）が共有ホスト手順の正本です。インストール・import・API 詳細は各 language hub を見てください。

## 前提

| | Go | Node |
|--|----|------|
| Runtime | Go **1.26+** | Node.js **≥ 20** |
| Package | `github.com/b4moss/ss-cachian` | `@b4moss/ss-cachian@0.9.0` |

公開パッケージ利用にリポジトリ clone は不要です。

## 使い方（要約）

1. ポートを選びインストール（`go get …@v0.9.0` / `npm i @b4moss/ss-cachian`）
2. コード組み立て（`Define`）か設定ファイル（YAML）かを決める
3. Loader / カスタム KeyBuilder は Registry に名前登録（設定は参照のみ）
4. Layer 順 = L1…Ln。TTL は `ParseDuration` 文字列。優先: layer TTL → type TTL → `0`
5. Firestore の Emulator は **環境変数のみ**（設定に書かない）
6. Get / Set / Delete / GetOrLoad / Purge を呼ぶ

詳細・フィールド表・落とし穴: **[English README — Config file](./README.md#config-file-yaml--json)**  
雛形: [`docs/plans/v0.9.0/sscachian.example.yaml`](./docs/plans/v0.9.0/sscachian.example.yaml)  
Node 向け長文: [`node/sscachian/README.md`](./node/sscachian/README.md#config-driven-v090)

## 開発

```bash
make lint test
make lint-node test-node
```

## ドキュメント索引

| Doc | 内容 |
|-----|------|
| [`README.md`](./README.md) | 英語ルート（正本） |
| [`docs/README.md`](./docs/README.md) | pillar |
| [`docs/specs/`](./docs/specs/) | 現行仕様 |
| [`docs/roadmap.md`](./docs/roadmap.md) | マイルストーン |
| [`go/sscachian/README.md`](./go/sscachian/README.md) | Go hub |
| [`node/sscachian/README.md`](./node/sscachian/README.md) | Node hub |

## License

MIT © Bicycle for Mind LLC. — [`LICENSE`](./LICENSE)
