package firestore_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/b4moss/ss-cachian"
	fsdriver "github.com/b4moss/ss-cachian/driver/firestore"
)

func requireEmulator(t *testing.T) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Fatal("FIRESTORE_EMULATOR_HOST is required for firestore driver tests")
	}
}

func newStore(t *testing.T) *fsdriver.Store {
	t.Helper()
	requireEmulator(t)
	ctx := context.Background()
	s, err := fsdriver.New(ctx, fsdriver.Options{
		ProjectID:  "ss-cachian-dev",
		Collection: "sscachian_test_" + t.Name(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = s.ClearCollection(context.Background())
		_ = s.Close()
	})
	return s
}

func TestGetSetRoundTrip(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	exp := now.Add(time.Hour)
	if err := s.Set(ctx, "k", sscachian.Entry{Value: "hello", CreatedAt: now, ExpiresAt: exp}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "k")
	if err != nil || !ok || e.Value != "hello" {
		t.Fatalf("ok=%v val=%v err=%v", ok, e.Value, err)
	}
}

func TestGet_ExpiredMiss(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Minute)
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 1, CreatedAt: past, ExpiresAt: past}, 0)
	_, ok, err := s.Get(ctx, "k")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestGet_Missing(t *testing.T) {
	s := newStore(t)
	_, ok, err := s.Get(context.Background(), "nope")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestGet_EmptyKey(t *testing.T) {
	s := newStore(t)
	_, _, err := s.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSet_OverwriteAndTTL(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "k", sscachian.Entry{Value: "a"}, 0)
	_ = s.Set(ctx, "k", sscachian.Entry{Value: "b"}, time.Hour)
	e, ok, _ := s.Get(ctx, "k")
	if !ok || e.Value != "b" || e.ExpiresAt.IsZero() {
		t.Fatalf("%+v ok=%v", e, ok)
	}
}

func TestSet_NegativeTTL(t *testing.T) {
	s := newStore(t)
	if err := s.Set(context.Background(), "k", sscachian.Entry{Value: 1}, -time.Second); err == nil {
		t.Fatal("expected error")
	}
}

func TestSet_NilValue(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if err := s.Set(ctx, "n", sscachian.Entry{Value: nil}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "n")
	if err != nil || !ok || e.Value != nil {
		t.Fatalf("ok=%v val=%v err=%v", ok, e.Value, err)
	}
}

func TestDelete(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 1}, 0)
	if err := s.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "k"); ok {
		t.Fatal("expected miss")
	}
	if err := s.Delete(ctx, "missing"); err != nil {
		t.Fatal(err)
	}
	_ = s.Set(ctx, "k", sscachian.Entry{Value: 2}, 0)
	e, ok, _ := s.Get(ctx, "k")
	if !ok || e.Value != float64(2) { // JSON numbers decode as float64
		t.Fatalf("%+v ok=%v", e, ok)
	}
}

func TestDelete_EmptyKey(t *testing.T) {
	s := newStore(t)
	if err := s.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestIncr(t *testing.T) {
	s := newStore(t)
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

func TestIncr_Concurrent(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := s.Incr(ctx, "c"); err != nil {
				t.Errorf("%v", err)
			}
		}()
	}
	wg.Wait()
	e, ok, err := s.Get(ctx, "c")
	if err != nil || !ok {
		t.Fatal(err)
	}
	got := int64(e.Value.(float64))
	if got != int64(n) {
		t.Fatalf("got %d want %d", got, n)
	}
}

func TestIncr_NonInteger(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "c", sscachian.Entry{Value: "x"}, 0)
	if _, err := s.Incr(ctx, "c"); err == nil {
		t.Fatal("expected error")
	}
}
