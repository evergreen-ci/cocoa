package ecs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/pkg/errors"
)

// PodDefinitionCache represents an external cache that tracks pod definitions.
type PodDefinitionCache interface {
	Put(ctx context.Context, res cocoa.ECSPodDefinitionOptions) error
}

// PodDefinitionManager manages pod definitions, which are configuration
// templates used to run pods. It can be optionally backed by an external
// PodDefinitionCache to keep track of the pod definitions.
type PodDefinitionManager struct {
	client cocoa.ECSClient    //nolint
	cache  PodDefinitionCache //nolint
}

// CreatePodDefinition creates a pod definition and caches it.
func (m *PodDefinitionManager) CreatePodDefinition(ctx context.Context, opts ...cocoa.ECSPodDefinitionOptions) (*ecs.TaskDefinition, error) {
	return nil, errors.New("TODO: implement")
}
