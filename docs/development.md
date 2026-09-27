---
type: Engineering
title: 開発・CI/CD
description: 開発環境、パッケージ配置、単体結合テスト、CI/CD、バッジ方針の決定事項。
tags: [engineering, decided]
timestamp: 2026-09-27T03:20:00Z
---

# 開発・CI/CD

規範は [憲章（charter）](./charter/) と `b4moss/crudian` を参考にする。  
憲章本体は編集せず、リモート `charter`（`b4moss/charter` の `docs` ブランチ）から [docs/charter/](./charter/) に直接取り込む。  
プロジェクト固有の上書きは [override-charter.md](./override-charter.md) に書く。

更新手順:

```bash
git remote add -t docs charter https://github.com/b4moss/charter.git  # 未追加時のみ
git fetch charter docs
git checkout charter/docs -- docs/charter
```

当リポジトリ固有の決定のみを以下に記す。

## リポジトリ配置（マルチランタイム）

他言語ポートを前提に、言語ごとにトップレベルディレクトリを並べる。

```text
go/sscachian/           # Go 公開モジュール（アプリが import）
go/sscachian/driver/    # Driver 実装（memory, firestore, …）
node/sscachian/         # Node.js / TypeScript 公開パッケージ（@b4moss/ss-cachian）
# 後続: php/ … など
docs/                   # 知識バンドル正本
```

- 先行実装は **Go 1.26**（Phase 1 PoC 完了 / v0.7.0）。
- Go 公開面は `go/sscachian`。Driver は `go/sscachian/driver/...`。
- Node 公開面は `node/sscachian`（npm: `@b4moss/ss-cachian`）。TS 実装は後続。

## 開発環境（devcontainer）

- 開発は **devcontainer** で行う。
- PoC 時点のコンテナ内容:
  - **Go 1.26**
  - **Firestore Emulator**（メモリ／Firestore の結合テストをコンテナ内で完結）
  - **`act`**（PR 前の Actions スモークを手元で回す）

## テスト

### 単体結合テスト

- 全関数または全メソッドに対してテストを書く。
- これを **単体結合テスト** と呼ぶ。
- 各テストに **正常系** と **異常系** を用意する。
- 具体的な書き方・テスト仕様書の形式は [憲章の TDD 方針](./charter/tdd.md) に従う。

### テスト仕様

- 配置は `docs/tests/`。**ドメイン別**に分割する。
- 想定ドメイン: `version` / `cache-type` / `layer` / `purge` / `driver-memory` / `driver-firestore`（[specs](./specs/) と同分割）
- 各ファイルの目安: 正常系おおよそ 3、異常系おおよそ 3〜5
- 索引は [docs/tests/](./tests/) を参照。

### 補足（charter 踏襲）

- 仕様 FIX のあと、テスト仕様を書き、先にテストを書いてから実装する（Red → Green → Refactor）。
- E2E は必須としない。過剰なカバレッジ追及はしない。

## CI

- `develop` / `dev-*` ブランチへの PR では CI を必須とする。
- 一度 CI に通った同一内容は、上位ブランチへのマージで再実行せずパス扱いにする（ancestor skip。実装は crudian を参考）。
- 可能な限りジョブを並行実行し、キャッシュを活用して壁時計時間を短くする。
- ドキュメントのみの PR / マージでは CI を走らせない（対象ジョブはスキップし、ゲートは成功としてよい）。
- PR 前に実装者が手元で `act` を回し、通ることを確認する。husky 等での強制はしない。
- `act` は devcontainer に同梱する（上記「開発環境」）。

### CI の参考モデル（crudian）

- path filter で変更パッケージだけをスケジュールする。
- 必須ステータスは集約ゲート（例: `CI result`）を想定する。
- `act` はワークフロー配線のスモーク。ゲート判定ロジックの正本はスクリプトテスト側とする。

## CD

1. **タグは `main` で打つ。**
2. **そのタグと同じ内容で GitHub Release を作る。**
3. **そのタグが `release` ブランチに乗るとリリースする。**（言語横断の正規 CD。整備中）
4. 各言語でバージョンアップがあったものだけリリースする。
5. 言語によっては SemVer が欠番になる場合がある。これは許容する。

### npm（暫定）

正規の `release` ブランチ CD が揃うまでの暫定経路:

1. リポジトリ Secret `NPM_TOKEN` に npm **Granular Access Token**（`@b4moss` 向け publish 権限）を置く。
2. `main` に `v*` タグを push すると [npm-publish.yml](../.github/workflows/npm-publish.yml) が走る。
3. タグから SemVer を取り（`v0.8.0` → `0.8.0`）、`node/sscachian` で `npm publish --access public` する。
4. パッケージ名: **`@b4moss/ss-cachian`**。

手元確認:

```bash
cd node/sscachian && npm ci && npm test && npm publish --dry-run
```

## バッジ

- 開発が進むと README 等にバッジを付与する。
- そのため Actions をあらかじめ整備する。
- 想定例: CI、Codecov、OpenSSF Scorecard、Release、License。
- バッジ構成は `b4moss/crudian` を参考にする。
