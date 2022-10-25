package testutil

import (
	"context"

	"github.com/evergreen-ci/cocoa"
)

// NoopSecretCache is an implementation of cocoa.SecretCache that no-ops for all
// cache modification operations.
type NoopSecretCache struct {
	Tag string
}

// Put is a no-op.
func (c *NoopSecretCache) Put(context.Context, cocoa.SecretCacheItem) error {
	return nil
}

// Delete is a no-op.
func (c *NoopSecretCache) Delete(context.Context, string) error {
	return nil
}

// GetTag returns the cache tag field.
func (c *NoopSecretCache) GetTag() string {
	return c.Tag
}
