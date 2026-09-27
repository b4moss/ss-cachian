package sscachian

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
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
	// PurgeExact deletes all version data keys "{logicalPrefix}:{n}" (n digits ≥1 shape).
	// It must not delete "{logicalPrefix}:__version__".
	PurgeExact(ctx context.Context, logicalPrefix string) error
}

// IsVersionDataKey reports whether key is "{logicalPrefix}:{digits}" (digits non-empty).
// Keys like "{logicalPrefix}:__version__" are not version data keys.
func IsVersionDataKey(logicalPrefix, key string) bool {
	if logicalPrefix == "" || key == "" {
		return false
	}
	want := logicalPrefix + ":"
	if !strings.HasPrefix(key, want) {
		return false
	}
	rest := key[len(want):]
	if rest == "" {
		return false
	}
	for _, r := range rest {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

var (
	ErrEmptyKey          = errors.New("sscachian: empty key")
	ErrNegativeTTL       = errors.New("sscachian: negative TTL")
	ErrNotInteger        = errors.New("sscachian: value is not an integer")
	ErrIncrOverflow      = errors.New("sscachian: incr overflow")
	ErrNoLayer           = errors.New("sscachian: no layers configured")
	ErrNoKeyBuilder      = errors.New("sscachian: key builder not configured")
	ErrInvalidVersion    = errors.New("sscachian: version must be >= 1")
	ErrNoLoader          = errors.New("sscachian: loader not configured")
	ErrTypeMismatch      = errors.New("sscachian: cached value type mismatch")
	ErrInvalidContext    = errors.New("sscachian: invalid key context")
	ErrCorruptVersion    = errors.New("sscachian: corrupt current-version value")
	ErrInvalidLayerIndex = errors.New("sscachian: invalid layer index")
	ErrInvalidConfig     = errors.New("sscachian: invalid config")
)
