package mock

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/ecs"
)

// ECSClient provides a mock implementation of a cocoa.ECSClient. This makes
// it possible to introspect on inputs to the client and control the client's
// output. It provides some default implementations where possible.
type ECSClient struct{}

// RegisterTaskDefinition saves the input and returns a new mock task
// definition. The mock output can be customized. By default, it will create a
// cached task definition based on the input.
func (c *ECSClient) RegisterTaskDefinition(ctx context.Context, in *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

// DeregisterTaskDefinition saves the input and deletes an existing mock task
// definition. The mock output can be customized. By default, it will delete a
// cached task definition if it exists.
func (c *ECSClient) DeregisterTaskDefinition(ctx context.Context, in *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

// ListTaskDefinitions saves the input and lists all matching task definitions.
// The mock output can be customized. By default, it will list all cached task
// definitions that match the input filters.
func (c *ECSClient) ListTaskDefinitions(ctx context.Context, in *ecs.ListTaskDefinitionsInput) (*ecs.ListTaskDefinitionsOutput, error) {
	return nil, errors.New("TODO: implement")
}

// RunTask saves the input options and returns the mock result of running a task
// definition. The mock output can be customized. By default, it will create
// mock output based on the input.
func (c *ECSClient) RunTask(ctx context.Context, in *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

// DescribeTasks saves the input and returns information about the existing
// tasks. The mock output can be customized. By default, it will describe all
// cached tasks that match.
func (c *ECSClient) DescribeTasks(ctx context.Context, in *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	return nil, errors.New("TODO: implement")
}

// ListTasks saves the input and lists all matching tasks. The mock output can
// be customized. By default, it will list all cached task definitions that
// match the input filters.
func (c *ECSClient) ListTasks(ctx context.Context, in *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	return nil, errors.New("TODO: implement")
}

// StopTask saves the input and stops a mock task. The mock output can be
// customized. By default, it will mark a cached task as stopped if it exists
// and is running.
func (c *ECSClient) StopTask(ctx context.Context, in *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

// Close closes the mock client. The mock output can be customized. By default,
// it is a no-op that returns no error.
func (c *ECSClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
