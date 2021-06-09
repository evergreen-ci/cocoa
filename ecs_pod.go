package cocoa

import (
	"context"
	"errors"
)

// ECSPod represents a pod that is backed by ECS.
type ECSPod struct{}

// ID is the pod's unique identifier, which must uniquely identify the
// backing resource in ECS.
func (p *ECSPod) ID() string {
	return ""
}

// DefinitionID is the unique identifier for the pod's template definition,
// which must uniquely identify the backing resource in ECS.
func (p *ECSPod) DefinitionID() string {
	return ""
}

// Stop stops the running pod.
func (p *ECSPod) Stop(ctx context.Context) error {
	return errors.New("TODO: implement")
}

// Delete deletes the pod and its owned resources.
func (p *ECSPod) Delete(ctx context.Context) error {
	return errors.New("TODO: implement")
}
