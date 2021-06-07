package mock

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
)

// MockECSClient provides a mock implementation of a cocoa.ECSClient. This makes
// it possible to introspect on inputs to the client and control the client's
// output. It provides some default implementations where possible.
type MockECSClient struct {
}

func NewMockECSClient() (cocoa.ECSClient, error) {
	return &MockECSClient{}, nil
}

func (c *MockECSClient) RegisterTaskDefinition(context.Context, *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockECSClient) DeregisterTaskDefinition(context.Context, *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockECSClient) RunTask(context.Context, *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockECSClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
