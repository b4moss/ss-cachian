package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/b4moss/ss-cachian"
	"github.com/b4moss/ss-cachian/driver/memory"
)

func TestGet_HitReturnsEntry(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	exp := now.Add(time.Hour)
	if err := s.Set(ctx, "k", sscachian.Entry{Value: "v", CreatedAt: now, ExpiresAt: exp}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "k")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if e.Value != "v" || !e.CreatedAt.Equal(now) || !e.ExpiresAt.Equal(exp) {
		t.Fatalf("entry mismatch: %+v", e)
	}
}

func TestGet_ExpiredIsMissAndRemoved(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Minute)
	if err := s.Set(ctx, "k", sscachian.Entry{Value: 1, CreatedAt: past, ExpiresAt: past}, 0); err != nil {
		t.Fatal(err)
	}
	_, ok, err := s.Get(ctx, "k")
	if err != nil || ok {
		t.Fatalf("expected miss, ok=%v err=%v", ok, err)
	}
	_, ok, err = s.Get(ctx, "k")
	if err != nil || ok {
		t.Fatalf("expected still miss after expiry purge, ok=%v err=%v", ok, err)
	}
}

func TestGet_MissingIsMiss(t *testing.T) {
	t.Parallel()
	s := memory.New()
	_, ok, err := s.Get(context.Background(), "nope")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestGet_EmptyKeyError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	_, _, err := s.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGet_CorruptValueNoPanic(t *testing.T) {
	t.Parallel()
	s := memory.New()
	s.InjectRaw("bad", "not-an-entry")
	_, ok, err := s.Get(context.Background(), "bad")
	if ok {
		t.Fatal("expected miss or error, not hit")
	}
	_ = err // either error or miss is acceptable
}

func TestGet_ConcurrentExpiredNoPanic(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Second)
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 1, ExpiresAt: past}, 0)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = s.Get(ctx, "k")
		}()
	}
	wg.Wait()
}

func TestSet_GetRoundTrip(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	if err := s.Set(ctx, "a", sscachian.Entry{Value: 42}, time.Hour); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "a")
	if err != nil || !ok || e.Value != 42 {
		t.Fatalf("ok=%v val=%v err=%v", ok, e.Value, err)
	}
}

func TestSet_Overwrite(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "a", sscachian.Entry{Value: "old"}, 0)
	_ = s.Set(ctx, "a", sscachian.Entry{Value: "new"}, 0)
	e, ok, _ := s.Get(ctx, "a")
	if !ok || e.Value != "new" {
		t.Fatalf("got %+v", e)
	}
}

func TestSet_TTLExpiry(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	if err := s.Set(ctx, "a", sscachian.Entry{Value: 1}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "a"); !ok {
		t.Fatal("expected hit before expiry")
	}
	// Force expiry via past ExpiresAt overwrite path used by tests.
	past := time.Now().UTC().Add(-time.Second)
	if err := s.Set(ctx, "a", sscachian.Entry{Value: 1, CreatedAt: past, ExpiresAt: past}, 0); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "a"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestSet_EmptyKeyError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if err := s.Set(context.Background(), "", sscachian.Entry{Value: 1}, 0); err == nil {
		t.Fatal("expected error")
	}
}

func TestSet_NilValueAllowed(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	if err := s.Set(ctx, "n", sscachian.Entry{Value: nil}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "n")
	if err != nil || !ok || e.Value != nil {
		t.Fatalf("ok=%v val=%v err=%v", ok, e.Value, err)
	}
}

func TestSet_NegativeTTLError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if err := s.Set(context.Background(), "k", sscachian.Entry{Value: 1}, -time.Second); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_Existing(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 1}, 0)
	if err := s.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("expected miss")
	}
}

func TestDelete_MissingOK(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if err := s.Delete(context.Background(), "missing"); err != nil {
		t.Fatal(err)
	}
}

func TestDelete_ThenSetAgain(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 1}, 0)
	_ = s.Delete(ctx, "k")
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 2}, 0)
	e, ok, _ := s.Get(ctx, "k")
	if !ok || e.Value != 2 {
		t.Fatalf("got %+v", e)
	}
}

func TestDelete_EmptyKeyError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if err := s.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestDelete_ConcurrentSetDeleteNoPanic(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = s.Set(ctx, "k", sscachian.Entry{Value: 1}, 0)
		}()
		go func() {
			defer wg.Done()
			_ = s.Delete(ctx, "k")
		}()
	}
	wg.Wait()
}

func TestDelete_OtherKeysIntact(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "a", sscachian.Entry{Value: 1}, 0)
	_ = s.Set(ctx, "b", sscachian.Entry{Value: 2}, 0)
	_ = s.Delete(ctx, "a")
	e, ok, _ := s.Get(ctx, "b")
	if !ok || e.Value != 2 {
		t.Fatalf("b corrupted: %+v", e)
	}
}

func TestIncr_MissingStartsAtOne(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	n, err := s.Incr(ctx, "c")
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	n, err = s.Incr(ctx, "c")
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestIncr_Existing(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "c", sscachian.Entry{Value: int64(10)}, 0)
	n, err := s.Incr(ctx, "c")
	if err != nil || n != 11 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestIncr_Concurrent(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := s.Incr(ctx, "c"); err != nil {
				t.Errorf("incr: %v", err)
			}
		}()
	}
	wg.Wait()
	e, ok, err := s.Get(ctx, "c")
	if err != nil || !ok {
		t.Fatalf("get: %v", err)
	}
	got, _ := e.Value.(int64)
	if got != int64(n) {
		t.Fatalf("got %d want %d", got, n)
	}
}

func TestIncr_EmptyKeyError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	if _, err := s.Incr(context.Background(), ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestIncr_NonIntegerError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "c", sscachian.Entry{Value: "x"}, 0)
	if _, err := s.Incr(ctx, "c"); err == nil {
		t.Fatal("expected error")
	}
}

func TestIncr_OverflowError(t *testing.T) {
	t.Parallel()
	s := memory.New()
	ctx := context.Background()
	_ = s.Set(ctx, "c", sscachian.Entry{Value: int64(^uint64(0) >> 1)}, 0) // max int64
	if _, err := s.Incr(ctx, "c"); err == nil {
		t.Fatal("expected overflow error")
	}
}
