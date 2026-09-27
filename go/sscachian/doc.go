// Package sscachian is a server-side cache strategy library.
//
// Applications declare Cache Types; the library runs them across Layers.
// v0.10.0: PurgePrefix/Tag, convenience APIs (Has/GetEntry/Remember/Forget).
// v0.9.0: config-driven LoadTypes (YAML/JSON) + Registry.
// v0.7.0: Exact Purge (PurgeExact), multilayer + Firestore (v0.5.0).
package sscachian
