# specs

現行プロダクト（**v0.7.0 / Phase 1 PoC 完了**）に存在する振る舞いの仕様正本。  
ドメイン分割は [tests](../tests/) と同じ。検証手順は tests、憲章ライフサイクルは [doc-rule](../charter/doc-rule.md)。

| ドメイン | パス |
| --- | --- |
| cache-type | [cache-type/](./cache-type/) |
| version | [version/](./version/) |
| layer | [layer/](./layer/) |
| purge | [purge/](./purge/) |
| driver-memory | [driver-memory/](./driver-memory/) |
| driver-firestore | [driver-firestore/](./driver-firestore/) |

横断の決定事項ハブ: [behavior](../behavior.md) / [api](../api.md) / [drivers](../drivers.md)。  
コード正本: `go/sscachian`（`VERSION=0.7.0`）。
