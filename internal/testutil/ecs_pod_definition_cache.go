package testutil

import (
	"context"

	"github.com/evergreen-ci/cocoa"
)

// NoopECSPodDefinitionCache is an implementation of cocoa.ECSPodDefinitionCache
// that no-ops for all operations.
type NoopECSPodDefinitionCache struct{}

// Put is a no-op.
func (c *NoopECSPodDefinitionCache) Put(context.Context, cocoa.ECSPodDefinitionItem) error {
	return nil
}

// Delete is a no-op.
func (c *NoopECSPodDefinitionCache) Delete(context.Context, string) error {
	return nil
}
