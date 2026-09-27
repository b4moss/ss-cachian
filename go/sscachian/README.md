# ss-cachian (Go)

Go module: `github.com/b4moss/ss-cachian` · **VERSION `0.10.0`**

Shared product overview, config schema, and pitfalls: [root README](../../README.md).  
Specs: [docs/specs](../../docs/specs/).

## Install

```bash
go get github.com/b4moss/ss-cachian@v0.10.0
```

Requires **Go 1.26+**.

## Quick start (code-built)

```go
package main

import (
  "context"

  "github.com/b4moss/ss-cachian"
  "github.com/b4moss/ss-cachian/driver/memory"
)

func main() {
  ctx := context.Background()
  ct, err := sscachian.Define[string]("users").
    WithLayers(memory.New()).
    WithKeyBuilder(sscachian.DefaultKeyBuilder).
    WithLayerTTL(0).
    Build()
  if err != nil {
    panic(err)
  }
  kc := sscachian.KeyContext{AppSlug: "app", TenantID: "t1", QueryType: "user"}
  _ = ct.Set(ctx, kc, "alice")
  v, ok, _ := ct.Get(ctx, kc)
  _, _ = v, ok
}
```

Firestore layer: `github.com/b4moss/ss-cachian/driver/firestore` (`New(ctx, Options{...})`). Honors `FIRESTORE_EMULATOR_HOST`.

## Config-driven (`LoadTypes`)

Blank-import drivers so factories register, then load YAML/JSON:

```go
import (
  "context"

  "github.com/b4moss/ss-cachian"
  _ "github.com/b4moss/ss-cachian/driver/firestore"
  _ "github.com/b4moss/ss-cachian/driver/memory"
)

reg := sscachian.NewRegistry()
_ = reg.RegisterLoader("load_client_list", func(ctx context.Context, kc sscachian.KeyContext) (MyT, error) {
  return MyT{}, nil
})
types, err := sscachian.LoadTypes[MyT](ctx, reg, "sscachian.yaml")
cache := types["client_list"]
```

Helpers: `LoadTypesYAML`, `LoadTypesJSON`, `LoadTypesReader`.  
Schema and field tables: [root README — Config file](../../README.md#config-file-yaml--json) · [example YAML](../../docs/plans/v0.9.0/sscachian.example.yaml).

## Layout

```text
go/sscachian/                 # module root
go/sscachian/driver/memory/
go/sscachian/driver/firestore/
```

## Develop

From repo root:

```bash
make lint test   # starts Firestore Emulator when needed
```

Package tests live next to the code (`*_test.go`).
