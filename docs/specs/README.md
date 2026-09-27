# specs

現行プロダクト（**v0.7.0 / Phase 1 PoC 完了**）に存在する振る舞いの仕様正本。  
ドメイン分割は [tests](../tests/) と同じ。検証手順は tests、憲章ライフサイクルは [doc-rule](../charter/doc-rule.md)。  
プロダクトの目的・スコープは [README.md](../README.md)（pillar）。

| ドメイン | パス |
| --- | --- |
| cache-type | [cache-type/](./cache-type/) |
| version | [version/](./version/) |
| layer | [layer/](./layer/) |
| purge | [purge/](./purge/) |
| driver-memory | [driver-memory/](./driver-memory/) |
| driver-firestore | [driver-firestore/](./driver-firestore/) |

コード正本: `go/sscachian`（`VERSION=0.7.0`）。未実装の拡張は [plans](../plans/)。
