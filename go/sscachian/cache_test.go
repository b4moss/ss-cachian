package sscachian_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/b4moss/ss-cachian"
	"github.com/b4moss/ss-cachian/driver/memory"
)

func sampleKC() sscachian.KeyContext {
	return sscachian.KeyContext{AppSlug: "my-app", TenantID: "0123456", QueryType: "client_list_page1"}
}

func newStringCache(t *testing.T, opts ...func(*sscachian.Builder[string])) *sscachian.CacheType[string] {
	t.Helper()
	b := sscachian.Define[string]("test").WithLayers(memory.New()).WithKeyBuilder(sscachian.DefaultKeyBuilder)
	for _, o := range opts {
		o(b)
	}
	ct, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return ct
}

func TestBuild_OK(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	if ct.Name() != "test" {
		t.Fatalf("name=%s", ct.Name())
	}
}

func TestBuild_WithTTLAndLoader(t *testing.T) {
	t.Parallel()
	var loads int32
	ct := newStringCache(t,
		func(b *sscachian.Builder[string]) { b.WithLayerTTL(time.Hour) },
		func(b *sscachian.Builder[string]) {
			b.WithLoader(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
				atomic.AddInt32(&loads, 1)
				return "loaded", nil
			})
		},
	)
	ctx := context.Background()
	kc := sampleKC()
	v, err := ct.GetOrLoad(ctx, kc)
	if err != nil || v != "loaded" || atomic.LoadInt32(&loads) != 1 {
		t.Fatalf("v=%q loads=%d err=%v", v, loads, err)
	}
	if err := ct.Set(ctx, kc, "x"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "x" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestBuild_Errors(t *testing.T) {
	t.Parallel()
	if _, err := sscachian.Define[string]("t").WithKeyBuilder(sscachian.DefaultKeyBuilder).Build(); err == nil {
		t.Fatal("expected no layer error")
	}
	if _, err := sscachian.Define[string]("t").WithLayers(memory.New()).WithKeyBuilder(nil).Build(); err == nil {
		t.Fatal("expected no keybuilder error")
	}
	if _, err := sscachian.Define[string]("t").WithLayers().WithKeyBuilder(sscachian.DefaultKeyBuilder).Build(); err == nil {
		t.Fatal("expected empty layers error")
	}
}

func TestBuildKey_OK(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	k, err := ct.BuildKey(ctx, kc, 7)
	if err != nil {
		t.Fatal(err)
	}
	want := "my-app:cache:0123456:client_list_page1:7"
	if k != want {
		t.Fatalf("got %s want %s", k, want)
	}
	vk, err := ct.VersionKey(ctx, kc)
	if err != nil {
		t.Fatal(err)
	}
	if vk != "my-app:cache:0123456:client_list_page1:__version__" {
		t.Fatalf("version key %s", vk)
	}
}

func TestBuildLatestKey_MatchesCurrent(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	v, err := ct.CurrentVersion(ctx, kc)
	if err != nil {
		t.Fatal(err)
	}
	k, err := ct.BuildLatestKey(ctx, kc)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := ct.BuildKey(ctx, kc, v)
	if k != want {
		t.Fatalf("got %s want %s", k, want)
	}
}

func TestBuildKey_Errors(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	if _, err := ct.BuildKey(ctx, sscachian.KeyContext{}, 1); err == nil {
		t.Fatal("expected invalid context")
	}
	if _, err := ct.BuildKey(ctx, sampleKC(), 0); err == nil {
		t.Fatal("expected invalid version")
	}
	ct2 := newStringCache(t, func(b *sscachian.Builder[string]) {
		b.WithKeyBuilder(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
			return "", errors.New("kb boom")
		})
	})
	if _, err := ct2.BuildKey(ctx, sampleKC(), 1); err == nil || err.Error() != "kb boom" {
		t.Fatalf("err=%v", err)
	}
}

