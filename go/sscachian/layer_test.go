package sscachian_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/b4moss/ss-cachian"
	"github.com/b4moss/ss-cachian/driver/memory"
)

type countingLayer struct {
	sscachian.Layer
	gets atomic.Int32
}

func (c *countingLayer) Get(ctx context.Context, key string) (sscachian.Entry, bool, error) {
	c.gets.Add(1)
	return c.Layer.Get(ctx, key)
}

type failGetLayer struct {
	sscachian.Layer
	err error
}

func (f *failGetLayer) Get(ctx context.Context, key string) (sscachian.Entry, bool, error) {
	if f.err != nil {
		return sscachian.Entry{}, false, f.err
	}
	return f.Layer.Get(ctx, key)
}

func TestGet_L1HitSkipsL2(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	l2base := memory.New()
	l2 := &countingLayer{Layer: l2base}
	ct, err := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	kc := sampleKC()
	if err := ct.Set(ctx, kc, "v"); err != nil {
		t.Fatal(err)
	}
	l2.gets.Store(0)
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "v" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
	if l2.gets.Load() != 0 {
		t.Fatalf("L2 gets=%d", l2.gets.Load())
	}
}

func TestGet_L2HitWriteBackL1(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	l2 := memory.New()
	ct, err := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	kc := sampleKC()
	_, _ = ct.CurrentVersion(ctx, kc)
	key, _ := ct.BuildLatestKey(ctx, kc)
	_ = l2.Set(ctx, key, sscachian.Entry{Value: "from-l2"}, 0)
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "from-l2" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
	e, ok, _ := l1.Get(ctx, key)
	if !ok || e.Value != "from-l2" {
		t.Fatalf("L1 write-back missing: %+v ok=%v", e, ok)
	}
}

func TestGet_AllMiss(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_, ok, err := ct.Get(ctx, kc)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	key, _ := ct.BuildLatestKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, key); ok {
		t.Fatal("L1 should be empty")
	}
	if _, ok, _ := l2.Get(ctx, key); ok {
		t.Fatal("L2 should be empty")
	}
}

func TestGet_L2ErrorStops(t *testing.T) {
	t.Parallel()
	boom := errors.New("l2 boom")
	l1 := memory.New()
	l2 := &failGetLayer{Layer: memory.New(), err: boom}
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	_, _, err := ct.Get(context.Background(), sampleKC())
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
}

func TestGet_WriteBackFailureStillReturns(t *testing.T) {
	t.Parallel()
	l1 := &failSetLayer{Layer: memory.New(), failSet: true}
	l2 := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	// init version on underlying store via temporary unlock of fail
	l1.failSet = false
	_, _ = ct.CurrentVersion(ctx, kc)
	key, _ := ct.BuildLatestKey(ctx, kc)
	_ = l2.Set(ctx, key, sscachian.Entry{Value: "x"}, 0)
	l1.failSet = true
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "x" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestGetOrLoad_Multilayer(t *testing.T) {
	t.Parallel()
	var loads int32
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).WithLoader(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		atomic.AddInt32(&loads, 1)
		return "L", nil
	}).Build()
	ctx := context.Background()
	kc := sampleKC()
	v, err := ct.GetOrLoad(ctx, kc)
	if err != nil || v != "L" || loads != 1 {
		t.Fatalf("v=%q loads=%d err=%v", v, loads, err)
	}
	before, _ := ct.CurrentVersion(ctx, kc)
	key, _ := ct.BuildLatestKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, key); !ok {
		t.Fatal("expected L1 fill")
	}
	if _, ok, _ := l2.Get(ctx, key); !ok {
		t.Fatal("expected L2 fill")
	}
	_, _ = ct.GetOrLoad(ctx, kc)
	after, _ := ct.CurrentVersion(ctx, kc)
	if before != after || loads != 1 {
		t.Fatalf("bump or reload: before=%d after=%d loads=%d", before, after, loads)
	}
}

func TestSet_WritesAllLayers(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).WithLayerTTLs(time.Hour, 2*time.Hour).Build()
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
	key, _ := ct.BuildLatestKey(ctx, kc)
	e1, ok1, _ := l1.Get(ctx, key)
	e2, ok2, _ := l2.Get(ctx, key)
	if !ok1 || !ok2 || e1.Value != "x" || e2.Value != "x" {
		t.Fatalf("l1=%v %+v l2=%v %+v", ok1, e1, ok2, e2)
	}
	if e1.ExpiresAt.IsZero() || e2.ExpiresAt.IsZero() {
		t.Fatal("expected expires_at from TTLs")
	}
	if !e2.ExpiresAt.After(e1.ExpiresAt) {
		t.Fatalf("L2 TTL should be longer: e1=%v e2=%v", e1.ExpiresAt, e2.ExpiresAt)
	}
}

func TestSet_L2FailureStillOK(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	l2 := &failSetLayer{Layer: memory.New(), failSet: true}
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	if err := ct.Set(ctx, kc, "x"); err != nil {
		t.Fatal(err)
	}
	got, ok, _ := ct.Get(ctx, kc)
	if !ok || got != "x" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestDelete_AllLayers(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	keyBefore, _ := ct.BuildLatestKey(ctx, kc)
	before, _ := ct.CurrentVersion(ctx, kc)
	if err := ct.Delete(ctx, kc); err != nil {
		t.Fatal(err)
	}
	after, _ := ct.CurrentVersion(ctx, kc)
	if after != before+1 {
		t.Fatalf("before=%d after=%d", before, after)
	}
	if _, ok, _ := l1.Get(ctx, keyBefore); ok {
		t.Fatal("L1 should miss old key")
	}
	if _, ok, _ := l2.Get(ctx, keyBefore); ok {
		t.Fatal("L2 should miss old key")
	}
	vk, _ := ct.VersionKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, vk); !ok {
		t.Fatal("__version__ should remain on L1")
	}
}

func TestBuild_WithLayerTTLs(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, err := sscachian.Define[string]("t").WithLayers(l1, l2).WithLayerTTLs(time.Second).Build()
	if err != nil {
		t.Fatal(err)
	}
	if ct.LayerTTL(0) != time.Second || ct.LayerTTL(1) != 0 {
		t.Fatalf("ttls %v %v", ct.LayerTTL(0), ct.LayerTTL(1))
	}
	_, err = sscachian.Define[string]("t").WithLayers(l1).WithLayerTTLs(-time.Second).Build()
	if !errors.Is(err, sscachian.ErrNegativeTTL) {
		t.Fatalf("err=%v", err)
	}
	ct2, err := sscachian.Define[string]("t").WithLayers(l1, l2).WithLayerTTL(time.Minute).Build()
	if err != nil {
		t.Fatal(err)
	}
	if ct2.LayerTTL(0) != time.Minute || ct2.LayerTTL(1) != time.Minute {
		t.Fatal("single TTL not applied to all")
	}
}
