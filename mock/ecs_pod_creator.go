package mock

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa"
)

// ECSPodCreator provides a mock implementation of a cocoa.ECSPodCreator
// that produces mock ECS pods. It can also be mocked to produce a pre-defined
// cocoa.ECSPod.
type ECSPodCreator struct{}

// CreatePod saves the input and returns a new mock pod. The mock output can be
// customized. By default, it will create a new pod based on the input that is
// backed by a mock ECSClient.
func (m *ECSPodCreator) CreatePod(ctx context.Context, opts ...*cocoa.ECSPodCreationOptions) (cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}

// CreatePodFromExistingDefinition saves the input and returns a new mock pod.
// The mock output can be customized. By default, it will create a new pod from
// the existing task definition that is backed by a mock ECSClient.
func (m *ECSPodCreator) CreatePodFromExistingDefinition(ctx context.Context, def cocoa.ECSTaskDefinition, opts ...*cocoa.ECSPodExecutionOptions) (cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}
