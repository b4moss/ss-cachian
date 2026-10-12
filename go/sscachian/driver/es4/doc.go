// Package es4 is an es4-backed Layer (embedded open + HTTP Server adapter).
//
// Entry JSON (decision B): {"value": <payload>, "expires_at": "<RFC3339>"} with
// expires_at omitted for no expiry. PurgeExact does not call Scan; PurgePrefix
// uses es4 Scan(prefix) then Delete. Incr retries match the Firestore driver.
package es4
