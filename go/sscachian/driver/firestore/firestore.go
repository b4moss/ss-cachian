package firestore

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/b4moss/ss-cachian"
)

const defaultCollection = "sscachian"

// Store is a Firestore-backed Layer (Emulator-friendly).
type Store struct {
	client     *firestore.Client
	collection string
}

// Options configures New.
type Options struct {
	ProjectID  string
	Collection string
}

// New creates a Store. Honors FIRESTORE_EMULATOR_HOST when set.
func New(ctx context.Context, opts Options) (*Store, error) {
	if opts.ProjectID == "" {
		opts.ProjectID = "ss-cachian-dev"
	}
	if opts.Collection == "" {
		opts.Collection = defaultCollection
	}
	client, err := firestore.NewClient(ctx, opts.ProjectID)
	if err != nil {
		return nil, err
	}
	return &Store{client: client, collection: opts.Collection}, nil
}

// Close closes the underlying client.
func (s *Store) Close() error {
	return s.client.Close()
}

type docFields struct {
	Value     []byte    `firestore:"value"`
	CreatedAt time.Time `firestore:"created_at"`
	ExpiresAt time.Time `firestore:"expires_at"`
}

func (s *Store) doc(key string) *firestore.DocumentRef {
	return s.client.Collection(s.collection).Doc(key)
}

func (s *Store) Get(ctx context.Context, key string) (sscachian.Entry, bool, error) {
	if key == "" {
		return sscachian.Entry{}, false, sscachian.ErrEmptyKey
	}
	snap, err := s.doc(key).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return sscachian.Entry{}, false, nil
		}
		return sscachian.Entry{}, false, err
	}
	var f docFields
	if err := snap.DataTo(&f); err != nil {
		return sscachian.Entry{}, false, err
	}
	if !f.ExpiresAt.IsZero() && !f.ExpiresAt.After(time.Now().UTC()) {
		return sscachian.Entry{}, false, nil
	}
	var val any
	if len(f.Value) == 0 {
		val = nil
	} else if err := json.Unmarshal(f.Value, &val); err != nil {
		return sscachian.Entry{}, false, err
	}
	return sscachian.Entry{Value: val, CreatedAt: f.CreatedAt, ExpiresAt: f.ExpiresAt}, true, nil
}

func (s *Store) Set(ctx context.Context, key string, entry sscachian.Entry, ttl time.Duration) error {
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
	raw, err := json.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("sscachian/firestore: marshal value: %w", err)
	}
	_, err = s.doc(key).Set(ctx, docFields{
		Value:     raw,
		CreatedAt: entry.CreatedAt,
		ExpiresAt: entry.ExpiresAt,
	})
	return err
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if key == "" {
		return sscachian.ErrEmptyKey
	}
	_, err := s.doc(key).Delete(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}
	return nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	if key == "" {
		return 0, sscachian.ErrEmptyKey
	}
	ref := s.doc(key)
	var out int64
	var last error
	for attempt := 0; attempt < 8; attempt++ {
		out = 0
		err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(ref)
			var n int64
			if err != nil {
				if status.Code(err) != codes.NotFound {
					return err
				}
				n = 0
			} else {
				var f docFields
				if err := snap.DataTo(&f); err != nil {
					return err
				}
				var val any
				if err := json.Unmarshal(f.Value, &val); err != nil {
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
			out = n
			raw, err := json.Marshal(n)
			if err != nil {
				return err
			}
			return tx.Set(ref, docFields{
				Value:     raw,
				CreatedAt: time.Now().UTC(),
			})
		})
		if err == nil {
			return out, nil
		}
		last = err
		if status.Code(err) != codes.Aborted {
			return 0, err
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return 0, last
}

func asInt64(v any) (int64, error) {
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return int64(n), nil
		}
	case int64:
		return n, nil
	case json.Number:
		return n.Int64()
	}
	return 0, sscachian.ErrNotInteger
}

func (s *Store) PurgeExact(ctx context.Context, logicalPrefix string) error {
	if logicalPrefix == "" {
		return sscachian.ErrEmptyKey
	}
	iter := s.client.Collection(s.collection).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if !sscachian.IsVersionDataKey(logicalPrefix, doc.Ref.ID) {
			continue
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) PurgePrefix(ctx context.Context, prefix string) error {
	if prefix == "" {
		return sscachian.ErrEmptyKey
	}
	iter := s.client.Collection(s.collection).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if !strings.HasPrefix(doc.Ref.ID, prefix) {
			continue
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ClearCollection deletes all docs (tests only).
func (s *Store) ClearCollection(ctx context.Context) error {
	iter := s.client.Collection(s.collection).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}

var _ sscachian.Layer = (*Store)(nil)
