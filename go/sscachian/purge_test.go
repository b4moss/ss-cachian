package sscachian_test

import (
	"context"
	"errors"
	"testing"

	"github.com/b4moss/ss-cachian"
	"github.com/b4moss/ss-cachian/driver/memory"
)

type failPurgeLayer struct {
	sscachian.Layer
	err     error
	calls   int
	failing bool
}

func (f *failPurgeLayer) PurgeExact(ctx context.Context, logicalPrefix string) error {
	f.calls++
	if f.failing && f.err != nil {
		return f.err
	}
	return f.Layer.PurgeExact(ctx, logicalPrefix)
}

func TestPurge_AllLayersKeepsVersion(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, err := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	kc := sampleKC()
	other := sscachian.KeyContext{AppSlug: "my-app", TenantID: "other", QueryType: "client_list_page1"}

	_ = ct.Set(ctx, kc, "v1")
	_ = ct.Set(ctx, kc, "v2")
	_ = ct.Set(ctx, other, "keep")
	before, _ := ct.CurrentVersion(ctx, kc)

	k1, _ := ct.BuildKey(ctx, kc, 1)
	k2, _ := ct.BuildKey(ctx, kc, 2)
	// After two Sets, versions are 2 then 3? ensure=1, bump->2 set, bump->3 set.
	// So data at :2 and :3 typically. Also check any version keys via prefix.

	if err := ct.Purge(ctx, kc); err != nil {
		t.Fatal(err)
	}
	after, _ := ct.CurrentVersion(ctx, kc)
	if before != after {
		t.Fatalf("version changed: before=%d after=%d", before, after)
	}
	vk, _ := ct.VersionKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, vk); !ok {
		t.Fatal("__version__ should remain on L1")
	}
	for _, store := range []sscachian.Layer{l1, l2} {
		for v := int64(1); v <= after+2; v++ {
			key, _ := ct.BuildKey(ctx, kc, v)
			if _, ok, _ := store.Get(ctx, key); ok {
				t.Fatalf("data key %s should be purged", key)
			}
		}
	}
	_ = k1
	_ = k2
	got, ok, err := ct.Get(ctx, other)
	if err != nil || !ok || got != "keep" {
		t.Fatalf("other key: got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestPurge_IdempotentNoData(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1).Build()
	ctx := context.Background()
	kc := sampleKC()
	if err := ct.Purge(ctx, kc); err != nil {
		t.Fatal(err)
	}
	vk, _ := ct.VersionKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, vk); ok {
		t.Fatal("Purge must not create __version__")
	}
}

func TestPurge_InvalidContext(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1).Build()
	if err := ct.Purge(context.Background(), sscachian.KeyContext{}); !errors.Is(err, sscachian.ErrInvalidContext) {
		t.Fatalf("err=%v", err)
	}
}

func TestPurge_L1FailureStops(t *testing.T) {
	t.Parallel()
	boom := errors.New("purge boom")
	l1 := &failPurgeLayer{Layer: memory.New(), err: boom, failing: true}
	l2 := &failPurgeLayer{Layer: memory.New()}
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x") // Set needs real L1 Set; failPurgeLayer only wraps PurgeExact
	// restore underlying for Set path: failPurge embeds memory for Set
	if err := ct.Purge(ctx, kc); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if l2.calls != 0 {
		t.Fatalf("L2 should not be called, calls=%d", l2.calls)
	}
}

func TestPurge_LayerSubset(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	key, _ := ct.BuildLatestKey(ctx, kc)
	if err := ct.Purge(ctx, kc, 1); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := l1.Get(ctx, key); !ok {
		t.Fatal("L1 data should remain")
	}
	if _, ok, _ := l2.Get(ctx, key); ok {
		t.Fatal("L2 data should be purged")
	}
	vk, _ := ct.VersionKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, vk); !ok {
		t.Fatal("__version__ should remain")
	}
}

func TestPurge_AllLayersExplicit(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	key, _ := ct.BuildLatestKey(ctx, kc)
	if err := ct.Purge(ctx, kc); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := l1.Get(ctx, key); ok {
		t.Fatal("L1 should miss")
	}
	if _, ok, _ := l2.Get(ctx, key); ok {
		t.Fatal("L2 should miss")
	}
}

