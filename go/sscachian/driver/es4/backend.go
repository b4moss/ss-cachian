package es4

import (
	"context"
	"encoding/json"
	"fmt"

	es4lib "github.com/b4moss/es4/packages/go/pkg/es4"
)

// Backend is the es4 State surface the Layer needs (embedded or HTTP).
type Backend interface {
	Set(ctx context.Context, key string, value json.RawMessage) error
	Get(ctx context.Context, key string) (json.RawMessage, error)
	Delete(ctx context.Context, key string) error
	BeginTx(ctx context.Context) (es4lib.Tx, error)
	Scan(ctx context.Context, prefix string) ([]string, error)
	Close() error
}

// scanner is satisfied when the underlying es4 DB exposes Scan (v0.10+).
type scanner interface {
	Scan(ctx context.Context, prefix string) ([]string, error)
}

type embeddedBackend struct {
	db *es4lib.DB
}

func (b *embeddedBackend) Set(ctx context.Context, key string, value json.RawMessage) error {
	return b.db.Set(ctx, key, value)
}

func (b *embeddedBackend) Get(ctx context.Context, key string) (json.RawMessage, error) {
	return b.db.Get(ctx, key)
}

func (b *embeddedBackend) Delete(ctx context.Context, key string) error {
	return b.db.Delete(ctx, key)
}

func (b *embeddedBackend) BeginTx(ctx context.Context) (es4lib.Tx, error) {
	return b.db.BeginTx(ctx)
}

func (b *embeddedBackend) Scan(ctx context.Context, prefix string) ([]string, error) {
	if s, ok := any(b.db).(scanner); ok {
		return s.Scan(ctx, prefix)
	}
	return nil, fmt.Errorf("sscachian/es4: Scan not available on es4.DB (need es4 v0.10+ with Scan)")
}

func (b *embeddedBackend) Close() error {
	return b.db.Close()
}
