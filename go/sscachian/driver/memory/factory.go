package memory

import (
	"context"
	"fmt"

	"github.com/b4moss/ss-cachian"
)

func init() {
	sscachian.RegisterDriverFactory("memory", func(_ context.Context, opts map[string]string) (sscachian.Layer, error) {
		for k := range opts {
			return nil, fmt.Errorf("unknown memory option %q", k)
		}
		return New(), nil
	})
}
