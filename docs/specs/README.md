# specs

現行プロダクト（**v0.10.0**）に存在する振る舞いの仕様正本。  
コード正本: `go/sscachian`（`VERSION=0.10.0`、モジュール `github.com/b4moss/ss-cachian`）および `node/sscachian`（`@b4moss/ss-cachian@0.10.0`）。  
ドメイン分割は [tests](../tests/) と同じ。検証手順は tests、憲章ライフサイクルは [doc-rule](../charter/doc-rule.md)。  
プロダクトの目的・スコープは [README.md](../README.md)（pillar）。

| ドメイン | パス | コード |
| --- | --- | --- |
| cache-type | [cache-type/](./cache-type/) | `cache.go`（Define / CacheType / 便利 API） |
| version | [version/](./version/) | `cache.go`（CurrentVersion / Bump / keys） |
| layer | [layer/](./layer/) | `types.go`（Layer）+ multilayer in `cache.go` |
| purge | [purge/](./purge/) | `Purge` / `PurgeExact` / `PurgePrefix` / `PurgeTag` / `IsVersionDataKey` |
| driver-memory | [driver-memory/](./driver-memory/) | `driver/memory` |
| driver-firestore | [driver-firestore/](./driver-firestore/) | `driver/firestore` |
| config | [config/](./config/) | `registry.go` / `config.go`（Go）、`config.ts`（Node） |

未実装の拡張は [plans](../plans/)。
