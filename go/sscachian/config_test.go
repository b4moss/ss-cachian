package sscachian_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/b4moss/ss-cachian"
	_ "github.com/b4moss/ss-cachian/driver/firestore"
	_ "github.com/b4moss/ss-cachian/driver/memory"
)

func TestRegistry_RegisterAndDefault(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	if err := reg.RegisterLoader("load_x", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "v", nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.RegisterKeyBuilder("custom", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "custom:" + kc.QueryType, nil
	}); err != nil {
		t.Fatal(err)
	}
	// last-wins
	if err := reg.RegisterKeyBuilder("custom", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "custom2:" + kc.QueryType, nil
	}); err != nil {
		t.Fatal(err)
	}

	yaml := `
schema_version: "0.9"
types:
  - name: a
    key_builder: default
    layers:
      - driver: memory
  - name: b
    key_builder: custom
    loader: load_x
    layers:
      - driver: memory
`
	types, err := sscachian.LoadTypesYAML[string](context.Background(), reg, []byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if types["a"] == nil || types["b"] == nil {
		t.Fatal("missing types")
	}
	ctx := context.Background()
	kc := sampleKC()
	v, err := types["b"].GetOrLoad(ctx, kc)
	if err != nil || v != "v" {
		t.Fatalf("loader resolve: v=%q err=%v", v, err)
	}
	key, err := types["b"].BuildKey(ctx, kc, 1)
	if err != nil || key != "custom2:client_list_page1:1" {
		t.Fatalf("keybuilder last-wins: key=%q err=%v", key, err)
	}
}

func TestRegistry_RegisterErrors(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	if err := reg.RegisterLoader("", func(ctx context.Context, kc sscachian.KeyContext) (string, error) { return "", nil }); err == nil {
		t.Fatal("expected empty name error")
	}
	if err := reg.RegisterLoader("x", nil); err == nil {
		t.Fatal("expected nil loader error")
	}
	if err := reg.RegisterKeyBuilder("", sscachian.DefaultKeyBuilder); err == nil {
		t.Fatal("expected empty kb name error")
	}
	if err := reg.RegisterKeyBuilder("x", nil); err == nil {
		t.Fatal("expected nil kb error")
	}
}

func TestLoadTypes_MinimalMemoryYAMLAndJSON(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	yaml := `
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default
    layers:
      - driver: memory
        ttl: 5m
`
	ctx := context.Background()
	fromYAML, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	ct := fromYAML["user_profile"]
	if ct == nil {
		t.Fatal("missing user_profile")
	}
	if ct.LayerTTL(0) != 5*time.Minute {
		t.Fatalf("ttl=%v", ct.LayerTTL(0))
	}
	kc := sampleKC()
	if err := ct.Set(ctx, kc, "hello"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "hello" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}

	js := map[string]any{
		"schema_version": "0.9",
		"types": []any{
			map[string]any{
				"name":        "user_profile",
				"key_builder": "default",
				"layers": []any{
					map[string]any{"driver": "memory", "ttl": "5m"},
				},
			},
		},
	}
	raw, _ := json.Marshal(js)
	fromJSON, err := sscachian.LoadTypesJSON[string](ctx, reg, raw)
	if err != nil {
		t.Fatal(err)
	}
	if fromJSON["user_profile"] == nil || fromJSON["user_profile"].LayerTTL(0) != 5*time.Minute {
		t.Fatal("json mismatch")
	}
	if err := fromJSON["user_profile"].Set(ctx, kc, "j"); err != nil {
		t.Fatal(err)
	}
	got, ok, err = fromJSON["user_profile"].Get(ctx, kc)
	if err != nil || !ok || got != "j" {
		t.Fatalf("json get got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestLoadTypes_MultipleIndependent(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	yaml := `
schema_version: "0.9"
types:
  - name: a
    layers:
      - driver: memory
  - name: b
    layers:
      - driver: memory
`
	types, err := sscachian.LoadTypesYAML[string](context.Background(), reg, []byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	kc := sampleKC()
	_ = types["a"].Set(ctx, kc, "A")
	_ = types["b"].Set(ctx, kc, "B")
	ga, _, _ := types["a"].Get(ctx, kc)
	gb, _, _ := types["b"].Get(ctx, kc)
	if ga != "A" || gb != "B" {
		t.Fatalf("independent: a=%q b=%q", ga, gb)
	}
}

func TestLoadTypes_ConfigErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	reg := sscachian.NewRegistry()

	mustErr := func(name, yaml string) {
		t.Helper()
		if _, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(yaml)); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}

	mustErr("bad schema", `schema_version: "0.8"
types:
  - name: a
    layers: [{driver: memory}]`)
	mustErr("missing schema", `types:
  - name: a
    layers: [{driver: memory}]`)
	mustErr("empty types", `schema_version: "0.9"
types: []`)
	mustErr("empty name", `schema_version: "0.9"
types:
  - name: ""
    layers: [{driver: memory}]`)
	mustErr("dup name", `schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory}]
  - name: a
    layers: [{driver: memory}]`)
	mustErr("empty layers", `schema_version: "0.9"
types:
  - name: a
    layers: []`)
	mustErr("unknown driver", `schema_version: "0.9"
types:
  - name: a
    layers: [{driver: valkey}]`)
	mustErr("unknown kb", `schema_version: "0.9"
types:
  - name: a
    key_builder: missing
    layers: [{driver: memory}]`)
	mustErr("unknown loader", `schema_version: "0.9"
types:
  - name: a
    loader: missing
    layers: [{driver: memory}]`)
	mustErr("bad ttl", `schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory, ttl: abc}]`)
	mustErr("neg ttl", `schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory, ttl: -1s}]`)
	mustErr("ttl not string", `schema_version: "0.9"
