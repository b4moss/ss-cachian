// Package es4 is an es4-backed Layer (embedded open + HTTP Server adapter).
//
// v0.12.0: values are the cache payload JSON. Expiry uses es4 Set options
// (WithExpiresAt) / HTTP ?expires_at=. Reads still accept the v0.11 Entry JSON
// envelope {"value":...,"expires_at":"<RFC3339>"} as a fallback.
// PurgeExact does not call Scan; PurgePrefix uses es4 Scan then Delete.
// Incr retries match the Firestore driver; Incr clears any prior expiry.
package es4
