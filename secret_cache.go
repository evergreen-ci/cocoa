package cocoa

import "context"

// SecretCache represents an external cache that tracks secrets.
type SecretCache interface {
	// Put adds a new secret with the given name and external resource
	// identifier in the cache.
	Put(ctx context.Context, item SecretCacheItem) error
}

// SecretCacheItem represents an item that can be cached in a SecretCache.
type SecretCacheItem struct {
	// ID is the unique resource identifier for the stored secret.
	ID string
	// Name is the friendly name of the secret.
	Name string
}