func TestCurrentVersion_InitAndRead(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	v, err := ct.CurrentVersion(ctx, kc)
	if err != nil || v != 1 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	// set to 7 via layer
	mem := memory.New()
	ct3, _ := sscachian.Define[string]("t").WithLayers(mem).Build()
	_ = mem.Set(ctx, mustVK(t, ct3, kc), sscachian.Entry{Value: int64(7)}, 0)
	v, err = ct3.CurrentVersion(ctx, kc)
	if err != nil || v != 7 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	if _, err := ct3.BumpVersion(ctx, kc); err != nil {
		t.Fatal(err)
	}
	v, err = ct3.CurrentVersion(ctx, kc)
	if err != nil || v != 8 {
		t.Fatalf("v=%d err=%v", v, err)
	}
}

func mustVK(t *testing.T, ct *sscachian.CacheType[string], kc sscachian.KeyContext) string {
	t.Helper()
	vk, err := ct.VersionKey(context.Background(), kc)
	if err != nil {
		t.Fatal(err)
	}
	return vk
}

func TestBumpVersion_AdvancesAndKeepsData(t *testing.T) {
	t.Parallel()
	mem := memory.New()
	ct, err := sscachian.Define[string]("t").WithLayers(mem).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	kc := sampleKC()
	_, _ = ct.CurrentVersion(ctx, kc) // 1
	k1, _ := ct.BuildKey(ctx, kc, 1)
	_ = mem.Set(ctx, k1, sscachian.Entry{Value: "old"}, 0)
	v, err := ct.BumpVersion(ctx, kc)
	if err != nil || v != 2 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	e, ok, _ := mem.Get(ctx, k1)
	if !ok || e.Value != "old" {
		t.Fatal("old data should remain")
	}
	for i := 0; i < 3; i++ {
		if _, err := ct.BumpVersion(ctx, kc); err != nil {
			t.Fatal(err)
		}
	}
	cur, err := ct.CurrentVersion(ctx, kc)
	if err != nil || cur < 2 {
		t.Fatalf("not monotonic: %d err=%v", cur, err)
	}
}

func TestBumpVersion_FromMissing(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	v, err := ct.BumpVersion(context.Background(), sampleKC())
	if err != nil || v != 2 {
		t.Fatalf("v=%d err=%v", v, err)
	}
}

