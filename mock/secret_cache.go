package mock

import (
	"context"

	"github.com/evergreen-ci/cocoa"
)

// SecretCache provides a mock implementation of a cocoa.SecretCache backed by
// another secret cache implementation.
type SecretCache struct {
	cocoa.SecretCache

	PutInput *cocoa.SecretCacheItem
	PutError error
}

// NewSecretCache creates a mock secret cache backed by the given secret cache.
func NewSecretCache(sc cocoa.SecretCache) *SecretCache {
	return &SecretCache{
		SecretCache: sc,
	}
}

// Put adds the secret to the mock cache. The mock output can be customized. By
// default, it will return the result of putting the secret in the backing
// secret cache.
func (c *SecretCache) Put(ctx context.Context, item cocoa.SecretCacheItem) error {
	c.PutInput = &item

	if c.PutError != nil {
		return c.PutError
	}

	return c.SecretCache.Put(ctx, item)
}
