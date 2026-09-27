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
	ttls   []time.Duration
	kb     KeyBuilder
	loader Loader[T]
}

// Builder constructs a CacheType.
type Builder[T any] struct {
	name   string
	layers []Layer
	ttls   []time.Duration
	single *time.Duration // WithLayerTTL / WithPolicy compatibility
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

// WithLayerTTL applies the same TTL to every layer (backward compatible).
func (b *Builder[T]) WithLayerTTL(ttl time.Duration) *Builder[T] {
	t := ttl
	b.single = &t
	return b
}

func (b *Builder[T]) WithPolicy(ttl time.Duration) *Builder[T] {
	return b.WithLayerTTL(ttl)
}

// WithLayerTTLs sets per-layer TTLs. Missing entries default to 0; extras ignored.
func (b *Builder[T]) WithLayerTTLs(ttls ...time.Duration) *Builder[T] {
	b.ttls = append([]time.Duration(nil), ttls...)
	b.single = nil
	return b
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
	ttls := make([]time.Duration, len(b.layers))
	if b.single != nil {
		if *b.single < 0 {
			return nil, ErrNegativeTTL
		}
		for i := range ttls {
			ttls[i] = *b.single
		}
	} else {
		for i := range ttls {
			if i < len(b.ttls) {
				if b.ttls[i] < 0 {
					return nil, ErrNegativeTTL
				}
				ttls[i] = b.ttls[i]
			}
		}
	}
	return &CacheType[T]{
		name:   b.name,
		layers: append([]Layer(nil), b.layers...),
		ttls:   ttls,
		kb:     b.kb,
		loader: b.loader,
	}, nil
}

func (c *CacheType[T]) l1() Layer { return c.layers[0] }

func (c *CacheType[T]) layerTTL(i int) time.Duration {
	if i < 0 || i >= len(c.ttls) {
		return 0
	}
	return c.ttls[i]
}

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
	case float64:
		if n == float64(int64(n)) {
			return int64(n), nil
		}
		return 0, ErrCorruptVersion
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

// BumpVersion best-effort increments current-version on L1.
func (c *CacheType[T]) BumpVersion(ctx context.Context, kc KeyContext) (int64, error) {
	vk, err := c.VersionKey(ctx, kc)
	if err != nil {
		return 0, err
	}
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
	tv, ok := v.(T)
	if !ok {
		return zero, ErrTypeMismatch
	}
	return tv, nil
}

func (c *CacheType[T]) writeBack(ctx context.Context, key string, e Entry, upToExclusive int) {
	for j := 0; j < upToExclusive; j++ {
		if err := c.layers[j].Set(ctx, key, e, c.layerTTL(j)); err != nil {
			log.Printf("sscachian: write-back to layer %d failed: %v", j, err)
		}
	}
}

func (c *CacheType[T]) writeAll(ctx context.Context, key string, e Entry) {
	for i := range c.layers {
		if err := c.layers[i].Set(ctx, key, e, c.layerTTL(i)); err != nil {
			log.Printf("sscachian: write-back to layer %d failed: %v", i, err)
		}
	}
}

// Get walks L1→Ln for the latest-version key and write-backs to upper layers on hit.
func (c *CacheType[T]) Get(ctx context.Context, kc KeyContext) (T, bool, error) {
	var zero T
	key, err := c.BuildLatestKey(ctx, kc)
	if err != nil {
		return zero, false, err
	}
	for i, layer := range c.layers {
		e, ok, err := layer.Get(ctx, key)
		if err != nil {
			return zero, false, err
		}
		if !ok {
			continue
		}
		c.writeBack(ctx, key, e, i)
		tv, err := castValue[T](e.Value)
		if err != nil {
			return zero, false, err
		}
		return tv, true, nil
	}
	return zero, false, nil
}

// Set bumps version on L1 then writes the new latest key to all layers.
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
	entry := Entry{Value: value}
	if err := c.l1().Set(ctx, key, entry, c.layerTTL(0)); err != nil {
		return err
	}
	for i := 1; i < len(c.layers); i++ {
		if err := c.layers[i].Set(ctx, key, entry, c.layerTTL(i)); err != nil {
			log.Printf("sscachian: set layer %d failed: %v", i, err)
		}
	}
	return nil
}

// Delete removes the latest key from all layers, then bumps L1 version.
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
	for i := 1; i < len(c.layers); i++ {
		if err := c.layers[i].Delete(ctx, key); err != nil {
			log.Printf("sscachian: delete layer %d failed: %v", i, err)
		}
	}
	_, err = c.BumpVersion(ctx, kc)
	return err
}

// GetOrLoad performs multilayer Get; on miss loads and write-backs to all layers without bump.
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
	c.writeAll(ctx, key, Entry{Value: loaded})
	return loaded, nil
}

// Name returns the cache type name (debug).
func (c *CacheType[T]) Name() string { return c.name }

// Layers returns a copy of configured layers (tests).
func (c *CacheType[T]) Layers() []Layer {
	out := make([]Layer, len(c.layers))
	copy(out, c.layers)
	return out
}

// LayerTTL returns the TTL for layer index i (tests).
func (c *CacheType[T]) LayerTTL(i int) time.Duration { return c.layerTTL(i) }
