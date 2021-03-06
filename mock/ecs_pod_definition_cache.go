package mock

import (
	"context"

	"github.com/evergreen-ci/cocoa"
)

// ECSPodDefinitionCache provides a mock implementation of a
// cocoa.ECSPodDefinitionCache backed by another ECS pod definition cache
// implementation.
type ECSPodDefinitionCache struct {
	cocoa.ECSPodDefinitionCache

	PutInput *cocoa.ECSPodDefinitionItem
	PutError error
}

// NewECSPodDefinitionCache creates a mock ECS pod definition cache backed
// by the given pod definition cache.
func NewECSPodDefinitionCache(pdc cocoa.ECSPodDefinitionCache) *ECSPodDefinitionCache {
	return &ECSPodDefinitionCache{
		ECSPodDefinitionCache: pdc,
	}
}

// Put adds the item to the mock cache. The mock output can be customized. By
// default, it will return the result of putting the item in the backing ECS pod
// definition cache.
func (c *ECSPodDefinitionCache) Put(ctx context.Context, item cocoa.ECSPodDefinitionItem) error {
	c.PutInput = &item

	if c.PutError != nil {
		return c.PutError
	}

	return c.ECSPodDefinitionCache.Put(ctx, item)
}
