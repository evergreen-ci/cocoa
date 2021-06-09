package cocoa

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/ecs"
)

// ECSClient provides a common interface to interact with an ECS client and its
// mock implementation for testing. Implementations must handle retrying and
// backoff.
type ECSClient interface {
	// RegisterTaskDefinition registers the definition for a new task with ECS.
	RegisterTaskDefinition(context.Context, *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	// DeregisterTaskDefinition deregisters an existing ECS task definition.
	DeregisterTaskDefinition(ctx context.Context, in *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error)
	// RunTask runs a registered task.
	RunTask(ctx context.Context, in *ecs.RunTaskInput) (*ecs.RunTaskOutput, error)
	// Close closes the client and cleans up its resources. Implementations
	// should ensure that this is idempotent.
	Close(ctx context.Context) error
}

// BasicECSClient provides an ECSClient implementation that wraps the ECS API.
// It supports retrying requests using exponential backoff and jitter.
type BasicECSClient struct{}

// RegisterTaskDefinition registers a new task definition.
func (c *BasicECSClient) RegisterTaskDefinition(context.Context, *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

// DeregisterTaskDefinition deregisters an existing task definition.
func (c *BasicECSClient) DeregisterTaskDefinition(context.Context, *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

// RunTask runs a new task.
func (c *BasicECSClient) RunTask(context.Context, *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

// Close closes the client.
func (c *BasicECSClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
