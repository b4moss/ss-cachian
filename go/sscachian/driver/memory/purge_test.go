package memory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/b4moss/ss-cachian"
	"github.com/b4moss/ss-cachian/driver/memory"
)

func TestPurgeExact_DeletesVersionKeysKeepsVersion(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	prefix := "app:cache:t:q"
	_ = s.Set(ctx, prefix+":1", sscachian.Entry{Value: "a"}, 0)
	_ = s.Set(ctx, prefix+":2", sscachian.Entry{Value: "b"}, 0)
	_ = s.Set(ctx, prefix+":__version__", sscachian.Entry{Value: int64(2)}, 0)
	_ = s.Set(ctx, "other:1", sscachian.Entry{Value: "o"}, 0)
	if err := s.PurgeExact(ctx, prefix); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, prefix+":1"); ok {
		t.Fatal("v1 should be gone")
	}
	if _, ok, _ := s.Get(ctx, prefix+":2"); ok {
		t.Fatal("v2 should be gone")
	}
	if _, ok, _ := s.Get(ctx, prefix+":__version__"); !ok {
		t.Fatal("__version__ should remain")
	}
	if _, ok, _ := s.Get(ctx, "other:1"); !ok {
		t.Fatal("other prefix should remain")
	}
}

func TestPurgeExact_IdempotentEmpty(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if err := s.PurgeExact(context.Background(), "app:cache:t:q"); err != nil {
		t.Fatal(err)
	}
}

func TestPurgeExact_EmptyPrefix(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if err := s.PurgeExact(context.Background(), ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestPurgeExact_VersionOnlyRemains(t *testing.T) {
	t.Parallel()
	s := memory.New()
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

func TestPurgeExact_ConcurrentWithSet(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	prefix := "c"
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = s.Set(ctx, prefix+":1", sscachian.Entry{Value: n}, 0)
		}(i)
		go func() {
			defer wg.Done()
			_ = s.PurgeExact(ctx, prefix)
		}()
	}
	wg.Wait()
}
