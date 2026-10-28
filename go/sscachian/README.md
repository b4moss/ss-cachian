# ss-cachian (Go)

Go module: `github.com/b4moss/ss-cachian/go/sscachian` · **VERSION `0.12.0`**

Shared product overview, config schema, and pitfalls: [root README](../../README.md).  
Specs: [docs/specs](../../docs/specs/).

## Install

This module lives under `go/sscachian/` (monorepo). Publish with the **subdirectory tag**
`go/sscachian/vX.Y.Z` (not a root `v*` tag alone — that zip omits nested modules).

```bash
go get github.com/b4moss/ss-cachian/go/sscachian@v0.12.0
```

Requires **Go 1.26+**.

```go
import (
  "github.com/b4moss/ss-cachian/go/sscachian"
  "github.com/b4moss/ss-cachian/go/sscachian/driver/memory"
)
```

## Quick start (code-built)

```go
package main

import (
  "context"

  "github.com/b4moss/ss-cachian/go/sscachian"
  "github.com/b4moss/ss-cachian/go/sscachian/driver/memory"
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

Firestore layer: `github.com/b4moss/ss-cachian/go/sscachian/driver/firestore` (`New(ctx, Options{...})`). Honors `FIRESTORE_EMULATOR_HOST`.

## Config-driven (`LoadTypes`)

Blank-import drivers so factories register, then load YAML/JSON:

```go
import (
  "context"

  "github.com/b4moss/ss-cachian/go/sscachian"
  _ "github.com/b4moss/ss-cachian/go/sscachian/driver/firestore"
  _ "github.com/b4moss/ss-cachian/go/sscachian/driver/memory"
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
go/sscachian/                 # module root (path …/go/sscachian)
go/sscachian/driver/memory/
go/sscachian/driver/firestore/
go/sscachian/driver/es4/
```

Release tags: `go/sscachian/v0.12.0` (Go). npm remains `@b4moss/ss-cachian@0.11.1` until Node parity.

## Develop

From repo root:

```bash
make lint test              # starts Firestore Emulator when needed
make verify-go-consumer     # module path + replace-based consumer build
```

Package tests live next to the code (`*_test.go`).
