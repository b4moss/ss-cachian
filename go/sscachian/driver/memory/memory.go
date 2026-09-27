package memory

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/b4moss/ss-cachian"
)

// Store is a process-local, goroutine-safe Layer.
type Store struct {
	mu sync.Mutex
	m  map[string]any
}

// New returns an empty in-memory Store.
func New() *Store {
	return &Store{m: make(map[string]any)}
}

// InjectRaw stores an arbitrary value for corrupt-entry tests.
func (s *Store) InjectRaw(key string, v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = v
}

func (s *Store) Get(_ context.Context, key string) (sscachian.Entry, bool, error) {
	if key == "" {
		return sscachian.Entry{}, false, sscachian.ErrEmptyKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, ok := s.m[key]
	if !ok {
		return sscachian.Entry{}, false, nil
	}
	e, ok := raw.(sscachian.Entry)
	if !ok {
		delete(s.m, key)
		return sscachian.Entry{}, false, nil
	}
	if !e.ExpiresAt.IsZero() && !e.ExpiresAt.After(time.Now().UTC()) {
		delete(s.m, key)
		return sscachian.Entry{}, false, nil
	}
	return e, true, nil
}

func (s *Store) Set(_ context.Context, key string, entry sscachian.Entry, ttl time.Duration) error {
	if key == "" {
		return sscachian.ErrEmptyKey
	}
	if ttl < 0 {
		return sscachian.ErrNegativeTTL
	}
	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if ttl > 0 {
		entry.ExpiresAt = now.Add(ttl)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = entry
	return nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	if key == "" {
		return sscachian.ErrEmptyKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	return nil
}

func (s *Store) Incr(_ context.Context, key string) (int64, error) {
	if key == "" {
		return 0, sscachian.ErrEmptyKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, ok := s.m[key]
	var n int64
	if ok {
		e, isEntry := raw.(sscachian.Entry)
		if !isEntry {
			return 0, sscachian.ErrNotInteger
		}
		switch v := e.Value.(type) {
		case int64:
			n = v
		case int:
			n = int64(v)
		default:
			return 0, sscachian.ErrNotInteger
		}
	}
	if n == math.MaxInt64 {
		return 0, sscachian.ErrIncrOverflow
	}
	n++
	s.m[key] = sscachian.Entry{Value: n, CreatedAt: time.Now().UTC()}
	return n, nil
}

// Interface guard.
var _ sscachian.Layer = (*Store)(nil)
