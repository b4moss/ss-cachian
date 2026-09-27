# ss-cachian: Server Side Cachian

サーバーサイドのキャッシュ戦略を抽象化するライブラリです。

詳細は [docs/README.md](./docs/README.md)（pillar）。OKF 索引は [docs/index.md](./docs/index.md)。現行仕様は [docs/specs/](./docs/specs/)。マイルストーンは [docs/roadmap.md](./docs/roadmap.md)。

## 現状（v0.7.0 / Phase 1 PoC 完了 · Node スキャフォールド）

| パス | 内容 |
| --- | --- |
| `go/sscachian/` | CacheType（多層・Version·Exact Purge） |
| `go/sscachian/driver/memory` | インメモリ Layer |
| `go/sscachian/driver/firestore` | Firestore Layer（Emulator 結合） |
| `node/sscachian/` | `@b4moss/ss-cachian`（npm 公開経路・実装は後続） |
| `.devcontainer/` + `docker/` | Go 1.26 / Firestore Emulator / `act` |
| `.github/workflows/ci.yml` | path filter・ancestor skip·Emulator 付き test |
| `.github/workflows/npm-publish.yml` | タグ `v*` / dispatch → `npm publish`（Trusted Publisher / OIDC） |

```bash
make lint test   # Go（Firestore Emulator 起動を含む）
make lint-node test-node
make act         # ローカル CI スモーク（Docker 必要）
```

## 憲章

リモート `charter`（`b4moss/charter` の **`main`**）から [docs/charter/](./docs/charter/) に取り込む（OKF v0.1）。

```bash
git remote add charter https://github.com/b4moss/charter.git
git fetch charter main
git checkout charter/main -- docs/charter
```
