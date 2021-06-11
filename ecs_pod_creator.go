package cocoa

import (
	"context"
	"errors"
)

// ECSPodCreator provides a means to create a new pod backed by ECS.
type ECSPodCreator interface {
	// CreatePod creates a new pod backed by ECS with the given options. Options
	// are applied in the order they're specified and conflicting options are
	// overwritten.
	CreatePod(ctx context.Context, opts ...*ECSPodCreationOptions) (*ECSPod, error)
}

// ECSPodCreationOptions provide options to create a pod backed by ECS.
type ECSPodCreationOptions struct {
	// TaskDefinition defines a task definition that should be used if the pod
	// is being created from an existing definition.
	TaskDefinition *ECSTaskDefinition
	// ContainerDefinitions defines settings that apply to individual containers
	// within the pod.
	ContainerDefinitions []ECSContainerDefinition
	// MemoryMB is the memory limit (in MB) across all containers in the pod.
	// This is ignored for pods running Windows containers.
	MemoryMB *int
	// CPU is the CPU limit (in CPU units) across all containers in the pod.
	// 1024 CPU units is equivalent to 1 vCPU on a machine. This is ignored for
	// pods running Windows containers.
	CPU *int
	// Tags are resource tags to apply to the pod.
	Tags []string
}

// SetTaskDefinition sets the task definition for the pod.
func (o *ECSPodCreationOptions) SetTaskDefinition(def ECSTaskDefinition) *ECSPodCreationOptions {
	o.TaskDefinition = &def
	return o
}

// SetContainerDefinitions sets the container definitions for the pod. This
// overwrites any existing container definitions.
func (o *ECSPodCreationOptions) SetContainerDefinitions(defs []ECSContainerDefinition) *ECSPodCreationOptions {
	o.ContainerDefinitions = defs
	return o
}

// AddContainerDefinitions add new container definitions to the existing ones
// for the pod.
func (o *ECSPodCreationOptions) AddContainerDefinitions(defs ...ECSContainerDefinition) *ECSPodCreationOptions {
	o.ContainerDefinitions = append(o.ContainerDefinitions, defs...)
	return o
}

// SetMemoryMB sets the memory limit (in MB) that applies across the entire
// pod's containers.
func (o *ECSPodCreationOptions) SetMemoryMB(mem int) *ECSPodCreationOptions {
	o.MemoryMB = &mem
	return o
}

// SetCPU sets the CPU limit (in CPU units) that applies across the entire pod's
// containers.
func (o *ECSPodCreationOptions) SetCPU(cpu int) *ECSPodCreationOptions {
	o.CPU = &cpu
	return o
}

// SetTags sets the tags for the pod. This overwrites any existing tags.
func (o *ECSPodCreationOptions) SetTags(tags []string) *ECSPodCreationOptions {
	o.Tags = tags
	return o
}

// AddTags adds new tags to the existing ones for the pod.
func (o *ECSPodCreationOptions) AddTags(tags ...string) *ECSPodCreationOptions {
	o.Tags = append(o.Tags, tags...)
	return o
}

//nolint:deadcode
func mergeECSPodCreationOptions(opts ...*ECSPodCreationOptions) *ECSPodCreationOptions {
	merged := ECSPodCreationOptions{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.ContainerDefinitions != nil {
			merged.ContainerDefinitions = opt.ContainerDefinitions
		}

		if opt.Tags != nil {
			merged.Tags = opt.Tags
		}
	}

	return &merged
}

// ECSTaskDefinition represents options for an existing ECS task definition.
type ECSTaskDefinition struct {
	// ID is the ID of the task definition, which should already exist.
	ID *string
	// Owned determines whether or not the task definition is owned by its pod
	// or not.
	Owned *bool
}

// SetID sets the task definition ID.
func (o *ECSTaskDefinition) SetID(id string) *ECSTaskDefinition {
	o.ID = &id
	return o
}

// SetOwned sets if the task definition should be owned by its pod.
func (o *ECSTaskDefinition) SetOwned(owned bool) *ECSTaskDefinition {
	o.Owned = &owned
	return o
}

// ECSContainerDefinition defines settings that apply to a single container
// within an ECS pod.
type ECSContainerDefinition struct {
	// Image is the Docker image to use.
	Image *string
	// Command is the command to run.
	Command *string
	// MemoryMB is the amount of memory (in MB) to allocate.
	MemoryMB *int
	// CPU is the number of CPU units to allocate. 1024 CPU units is equivalent
	// to 1 vCPU on a machine.
	CPU *int
	// EnvVars are environment variables.
	EnvVars []EnvironmentVariable
	// Tags are resource tags to apply.
	Tags []string
}

