package es4_test

import (
	"context"
	"strings"
	"testing"

	"github.com/b4moss/ss-cachian/go/sscachian"
	es4driver "github.com/b4moss/ss-cachian/go/sscachian/driver/es4"
)

type scanCountingBackend struct {
	*memBackend
	scanCalls int
}

func (b *scanCountingBackend) Scan(ctx context.Context, prefix string) ([]string, error) {
	b.scanCalls++
	return b.memBackend.Scan(ctx, prefix)
}

func TestPurgeExact_DeletesVersionKeysKeepsVersion(t *testing.T) {
	t.Parallel()
	inner := newMemBackend()
	b := &scanCountingBackend{memBackend: inner}
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	prefix := "app:cache:t:q"
	_ = s.Set(ctx, prefix+":1", sscachian.Entry{Value: "a"}, 0)
	_ = s.Set(ctx, prefix+":2", sscachian.Entry{Value: "b"}, 0)
	_ = s.Set(ctx, prefix+":__version__", sscachian.Entry{Value: int64(2)}, 0)
	_ = s.Set(ctx, "other:1", sscachian.Entry{Value: "keep"}, 0)

	if err := s.PurgeExact(ctx, prefix); err != nil {
		t.Fatal(err)
	}
	if b.scanCalls != 0 {
		t.Fatalf("PurgeExact must not call Scan, calls=%d", b.scanCalls)
	}
	if _, ok, _ := s.Get(ctx, prefix+":1"); ok {
		t.Fatal(":1 should be gone")
	}
	if _, ok, _ := s.Get(ctx, prefix+":2"); ok {
		t.Fatal(":2 should be gone")
	}
	if _, ok, _ := s.Get(ctx, prefix+":__version__"); !ok {
		t.Fatal("__version__ should remain")
	}
	if _, ok, _ := s.Get(ctx, "other:1"); !ok {
		t.Fatal("other prefix should remain")
	}
}

func TestPurgeExact_IdempotentAndEmptyPrefix(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	if err := s.PurgeExact(context.Background(), "app:cache:t:q"); err != nil {
		t.Fatal(err)
	}
	if err := s.PurgeExact(context.Background(), ""); err == nil {
		t.Fatal("expected empty key")
	}
}

func TestPurgeExact_VersionOnlyRemains(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	prefix := "p"
	_ = s.Set(ctx, prefix+":__version__", sscachian.Entry{Value: int64(1)}, 0)
	if err := s.PurgeExact(ctx, prefix); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, prefix+":__version__"); !ok {
		t.Fatal("__version__ should remain")
	}
}

func TestPurgePrefix_ScanAndDelete(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	prefix := "app:cache:t:q"
	for _, k := range []string{prefix + ":1", prefix + ":2", prefix + ":__version__", prefix + ":extra", "other:1"} {
		_ = s.Set(ctx, k, sscachian.Entry{Value: 1}, 0)
	}
	if err := s.PurgePrefix(ctx, prefix); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{prefix + ":1", prefix + ":2", prefix + ":__version__", prefix + ":extra"} {
		if _, ok, _ := s.Get(ctx, k); ok {
			t.Fatalf("%s should be gone", k)
		}
	}
	if _, ok, _ := s.Get(ctx, "other:1"); !ok {
		t.Fatal("other:1 should remain")
	}
}

func TestPurgePrefix_PrefixMatchOnly(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "ab:1", sscachian.Entry{Value: 1}, 0)
	_ = s.Set(ctx, "a:1", sscachian.Entry{Value: 1}, 0)
	if err := s.PurgePrefix(ctx, "a:"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "a:1"); ok {
		t.Fatal("a:1 should be gone")
	}
	if _, ok, _ := s.Get(ctx, "ab:1"); !ok {
		t.Fatal("ab:1 should remain (not a prefix match of a:)")
	}
}

func TestPurgePrefix_EmptyAndIdempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	if err := s.PurgePrefix(context.Background(), ""); err == nil {
		t.Fatal("expected empty key")
	}
	if err := s.PurgePrefix(context.Background(), "missing:"); err != nil {
		t.Fatal(err)
	}
}

func TestPurgePrefix_ScanFailurePropagates(t *testing.T) {
	t.Parallel()
	b := &failScanBackend{memBackend: newMemBackend(), err: context.Canceled}
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })
	err := s.PurgePrefix(context.Background(), "p")
	if err == nil || !strings.Contains(err.Error(), "canceled") && err != context.Canceled {
		t.Fatalf("expected canceled, got %v", err)
	}
}

func TestPurgePrefix_EmbeddedEs4Scan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := es4driver.New(ctx, es4driver.Options{MemoryOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	prefix := "app:cache:t:q"
	for _, k := range []string{prefix + ":1", prefix + ":2", prefix + ":__version__", prefix + ":extra", "other:1"} {
		if err := s.Set(ctx, k, sscachian.Entry{Value: 1}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PurgePrefix(ctx, prefix); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{prefix + ":1", prefix + ":2", prefix + ":__version__", prefix + ":extra"} {
		if _, ok, _ := s.Get(ctx, k); ok {
			t.Fatalf("%s should be gone via real es4.Scan", k)
		}
	}
	if _, ok, _ := s.Get(ctx, "other:1"); !ok {
		t.Fatal("other:1 should remain")
	}
}

func TestSet_InvalidColonKeyFromEs4(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := es4driver.New(ctx, es4driver.Options{MemoryOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	for _, bad := range []string{":a", "a:", "a::b"} {
		if err := s.Set(ctx, bad, sscachian.Entry{Value: 1}, 0); err == nil {
			t.Fatalf("expected invalid key error for %q", bad)
		}
	}
}

type failScanBackend struct {
	*memBackend
	err error
}

func (b *failScanBackend) Scan(context.Context, string) ([]string, error) {
	return nil, b.err
}
