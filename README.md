# ss-cachian: Server Side Cachian

サーバーサイドのキャッシュ戦略を抽象化するライブラリです。

詳細は [docs/index.md](./docs/index.md)。現行仕様は [docs/specs/](./docs/specs/)。マイルストーンは [docs/roadmap.md](./docs/roadmap.md)。

## 現状（v0.7.0 / Phase 1 PoC 完了）

| パス | 内容 |
| --- | --- |
| `go/sscachian/` | CacheType（多層・Version・Exact Purge） |
| `go/sscachian/driver/memory` | インメモリ Layer |
| `go/sscachian/driver/firestore` | Firestore Layer（Emulator 結合） |
| `.devcontainer/` + `docker/` | Go 1.26 / Firestore Emulator / `act` |
| `.github/workflows/ci.yml` | path filter・ancestor skip・Emulator 付き test |

```bash
make lint test   # Go（Firestore Emulator 起動を含む）
make act         # ローカル CI スモーク（Docker 必要）
```

## 憲章

リモート `charter`（`b4moss/charter` の `docs` ブランチ）から [docs/charter/](./docs/charter/) に直接取り込み。

```bash
git remote add -t docs charter https://github.com/b4moss/charter.git
git fetch charter docs
git checkout charter/docs -- docs/charter
```