types:
  - name: a
    layers: [{driver: memory, ttl: 60}]`)
}

func TestLoadTypes_TTLPrecedence(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	ctx := context.Background()

	onlyLayer, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
        ttl: 1m
`))
	if err != nil {
		t.Fatal(err)
	}
	if onlyLayer["t"].LayerTTL(0) != time.Minute {
		t.Fatal(onlyLayer["t"].LayerTTL(0))
	}

	onlyType, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    ttl: 30m
    layers:
      - driver: memory
      - driver: memory
`))
	if err != nil {
		t.Fatal(err)
	}
	if onlyType["t"].LayerTTL(0) != 30*time.Minute || onlyType["t"].LayerTTL(1) != 30*time.Minute {
		t.Fatalf("%v %v", onlyType["t"].LayerTTL(0), onlyType["t"].LayerTTL(1))
	}

	mixed, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    ttl: 30m
    layers:
      - driver: memory
        ttl: 1m
      - driver: memory
`))
	if err != nil {
		t.Fatal(err)
	}
	if mixed["t"].LayerTTL(0) != time.Minute || mixed["t"].LayerTTL(1) != 30*time.Minute {
		t.Fatalf("%v %v", mixed["t"].LayerTTL(0), mixed["t"].LayerTTL(1))
	}
}

func TestLoadTypes_DriverOptions(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	ctx := context.Background()

	if _, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: firestore
        options:
          emulator_host: localhost:8080
`)); err == nil {
		t.Fatal("expected unknown firestore option error")
	}
	if _, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
        options:
          foo: bar
`)); err == nil {
		t.Fatal("expected unknown memory option error")
	}

	okMem, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
`))
	if err != nil || okMem["t"] == nil {
		t.Fatalf("memory ok: %v", err)
	}
}

func TestLoadTypes_FirestoreCollection(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Fatal("FIRESTORE_EMULATOR_HOST is required")
	}
	reg := sscachian.NewRegistry()
	ctx := context.Background()
	coll := "sscachian_cfg_" + strings.ReplaceAll(t.Name(), "/", "_") + "_" + time.Now().Format("150405.000")
	yaml := `
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: firestore
        options:
          project_id: ss-cachian-dev
          collection: ` + coll + `
`
	types, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	kc := sampleKC()
	if err := types["t"].Set(ctx, kc, "fs"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := types["t"].Get(ctx, kc)
	if err != nil || !ok || got != "fs" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}

	// defaults omit options
	types2, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: firestore
`))
	if err != nil {
		t.Fatal(err)
	}
	if types2["t"] == nil || len(types2["t"].Layers()) != 1 {
		t.Fatal("default firestore")
	}
}