func TestGet_SetRoundTrip(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	if err := ct.Set(ctx, kc, "hello"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "hello" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestGet_FirstMissInitsVersion(t *testing.T) {
	t.Parallel()
	mem := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(mem).Build()
	ctx := context.Background()
	kc := sampleKC()
	_, ok, err := ct.Get(ctx, kc)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	v, err := ct.CurrentVersion(ctx, kc)
	if err != nil || v != 1 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	vk := mustVK(t, ct, kc)
	if _, ok, _ := mem.Get(ctx, vk); !ok {
		t.Fatal("expected __version__ created")
	}
}

func TestGet_TypeMismatch(t *testing.T) {
	t.Parallel()
	mem := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(mem).Build()
	ctx := context.Background()
	kc := sampleKC()
	v, _ := ct.CurrentVersion(ctx, kc)
	k, _ := ct.BuildKey(ctx, kc, v)
	_ = mem.Set(ctx, k, sscachian.Entry{Value: 123}, 0)
	_, _, err := ct.Get(ctx, kc)
	if err == nil {
		t.Fatal("expected type mismatch")
	}
}

func TestSet_BumpsVersion(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	before, _ := ct.CurrentVersion(ctx, kc)
	if err := ct.Set(ctx, kc, "x"); err != nil {
		t.Fatal(err)
	}
	after, _ := ct.CurrentVersion(ctx, kc)
	if after != before+1 {
		t.Fatalf("before=%d after=%d", before, after)
	}
}

func TestSet_WithTTL(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t, func(b *sscachian.Builder[string]) { b.WithLayerTTL(time.Hour) })
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "x" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestSet_InvalidContext(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	if err := ct.Set(context.Background(), sscachian.KeyContext{}, "x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_BumpsAndKeepsVersionKey(t *testing.T) {
	t.Parallel()
	mem := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(mem).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	before, _ := ct.CurrentVersion(ctx, kc)
	if err := ct.Delete(ctx, kc); err != nil {
		t.Fatal(err)
	}
	after, _ := ct.CurrentVersion(ctx, kc)
	if after != before+1 {
		t.Fatalf("before=%d after=%d", before, after)
	}
	_, ok, _ := ct.Get(ctx, kc)
	if ok {
		t.Fatal("expected miss after delete")
	}
	vk := mustVK(t, ct, kc)
	if _, ok, _ := mem.Get(ctx, vk); !ok {
		t.Fatal("__version__ should remain")
	}
}

func TestDelete_MissingStillBumps(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	before, _ := ct.CurrentVersion(ctx, kc)
	if err := ct.Delete(ctx, kc); err != nil {
		t.Fatal(err)
	}
	after, _ := ct.CurrentVersion(ctx, kc)
	if after != before+1 {
		t.Fatalf("before=%d after=%d", before, after)
	}
}

func TestGetOrLoad_HitSkipsLoader(t *testing.T) {
	t.Parallel()
	var loads int32
	ct := newStringCache(t, func(b *sscachian.Builder[string]) {
		b.WithLoader(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
			atomic.AddInt32(&loads, 1)
			return "L", nil
		})
	})
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "cached")
	v, err := ct.GetOrLoad(ctx, kc)
	if err != nil || v != "cached" || atomic.LoadInt32(&loads) != 0 {
		t.Fatalf("v=%q loads=%d err=%v", v, loads, err)
	}
}

func TestGetOrLoad_MissLoads(t *testing.T) {
	t.Parallel()
	var loads int32
	ct := newStringCache(t, func(b *sscachian.Builder[string]) {
		b.WithLoader(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
			atomic.AddInt32(&loads, 1)
			return "L", nil
		})
	})
	ctx := context.Background()
	kc := sampleKC()
	v, err := ct.GetOrLoad(ctx, kc)
	if err != nil || v != "L" || atomic.LoadInt32(&loads) != 1 {
		t.Fatalf("v=%q loads=%d err=%v", v, loads, err)
	}
	got, ok, _ := ct.Get(ctx, kc)
	if !ok || got != "L" {
		t.Fatalf("cached got=%q ok=%v", got, ok)
	}
	before, _ := ct.CurrentVersion(ctx, kc)
	_, _ = ct.GetOrLoad(ctx, kc)
	after, _ := ct.CurrentVersion(ctx, kc)
	if before != after {
		t.Fatal("GetOrLoad must not bump")
	}
}

func TestGetOrLoad_NoLoaderError(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	if _, err := ct.GetOrLoad(context.Background(), sampleKC()); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetOrLoad_LoaderError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	ct := newStringCache(t, func(b *sscachian.Builder[string]) {
		b.WithLoader(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
			return "", boom
		})
	})
	if _, err := ct.GetOrLoad(context.Background(), sampleKC()); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
}

func TestEnsureCurrentVersion_Concurrent(t *testing.T) {
	t.Parallel()
	ct := newStringCache(t)
	ctx := context.Background()
	kc := sampleKC()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ct.CurrentVersion(ctx, kc); err != nil {
				t.Errorf("%v", err)
			}
		}()
	}
	wg.Wait()
	v, err := ct.CurrentVersion(ctx, kc)
	if err != nil || v < 1 {
		t.Fatalf("v=%d err=%v", v, err)
	}
}

type failSetLayer struct {
	sscachian.Layer
	failSet bool
}

func (f *failSetLayer) Set(ctx context.Context, key string, entry sscachian.Entry, ttl time.Duration) error {
	if f.failSet {
		return errors.New("set failed")
	}
	return f.Layer.Set(ctx, key, entry, ttl)
}

func TestGetOrLoad_WriteBackFailureStillReturns(t *testing.T) {
	t.Parallel()
	base := memory.New()
	fl := &failSetLayer{Layer: base}
	ct, _ := sscachian.Define[string]("t").WithLayers(fl).WithLoader(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "L", nil
	}).Build()
	ctx := context.Background()
	kc := sampleKC()
	// init version without failing
	_, _ = ct.CurrentVersion(ctx, kc)
	fl.failSet = true
	v, err := ct.GetOrLoad(ctx, kc)
	if err != nil || v != "L" {
		t.Fatalf("v=%q err=%v", v, err)
	}
}
