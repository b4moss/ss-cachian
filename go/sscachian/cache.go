package sscachian

import (
	"context"
	"fmt"
	"log"
	"time"
)

// KeyContext carries business fields for key building.
type KeyContext struct {
	AppSlug   string
	TenantID  string
	QueryType string
}

// KeyBuilder maps a context to the logical key prefix
// (without trailing :{version} or :__version__).
type KeyBuilder func(ctx context.Context, kc KeyContext) (string, error)

// Loader loads a value on cache miss (read-through).
type Loader[T any] func(ctx context.Context, kc KeyContext) (T, error)

// DefaultKeyBuilder builds "{app}:cache:{tenant}:{query}".
func DefaultKeyBuilder(_ context.Context, kc KeyContext) (string, error) {
	if kc.AppSlug == "" || kc.TenantID == "" || kc.QueryType == "" {
		return "", ErrInvalidContext
	}
	return fmt.Sprintf("%s:cache:%s:%s", kc.AppSlug, kc.TenantID, kc.QueryType), nil
}

// CacheType is a typed cache strategy bound to layers.
type CacheType[T any] struct {
	name   string
	layers []Layer
	ttl    time.Duration
	kb     KeyBuilder
	loader Loader[T]
}

// Builder constructs a CacheType.
type Builder[T any] struct {
	name   string
	layers []Layer
	ttl    time.Duration
	kb     KeyBuilder
	loader Loader[T]
}

// Define starts a CacheType builder.
func Define[T any](name string) *Builder[T] {
	return &Builder[T]{name: name, kb: DefaultKeyBuilder}
}

func (b *Builder[T]) WithLayers(layers ...Layer) *Builder[T] {
	b.layers = append([]Layer(nil), layers...)
	return b
}

func (b *Builder[T]) WithKeyBuilder(kb KeyBuilder) *Builder[T] {
	b.kb = kb
	return b
}

func (b *Builder[T]) WithLayerTTL(ttl time.Duration) *Builder[T] {
	b.ttl = ttl
	return b
}

func (b *Builder[T]) WithPolicy(ttl time.Duration) *Builder[T] {
	return b.WithLayerTTL(ttl)
}

func (b *Builder[T]) WithLoader(loader Loader[T]) *Builder[T] {
	b.loader = loader
	return b
}

func (b *Builder[T]) Build() (*CacheType[T], error) {
	if len(b.layers) == 0 {
		return nil, ErrNoLayer
	}
	if b.kb == nil {
		return nil, ErrNoKeyBuilder
	}
	return &CacheType[T]{
		name:   b.name,
		layers: append([]Layer(nil), b.layers...),
		ttl:    b.ttl,
		kb:     b.kb,
		loader: b.loader,
	}, nil
}

func (c *CacheType[T]) l1() Layer { return c.layers[0] }

func (c *CacheType[T]) logicalPrefix(ctx context.Context, kc KeyContext) (string, error) {
	return c.kb(ctx, kc)
}

// BuildKey returns the data key for an explicit version (>= 1).
func (c *CacheType[T]) BuildKey(ctx context.Context, kc KeyContext, version int64) (string, error) {
	if version < 1 {
		return "", ErrInvalidVersion
	}
	prefix, err := c.logicalPrefix(ctx, kc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", prefix, version), nil
}

// VersionKey returns the __version__ key for the logical scope.
func (c *CacheType[T]) VersionKey(ctx context.Context, kc KeyContext) (string, error) {
	prefix, err := c.logicalPrefix(ctx, kc)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:__version__", prefix), nil
}

func asInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	default:
		return 0, ErrCorruptVersion
	}
}

// ensureCurrentVersion creates __version__=1 when missing; returns the current value.
func (c *CacheType[T]) ensureCurrentVersion(ctx context.Context, kc KeyContext) (int64, error) {
	vk, err := c.VersionKey(ctx, kc)
	if err != nil {
		return 0, err
	}
	e, ok, err := c.l1().Get(ctx, vk)
	if err != nil {
		return 0, err
	}
	if ok {
		return asInt64(e.Value)
	}
	if err := c.l1().Set(ctx, vk, Entry{Value: int64(1)}, 0); err != nil {
		return 0, err
	}
	return 1, nil
}

