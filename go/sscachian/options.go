package sscachian

import "time"

const tagIndexKeyPrefix = "__sscachian_tag__:"

// WriteOption customizes mutate / fill writes (tags).
type WriteOption func(*writeOpts)

type writeOpts struct {
	tags []string
}

// WithTags attaches tags for L1 reverse index updates on successful write.
// Empty tag strings are ignored.
func WithTags(tags ...string) WriteOption {
	return func(o *writeOpts) {
		o.tags = append(o.tags, tags...)
	}
}

func applyWriteOpts(opts []WriteOption) writeOpts {
	var o writeOpts
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

func mergeTags(defaultTags, callTags []string) []string {
	seen := make(map[string]struct{}, len(defaultTags)+len(callTags))
	out := make([]string, 0, len(defaultTags)+len(callTags))
	for _, t := range append(append([]string{}, defaultTags...), callTags...) {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func tagIndexKey(tag string) string {
	return tagIndexKeyPrefix + tag
}

// CacheEntry is the public GetEntry payload (value + meta).
type CacheEntry[T any] struct {
	Value     T
	CreatedAt time.Time
	ExpiresAt time.Time
}
