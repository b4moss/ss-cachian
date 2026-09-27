package firestore

import (
	"context"
	"fmt"

	"github.com/b4moss/ss-cachian"
)

func init() {
	sscachian.RegisterDriverFactory("firestore", func(ctx context.Context, opts map[string]string) (sscachian.Layer, error) {
		o := Options{}
		for k, v := range opts {
			switch k {
			case "project_id":
				o.ProjectID = v
			case "collection":
				o.Collection = v
			default:
				return nil, fmt.Errorf("unknown firestore option %q", k)
			}
		}
		return New(ctx, o)
	})
}
