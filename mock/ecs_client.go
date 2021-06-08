package mock

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/ecs"
)

// MockECSClient provides a mock implementation of a cocoa.ECSClient. This makes
// it possible to introspect on inputs to the client and control the client's
// output. It provides some default implementations where possible.
type MockECSClient struct{}

func (c *MockECSClient) RegisterTaskDefinition(ctx context.Context, in *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockECSClient) DeregisterTaskDefinition(ctx context.Context, in *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockECSClient) RunTask(ctx context.Context, in *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockECSClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
