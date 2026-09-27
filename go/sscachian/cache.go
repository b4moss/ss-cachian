package sscachian

import (
	"context"
	"fmt"
	"log"
	"sort"
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
	name        string
	layers      []Layer
	ttls        []time.Duration
	kb          KeyBuilder
	loader      Loader[T]
	defaultTags []string
}

// Builder constructs a CacheType.
type Builder[T any] struct {
	name        string
	layers      []Layer
	ttls        []time.Duration
	single      *time.Duration // WithLayerTTL / WithPolicy compatibility
	kb          KeyBuilder
	loader      Loader[T]
	defaultTags []string
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

// WithDefaultTags sets type-wide tags merged into every tagged write.
func (b *Builder[T]) WithDefaultTags(tags ...string) *Builder[T] {
	b.defaultTags = append([]string{}, tags...)
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
		name:        b.name,
		layers:      append([]Layer(nil), b.layers...),
		ttls:        ttls,
		kb:          b.kb,
		loader:      b.loader,
		defaultTags: append([]string{}, b.defaultTags...),
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

func (c *CacheType[T]) writeAllTTL(ctx context.Context, key string, e Entry, ttl time.Duration) {
	for i := range c.layers {
		if err := c.layers[i].Set(ctx, key, e, ttl); err != nil {
			log.Printf("sscachian: write-back to layer %d failed: %v", i, err)
		}
	}
}

func prefixesFromTagValue(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string{}, x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (c *CacheType[T]) recordTags(ctx context.Context, logicalPrefix string, tags []string) {
	tags = mergeTags(nil, tags)
	if len(tags) == 0 || logicalPrefix == "" {
		return
	}
	for _, tag := range tags {
		tk := tagIndexKey(tag)
		e, ok, err := c.l1().Get(ctx, tk)
		if err != nil {
			log.Printf("sscachian: tag index get failed: %v", err)
			continue
		}
		set := map[string]struct{}{}
		if ok {
			for _, p := range prefixesFromTagValue(e.Value) {
				set[p] = struct{}{}
			}
		}
		set[logicalPrefix] = struct{}{}
		list := make([]string, 0, len(set))
		for p := range set {
			list = append(list, p)
		}
		sort.Strings(list)
		if err := c.l1().Set(ctx, tk, Entry{Value: list}, 0); err != nil {
			log.Printf("sscachian: tag index set failed: %v", err)
		}
	}
}

func (c *CacheType[T]) readTagPrefixes(ctx context.Context, tag string) ([]string, error) {
	e, ok, err := c.l1().Get(ctx, tagIndexKey(tag))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return prefixesFromTagValue(e.Value), nil
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

// GetEntry is like Get but also returns created_at / expires_at meta.
func (c *CacheType[T]) GetEntry(ctx context.Context, kc KeyContext) (CacheEntry[T], bool, error) {
	var zero CacheEntry[T]
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
		return CacheEntry[T]{Value: tv, CreatedAt: e.CreatedAt, ExpiresAt: e.ExpiresAt}, true, nil
	}
	return zero, false, nil
}

// Has reports whether the latest key is present (Get then ok).
func (c *CacheType[T]) Has(ctx context.Context, kc KeyContext) (bool, error) {
	_, ok, err := c.Get(ctx, kc)
	return ok, err
}

// Exists is an alias of Has.
func (c *CacheType[T]) Exists(ctx context.Context, kc KeyContext) (bool, error) {
	return c.Has(ctx, kc)
}

// Set bumps version on L1 then writes the new latest key to all layers.
func (c *CacheType[T]) Set(ctx context.Context, kc KeyContext, value T, opts ...WriteOption) error {
	wo := applyWriteOpts(opts)
	tags := mergeTags(c.defaultTags, wo.tags)
	prefix, err := c.logicalPrefix(ctx, kc)
	if err != nil {
		return err
	}
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
	c.recordTags(ctx, prefix, tags)
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

// Forget is an alias of Delete.
func (c *CacheType[T]) Forget(ctx context.Context, kc KeyContext) error {
	return c.Delete(ctx, kc)
}

// Purge removes all version data keys for the logical key on selected layers.
// Omitting layerIdx purges every layer. Does not touch __version__ or bump.
func (c *CacheType[T]) Purge(ctx context.Context, kc KeyContext, layerIdx ...int) error {
	prefix, err := c.logicalPrefix(ctx, kc)
	if err != nil {
		return err
	}
	return c.purgeExactOnPrefix(ctx, prefix, layerIdx...)
}

// PurgeExact is an alias of Purge (Exact semantics).
func (c *CacheType[T]) PurgeExact(ctx context.Context, kc KeyContext, layerIdx ...int) error {
	return c.Purge(ctx, kc, layerIdx...)
}

// PurgePrefix deletes all keys starting with prefix on selected layers.
func (c *CacheType[T]) PurgePrefix(ctx context.Context, prefix string, layerIdx ...int) error {
	if prefix == "" {
		return ErrEmptyKey
	}
	targets, err := c.resolveLayerIndexes(layerIdx...)
	if err != nil {
		return err
	}
	includesL1 := false
	for _, i := range targets {
		if i == 0 {
			includesL1 = true
			break
		}
	}
	for _, i := range targets {
		if err := c.layers[i].PurgePrefix(ctx, prefix); err != nil {
			if includesL1 {
				if i == 0 {
					return err
				}
				log.Printf("sscachian: purgePrefix layer %d failed: %v", i, err)
				continue
			}
			return err
		}
	}
	return nil
}

// PurgeTag Exact-purges every logical prefix indexed under tag, then drops the index.
func (c *CacheType[T]) PurgeTag(ctx context.Context, tag string, layerIdx ...int) error {
	if tag == "" {
		return ErrEmptyKey
	}
	prefixes, err := c.readTagPrefixes(ctx, tag)
	if err != nil {
		return err
	}
	for _, p := range prefixes {
		if err := c.purgeExactOnPrefix(ctx, p, layerIdx...); err != nil {
			return err
		}
	}
	if err := c.l1().Delete(ctx, tagIndexKey(tag)); err != nil {
		return err
	}
	return nil
}

func (c *CacheType[T]) purgeExactOnPrefix(ctx context.Context, prefix string, layerIdx ...int) error {
	targets, err := c.resolveLayerIndexes(layerIdx...)
	if err != nil {
		return err
	}
	includesL1 := false
	for _, i := range targets {
		if i == 0 {
			includesL1 = true
			break
		}
	}
	for _, i := range targets {
		if err := c.layers[i].PurgeExact(ctx, prefix); err != nil {
			if includesL1 {
				if i == 0 {
					return err
				}
				log.Printf("sscachian: purge layer %d failed: %v", i, err)
				continue
			}
			return err
		}
	}
	return nil
}

func (c *CacheType[T]) resolveLayerIndexes(layerIdx ...int) ([]int, error) {
	n := len(c.layers)
	if len(layerIdx) == 0 {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out, nil
	}
	seen := make(map[int]struct{}, len(layerIdx))
	out := make([]int, 0, len(layerIdx))
	for _, i := range layerIdx {
		if i < 0 || i >= n {
			return nil, ErrInvalidLayerIndex
		}
		if _, ok := seen[i]; ok {
			continue
		}
		seen[i] = struct{}{}
		out = append(out, i)
	}
	sort.Ints(out)
	return out, nil
}

// GetOrLoad performs multilayer Get; on miss loads and write-backs to all layers without bump.
func (c *CacheType[T]) GetOrLoad(ctx context.Context, kc KeyContext, opts ...WriteOption) (T, error) {
	var zero T
	wo := applyWriteOpts(opts)
	tags := mergeTags(c.defaultTags, wo.tags)
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
	prefix, err := c.logicalPrefix(ctx, kc)
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
	c.recordTags(ctx, prefix, tags)
	return loaded, nil
}

// Remember is Get-or-fill with an explicit TTL (Bump=false). Negative TTL is rejected.
func (c *CacheType[T]) Remember(ctx context.Context, kc KeyContext, ttl time.Duration, loader Loader[T], opts ...WriteOption) (T, error) {
	var zero T
	if ttl < 0 {
		return zero, ErrNegativeTTL
	}
	wo := applyWriteOpts(opts)
	tags := mergeTags(c.defaultTags, wo.tags)
	v, ok, err := c.Get(ctx, kc)
	if err != nil {
		return zero, err
	}
	if ok {
		return v, nil
	}
	if loader == nil {
		return zero, ErrNoLoader
	}
	loaded, err := loader(ctx, kc)
	if err != nil {
		return zero, err
	}
	prefix, err := c.logicalPrefix(ctx, kc)
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
	c.writeAllTTL(ctx, key, Entry{Value: loaded}, ttl)
	c.recordTags(ctx, prefix, tags)
	return loaded, nil
}

// RememberForever is Remember with TTL 0 (no expiry).
func (c *CacheType[T]) RememberForever(ctx context.Context, kc KeyContext, loader Loader[T], opts ...WriteOption) (T, error) {
	return c.Remember(ctx, kc, 0, loader, opts...)
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
