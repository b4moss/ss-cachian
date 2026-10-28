package es4_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	es4lib "github.com/b4moss/es4/packages/go/pkg/es4"
	es4driver "github.com/b4moss/ss-cachian/driver/es4"
)

var _ es4driver.Backend = (*memBackend)(nil)
var _ es4driver.Tx = (*memTx)(nil)

type memEntry struct {
	value     json.RawMessage
	expiresAt *time.Time
}

// memBackend is an in-process backend with Scan + expiry for unit tests.
type memBackend struct {
	mu     sync.Mutex
	m      map[string]memEntry
	closed bool
	// lastSetExpiresAt records the expiresAt pointer passed to the latest Set (tests).
	lastSetExpiresAt **time.Time
}

func newMemBackend() *memBackend {
	return &memBackend{m: make(map[string]memEntry)}
}

func (b *memBackend) guard() error {
	if b.closed {
		return es4lib.ErrClosed
	}
	return nil
}

func (b *memBackend) alive(e memEntry) bool {
	if e.expiresAt == nil {
		return true
	}
	return e.expiresAt.After(time.Now().UTC())
}

func (b *memBackend) Set(_ context.Context, key string, value json.RawMessage, expiresAt *time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return err
	}
	if b.lastSetExpiresAt != nil {
		*b.lastSetExpiresAt = expiresAt
	}
	cp := append(json.RawMessage(nil), value...)
	var expCopy *time.Time
	if expiresAt != nil {
		u := expiresAt.UTC()
		expCopy = &u
	}
	b.m[key] = memEntry{value: cp, expiresAt: expCopy}
	return nil
}

func (b *memBackend) Get(_ context.Context, key string) (json.RawMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return nil, err
	}
	e, ok := b.m[key]
	if !ok || !b.alive(e) {
		return nil, es4lib.ErrNotFound
	}
	return append(json.RawMessage(nil), e.value...), nil
}

func (b *memBackend) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return err
	}
	if _, ok := b.m[key]; !ok {
		return es4lib.ErrNotFound
	}
	delete(b.m, key)
	return nil
}

func (b *memBackend) Scan(_ context.Context, prefix string) ([]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return nil, err
	}
	var out []string
	for k, e := range b.m {
		if !b.alive(e) {
			continue
		}
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	return out, nil
}

func (b *memBackend) BeginTx(ctx context.Context) (es4driver.Tx, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return nil, err
	}
	snap := make(map[string]memEntry, len(b.m))
	for k, v := range b.m {
		cp := append(json.RawMessage(nil), v.value...)
		var expCopy *time.Time
		if v.expiresAt != nil {
			u := *v.expiresAt
			expCopy = &u
		}
		snap[k] = memEntry{value: cp, expiresAt: expCopy}
	}
	return &memTx{b: b, overlay: make(map[string]*memEntry), snap: snap}, nil
}

func (b *memBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

type memTx struct {
	b       *memBackend
	overlay map[string]*memEntry // nil pointer = deleted
	snap    map[string]memEntry
	done    bool
}

func (t *memTx) getLocked(key string) (memEntry, error) {
	if v, ok := t.overlay[key]; ok {
		if v == nil {
			return memEntry{}, es4lib.ErrNotFound
		}
		if v.expiresAt != nil && !v.expiresAt.After(time.Now().UTC()) {
			return memEntry{}, es4lib.ErrNotFound
		}
		cp := append(json.RawMessage(nil), v.value...)
		return memEntry{value: cp, expiresAt: v.expiresAt}, nil
	}
	v, ok := t.snap[key]
	if !ok {
		return memEntry{}, es4lib.ErrNotFound
	}
	if v.expiresAt != nil && !v.expiresAt.After(time.Now().UTC()) {
		return memEntry{}, es4lib.ErrNotFound
	}
	cp := append(json.RawMessage(nil), v.value...)
	return memEntry{value: cp, expiresAt: v.expiresAt}, nil
}

func (t *memTx) Set(_ context.Context, key string, value json.RawMessage, expiresAt *time.Time) error {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	cp := append(json.RawMessage(nil), value...)
	var expCopy *time.Time
	if expiresAt != nil {
		u := expiresAt.UTC()
		expCopy = &u
	}
	e := memEntry{value: cp, expiresAt: expCopy}
	t.overlay[key] = &e
	return nil
}

func (t *memTx) Get(_ context.Context, key string) (json.RawMessage, error) {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return nil, es4lib.ErrTxDone
	}
	e, err := t.getLocked(key)
	if err != nil {
		return nil, err
	}
	return e.value, nil
}

func (t *memTx) Delete(_ context.Context, key string) error {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	if _, err := t.getLocked(key); err != nil {
		return err
	}
	t.overlay[key] = nil
	return nil
}

func (t *memTx) Commit() error {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	for k, v := range t.overlay {
		if v == nil {
			delete(t.b.m, k)
		} else {
			cp := append(json.RawMessage(nil), v.value...)
			var expCopy *time.Time
			if v.expiresAt != nil {
				u := *v.expiresAt
				expCopy = &u
			}
			t.b.m[k] = memEntry{value: cp, expiresAt: expCopy}
		}
	}
	t.done = true
	return nil
}

func (t *memTx) Rollback() error {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	t.done = true
	return nil
}