func TestLoadTypes_IntegrationLoaderAndKeyBuilder(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	var loads int32
	_ = reg.RegisterLoader("load_client_list", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		atomic.AddInt32(&loads, 1)
		return "loaded", nil
	})
	_ = reg.RegisterKeyBuilder("report_v2", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "report:" + kc.TenantID + ":" + kc.QueryType, nil
	})

	yaml := `
schema_version: "0.9"
types:
  - name: client_list
    key_builder: default
    loader: load_client_list
    layers:
      - driver: memory
        ttl: 1m
  - name: bare
    layers:
      - driver: memory
  - name: report_cache
    key_builder: report_v2
    layers:
      - driver: memory
`
	ctx := context.Background()
	types, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	kc := sampleKC()

	v, err := types["client_list"].GetOrLoad(ctx, kc)
	if err != nil || v != "loaded" || atomic.LoadInt32(&loads) != 1 {
		t.Fatalf("v=%q loads=%d err=%v", v, loads, err)
	}
	verBefore, _ := types["client_list"].CurrentVersion(ctx, kc)
	_, _ = types["client_list"].GetOrLoad(ctx, kc)
	verAfter, _ := types["client_list"].CurrentVersion(ctx, kc)
	if verBefore != verAfter || atomic.LoadInt32(&loads) != 1 {
		t.Fatalf("bump/load: before=%d after=%d loads=%d", verBefore, verAfter, loads)
	}

	_, err = types["bare"].GetOrLoad(ctx, kc)
	if !errors.Is(err, sscachian.ErrNoLoader) {
		t.Fatalf("want ErrNoLoader got %v", err)
	}

	key, err := types["report_cache"].BuildKey(ctx, kc, 1)
	if err != nil || key != "report:0123456:client_list_page1:1" {
		t.Fatalf("custom kb %q err=%v", key, err)
	}

	// snapshot: changing registry after load must not affect existing type
	_ = reg.RegisterLoader("load_client_list", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "other", nil
	})
	ct2, _ := sscachian.Define[string]("x").WithLayers(types["bare"].Layers()...).Build()
	_ = ct2 // silence
	// force miss on a fresh type that still uses old loader via client_list
	freshKC := sscachian.KeyContext{AppSlug: "my-app", TenantID: "snap", QueryType: "q"}
	v2, err := types["client_list"].GetOrLoad(ctx, freshKC)
	if err != nil || v2 != "loaded" {
		t.Fatalf("snapshot loader: v=%q err=%v", v2, err)
	}
}

func TestLoadTypes_MultilayerWriteBack(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Fatal("FIRESTORE_EMULATOR_HOST is required")
	}
	reg := sscachian.NewRegistry()
	ctx := context.Background()
	coll := "sscachian_wb_" + time.Now().Format("150405.000000")
	types, err := sscachian.LoadTypesYAML[string](ctx, reg, []byte(`
schema_version: "0.9"
types:
  - name: client_list
    layers:
      - driver: memory
        ttl: 1m
      - driver: firestore
        ttl: 1h
        options:
          project_id: ss-cachian-dev
          collection: `+coll+`
`))
	if err != nil {
		t.Fatal(err)
	}
	ct := types["client_list"]
	kc := sampleKC()
	if err := ct.Set(ctx, kc, "wb"); err != nil {
		t.Fatal(err)
	}
	// clear L1 only
	l1 := ct.Layers()[0]
	key, err := ct.BuildLatestKey(ctx, kc)
	if err != nil {
		t.Fatal(err)
	}
	if err := l1.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "wb" {
		t.Fatalf("write-back get got=%q ok=%v err=%v", got, ok, err)
	}
	// L1 should be filled again
	e, hit, err := l1.Get(ctx, key)
	if err != nil || !hit || e.Value != "wb" {
		t.Fatalf("l1 fill hit=%v val=%v err=%v", hit, e.Value, err)
	}
}

func TestLoadTypes_FromFile(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	_ = reg.RegisterLoader("load_client_list", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "x", nil
	})
	_ = reg.RegisterKeyBuilder("report_v2", func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "r", nil
	})
	// Use a trimmed fixture without firestore so file load works offline.
	dir := t.TempDir()
	path := filepath.Join(dir, "sscachian.yaml")
	content := []byte(`
schema_version: "0.9"
types:
  - name: user_profile
    key_builder: default
    layers:
      - driver: memory
        ttl: 5m
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	types, err := sscachian.LoadTypes[string](context.Background(), reg, path)
	if err != nil {
		t.Fatal(err)
	}
	if types["user_profile"] == nil {
		t.Fatal("missing")
	}
}

func TestLoadTypes_InvalidContextSameAsDefine(t *testing.T) {
	t.Parallel()
	reg := sscachian.NewRegistry()
	types, err := sscachian.LoadTypesYAML[string](context.Background(), reg, []byte(`
schema_version: "0.9"
types:
  - name: t
    layers:
      - driver: memory
`))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = types["t"].Get(context.Background(), sscachian.KeyContext{})
	if !errors.Is(err, sscachian.ErrInvalidContext) {
		t.Fatalf("got %v", err)
	}
}
