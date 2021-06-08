package mock

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa"
)

// MockECSClient provides a mock implementation of a cocoa.ECSClient. This makes
// it possible to introspect on inputs to the pod manager and control the pod
// manager's output. It provides some default implementations where possible.
type MockECSPodManager struct{}

func (m *MockECSPodManager) CreatePod(ctx context.Context, opts ...*cocoa.ECSPodCreationOptions) (cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}

func (m *MockECSPodManager) StopPod(ctx context.Context, p cocoa.ECSPod) error {
	return errors.New("TODO: implement")
}

func (m *MockECSPodManager) DeletePod(ctx context.Context, p cocoa.ECSPod, opts ...*cocoa.ECSPodDeletionOptions) error {
	return errors.New("TODO: implement")
}

// MockECSPod provides a mock implementation of a cocoa.ECSPod to be used with
// the MockECSPodManager.
type MockECSPod struct{}

func (p *MockECSPod) ID() string {
	return ""
}

func (p *MockECSPod) DefinitionID() string {
	return ""
}
