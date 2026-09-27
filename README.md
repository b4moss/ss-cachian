# ss-cachian: Server Side Cachian

サーバーサイドのキャッシュ戦略を抽象化するライブラリです。

詳細は [docs/index.md](./docs/index.md)。マイルストーンは [docs/roadmap.md](./docs/roadmap.md)。

## 現状（v0.1.0 スキャフォールド）

| パス | 内容 |
| --- | --- |
| `go/sscachian/` | Go 公開モジュール骨格 |
| `.devcontainer/` + `docker/` | Go 1.26 / Firestore Emulator / `act` |
| `.github/workflows/ci.yml` | path filter・ancestor skip・集約ゲート |

```bash
make lint test   # Go
make act         # ローカル CI スモーク（Docker 必要）
```

## 憲章

リモート `charter`（`b4moss/charter` の `docs` ブランチ）から [docs/charter/](./docs/charter/) に直接取り込み。

```bash
git remote add -t docs charter https://github.com/b4moss/charter.git
git fetch charter docs
git checkout charter/docs -- docs/charter
```
