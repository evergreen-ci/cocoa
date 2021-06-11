package cocoa

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa/secret"
)

// ECSPod represents a pod that is backed by ECS.
type ECSPod struct {
	TaskID           string      `bson:"-" json:"-" yaml:"-"`
	TaskDefinitionID string      `bson:"-" json:"-" yaml:"-"`
	Secrets          []PodSecret `bson:"-" json:"-" yaml:"-"`
}

// Info returns information about the current state of the pod.
func (p *ECSPod) Info(ctx context.Context) (*ECSPodInfo, error) {
	return nil, errors.New("TODO: implement")
}

// Stop stops the running pod.
func (p *ECSPod) Stop(ctx context.Context) error {
	return errors.New("TODO: implement")
}

// Delete deletes the pod and its owned resources.
func (p *ECSPod) Delete(ctx context.Context) error {
	return errors.New("TODO: implement")
}

// ECSPodInfo provides information about the current status of the pod.
type ECSPodInfo struct {
	// Status is the current status of the pod.
	Status string
}

// PodSecret is a named secret that may or may not be owned by its pod.
type PodSecret struct {
	secret.NamedSecret
	// Owned determines whether or not the secret is owned by its pod or not.
	Owned *bool
}
