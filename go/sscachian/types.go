package sscachian

import (
	"context"
	"errors"
	"time"
)

// Entry is the on-Layer storage wrapper.
type Entry struct {
	Value     any
	CreatedAt time.Time
	ExpiresAt time.Time // zero means no expiry
}

// Layer is the thin storage contract for PoC.
type Layer interface {
	Get(ctx context.Context, key string) (Entry, bool, error)
	Set(ctx context.Context, key string, entry Entry, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int64, error)
}

var (
	ErrEmptyKey       = errors.New("sscachian: empty key")
	ErrNegativeTTL    = errors.New("sscachian: negative TTL")
	ErrNotInteger     = errors.New("sscachian: value is not an integer")
	ErrIncrOverflow   = errors.New("sscachian: incr overflow")
	ErrNoLayer        = errors.New("sscachian: no layers configured")
	ErrNoKeyBuilder   = errors.New("sscachian: key builder not configured")
	ErrInvalidVersion = errors.New("sscachian: version must be >= 1")
	ErrNoLoader       = errors.New("sscachian: loader not configured")
	ErrTypeMismatch   = errors.New("sscachian: cached value type mismatch")
	ErrInvalidContext = errors.New("sscachian: invalid key context")
	ErrCorruptVersion = errors.New("sscachian: corrupt current-version value")
)
