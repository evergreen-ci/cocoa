package mock

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa"
)

// MockECSPodCreator provides a mock implementation of a cocoa.ECSPodCreator
// that produces ECS pods backed by MockECSClients. It can also be mocked to
// produce a pre-defined cocoa.ECSPod.
type MockECSPodCreator struct{}

func (m *MockECSPodCreator) CreatePod(ctx context.Context, opts ...*cocoa.ECSPodCreationOptions) (*cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}
