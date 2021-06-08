package cocoa

import "context"

// ECSPodManager allows you to interact with pods backed by ECS without needing
// to make direct API calls to ECS to perform common operations.
type ECSPodManager interface {
	// Create creates a new pod backed by ECS with the given options. Options
	// are applied in the order they're specified and conflicting options are
	// overwritten.
	CreatePod(ctx context.Context, opts ...*ECSPodCreationOptions) (ECSPod, error)
	// Stop stops a pod.
	StopPod(ctx context.Context, p ECSPod) error
	// Delete deletes a pod with the given options, cleaning up all the
	// resources that it uses. Options are applied in the order they're
	// specified and conflicting options are overwritten.
	DeletePod(ctx context.Context, p ECSPod, opts ...*ECSPodDeletionOptions) error
}

// ECSPodCreationOptions provide options to create a pod backed by ECS.
type ECSPodCreationOptions struct {
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
	// Secrets are secret values to be passed into the environment.
	Secrets []string
	// Tags are resource tags to apply.
	Tags []string
}

// SetImage sets the image for the container.
func (o *ECSContainerDefinition) SetImage(img string) *ECSContainerDefinition {
	o.Image = &img
	return o
}

func (o *ECSContainerDefinition) SetCommand(cmd string) *ECSContainerDefinition {
	o.Command = &cmd
	return o
}

func (o *ECSContainerDefinition) SetMemoryMB(mem int) *ECSContainerDefinition {
	o.MemoryMB = &mem
	return o
}

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

// SetSecrets sets the secrets for the container. This overwrites any existing
// secrets.
func (o *ECSContainerDefinition) SetSecrets(secrets []string) *ECSContainerDefinition {
	o.Secrets = secrets
	return o
}

// AddSecrets adds new secrets to the existing ones for the container.
func (o *ECSContainerDefinition) AddSecrets(secrets ...string) *ECSContainerDefinition {
	o.Secrets = append(o.Secrets, secrets...)
	return o
}

// ECSPodDeletionOptions provide options to delete a pod backed by ECS.
type ECSPodDeletionOptions struct {
	// KeepDefinition determines whether or not the pod's definition will be
	// deleted. If true, only the pod instance will be deleted; otherwise, the
	// pod's underlying definition will also be deleted. By default, this is
	// false.
	KeepDefinition *bool
}

func (o *ECSPodDeletionOptions) SetKeepDefinition(keep bool) *ECSPodDeletionOptions {
	o.KeepDefinition = &keep
	return o
}

func mergeECSPodDeletionOptions(opts ...*ECSPodDeletionOptions) *ECSPodDeletionOptions {
	merged := ECSPodDeletionOptions{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.KeepDefinition != nil {
			merged.KeepDefinition = opt.KeepDefinition
		}
	}

	return &merged
}