// CurrentVersion returns the L1 current-version, initializing to 1 when absent.
func (c *CacheType[T]) CurrentVersion(ctx context.Context, kc KeyContext) (int64, error) {
	return c.ensureCurrentVersion(ctx, kc)
}

// BumpVersion best-effort increments current-version (read → +1 → write via Incr).
func (c *CacheType[T]) BumpVersion(ctx context.Context, kc KeyContext) (int64, error) {
	vk, err := c.VersionKey(ctx, kc)
	if err != nil {
		return 0, err
	}
	// Ensure a starting point exists so first bump yields 2 from 1, or 1→2 via incr from missing→1 then...
	// Spec: missing → create 1 then become 2. Using ensure then Incr: 1 → 2.
	if _, err := c.ensureCurrentVersion(ctx, kc); err != nil {
		return 0, err
	}
	return c.l1().Incr(ctx, vk)
}

// BuildLatestKey resolves current-version then builds the data key.
func (c *CacheType[T]) BuildLatestKey(ctx context.Context, kc KeyContext) (string, error) {
	v, err := c.ensureCurrentVersion(ctx, kc)
	if err != nil {
		return "", err
	}
	return c.BuildKey(ctx, kc, v)
}

func castValue[T any](v any) (T, error) {
	var zero T
	if v == nil {
		// Allow nil only when T is a pointer/interface — try assertion.
		tv, ok := any(v).(T)
		if !ok {
			return zero, ErrTypeMismatch
		}
		return tv, nil
	}
	tv, ok := v.(T)
	if !ok {
		return zero, ErrTypeMismatch
	}
	return tv, nil
}

// Get returns the latest-version value from L1. ok=false means miss.
func (c *CacheType[T]) Get(ctx context.Context, kc KeyContext) (T, bool, error) {
	var zero T
	key, err := c.BuildLatestKey(ctx, kc)
	if err != nil {
		return zero, false, err
	}
	e, ok, err := c.l1().Get(ctx, key)
	if err != nil {
		return zero, false, err
	}
	if !ok {
		return zero, false, nil
	}
	tv, err := castValue[T](e.Value)
	if err != nil {
		return zero, false, err
	}
	return tv, true, nil
}

// Set writes under the post-bump latest version (bump then write) and returns the new version.
func (c *CacheType[T]) Set(ctx context.Context, kc KeyContext, value T) error {
	if _, err := c.ensureCurrentVersion(ctx, kc); err != nil {
		return err
	}
	newV, err := c.BumpVersion(ctx, kc)
	if err != nil {
		return err
	}
	key, err := c.BuildKey(ctx, kc, newV)
	if err != nil {
		return err
	}
	return c.l1().Set(ctx, key, Entry{Value: value}, c.ttl)
}

// Delete removes the current latest data key, then bumps version. Idempotent.
func (c *CacheType[T]) Delete(ctx context.Context, kc KeyContext) error {
	v, err := c.ensureCurrentVersion(ctx, kc)
	if err != nil {
		return err
	}
	key, err := c.BuildKey(ctx, kc, v)
	if err != nil {
		return err
	}
	if err := c.l1().Delete(ctx, key); err != nil {
		return err
	}
	_, err = c.BumpVersion(ctx, kc)
	return err
}

// GetOrLoad returns cached value or loads, writes back to L1 without bumping.
func (c *CacheType[T]) GetOrLoad(ctx context.Context, kc KeyContext) (T, error) {
	var zero T
	v, ok, err := c.Get(ctx, kc)
	if err != nil {
		return zero, err
	}
	if ok {
		return v, nil
	}
	if c.loader == nil {
		return zero, ErrNoLoader
	}
	loaded, err := c.loader(ctx, kc)
	if err != nil {
		return zero, err
	}
	ver, err := c.ensureCurrentVersion(ctx, kc)
	if err != nil {
		return zero, err
	}
	key, err := c.BuildKey(ctx, kc, ver)
	if err != nil {
		return zero, err
	}
	if err := c.l1().Set(ctx, key, Entry{Value: loaded}, c.ttl); err != nil {
		log.Printf("sscachian: write-back failed: %v", err)
		return loaded, nil
	}
	return loaded, nil
}

// Name returns the cache type name (debug).
func (c *CacheType[T]) Name() string { return c.name }
