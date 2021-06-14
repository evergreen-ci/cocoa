package cocoa

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa/secret"
)

// ECSPod provides an abstraction of a pod backed by ECS.
type ECSPod interface {
	// Info returns information about the current state of the pod.
	Info(ctx context.Context) (*ECSPodInfo, error)
	// Stop stops the running pod without cleaning up any of its underlying
	// resources.
	Stop(ctx context.Context) error
	// Delete deletes the pod and its owned resources.
	Delete(ctx context.Context) error
}

// BasicECSPod represents a pod that is backed by ECS.
type BasicECSPod struct {
	client    ECSClient
	resources ECSPodResources
	status    ECSPodStatus
}

// NewBasicECSPod initializes a new pod that is backed by ECS.
func NewBasicECSPod(c ECSClient, res ECSPodResources, stat ECSPodStatus) *BasicECSPod {
	return &BasicECSPod{
		client:    c,
		resources: res,
		status:    stat,
	}
}

// Info returns information about the current state of the pod.
func (p *BasicECSPod) Info(ctx context.Context) (*ECSPodInfo, error) {
	return nil, errors.New("TODO: implement")
}

// Stop stops the running pod without cleaning up any of its underlying
// resources.
func (p *BasicECSPod) Stop(ctx context.Context) error {
	return errors.New("TODO: implement")
}

// Delete deletes the pod and its owned resources.
func (p *BasicECSPod) Delete(ctx context.Context) error {
	return errors.New("TODO: implement")
}

// ECSPodInfo provides information about the current status of the pod.
type ECSPodInfo struct {
	// Status is the current status of the pod.
	Status ECSPodStatus `bson:"-" json:"-" yaml:"-"`
	// Resources provides information about the underlying ECS resources being
	// used by the pod.
	Resources ECSPodResources `bson:"-" json:"-" yaml:"-"`
}

// PodSecret is a named secret that may or may not be owned by its pod.
type PodSecret struct {
	secret.NamedSecret
	// Owned determines whether or not the secret is owned by its pod or not.
	Owned *bool
}

// SetName sets the secret's name.
func (s *PodSecret) SetName(name string) *PodSecret {
	s.Name = &name
	return s
}

// SetValue sets the secret's value.
func (s *PodSecret) SetValue(val string) *PodSecret {
	s.Value = &val
	return s
}

// SetOwned sets if the secret should be owned by its pod.
func (s *PodSecret) SetOwned(owned bool) *PodSecret {
	s.Owned = &owned
	return s
}

// ECSPodResources are ECS-specific resources that a pod uses.
type ECSPodResources struct {
	TaskID         string            `bson:"-" json:"-" yaml:"-"`
	TaskDefinition ECSTaskDefinition `bson:"-" json:"-" yaml:"-"`
	Secrets        []PodSecret       `bson:"-" json:"-" yaml:"-"`
}

// SetTaskID sets the ECS task ID associated with the pod.
func (r *ECSPodResources) SetTaskID(id string) *ECSPodResources {
	r.TaskID = id
	return r
}

// SetTaskDefinition sets the ECS task definition associated with the pod.
func (r *ECSPodResources) SetTaskDefinition(def ECSTaskDefinition) *ECSPodResources {
	r.TaskDefinition = def
	return r
}

// SetSecrets sets the secrets associated with the pod. This overwrites any
// existing secrets.
func (r *ECSPodResources) SetSecrets(secrets []PodSecret) *ECSPodResources {
	r.Secrets = secrets
	return r
}

// AddSecrets adds new secrets to the existing ones associated with the pod.
func (r *ECSPodResources) AddSecrets(secrets ...PodSecret) *ECSPodResources {
	r.Secrets = append(r.Secrets, secrets...)
	return r
}

// ECSPodStatus represents the different statuses possible for an ECS pod.
type ECSPodStatus = string

const (
	// Starting indicates that the ECS pod is being prepared to run.
	Starting ECSPodStatus = "starting"
	// Running indicates that the ECS pod is actively running.
	Running ECSPodStatus = "running"
	// Stopped indicates the that ECS pod is stopped, but all of its resources
	// are still available.
	Stopped ECSPodStatus = "stopped"
	// Deleted indicates that the ECS pod has been cleaned up completely,
	// including all of its resources.
	Deleted ECSPodStatus = "deleted"
)
