package es4

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	es4lib "github.com/b4moss/es4/packages/go/pkg/es4"
	"github.com/b4moss/es4/packages/go/pkg/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/b4moss/ss-cachian"
)

const incrMaxAttempts = 8

// Options configures New (embedded es4.Open). Driver owns Open/Close.
type Options struct {
	MemoryOnly bool

	StateBackend            string
	StatePath               string
	StateRedisURL           string
	StateRedisKeyPrefix     string
	StateFirestoreProjectID  string
	StateFirestoreDatabaseID string
	StateFirestoreCollection string
}

// Store is an es4-backed Layer.
type Store struct {
	backend Backend
	closed  atomic.Bool
	ownsDB  bool
}

// New opens an embedded es4 DB and returns a Layer. Caller must Close.
func New(ctx context.Context, opts Options) (*Store, error) {
	es4Opts := options.Options{
		MemoryOnly:              opts.MemoryOnly,
		StateBackend:            opts.StateBackend,
		StatePath:               opts.StatePath,
		StateRedisURL:           opts.StateRedisURL,
		StateRedisKeyPrefix:     opts.StateRedisKeyPrefix,
		StateFirestoreProjectID:  opts.StateFirestoreProjectID,
		StateFirestoreDatabaseID: opts.StateFirestoreDatabaseID,
		StateFirestoreCollection: opts.StateFirestoreCollection,
	}
	// Prefer memory_only for CI / default when nothing else is set.
	if !opts.MemoryOnly && opts.StateBackend == "" && opts.StatePath == "" {
		es4Opts.MemoryOnly = true
	}
	db, err := es4lib.Open(ctx, es4Opts)
	if err != nil {
		return nil, err
	}
	return &Store{backend: &embeddedBackend{db: db}, ownsDB: true}, nil
}

// NewWithBackend injects a Backend (tests / advanced wiring). Owns Close of backend.
func NewWithBackend(b Backend) *Store {
	return &Store{backend: b, ownsDB: true}
}

// Close closes the underlying es4 DB (idempotent).
func (s *Store) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	if s.ownsDB && s.backend != nil {
		return s.backend.Close()
	}
	return nil
}

func (s *Store) guard() error {
	if s.closed.Load() {
		return es4lib.ErrClosed
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (sscachian.Entry, bool, error) {
	if key == "" {
		return sscachian.Entry{}, false, sscachian.ErrEmptyKey
	}
	if err := s.guard(); err != nil {
		return sscachian.Entry{}, false, err
	}
	raw, err := s.backend.Get(ctx, key)
	if err != nil {
		if es4lib.IsNotFound(err) {
			return sscachian.Entry{}, false, nil
		}
		return sscachian.Entry{}, false, err
	}
	val, exp, ok := decodeEntry(raw)
	if !ok {
		_ = s.deleteIgnoreMissing(ctx, key)
		return sscachian.Entry{}, false, nil
	}
	if !exp.IsZero() && !exp.After(time.Now().UTC()) {
		_ = s.deleteIgnoreMissing(ctx, key)
		return sscachian.Entry{}, false, nil
	}
	return sscachian.Entry{Value: val, ExpiresAt: exp}, true, nil
}

func (s *Store) Set(ctx context.Context, key string, entry sscachian.Entry, ttl time.Duration) error {
	if key == "" {
		return sscachian.ErrEmptyKey
	}
	if ttl < 0 {
		return sscachian.ErrNegativeTTL
	}
	if err := s.guard(); err != nil {
		return err
	}
	now := time.Now().UTC()
	exp := entry.ExpiresAt
	if ttl > 0 {
		exp = now.Add(ttl)
	} else if ttl == 0 && entry.ExpiresAt.IsZero() {
		exp = time.Time{}
	}
	raw, err := encodeEntry(entry.Value, exp)
	if err != nil {
		return fmt.Errorf("sscachian/es4: marshal entry: %w", err)
	}
	return s.backend.Set(ctx, key, raw)
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if key == "" {
		return sscachian.ErrEmptyKey
	}
	if err := s.guard(); err != nil {
		return err
	}
	return s.deleteIgnoreMissing(ctx, key)
}

func (s *Store) deleteIgnoreMissing(ctx context.Context, key string) error {
	err := s.backend.Delete(ctx, key)
	if err != nil && !es4lib.IsNotFound(err) {
		return err
	}
	return nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	if key == "" {
		return 0, sscachian.ErrEmptyKey
	}
	if err := s.guard(); err != nil {
		return 0, err
	}
	var out int64
	var last error
	for attempt := 0; attempt < incrMaxAttempts; attempt++ {
		out = 0
		err := s.incrOnce(ctx, key, &out)
		if err == nil {
			return out, nil
		}
		last = err
		if errors.Is(err, sscachian.ErrNotInteger) || errors.Is(err, sscachian.ErrIncrOverflow) {
			return 0, err
		}
		// Firestore Driver retries on Aborted; es4 Memory serializes Tx via ErrNestedTx.
		if status.Code(err) != codes.Aborted && !errors.Is(err, es4lib.ErrNestedTx) {
			return 0, err
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return 0, last
}

func (s *Store) incrOnce(ctx context.Context, key string, out *int64) error {
	tx, err := s.backend.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var n int64
	raw, err := tx.Get(ctx, key)
	if err != nil {
		if !es4lib.IsNotFound(err) {
			return err
		}
		n = 0
	} else {
		val, _, ok := decodeEntry(raw)
		if !ok {
			return sscachian.ErrNotInteger
		}
		parsed, err := asInt64(val)
		if err != nil {
			return sscachian.ErrNotInteger
		}
		n = parsed
	}
	if n == math.MaxInt64 {
		return sscachian.ErrIncrOverflow
	}
	n++
	*out = n
	encoded, err := encodeEntry(n, time.Time{})
	if err != nil {
		return err
	}
	if err := tx.Set(ctx, key, encoded); err != nil {
		return err
	}
	return tx.Commit()
}

func asInt64(v any) (int64, error) {
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return int64(n), nil
		}
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	}
	return 0, sscachian.ErrNotInteger
}

// PurgeExact deletes "{logicalPrefix}:{n}" for n in 1..currentVersion without Scan.
func (s *Store) PurgeExact(ctx context.Context, logicalPrefix string) error {
	if logicalPrefix == "" {
		return sscachian.ErrEmptyKey
	}
	if err := s.guard(); err != nil {
		return err
	}
	vk := logicalPrefix + ":__version__"
	raw, err := s.backend.Get(ctx, vk)
	var max int64
	if err != nil {
		if !es4lib.IsNotFound(err) {
			return err
		}
		max = 0
	} else {
		val, _, ok := decodeEntry(raw)
		if ok {
			if n, err := asInt64(val); err == nil {
				max = n
			}
		}
	}
	for i := int64(1); i <= max; i++ {
		k := fmt.Sprintf("%s:%d", logicalPrefix, i)
		if !sscachian.IsVersionDataKey(logicalPrefix, k) {
			continue
		}
		if err := s.deleteIgnoreMissing(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

// PurgePrefix uses es4 Scan(prefix) then Delete for each key.
func (s *Store) PurgePrefix(ctx context.Context, prefix string) error {
	if prefix == "" {
		return sscachian.ErrEmptyKey
	}
	if err := s.guard(); err != nil {
		return err
	}
	keys, err := s.backend.Scan(ctx, prefix)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := s.deleteIgnoreMissing(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

var _ sscachian.Layer = (*Store)(nil)