// SetImage sets the image for the container.
func (o *ECSContainerDefinition) SetImage(img string) *ECSContainerDefinition {
	o.Image = &img
	return o
}

// SetCommand sets the command for the container to run.
func (o *ECSContainerDefinition) SetCommand(cmd string) *ECSContainerDefinition {
	o.Command = &cmd
	return o
}

// SetMemoryMB sets the amount of memory (in MB) to allocate.
func (o *ECSContainerDefinition) SetMemoryMB(mem int) *ECSContainerDefinition {
	o.MemoryMB = &mem
	return o
}

// SetCPU sets the number of CPU units to allocate.
func (o *ECSContainerDefinition) SetCPU(cpu int) *ECSContainerDefinition {
	o.CPU = &cpu
	return o
}

// SetTags sets the tags for the container. This overwrites any existing tags.
func (o *ECSContainerDefinition) SetTags(tags []string) *ECSContainerDefinition {
	o.Tags = tags
	return o
}

// AddTags adds new tags to the existing ones for the container.
func (o *ECSContainerDefinition) AddTags(tags ...string) *ECSContainerDefinition {
	o.Tags = append(o.Tags, tags...)
	return o
}

// SetEnvironmentVariables sets the environment variables for the container.
// This overwrites any existing environment variables.
func (o *ECSContainerDefinition) SetEnvironmentVariables(envVars []EnvironmentVariable) *ECSContainerDefinition {
	o.EnvVars = envVars
	return o
}

// AddEnvironmentVariables adds new environment variables to the existing ones
// for the container.
func (o *ECSContainerDefinition) AddEnvironmentVariables(envVars ...EnvironmentVariable) *ECSContainerDefinition {
	o.EnvVars = append(o.EnvVars, envVars...)
	return o
}

// SecretOptions represents a secret with a name and value that may or may not
// be owned by its pod.
type SecretOptions struct {
	OwnedSecret
	// Exists determines whether or not the secret already exists or must be
	// created before it can be used.
	Exists *bool
}

// SetName sets the secret's name.
func (s *SecretOptions) SetName(name string) *SecretOptions {
	s.Name = &name
	return s
}

// SetValue sets the secret's value.
func (s *SecretOptions) SetValue(val string) *SecretOptions {
	s.Value = &val
	return s
}

// SetOwned sets if the secret should be owned by its pod.
func (s *SecretOptions) SetOwned(owned bool) *SecretOptions {
	s.Owned = &owned
	return s
}

// SetExists sets whether or not the secret already exists or or must be
// created.
func (s *SecretOptions) SetExists(exists bool) *SecretOptions {
	s.Exists = &exists
	return s
}

// EnvironmentVariable represents an environment variable, which can be
// optionally backed by a secret.
type EnvironmentVariable struct {
	Name       *string
	Value      *string
	SecretOpts *SecretOptions
}

// SetName sets the environment variable name.
func (e *EnvironmentVariable) SetName(name string) *EnvironmentVariable {
	e.Name = &name
	return e
}

// SetValue sets the environment variable's value. This is mutually exclusive
// with setting the (EnvironmentVariable).SecretOptions.
func (e *EnvironmentVariable) SetValue(val string) *EnvironmentVariable {
	e.Value = &val
	return e
}

// SetSecretOptions sets the environment variable's secret value. This is
// mutually exclusive with setting the non-secret (EnvironmentVariable).Value.
func (e *EnvironmentVariable) SetSecretOptions(opts SecretOptions) *EnvironmentVariable {
	e.SecretOpts = &opts
	return e
}

// BasicECSPodCreator provides an ECSPodCreator implementation to create
// ECS pods.
type BasicECSPodCreator struct {
	client ECSClient
}

// NewBasicECSPodCreator creates a helper to create pods backed by ECS.
func NewBasicECSPodCreator(c ECSClient) *BasicECSPodCreator {
	return &BasicECSPodCreator{
		client: c,
	}
}

// CreatePod creates a new pod backed by ECS.
func (m *BasicECSPodCreator) CreatePod(ctx context.Context, opts ...*ECSPodCreationOptions) (*ECSPod, error) {
	return nil, errors.New("TODO: implement")
}
