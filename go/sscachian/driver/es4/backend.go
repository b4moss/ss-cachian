package es4

import (
	"context"
	"encoding/json"
	"time"

	es4lib "github.com/b4moss/es4/packages/go/pkg/es4"
)

// Backend is the es4 State surface the Layer needs (embedded or HTTP).
type Backend interface {
	// Set writes value. When expiresAt != nil, the absolute expiry is applied
	// (es4 WithExpiresAt / HTTP ?expires_at=). nil means no expiry (field cleared).
	Set(ctx context.Context, key string, value json.RawMessage, expiresAt *time.Time) error
	Get(ctx context.Context, key string) (json.RawMessage, error)
	Delete(ctx context.Context, key string) error
	BeginTx(ctx context.Context) (Tx, error)
	Scan(ctx context.Context, prefix string) ([]string, error)
	Close() error
}

// Tx is a Backend transaction (mirrors es4 Tx without opaque SetOption).
type Tx interface {
	Set(ctx context.Context, key string, value json.RawMessage, expiresAt *time.Time) error
	Get(ctx context.Context, key string) (json.RawMessage, error)
	Delete(ctx context.Context, key string) error
	Commit() error
	Rollback() error
}

type embeddedBackend struct {
	db *es4lib.DB
}

func (b *embeddedBackend) Set(ctx context.Context, key string, value json.RawMessage, expiresAt *time.Time) error {
	if expiresAt != nil {
		return b.db.Set(ctx, key, value, es4lib.WithExpiresAt(*expiresAt))
	}
	return b.db.Set(ctx, key, value)
}

func (b *embeddedBackend) Get(ctx context.Context, key string) (json.RawMessage, error) {
	return b.db.Get(ctx, key)
}

func (b *embeddedBackend) Delete(ctx context.Context, key string) error {
	return b.db.Delete(ctx, key)
}

func (b *embeddedBackend) BeginTx(ctx context.Context) (Tx, error) {
	inner, err := b.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return &embeddedTx{inner: inner}, nil
}

func (b *embeddedBackend) Scan(ctx context.Context, prefix string) ([]string, error) {
	return b.db.Scan(ctx, prefix)
}

func (b *embeddedBackend) Close() error {
	return b.db.Close()
}

type embeddedTx struct {
	inner es4lib.Tx
}

func (t *embeddedTx) Set(ctx context.Context, key string, value json.RawMessage, expiresAt *time.Time) error {
	if expiresAt != nil {
		return t.inner.Set(ctx, key, value, es4lib.WithExpiresAt(*expiresAt))
	}
	return t.inner.Set(ctx, key, value)
}

func (t *embeddedTx) Get(ctx context.Context, key string) (json.RawMessage, error) {
	return t.inner.Get(ctx, key)
}

func (t *embeddedTx) Delete(ctx context.Context, key string) error {
	return t.inner.Delete(ctx, key)
}

func (t *embeddedTx) Commit() error   { return t.inner.Commit() }
func (t *embeddedTx) Rollback() error { return t.inner.Rollback() }
