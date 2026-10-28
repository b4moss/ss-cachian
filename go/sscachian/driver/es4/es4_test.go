package es4_test

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/b4moss/ss-cachian"
	es4driver "github.com/b4moss/ss-cachian/driver/es4"
)

func newTestStore(t *testing.T) *es4driver.Store {
	t.Helper()
	s := es4driver.NewWithBackend(newMemBackend())
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpen_MemoryOnlyRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s, err := es4driver.New(ctx, es4driver.Options{MemoryOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Set(ctx, "a:b", sscachian.Entry{Value: "v"}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "a:b")
	if err != nil || !ok || e.Value != "v" {
		t.Fatalf("ok=%v val=%v err=%v", ok, e.Value, err)
	}
}

func TestClose_IdempotentAndBlocksOps(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(ctx, "k", sscachian.Entry{Value: 1}, 0); err == nil {
		t.Fatal("expected error after close")
	}
	if _, _, err := s.Get(ctx, "k"); err == nil {
		t.Fatal("expected error after close")
	}
	if err := s.Delete(ctx, "k"); err == nil {
		t.Fatal("expected error after close")
	}
}

func TestOpen_InvalidFirestoreOptions(t *testing.T) {
	t.Parallel()
	_, err := es4driver.New(context.Background(), es4driver.Options{
		StateBackend:            "firestore",
		StateFirestoreProjectID: "", // required
	})
	if err == nil {
		t.Fatal("expected open error")
	}
}

func TestGetSet_RoundTripTypes(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	cases := []any{
		"str",
		float64(3),
		map[string]any{"a": float64(1)},
	}
	for i, v := range cases {
		key := "k:" + string(rune('a'+i))
		if err := s.Set(ctx, key, sscachian.Entry{Value: v}, 0); err != nil {
			t.Fatal(err)
		}
		e, ok, err := s.Get(ctx, key)
		if err != nil || !ok {
			t.Fatalf("%v: ok=%v err=%v", v, ok, err)
		}
		got, _ := json.Marshal(e.Value)
		want, _ := json.Marshal(v)
		if string(got) != string(want) {
			t.Fatalf("value mismatch got=%s want=%s", got, want)
		}
	}
}

func TestGetSet_TTLHitAndMiss(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.Set(ctx, "ttl:k", sscachian.Entry{Value: 1}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.Get(ctx, "ttl:k"); err != nil || !ok {
		t.Fatalf("expected hit: ok=%v err=%v", ok, err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if err := s.Set(ctx, "ttl:old", sscachian.Entry{Value: 2, ExpiresAt: past}, 0); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.Get(ctx, "ttl:old"); err != nil || ok {
		t.Fatalf("expected miss: ok=%v err=%v", ok, err)
	}
}

func TestGetSet_NoExpiresAtPersists(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.Set(ctx, "forever", sscachian.Entry{Value: "x"}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "forever")
	if err != nil || !ok || e.Value != "x" || !e.ExpiresAt.IsZero() {
		t.Fatalf("entry=%+v ok=%v err=%v", e, ok, err)
	}
}

func TestGetSet_EmptyKeyAndNegativeTTL(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	if _, _, err := s.Get(ctx, ""); err == nil {
		t.Fatal("expected empty key")
	}
	if err := s.Set(ctx, "", sscachian.Entry{Value: 1}, 0); err == nil {
		t.Fatal("expected empty key")
	}
	if err := s.Set(ctx, "k", sscachian.Entry{Value: 1}, -time.Second); err == nil {
		t.Fatal("expected negative ttl")
	}
}

func TestGet_CorruptEntryIsMiss(t *testing.T) {
	t.Parallel()
	b := newMemBackend()
	_ = b.Set(context.Background(), "bad", json.RawMessage(`{`), nil)
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })
	_, ok, err := s.Get(context.Background(), "bad")
	if err != nil || ok {
		t.Fatalf("expected miss, ok=%v err=%v", ok, err)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "d", sscachian.Entry{Value: 1}, 0)
	if err := s.Delete(ctx, "d"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "d"); ok {
		t.Fatal("expected miss")
	}
	if err := s.Delete(ctx, "d"); err != nil {
		t.Fatal(err)
	}
	_ = s.Set(ctx, "d", sscachian.Entry{Value: 2}, 0)
	e, ok, _ := s.Get(ctx, "d")
	if !ok || e.Value != float64(2) && e.Value != 2 && e.Value != int64(2) {
		// JSON numbers decode as float64
		if n, ok := e.Value.(float64); !ok || n != 2 {
			t.Fatalf("got %+v", e.Value)
		}
	}
}

func TestIncr_Basics(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
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

func TestIncr_NotIntegerAndOverflow(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.Set(ctx, "s", sscachian.Entry{Value: "nope"}, 0)
	if _, err := s.Incr(ctx, "s"); err == nil {
		t.Fatal("expected not integer")
	}
	_ = s.Set(ctx, "max", sscachian.Entry{Value: math.MaxInt64}, 0)
	if _, err := s.Incr(ctx, "max"); err == nil {
		t.Fatal("expected overflow")
	}
}

func TestIncr_Concurrent(t *testing.T) {
	t.Parallel()
	// Use embedded es4 memory (serialized Tx) rather than the test fake.
	ctx := context.Background()
	s, err := es4driver.New(ctx, es4driver.Options{MemoryOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Incr(ctx, "par"); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("incr: %v", err)
	}
	e, ok, err := s.Get(ctx, "par")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	f, ok := e.Value.(float64)
	if !ok || int64(f) != n {
		t.Fatalf("got %v want %d", e.Value, n)
	}
}

func mustJSONNumber(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestSet_StoresPayloadOnlyWithoutExpiresOption(t *testing.T) {
	t.Parallel()
	b := newMemBackend()
	var lastExp *time.Time
	b.lastSetExpiresAt = &lastExp
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	if err := s.Set(ctx, "e", sscachian.Entry{Value: "x"}, 0); err != nil {
		t.Fatal(err)
	}
	if lastExp != nil {
		t.Fatalf("expected nil expiresAt option, got %v", *lastExp)
	}
	raw, err := b.Get(ctx, "e")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `"x"` {
		t.Fatalf("want payload JSON string, got %s", raw)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err == nil {
		if _, ok := m["expires_at"]; ok {
			t.Fatalf("payload must not wrap expires_at: %s", raw)
		}
	}
}

func TestSet_TTLPassesWithExpiresAt(t *testing.T) {
	t.Parallel()
	b := newMemBackend()
	var lastExp *time.Time
	b.lastSetExpiresAt = &lastExp
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	if err := s.Set(ctx, "e", sscachian.Entry{Value: 1}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if lastExp == nil || !lastExp.After(time.Now().UTC()) {
		t.Fatalf("expected future expiresAt, got %v", lastExp)
	}
}

func TestGet_LegacyEntryJSONFallback(t *testing.T) {
	t.Parallel()
	b := newMemBackend()
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	_ = b.Set(ctx, "old-ok", json.RawMessage(`{"value":"hi","expires_at":"`+future+`"}`), nil)
	_ = b.Set(ctx, "old-exp", json.RawMessage(`{"value":"bye","expires_at":"`+past+`"}`), nil)
	_ = b.Set(ctx, "old-forever", json.RawMessage(`{"value":42}`), nil)
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })

	e, ok, err := s.Get(ctx, "old-ok")
	if err != nil || !ok || e.Value != "hi" {
		t.Fatalf("old-ok: ok=%v val=%v err=%v", ok, e.Value, err)
	}
	if _, ok, err := s.Get(ctx, "old-exp"); err != nil || ok {
		t.Fatalf("old-exp expected miss: ok=%v err=%v", ok, err)
	}
	e, ok, err = s.Get(ctx, "old-forever")
	if err != nil || !ok {
		t.Fatalf("old-forever: ok=%v err=%v", ok, err)
	}
	if n, ok := e.Value.(float64); !ok || n != 42 {
		t.Fatalf("old-forever value=%v", e.Value)
	}
}

func TestIncr_ClearsExpiry(t *testing.T) {
	t.Parallel()
	b := newMemBackend()
	ctx := context.Background()
	future := time.Now().UTC().Add(time.Hour)
	_ = b.Set(ctx, "c", json.RawMessage(`1`), &future)
	s := es4driver.NewWithBackend(b)
	t.Cleanup(func() { _ = s.Close() })
	n, err := s.Incr(ctx, "c")
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	b.mu.Lock()
	e := b.m["c"]
	b.mu.Unlock()
	if e.expiresAt != nil {
		t.Fatalf("expected expiry cleared, got %v", *e.expiresAt)
	}
}
