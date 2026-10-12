package es4

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/b4moss/ss-cachian"
)

func init() {
	sscachian.RegisterDriverFactory("es4", func(ctx context.Context, opts map[string]string) (sscachian.Layer, error) {
		if baseURL, ok := opts["base_url"]; ok && baseURL != "" {
			for k := range opts {
				if k != "base_url" {
					return nil, fmt.Errorf("unknown es4 http option %q", k)
				}
			}
			return NewHTTP(ctx, HTTPOptions{BaseURL: baseURL})
		}

		o := Options{}
		for k, v := range opts {
			switch k {
			case "memory_only":
				b, err := strconv.ParseBool(v)
				if err != nil {
					return nil, fmt.Errorf("sscachian/es4: memory_only: %w", err)
				}
				o.MemoryOnly = b
			case "state_backend":
				o.StateBackend = strings.ToLower(v)
			case "state_path":
				o.StatePath = v
			case "state_redis_url":
				o.StateRedisURL = v
			case "state_redis_key_prefix":
				o.StateRedisKeyPrefix = v
			case "state_firestore_project_id":
				o.StateFirestoreProjectID = v
			case "state_firestore_database_id":
				o.StateFirestoreDatabaseID = v
			case "state_firestore_collection":
				o.StateFirestoreCollection = v
			default:
				return nil, fmt.Errorf("unknown es4 option %q", k)
			}
		}
		if !o.MemoryOnly && o.StateBackend == "" && o.StatePath == "" {
			o.MemoryOnly = true
		}
		return New(ctx, o)
	})
}
