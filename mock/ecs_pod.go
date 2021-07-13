package mock

import (
	"context"

	"github.com/evergreen-ci/cocoa"
)

// ECSPod provides a mock implementation of a cocoa.ECSPod. By default, it is
// backed by a mock ECS pod.
type ECSPod struct {
	cocoa.ECSPod

	Input      context.Context
	InfoOutput *cocoa.ECSPodInfo
}

// NewECSPod creates a mock ECS Pod backed by the given ECSPod.
func NewECSPod(p cocoa.ECSPod) *ECSPod {
	return &ECSPod{
		ECSPod: p,
	}
}

// Info returns mock information about the pod. The mock output can be
// customized. By default, it will return its cached information.
func (p *ECSPod) Info(ctx context.Context) (*cocoa.ECSPodInfo, error) {
	p.Input = ctx

	if p.InfoOutput != nil {
		return p.InfoOutput, nil
	}

	return p.ECSPod.Info(ctx)
}

// Stop stops the mock pod. The mock output can be customized. By default, it
// will set the cached status to stopped.
func (p *ECSPod) Stop(ctx context.Context) error {
	p.Input = ctx

	return p.ECSPod.Stop(ctx)
}

// Delete deletes the mock pod and all of its underlying resources. The mock
// output can be customized. By default, it will delete its secrets from its
// Vault. If it succeeds, it will set the cached status to deleted.
func (p *ECSPod) Delete(ctx context.Context) error {
	p.Input = ctx

	return p.ECSPod.Delete(ctx)
}
