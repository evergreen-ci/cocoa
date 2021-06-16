package cocoa

import (
	"context"

	"github.com/pkg/errors"

	"github.com/evergreen-ci/cocoa/secret"
	"github.com/mongodb/grip"
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
	vault     secret.Vault
	resources ECSPodResources
	status    ECSPodStatus
}

// BasicECSPodOptions are options to create a basic ECS pod.
type BasicECSPodOptions struct {
	Client    ECSClient
	Vault     secret.Vault
	Resources *ECSPodResources
	Status    *ECSPodStatus
}

// NewBasicECSPodOptions returns new uninitialized options to create a basic ECS
// pod.
func NewBasicECSPodOptions() *BasicECSPodOptions {
	return &BasicECSPodOptions{}
}

// SetClient sets the client the pod uses to communicate with ECS.
func (o *BasicECSPodOptions) SetClient(c ECSClient) *BasicECSPodOptions {
	o.Client = c
	return o
}

// SetVault sets the vault that the pod uses to manage secrets.
func (o *BasicECSPodOptions) SetVault(v secret.Vault) *BasicECSPodOptions {
	o.Vault = v
	return o
}

// SetResources sets the resources used by the pod.
func (o *BasicECSPodOptions) SetResources(res ECSPodResources) *BasicECSPodOptions {
	o.Resources = &res
	return o
}

// SetStatus sets the current status for the pod.
func (o *BasicECSPodOptions) SetStatus(s ECSPodStatus) *BasicECSPodOptions {
	o.Status = &s
	return o
}

// Validate checks that the required parameters to initialize a pod are given.
func (o *BasicECSPodOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(o.Client == nil, "must specify a client")
	catcher.NewWhen(o.Resources == nil, "must specify at least one underlying resource being used by the pod")
	catcher.NewWhen(o.Resources != nil && o.Resources.TaskID == nil, "must specify task ID")
	if o.Status != nil {
		catcher.Add(o.Status.Validate())
	} else {
		catcher.New("must specify a status")
	}
	return catcher.Resolve()
}

// MergeECSPodOptions merges all the given options describing an ECS pod.
// Options are applied in the order that they're specified and conflicting
// options are overwritten.
func MergeECSPodOptions(opts ...*BasicECSPodOptions) BasicECSPodOptions {
	merged := BasicECSPodOptions{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.Client != nil {
			merged.Client = opt.Client
		}

		if opt.Vault != nil {
			merged.Vault = opt.Vault
		}

		if opt.Resources != nil {
			merged.Resources = opt.Resources
		}

		if opt.Status != nil {
			merged.Status = opt.Status
		}
	}

	return merged
}

// NewBasicECSPod initializes a new pod that is backed by ECS.
func NewBasicECSPod(opts ...*BasicECSPodOptions) (*BasicECSPod, error) {
	merged := MergeECSPodOptions(opts...)
	if err := merged.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid options")
	}
	return &BasicECSPod{
		client:    merged.Client,
		vault:     merged.Vault,
		resources: *merged.Resources,
		status:    *merged.Status,
	}, nil
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

// NewPodSecret creates a new uninitialized pod secret.
func NewPodSecret() *PodSecret {
	return &PodSecret{}
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
	TaskID         *string            `bson:"-" json:"-" yaml:"-"`
	TaskDefinition *ECSTaskDefinition `bson:"-" json:"-" yaml:"-"`
	Secrets        []PodSecret        `bson:"-" json:"-" yaml:"-"`
}

// NewECSPodResources returns a new uninitialized set of resources used by a
// pod.
func NewECSPodResources() *ECSPodResources {
	return &ECSPodResources{}
}

// SetTaskID sets the ECS task ID associated with the pod.
func (r *ECSPodResources) SetTaskID(id string) *ECSPodResources {
	r.TaskID = &id
	return r
}

// SetTaskDefinition sets the ECS task definition associated with the pod.
func (r *ECSPodResources) SetTaskDefinition(def ECSTaskDefinition) *ECSPodResources {
	r.TaskDefinition = &def
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
type ECSPodStatus string

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

// Validate checks that the ECS pod status is one of the recognized statuses.
func (s ECSPodStatus) Validate() error {
	switch s {
	case Starting, Running, Stopped, Deleted:
		return nil
	default:
		return errors.Errorf("unrecognized status '%s'", s)
	}
}
