package es4_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	es4lib "github.com/b4moss/es4/packages/go/pkg/es4"
)

// memBackend is an in-process backend with Scan for unit tests.
type memBackend struct {
	mu     sync.Mutex
	m      map[string]json.RawMessage
	closed bool
}

func newMemBackend() *memBackend {
	return &memBackend{m: make(map[string]json.RawMessage)}
}

func (b *memBackend) guard() error {
	if b.closed {
		return es4lib.ErrClosed
	}
	return nil
}

func (b *memBackend) Set(_ context.Context, key string, value json.RawMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return err
	}
	cp := append(json.RawMessage(nil), value...)
	b.m[key] = cp
	return nil
}

func (b *memBackend) Get(_ context.Context, key string) (json.RawMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return nil, err
	}
	v, ok := b.m[key]
	if !ok {
		return nil, es4lib.ErrNotFound
	}
	return append(json.RawMessage(nil), v...), nil
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
	for k := range b.m {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	return out, nil
}

func (b *memBackend) BeginTx(ctx context.Context) (es4lib.Tx, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.guard(); err != nil {
		return nil, err
	}
	snap := make(map[string]json.RawMessage, len(b.m))
	for k, v := range b.m {
		snap[k] = append(json.RawMessage(nil), v...)
	}
	return &memTx{b: b, overlay: make(map[string]*json.RawMessage), snap: snap}, nil
}

func (b *memBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

type memTx struct {
	b       *memBackend
	overlay map[string]*json.RawMessage // nil pointer = deleted
	snap    map[string]json.RawMessage
	done    bool
}

func (t *memTx) getLocked(key string) (json.RawMessage, error) {
	if v, ok := t.overlay[key]; ok {
		if v == nil {
			return nil, es4lib.ErrNotFound
		}
		return append(json.RawMessage(nil), (*v)...), nil
	}
	v, ok := t.snap[key]
	if !ok {
		return nil, es4lib.ErrNotFound
	}
	return append(json.RawMessage(nil), v...), nil
}

func (t *memTx) Set(_ context.Context, key string, value json.RawMessage) error {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	cp := append(json.RawMessage(nil), value...)
	t.overlay[key] = &cp
	return nil
}

func (t *memTx) Get(_ context.Context, key string) (json.RawMessage, error) {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return nil, es4lib.ErrTxDone
	}
	return t.getLocked(key)
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

func (t *memTx) Exists(ctx context.Context, key string) (bool, error) {
	_, err := t.Get(ctx, key)
	if err == nil {
		return true, nil
	}
	if es4lib.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

func (t *memTx) Clear(_ context.Context) error {
	t.b.mu.Lock()
	defer t.b.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	t.overlay = make(map[string]*json.RawMessage)
	for k := range t.snap {
		t.overlay[k] = nil
	}
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
			t.b.m[k] = append(json.RawMessage(nil), (*v)...)
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
