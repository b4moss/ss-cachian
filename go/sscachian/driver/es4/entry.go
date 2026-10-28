package es4

import (
	"encoding/json"
	"time"
)

// legacyWire is the v0.11 Entry JSON envelope (read fallback only).
// New writes store the payload JSON directly; expiry is es4 meta / WithExpiresAt.
type legacyWire struct {
	Value     any    `json:"value"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func encodePayload(value any) (json.RawMessage, error) {
	return json.Marshal(value)
}

// decodeStored parses a stored es4 value.
// New format: raw payload JSON.
// Legacy format: {"value":...,"expires_at":"<RFC3339>"} (keys ⊆ {value,expires_at}).
func decodeStored(raw json.RawMessage) (value any, expiresAt time.Time, ok bool) {
	if len(raw) == 0 {
		return nil, time.Time{}, false
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err == nil {
		if _, hasValue := probe["value"]; hasValue {
			legacyOnly := true
			for k := range probe {
				if k != "value" && k != "expires_at" {
					legacyOnly = false
					break
				}
			}
			if legacyOnly {
				var w legacyWire
				if err := json.Unmarshal(raw, &w); err != nil {
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
		}
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, time.Time{}, false
	}
	return v, time.Time{}, true
}
