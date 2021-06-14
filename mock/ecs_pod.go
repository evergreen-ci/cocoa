package mock

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa"
)

// ECSPod provides a mock implementation of a cocoa.ECSPod. By default, it is
// backed by a mock ECS client.
type ECSPod struct{}

// Info returns mock information about the pod. The mock output can be
// customized. By default, it will return its cached information.
func (p *ECSPod) Info(ctx context.Context) (*cocoa.ECSPodInfo, error) {
	return nil, errors.New("TODO: implement")
}

// Stop stops the mock pod. The mock output can be customized. By default, it
// will set the cached status to stopped.
func (p *ECSPod) Stop(ctx context.Context) error {
	return errors.New("TODO: implement")
}

// Delete deletes the mock pod and all of its underlying resources. The mock
// output can be customized. By default, it will delete its secrets from its
// Vault. If it succeeds, it will set the cached status to deleted.
func (p *ECSPod) Delete(ctx context.Context) error {
	return errors.New("TODO: implement")
}
