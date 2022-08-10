package testutil

import (
	"context"

	"github.com/evergreen-ci/cocoa"
)

// NoopSecretCache is an implementation of cocoa.SecretCache that no-ops for all
// operations.
type NoopSecretCache struct{}

// Put is a no-op.
func (c *NoopSecretCache) Put(context.Context, cocoa.SecretCacheItem) error {
	return nil
}

// Delete is a no-op.
func (c *NoopSecretCache) Delete(context.Context, string) error {
	return nil
}
