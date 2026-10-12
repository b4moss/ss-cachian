package es4

import (
	"encoding/json"
	"time"
)

// wireEntry is Entry JSON B stored as the es4 value.
// expires_at omitted means no expiry. created_at is intentionally absent.
type wireEntry struct {
	Value     any    `json:"value"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func encodeEntry(value any, expiresAt time.Time) (json.RawMessage, error) {
	w := wireEntry{Value: value}
	if !expiresAt.IsZero() {
		w.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
	}
	return json.Marshal(w)
}

// decodeEntry parses Entry JSON B. Corrupt / unusable payloads return ok=false.
func decodeEntry(raw json.RawMessage) (value any, expiresAt time.Time, ok bool) {
	if len(raw) == 0 {
		return nil, time.Time{}, false
	}
	var w wireEntry
	if err := json.Unmarshal(raw, &w); err != nil {
		return nil, time.Time{}, false
	}
	// Reject payloads that are not objects with a value field conceptually:
	// bare JSON (number/string) is treated as corrupt → miss.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, time.Time{}, false
	}
	if _, hasValue := probe["value"]; !hasValue {
		return nil, time.Time{}, false
	}
	if w.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, w.ExpiresAt)
		if err != nil {
			return nil, time.Time{}, false
		}
		expiresAt = t.UTC()
	}
	return w.Value, expiresAt, true
}
