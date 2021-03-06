package cocoa

import "context"

// ECSPodDefinitionCache represents an external cache that tracks pod
// definitions.
type ECSPodDefinitionCache interface {
	// Put adds a new pod definition item or or updates an existing pod
	// definition item.
	Put(ctx context.Context, item ECSPodDefinitionItem) error
}

// ECSPodDefinitionItem represents an item that can be cached in a
// ECSPodDefinitionCache.
type ECSPodDefinitionItem struct {
	// ID is the unique external identifier in ECS for pod definition
	// represented by the item.
	ID string
	// DefinitionOpts are the options used to create the pod definition.
	DefinitionOpts ECSPodDefinitionOptions
}

// ECSPodDefinitionManager manages pod definitions, which are configuration
// templates used to run pods.
type ECSPodDefinitionManager interface {
	// CreatePodDefinition creates a pod definition.
	CreatePodDefinition(ctx context.Context, opts ...ECSPodDefinitionOptions) (*ECSPodDefinitionItem, error)
}