func TestPurge_InvalidIndex(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	key, _ := ct.BuildLatestKey(ctx, kc)
	if err := ct.Purge(ctx, kc, 99); !errors.Is(err, sscachian.ErrInvalidLayerIndex) {
		t.Fatalf("err=%v", err)
	}
	if _, ok, _ := l1.Get(ctx, key); !ok {
		t.Fatal("should not purge on invalid index")
	}
}

func TestPurge_L2OnlyFailureIsError(t *testing.T) {
	t.Parallel()
	boom := errors.New("l2 purge boom")
	l1 := memory.New()
	l2 := &failPurgeLayer{Layer: memory.New(), err: boom, failing: true}
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	if err := ct.Purge(ctx, kc, 1); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
}

func TestPurge_L2FailureWhenAllLayersStillOK(t *testing.T) {
	t.Parallel()
	boom := errors.New("l2 purge boom")
	l1 := memory.New()
	l2 := &failPurgeLayer{Layer: memory.New(), err: boom, failing: true}
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	key, _ := ct.BuildLatestKey(ctx, kc)
	if err := ct.Purge(ctx, kc); err != nil {
		t.Fatalf("expected success with logged L2 failure, err=%v", err)
	}
	if _, ok, _ := l1.Get(ctx, key); ok {
		t.Fatal("L1 should be purged")
	}
}

func TestPurge_DuplicateIndexesIdempotent(t *testing.T) {
	t.Parallel()
	l1, l2 := memory.New(), memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1, l2).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "x")
	if err := ct.Purge(ctx, kc, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
}

func TestPurge_AfterMultipleSetsThenBumpSet(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "a")
	_ = ct.Set(ctx, kc, "b")
	before, _ := ct.CurrentVersion(ctx, kc)
	if err := ct.Purge(ctx, kc); err != nil {
		t.Fatal(err)
	}
	after, _ := ct.CurrentVersion(ctx, kc)
	if before != after {
		t.Fatalf("version changed %d -> %d", before, after)
	}
	if _, ok, _ := ct.Get(ctx, kc); ok {
		t.Fatal("expected miss after purge")
	}
	if err := ct.Set(ctx, kc, "c"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ct.Get(ctx, kc)
	if err != nil || !ok || got != "c" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
}

func TestPurge_CleansOldKeysAfterDelete(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	ct, _ := sscachian.Define[string]("t").WithLayers(l1).Build()
	ctx := context.Background()
	kc := sampleKC()
	_ = ct.Set(ctx, kc, "a")
	oldKey, _ := ct.BuildLatestKey(ctx, kc)
	_ = ct.Delete(ctx, kc) // bumps; old key already deleted for latest, but if we had older versions...
	// Plant an old version key explicitly.
	_ = l1.Set(ctx, oldKey, sscachian.Entry{Value: "stale"}, 0)
	_ = ct.Set(ctx, kc, "b")
	if err := ct.Purge(ctx, kc); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := l1.Get(ctx, oldKey); ok {
		t.Fatal("old key should be purged")
	}
	vk, _ := ct.VersionKey(ctx, kc)
	if _, ok, _ := l1.Get(ctx, vk); !ok {
		t.Fatal("__version__ should remain")
	}
}

func TestPurge_EmptyPrefixFromKeyBuilder(t *testing.T) {
	t.Parallel()
	l1 := memory.New()
	_ = l1.Set(context.Background(), "keep:1", sscachian.Entry{Value: 1}, 0)
	ct, _ := sscachian.Define[string]("t").WithLayers(l1).WithKeyBuilder(func(ctx context.Context, kc sscachian.KeyContext) (string, error) {
		return "", nil
	}).Build()
	if err := ct.Purge(context.Background(), sampleKC()); !errors.Is(err, sscachian.ErrEmptyKey) {
		t.Fatalf("err=%v", err)
	}
	if _, ok, _ := l1.Get(context.Background(), "keep:1"); !ok {
		t.Fatal("other keys must stay")
	}
}

func TestIsVersionDataKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		prefix, key string
		want        bool
	}{
		{"p", "p:1", true},
		{"p", "p:12", true},
		{"p", "p:__version__", false},
		{"p", "p:", false},
		{"p", "p:1a", false},
		{"p", "other:1", false},
		{"", "p:1", false},
	}
	for _, tc := range cases {
		if got := sscachian.IsVersionDataKey(tc.prefix, tc.key); got != tc.want {
			t.Fatalf("%q %q: got %v want %v", tc.prefix, tc.key, got, tc.want)
		}
	}
}
