# CI / CD policy (ss-cachian)

Aligned with `b4moss/crudian` and [docs/README.md](../docs/README.md)（技術方針）。  
憲章は [docs/charter/](../docs/charter/)。上書きは [override-charter.md](../docs/override-charter.md)。

## テスト方針（要約）

- 全関数または全メソッドに対してテストを書く（**単体結合テスト**）。各テストに正常系と異常系。
- テスト仕様は `docs/tests/`（[索引](../docs/tests/)）。ドメイン分割は `docs/specs/` と同じ。
- 各ファイルの目安: 正常系おおよそ 3、異常系おおよそ 3〜5。
- 仕様 FIX のあと、テスト仕様 → テスト → 実装（Red → Green → Refactor）。E2E は必須としない。
- 詳細形式は [憲章の TDD 方針](../docs/charter/tdd.md)。

## Goals（CI）

- Lint + unit/integration on PRs (and push) for touched Go / Node packages.
- Path filter: only schedule jobs for changed areas (`go/**`, `node/**`).
- Ancestor skip (Go): same package + workflow identity already green → short-circuit.
- Docs-only changes: language jobs skipped; aggregate gate still succeeds.
- No product E2E in CI.
- `develop` / `dev-*` への PR では CI を必須とする。
- 可能な限りジョブを並行実行し、キャッシュを活用する。
- PR 前に実装者が手元で `act` を回す（husky 等での強制はしない）。`act` は devcontainer に同梱。

### 参考モデル（crudian）

- path filter で変更パッケージだけをスケジュールする。
- 必須ステータスは集約ゲート（例: `CI result`）を想定する。
- `act` はワークフロー配線のスモーク。ゲート判定ロジックの正本はスクリプトテスト側とする。

## CD

1. **タグは `main` で打つ。**
2. **そのタグと同じ内容で GitHub Release を作る。**
3. **そのタグが `release` ブランチに乗るとリリースする。**（言語横断の正規 CD。整備中）
4. 各言語でバージョンアップがあったものだけリリースする。
5. 言語によっては SemVer が欠番になる場合がある。これは許容する。

### npm（Trusted Publisher）

このリポジトリの `@b4moss/ss-cachian` は **npm Trusted Publisher（OIDC）** で公開する。

- トリガー: `main` への `v*` タグ、または Actions → **npm publish** → `Run workflow`
- ワークフロー: [npm-publish.yml](./workflows/npm-publish.yml)（`id-token: write`、**`NODE_AUTH_TOKEN` は使わない**）
- Org の `NPM_TOKEN` は他リポの **初回 publish 用**に残してよい。本リポの publish ジョブからは参照しない。

初回パッケージ作成時のみ Trusted Publisher が使えないため、そのときだけ一時 GAT + Secret で bootstrap する（実施済み: 0.7.0）。

## Required status

Use the aggregate check **CI result** as the required status when branch protection is enabled.

## Local smoke

```bash
make act
```

`act` is wiring smoke only; gate logic correctness is covered later by `.github/tests/` when added.

## バッジ

- 開発が進むと README 等にバッジを付与する。
- 想定例: CI、Codecov、OpenSSF Scorecard、Release、License。
- バッジ構成は `b4moss/crudian` を参考にする。
