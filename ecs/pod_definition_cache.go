package ecs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/pkg/errors"
)

type PodDefinitionCache interface {
	Put(ctx context.Context, res cocoa.ECSPodDefinitionOptions) error
}

type PodDefinitionManager struct {
	cache PodDefinitionCache
}

// CreatePodDefinition creates a pod definition and caches it.
// kim: TODO: pass PodDefinitionOptions to CreatePodDefinition
func (m *PodDefinitionManager) CreatePodDefinition(ctx context.Context, opts ...cocoa.ECSPodDefinitionOptions) (*ecs.TaskDefinition, error) {
	// kim: TODO: register task definition with "untracked" tag.

	// kim: TODO: cache task definition.

	// kim: TODO: re-tag task definition with "tracked" tag.
	return nil, errors.New("kim: TODO: implement")
}
