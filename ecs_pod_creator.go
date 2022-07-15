package cocoa

import (
	"context"
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"sort"
	"strconv"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/grip"
)

// ECSPodCreator provides a means to create a new pod backed by AWS ECS.
type ECSPodCreator interface {
	// CreatePod creates a new pod backed by ECS with the given options. Options
	// are applied in the order they're specified and conflicting options are
	// overwritten.
	CreatePod(ctx context.Context, opts ...ECSPodCreationOptions) (ECSPod, error)
	// CreatePodFromExistingDefinition creates a new pod backed by ECS from an
	// existing task definition.
	CreatePodFromExistingDefinition(ctx context.Context, def ECSTaskDefinition, opts ...ECSPodExecutionOptions) (ECSPod, error)
}

// ECSPodCreationOptions provide options to create a pod backed by ECS.
type ECSPodCreationOptions struct {
	// DefinitionOpts specify options to configure the pod's definition.
	DefinitionOpts ECSPodDefinitionOptions
	// ExecutionOpts specify options to configure how the pod executes.
	ExecutionOpts *ECSPodExecutionOptions
}

// NewECSPodCreationOptions returns new uninitialized options to create a pod.
func NewECSPodCreationOptions() *ECSPodCreationOptions {
	return &ECSPodCreationOptions{}
}

// SetDefinitionOptions sets the options to configure the pod definition.
func (o *ECSPodCreationOptions) SetDefinitionOptions(opts ECSPodDefinitionOptions) *ECSPodCreationOptions {
	o.DefinitionOpts = opts
	return o
}

// SetExecutionOptions sets the options to configure how the pod executes.
func (o *ECSPodCreationOptions) SetExecutionOptions(opts ECSPodExecutionOptions) *ECSPodCreationOptions {
	o.ExecutionOpts = &opts
	return o
}

// Validate checks that all the required parameters are given and the values are
// valid. It sets defaults where possible.
func (o *ECSPodCreationOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.Wrap(o.DefinitionOpts.Validate(), "invalid pod definition options")
	networkMode := o.DefinitionOpts.getNetworkMode()
	catcher.NewWhen(networkMode == NetworkModeAWSVPC && (o.ExecutionOpts == nil || o.ExecutionOpts.AWSVPCOpts == nil), "must specify AWSVPC configuration when using AWSVPC network mode")
	catcher.NewWhen(networkMode != NetworkModeAWSVPC && o.ExecutionOpts != nil && o.ExecutionOpts.AWSVPCOpts != nil, "cannot specify AWSVPC configuration when network mode is not AWSVPC")

	if o.ExecutionOpts != nil {
		catcher.Wrap(o.ExecutionOpts.Validate(), "invalid execution options")
	}

	if catcher.HasErrors() {
		return catcher.Resolve()
	}

	if o.ExecutionOpts == nil {
		placementOpts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack).SetStrategyParameter(StrategyParamBinpackMemory)
		o.ExecutionOpts = NewECSPodExecutionOptions().SetPlacementOptions(*placementOpts)
	}

	return nil
}

// MergeECSPodCreationOptions merges all the given options to create an ECS pod.
// Options are applied in the order that they're specified and conflicting
// options are overwritten.
func MergeECSPodCreationOptions(opts ...ECSPodCreationOptions) ECSPodCreationOptions {
	merged := ECSPodCreationOptions{}

	for _, opt := range opts {
		merged.DefinitionOpts = MergeECSPodDefinitionOptions(merged.DefinitionOpts, opt.DefinitionOpts)

		if opt.ExecutionOpts != nil {
			var execOpts ECSPodExecutionOptions
			if merged.ExecutionOpts != nil {
				execOpts = MergeECSPodExecutionOptions(*merged.ExecutionOpts, *opt.ExecutionOpts)
			} else {
				execOpts = *opt.ExecutionOpts
			}
			merged.ExecutionOpts = &execOpts
		}
	}

	return merged
}

// ECSPodDefinitionOptions represent options to configure a template for running
// a pod.
type ECSPodDefinitionOptions struct {
	// Name is the friendly name of the pod. By default, this is a random
	// string.
	Name *string
	// ContainerDefinitions defines settings that apply to individual containers
	// within the pod. This is required.
	ContainerDefinitions []ECSContainerDefinition
	// MemoryMB is the hard memory limit (in MB) across all containers in the
	// pod. If this is not specified, then each container is required to specify
	// its own memory. This is ignored for pods running Windows containers.
	MemoryMB *int
	// CPU is the hard CPU limit (in CPU units) across all containers in the
	// pod. 1024 CPU units is equivalent to 1 vCPU on a machine. If this is not
	// specified, then each container is required to specify its own CPU.
	// This is ignored for pods running Windows containers.
	CPU *int
	// NetworkMode describes the networking capabilities of the pod's
	// containers. If the NetworkMode is unspecified for a pod running Linux
	// containers, the default value is NetworkModeBridge. If the NetworkMode is
	// unspecified for a pod running Windows containers, the default network
	// mode is to use the Windows NAT network.
	NetworkMode *ECSNetworkMode
	// TaskRole is the role that the pod can use. Depending on the
	// configuration, this may be required if
	// (ECSPodExecutionOptions).SupportsDebugMode is true.
	TaskRole *string
	// ExecutionRole is the role that ECS container agent can use. Depending on
	// the configuration, this may be required if the container uses secrets.
	ExecutionRole *string
	// Tags are resource tags to apply to the pod definition.
	Tags map[string]string
}

// NewECSPodDefinitionOptions returns new uninitialized options to create a pod
// definition.
func NewECSPodDefinitionOptions() *ECSPodDefinitionOptions {
	return &ECSPodDefinitionOptions{}
}

// SetName sets the friendly name of the pod.
func (o *ECSPodDefinitionOptions) SetName(name string) *ECSPodDefinitionOptions {
	o.Name = &name
	return o
}

// SetContainerDefinitions sets the container definitions for the pod. This
// overwrites any existing container definitions.
func (o *ECSPodDefinitionOptions) SetContainerDefinitions(defs []ECSContainerDefinition) *ECSPodDefinitionOptions {
	o.ContainerDefinitions = defs
	return o
}

// AddContainerDefinitions add new container definitions to the existing ones
// for the pod.
func (o *ECSPodDefinitionOptions) AddContainerDefinitions(defs ...ECSContainerDefinition) *ECSPodDefinitionOptions {
	o.ContainerDefinitions = append(o.ContainerDefinitions, defs...)
	return o
}

// SetMemoryMB sets the memory limit (in MB) that applies across the entire
// pod's containers.
func (o *ECSPodDefinitionOptions) SetMemoryMB(mem int) *ECSPodDefinitionOptions {
	o.MemoryMB = &mem
	return o
}

// SetCPU sets the CPU limit (in CPU units) that applies across the entire pod's
// containers.
func (o *ECSPodDefinitionOptions) SetCPU(cpu int) *ECSPodDefinitionOptions {
	o.CPU = &cpu
	return o
}

// SetNetworkMode sets the network mode that applies for all the pod's
// containers.
func (o *ECSPodDefinitionOptions) SetNetworkMode(mode ECSNetworkMode) *ECSPodDefinitionOptions {
	o.NetworkMode = &mode
	return o
}

// SetTaskRole sets the task role that the pod can use.
func (o *ECSPodDefinitionOptions) SetTaskRole(role string) *ECSPodDefinitionOptions {
	o.TaskRole = &role
	return o
}

// SetExecutionRole sets the execution role that the pod can use.
func (o *ECSPodDefinitionOptions) SetExecutionRole(role string) *ECSPodDefinitionOptions {
	o.ExecutionRole = &role
	return o
}

// SetTags sets the tags for the pod definition. This overwrites any existing
// tags.
func (o *ECSPodDefinitionOptions) SetTags(tags map[string]string) *ECSPodDefinitionOptions {
	o.Tags = tags
	return o
}

// AddTags adds new tags to the existing ones for the pod definition.
func (o *ECSPodDefinitionOptions) AddTags(tags map[string]string) *ECSPodDefinitionOptions {
	if o.Tags == nil {
		o.Tags = map[string]string{}
	}
	for k, v := range tags {
		o.Tags[k] = v
	}
	return o
}

// getNetworkMode returns the network mode. If no network mode is explicitly
// set, this returns the default network mode.
func (o *ECSPodDefinitionOptions) getNetworkMode() ECSNetworkMode {
	if o.NetworkMode != nil {
		return *o.NetworkMode
	}
	return NetworkModeBridge
}

// Validate checks that all the required parameters are given and the values are
// valid. It sets default values where possible.
func (o *ECSPodDefinitionOptions) Validate() error {
	catcher := grip.NewBasicCatcher()

	catcher.NewWhen(o.Name != nil && *o.Name == "", "cannot specify an empty name")
	catcher.NewWhen(o.MemoryMB != nil && *o.MemoryMB <= 0, "must have positive memory value if non-default")
	catcher.NewWhen(o.CPU != nil && *o.CPU <= 0, "must have positive CPU value if non-default")

	catcher.Wrap(o.validateContainerDefinitions(), "invalid container definitions")

	networkMode := o.getNetworkMode()
	catcher.Wrap(networkMode.Validate(), "invalid network mode")

	if o.Name == nil {
		o.Name = utility.ToStringPtr(utility.RandomString())
	}

	return catcher.Resolve()
}

// validateContainerDefinitions checks that all the individual container
// definitions are valid.
func (o *ECSPodDefinitionOptions) validateContainerDefinitions() error {
	catcher := grip.NewBasicCatcher()

	catcher.NewWhen(len(o.ContainerDefinitions) == 0, "must specify at least one container definition")

	networkMode := o.getNetworkMode()
	var totalContainerMemMB, totalContainerCPU int
	for i, def := range o.ContainerDefinitions {
		catcher.Wrapf(o.ContainerDefinitions[i].Validate(), "container definition '%s'", utility.FromStringPtr(def.Name))

		switch networkMode {
		case NetworkModeNone:
			catcher.NewWhen(len(def.PortMappings) != 0, "cannot specify port mappings because networking is disabled")
		case NetworkModeHost, NetworkModeAWSVPC:
			for _, pm := range def.PortMappings {
				containerPort := utility.FromIntPtr(pm.ContainerPort)
				if pm.HostPort != nil {
					hostPort := utility.FromIntPtr(pm.HostPort)
					catcher.ErrorfWhen(hostPort != containerPort,
						"host port '%d' must be omitted or identical to the container port '%d' when network mode is '%s'", hostPort, containerPort, networkMode)
				}
			}
		}

		if def.MemoryMB != nil {
			totalContainerMemMB += *def.MemoryMB
		} else if o.MemoryMB == nil {
			catcher.Errorf("must specify container-level memory to allocate for each container if pod-level memory is not specified")
		}

		if o.ContainerDefinitions[i].CPU != nil {
			totalContainerCPU += *o.ContainerDefinitions[i].CPU
		} else if o.CPU == nil {
			catcher.Errorf("must specify container-level CPU to allocate for each container if pod-level CPU is not specified")
		}

		for _, envVar := range def.EnvVars {
			if envVar.SecretOpts != nil && o.ExecutionRole == nil {
				catcher.Errorf("must specify execution role ARN when specifying container secrets")
				break
			}
		}
	}

	if o.MemoryMB != nil {
		catcher.ErrorfWhen(*o.MemoryMB < totalContainerMemMB, "total memory requested for the individual containers (%d MB) is greater than the memory available for the entire task (%d MB)", totalContainerMemMB, *o.MemoryMB)
	}
	if o.CPU != nil {
		catcher.ErrorfWhen(*o.CPU < totalContainerCPU, "total CPU requested for the individual containers (%d units) is greater than the memory available for the entire task (%d units)", totalContainerCPU, *o.CPU)
	}

	return catcher.Resolve()
}

// tagPair represents a tag with a key and value pair.
type tagPair struct {
	key   string
	value string
}

// hash returns the hash digest of the tag pair.
func (tp tagPair) hash() string {
	h := newSHA1Hasher()
	h.add(tp.key)
	h.add(tp.value)
	return h.sum()
}

// hashableTagPairs represents a slice of key-value tag that can be hashed.
type hashableTagPairs []tagPair

// newHashableTagPairs returns a sorted slice of hashable tag pairs.
func newHashableTagPairs(tags map[string]string) hashableTagPairs {
	var htp hashableTagPairs
	for k, v := range tags {
		htp = append(htp, tagPair{key: k, value: v})
	}
	sort.Sort(htp)
	return htp
}

// Len returns the number of container definitions.
func (htp hashableTagPairs) Len() int {
	return len(htp)
}

// Less returns whether or not the key for the pair at index i is
// lexicographically before the key for the pair at index j.
func (htp hashableTagPairs) Less(i, j int) bool {
	return htp[i].key < htp[j].key
}

// Swap swaps the tag pairs at indexes i and j.
func (htp hashableTagPairs) Swap(i, j int) {
	htp[i], htp[j] = htp[j], htp[i]
}

// hash returns the hash digest of the tag pairs.
func (htp hashableTagPairs) hash() string {
	if !sort.IsSorted(htp) {
		sort.Sort(htp)
	}

	h := newSHA1Hasher()

	for _, tp := range htp {
		h.add(tp.hash())
	}

	return h.sum()
}

// kim: TODO: test hash fn for same/different inputs incrementally for each
// field/substruct. Test order of fields doesn't matter.

// Hash returns the hash digest of the pod definition.
func (o *ECSPodDefinitionOptions) Hash() string {
	h := newSHA1Hasher()

	if o.Name != nil {
		h.add(utility.FromStringPtr(o.Name))
	}

	if len(o.ContainerDefinitions) != 0 {
		h.add(newHashableContainerDefinitions(o.ContainerDefinitions).hash())
	}

	if o.MemoryMB != nil {
		h.add(strconv.Itoa(utility.FromIntPtr(o.MemoryMB)))
	}

	if o.CPU != nil {
		h.add(strconv.Itoa(utility.FromIntPtr(o.CPU)))
	}

	if o.NetworkMode != nil {
		h.add(string(*o.NetworkMode))
	}

	if o.TaskRole != nil {
		h.add(utility.FromStringPtr(o.TaskRole))
	}

	if o.ExecutionRole != nil {
		h.add(utility.FromStringPtr(o.ExecutionRole))
	}

	if len(o.Tags) != 0 {
		h.add(newHashableTagPairs(o.Tags).hash())
	}

	return h.sum()
}

// hasher is a wrapper around a hashing algorithm.
type hasher struct {
	hash.Hash
}

// newSHA1Hasher returns a hasher that uses the SHA1 algorithm.
func newSHA1Hasher() hasher {
	return hasher{Hash: sha1.New()}
}

// add adds data to the hasher.
func (h hasher) add(data string) {
	// The hash.Hash interface says the io.Writer will never return an error, so
	// the returned error can be squashed.
	_, _ = io.WriteString(h, data)
}

// sum returns the hash sum of the accumulated data.
func (h hasher) sum() string {
	return fmt.Sprintf("%x", h.Sum(nil))
}

// MergeECSPodDefinitionOptions merges all the given options to create an ECS
// pod definition. Options are applied in the order that they're specified and
// conflicting options are overwritten.
func MergeECSPodDefinitionOptions(opts ...ECSPodDefinitionOptions) ECSPodDefinitionOptions {
	merged := ECSPodDefinitionOptions{}

	for _, opt := range opts {
		if opt.Name != nil {
			merged.Name = opt.Name
		}

		if opt.ContainerDefinitions != nil {
			merged.ContainerDefinitions = opt.ContainerDefinitions
		}

		if opt.MemoryMB != nil {
			merged.MemoryMB = opt.MemoryMB
		}

		if opt.CPU != nil {
			merged.CPU = opt.CPU
		}

		if opt.NetworkMode != nil {
			merged.NetworkMode = opt.NetworkMode
		}

		if opt.TaskRole != nil {
			merged.TaskRole = opt.TaskRole
		}

		if opt.ExecutionRole != nil {
			merged.ExecutionRole = opt.ExecutionRole
		}

		if opt.Tags != nil {
			merged.Tags = opt.Tags
		}
	}

	return merged
}

// ECSContainerDefinition defines settings that apply to a single container
// within an ECS pod.
type ECSContainerDefinition struct {
	// Name is the friendly name of the container. By default, this is a random
	// string.
	Name *string
	// Image is the Docker image to use. This is required.
	Image *string
	// Command is the command to run, separated into individual arguments. By
	// default, there is no command.
	Command []string
	// WorkingDir is the container working directory in which commands will be
	// run.
	WorkingDir *string
	// MemoryMB is the amount of memory (in MB) to allocate. This must be
	// set if a pod-level memory limit is not given.
	MemoryMB *int
	// CPU is the number of CPU units to allocate. 1024 CPU units is equivalent
	// to 1 vCPU on a machine. This must be set if a pod-level CPU limit is not
	// given.
	CPU *int
	// EnvVars are environment variables to make available in the container.
	EnvVars []EnvironmentVariable
	// RepoCreds are private repository credentials for using images that
	// require authentication.
	RepoCreds *RepositoryCredentials
	// PortMappings are mappings between the ports within the container to
	// allow network traffic.
	PortMappings []PortMapping
}

// NewECSContainerDefinition returns a new uninitialized container definition.
func NewECSContainerDefinition() *ECSContainerDefinition {
	return &ECSContainerDefinition{}
}

// SetName sets the friendly name for the container.
func (d *ECSContainerDefinition) SetName(name string) *ECSContainerDefinition {
	d.Name = &name
	return d
}

// SetImage sets the image for the container.
func (d *ECSContainerDefinition) SetImage(img string) *ECSContainerDefinition {
	d.Image = &img
	return d
}

// SetCommand sets the command for the container to run.
func (d *ECSContainerDefinition) SetCommand(cmd []string) *ECSContainerDefinition {
	d.Command = cmd
	return d
}

// SetWorkingDir sets the working directory where the container's commands
// will run.
func (d *ECSContainerDefinition) SetWorkingDir(dir string) *ECSContainerDefinition {
	d.WorkingDir = &dir
	return d
}

// SetMemoryMB sets the amount of memory (in MB) to allocate.
func (d *ECSContainerDefinition) SetMemoryMB(mem int) *ECSContainerDefinition {
	d.MemoryMB = &mem
	return d
}

// SetCPU sets the number of CPU units to allocate.
func (d *ECSContainerDefinition) SetCPU(cpu int) *ECSContainerDefinition {
	d.CPU = &cpu
	return d
}

// SetEnvironmentVariables sets the environment variables for the container.
// This overwrites any existing environment variables.
func (d *ECSContainerDefinition) SetEnvironmentVariables(envVars []EnvironmentVariable) *ECSContainerDefinition {
	d.EnvVars = envVars
	return d
}

// AddEnvironmentVariables adds new environment variables to the existing ones
// for the container.
func (d *ECSContainerDefinition) AddEnvironmentVariables(envVars ...EnvironmentVariable) *ECSContainerDefinition {
	d.EnvVars = append(d.EnvVars, envVars...)
	return d
}

// SetRepositoryCredentials sets the private repository credentials for using
// images that require authentication.
func (d *ECSContainerDefinition) SetRepositoryCredentials(creds RepositoryCredentials) *ECSContainerDefinition {
	d.RepoCreds = &creds
	return d
}

// SetPortMappings sets the port mappings for the container. This overwrites any
// existing port mappings.
func (d *ECSContainerDefinition) SetPortMappings(mappings []PortMapping) *ECSContainerDefinition {
	d.PortMappings = mappings
	return d
}

// AddPortMappings adds new port mappings to the existing ones for the
// container.
func (d *ECSContainerDefinition) AddPortMappings(mappings ...PortMapping) *ECSContainerDefinition {
	d.PortMappings = append(d.PortMappings, mappings...)
	return d
}

// Validate checks that the container definition is valid and sets defaults
// where possible.
func (d *ECSContainerDefinition) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(d.Image == nil, "must specify an image")
	catcher.NewWhen(d.Image != nil && *d.Image == "", "cannot specify an empty image")
	catcher.NewWhen(d.MemoryMB != nil && *d.MemoryMB <= 0, "must have positive memory value if non-default")
	catcher.NewWhen(d.CPU != nil && *d.CPU <= 0, "must have positive CPU value if non-default")
	for _, ev := range d.EnvVars {
		catcher.Wrapf(ev.Validate(), "environment variable '%s'", utility.FromStringPtr(ev.Name))
	}
	if d.RepoCreds != nil {
		catcher.Wrap(d.RepoCreds.Validate(), "invalid repository credentials")
	}
	for _, pm := range d.PortMappings {
		catcher.Wrapf(pm.Validate(), "invalid port mapping")
	}
	if catcher.HasErrors() {
		return catcher.Resolve()
	}

	if d.Name == nil {
		d.Name = utility.ToStringPtr(utility.RandomString())
	}

	return nil
}

// hash returns the hash digest of the definition.
func (d *ECSContainerDefinition) hash() string {
	h := newSHA1Hasher()
	if d.Name != nil {
		h.add(utility.FromStringPtr(d.Name))
	}

	if d.Image != nil {
		h.add(utility.FromStringPtr(d.Image))
	}

	if len(d.Command) != 0 {
		for _, arg := range d.Command {
			h.add(arg)
		}
	}

	if d.WorkingDir != nil {
		h.add(utility.FromStringPtr(d.WorkingDir))
	}

	if d.MemoryMB != nil {
		h.add(strconv.Itoa(utility.FromIntPtr(d.MemoryMB)))
	}

	if d.CPU != nil {
		h.add(strconv.Itoa(utility.FromIntPtr(d.CPU)))
	}

	if len(d.EnvVars) != 0 {
		h.add(newHashableEnvironmentVariables(d.EnvVars).hash())
	}

	if d.RepoCreds != nil {
		h.add(d.RepoCreds.hash())
	}

	if len(d.PortMappings) != 0 {
		h.add(newHashablePortMappings(d.PortMappings).hash())
	}

	return h.sum()
}

// hashableECSContainerDefinitions represents a hashable slice of ECS container
// definitions ordered by container name.
type hashableECSContainerDefinitions []ECSContainerDefinition

func newHashableContainerDefinitions(containerDefs []ECSContainerDefinition) hashableECSContainerDefinitions {
	hcd := hashableECSContainerDefinitions(containerDefs)
	sort.Sort(hcd)
	return hcd
}

// Len returns the number of container definitions.
func (hcd hashableECSContainerDefinitions) Len() int {
	return len(hcd)
}

// Less returns whether or not the name of the container definition at index i
// is lexicographically before the name of the container definition at index j.
func (hcd hashableECSContainerDefinitions) Less(i, j int) bool {
	return utility.FromStringPtr(hcd[i].Name) < utility.FromStringPtr(hcd[j].Name)
}

// Swap swaps the container definitions at indexes i and j.
func (hcd hashableECSContainerDefinitions) Swap(i, j int) {
	hcd[i], hcd[j] = hcd[j], hcd[i]
}

// hash returns the hash digest of the container definitions.
func (hcd hashableECSContainerDefinitions) hash() string {
	if !sort.IsSorted(hcd) {
		sort.Sort(hcd)
	}

	h := newSHA1Hasher()

	for _, cd := range hcd {
		h.add(cd.hash())
	}

	return h.sum()
}

// EnvironmentVariable represents an environment variable, which can be
// optionally backed by a secret.
type EnvironmentVariable struct {
	// Name is the name of the environment variable.
	Name *string
	// Value is the environment variable's non-secret value. This is required if
	// SecretOpts is not given.
	Value *string
	// SecretOpts are options to define a stored secret that the environment
	// variable refers to. This is required if the non-secret Value is not
	// given.
	SecretOpts *SecretOptions
}

// NewEnvironmentVariable returns a new uninitialized environment variable.
func NewEnvironmentVariable() *EnvironmentVariable {
	return &EnvironmentVariable{}
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

// SetSecretOptions sets the environment variable's secret options. This is
// mutually exclusive with setting the non-secret (EnvironmentVariable).Value.
func (e *EnvironmentVariable) SetSecretOptions(opts SecretOptions) *EnvironmentVariable {
	e.SecretOpts = &opts
	return e
}

// Validate checks that the environment variable name is given and that either
// the raw environment variable value or the referenced secret is given.
func (e *EnvironmentVariable) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(e.Name == nil, "must specify a name")
	catcher.NewWhen(e.Name != nil && *e.Name == "", "cannot specify an empty name")
	catcher.NewWhen(e.Value == nil && e.SecretOpts == nil, "must either specify a value or reference a secret")
	catcher.NewWhen(e.Value != nil && e.SecretOpts != nil, "cannot both specify a value and reference a secret")
	if e.SecretOpts != nil {
		catcher.Wrap(e.SecretOpts.Validate(), "invalid secret options")
	}
	return catcher.Resolve()
}

// hash is the hash digest of the environment variable.
func (e *EnvironmentVariable) hash() string {
	h := newSHA1Hasher()
	if e.Name != nil {
		h.add(utility.FromStringPtr(e.Name))
	}

	if e.Value != nil {
		h.add(utility.FromStringPtr(e.Value))
	}

	if e.SecretOpts != nil {
		h.add(e.SecretOpts.hash())
	}

	return h.sum()
}

// hashableEnvironmentVariables represents a slice of environment variables that
// can be hashed.
type hashableEnvironmentVariables []EnvironmentVariable

// newHashableEnvironmentVariables returns a sorted slice of hashable
// environment variables.
func newHashableEnvironmentVariables(ev []EnvironmentVariable) hashableEnvironmentVariables {
	hev := hashableEnvironmentVariables(ev)
	sort.Sort(hev)
	return hev
}

// Len returns the number of environment variables.
func (hev hashableEnvironmentVariables) Len() int {
	return len(hev)
}

// Less returns whether or not the name of the environment variable at index i
// is lexicographically before the name of the environment variable at index j.
func (hev hashableEnvironmentVariables) Less(i, j int) bool {
	return utility.FromStringPtr(hev[i].Name) < utility.FromStringPtr(hev[j].Name)
}

// Swap swaps the environment variables at indexes i and j.
func (hev hashableEnvironmentVariables) Swap(i, j int) {
	hev[i], hev[j] = hev[j], hev[i]
}

// hash returns the hash digest of the environment variables.
func (hev hashableEnvironmentVariables) hash() string {
	if !sort.IsSorted(hev) {
		sort.Sort(hev)
	}

	h := newSHA1Hasher()
	for _, ev := range hev {
		h.add(ev.hash())
	}

	return h.sum()
}

// SecretOptions represents a secret with a name and value that may or may not
// be owned by its container.
type SecretOptions struct {
	// ID is the unique resource identfier for an existing secret.
	ID *string
	// Name is the friendly name of the secret.
	Name *string
	// NewValue is the value of the secret if it must be created.
	NewValue *string
	// Owned determines whether or not the secret is owned by its container or
	// not.
	Owned *bool
}

// NewSecretOptions returns new uninitialized options for a secret.
func NewSecretOptions() *SecretOptions {
	return &SecretOptions{}
}

// SetID sets the unique resource identifier for an existing secret.
func (s *SecretOptions) SetID(id string) *SecretOptions {
	s.ID = &id
	return s
}

// SetName sets the friendly name of the secret.
func (s *SecretOptions) SetName(name string) *SecretOptions {
	s.Name = &name
	return s
}

// SetNewValue sets the value of the new secret to be created.
func (s *SecretOptions) SetNewValue(val string) *SecretOptions {
	s.NewValue = &val
	return s
}

// SetOwned returns whether or not the secret is owned by its container.
func (s *SecretOptions) SetOwned(owned bool) *SecretOptions {
	s.Owned = &owned
	return s
}

// Validate validates that the secret name is given and that either the secret
// already exists or the new secret's value is given.
func (s *SecretOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(s.ID == nil && s.NewValue == nil, "must specify either an existing secret ID or a new secret to be created")
	catcher.NewWhen(s.ID != nil && s.NewValue != nil, "cannot specify both an existing secret ID and a new secret to be created")
	catcher.NewWhen(s.NewValue != nil && s.Name == nil, "cannot specify a new secret to be created without a name")
	catcher.NewWhen(s.ID != nil && utility.FromStringPtr(s.ID) == "", "cannot specify an empty secret ID")
	return catcher.Resolve()
}

// hash returns the hash digest of the secret options.
func (s *SecretOptions) hash() string {
	h := newSHA1Hasher()
	if s.ID != nil {
		h.add(utility.FromStringPtr(s.ID))
	}

	if s.Name != nil {
		h.add(utility.FromStringPtr(s.Name))
	}

	if s.NewValue != nil {
		h.add(utility.FromStringPtr(s.NewValue))
	}

	if s.Owned != nil {
		h.add(strconv.FormatBool(utility.FromBoolPtr(s.Owned)))
	}

	return h.sum()
}

// RepositoryCredentials are credentials for using images from private
// repositories. The credentials must be stored in a secret vault.
type RepositoryCredentials struct {
	// ID is the unique resource identifier for an existing secret containing
	// the credentials for a private repository.
	ID *string
	// Name is the friendly name of the secret containing the credentials
	// for a private repository.
	Name *string
	// NewCreds are the new credentials to be stored. If this is unspecified,
	// the secrets are assumed to already exist.
	NewCreds *StoredRepositoryCredentials
	// Owned determines whether or not the secret is owned by its pod or not.
	Owned *bool
}

// NewRepositoryCredentials returns a new uninitialized set of repository
// credentials.
func NewRepositoryCredentials() *RepositoryCredentials {
	return &RepositoryCredentials{}
}

// SetID sets the unique resource identifier for an existing secret.
func (c *RepositoryCredentials) SetID(id string) *RepositoryCredentials {
	c.ID = &id
	return c
}

// SetName sets the friendly name of the secret containing the credentials.
func (c *RepositoryCredentials) SetName(name string) *RepositoryCredentials {
	c.Name = &name
	return c
}

// SetNewCredentials sets the new credentials to be stored.
func (c *RepositoryCredentials) SetNewCredentials(creds StoredRepositoryCredentials) *RepositoryCredentials {
	c.NewCreds = &creds
	return c
}

// SetOwned sets whether or not the secret credentials are owned by its pod or
// not.
func (c *RepositoryCredentials) SetOwned(owned bool) *RepositoryCredentials {
	c.Owned = &owned
	return c
}

// Validate check that the secret options are given and that either the
// new credentials to create are specified, or the secret already exists.
func (c *RepositoryCredentials) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(c.ID == nil && c.NewCreds == nil, "must specify either an existing secret ID or new credentials to create")
	catcher.NewWhen(c.ID != nil && c.NewCreds != nil, "cannot specify both an existing secret ID and a new secret to create")
	catcher.NewWhen(c.NewCreds != nil && c.Name == nil, "cannot specify a new secret to be created without a name")
	catcher.NewWhen(c.ID != nil && utility.FromStringPtr(c.ID) == "", "cannot specify an empty secret ID")
	if c.NewCreds != nil {
		catcher.Wrap(c.NewCreds.Validate(), "invalid new credentials to create")
	}
	return catcher.Resolve()
}

// hash returns the hash digest of the repository credentials.
func (c *RepositoryCredentials) hash() string {
	h := newSHA1Hasher()
	if c.ID != nil {
		h.add(utility.FromStringPtr(c.ID))
	}

	if c.Name != nil {
		h.add(utility.FromStringPtr(c.Name))
	}

	if c.NewCreds != nil {
		h.add(c.NewCreds.hash())
	}

	if c.Owned != nil {
		h.add(strconv.FormatBool(utility.FromBoolPtr(c.Owned)))
	}

	return h.sum()
}

// StoredRepositoryCredentials represents the storage format of repository
// credentials for using images from private repositories.
type StoredRepositoryCredentials struct {
	// Username is the username for authentication.
	Username *string `json:"username"`
	// Password is the password for authentication.
	Password *string `json:"password"`
}

// NewStoredRepositoryCredentials returns a new uninitialized set of repository
// credentials for storage.
func NewStoredRepositoryCredentials() *StoredRepositoryCredentials {
	return &StoredRepositoryCredentials{}
}

// SetUsername sets the stored repository credential's username.
func (c *StoredRepositoryCredentials) SetUsername(name string) *StoredRepositoryCredentials {
	c.Username = &name
	return c
}

// SetPassword sets the stored repository credential's password.
func (c *StoredRepositoryCredentials) SetPassword(pwd string) *StoredRepositoryCredentials {
	c.Password = &pwd
	return c
}

// Validate checks that the username and password are set.
func (c *StoredRepositoryCredentials) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(utility.FromStringPtr(c.Username) == "", "must specify a username")
	catcher.NewWhen(utility.FromStringPtr(c.Password) == "", "must specify a password")
	return catcher.Resolve()
}

// hash returns the hash digest of the stored repository credentials.
func (c *StoredRepositoryCredentials) hash() string {
	h := newSHA1Hasher()
	if c.Username != nil {
		h.add(utility.FromStringPtr(c.Username))
	}

	if c.Password != nil {
		h.add(utility.FromStringPtr(c.Password))
	}

	return h.sum()
}

// PortMapping represents a mapping from a container port to a port in the
// container instance. The transport protocol is TCP.
type PortMapping struct {
	// ContainerPort is the port within the container to expose to network
	// traffic.
	ContainerPort *int
	// HostPort is the port within the container instance to which the container
	// port will be bound.
	// If the pod's network mode is NetworkModeAWSVPC or NetworkModeHost, then
	// this will be set to the same value as ContainerPort.
	// If the pod's network mode is NetworkModeBridge, this can either be
	// explicitly set or omitted to be assigned a port at random.
	HostPort *int
}

// NewPortMapping returns a new uninitialized port mapping.
func NewPortMapping() *PortMapping {
	return &PortMapping{}
}

// SetContainerPort sets the port within the container to expose to network
// traffic.
func (m *PortMapping) SetContainerPort(port int) *PortMapping {
	m.ContainerPort = &port
	return m
}

// SetHostPort sets the port within the container instance to which the
// container port will be bound.
func (m *PortMapping) SetHostPort(port int) *PortMapping {
	m.HostPort = &port
	return m
}

// Validate checks that the required port mapping settings are given. It does
// not check that the pod-level network mode is valid with the port mapping.
func (m *PortMapping) Validate() error {
	const (
		minPort = 0
		maxPort = 2 << 15
	)
	catcher := grip.NewBasicCatcher()
	containerPort := utility.FromIntPtr(m.ContainerPort)
	catcher.NewWhen(m.ContainerPort == nil, "must specify a container port")
	catcher.ErrorfWhen(containerPort <= minPort || containerPort >= maxPort, "must specify a container port between %d-%d", minPort, maxPort)
	if m.HostPort != nil {
		hostPort := utility.FromIntPtr(m.HostPort)
		catcher.ErrorfWhen(hostPort <= minPort || hostPort >= maxPort, "must specify a container port between %d-%d", minPort, maxPort)
	}
	return catcher.Resolve()
}

// hash returns the hash digest of the port mapping.
func (m *PortMapping) hash() string {
	h := newSHA1Hasher()
	if m.ContainerPort != nil {
		h.add(strconv.Itoa(utility.FromIntPtr(m.ContainerPort)))
	}

	if m.HostPort != nil {
		h.add(strconv.Itoa(utility.FromIntPtr(m.HostPort)))
	}

	return h.sum()
}

type hashablePortMappings []PortMapping

// newHashablePortMappings returns a sorted slice of hashable port mappings.
func newHashablePortMappings(pm []PortMapping) hashablePortMappings {
	hpm := hashablePortMappings(pm)
	sort.Sort(hpm)
	return hpm
}

// Len returns the number of port mappings.
func (hpm hashablePortMappings) Len() int {
	return len(hpm)
}

// Less returns whether or not the container port for the mapping at index i is
// less than the container port for the mapping at index j. If they're equal,
// the host ports are compared.
func (hpm hashablePortMappings) Less(i, j int) bool {
	cpi, cpj := utility.FromIntPtr(hpm[i].ContainerPort), utility.FromIntPtr(hpm[j].ContainerPort)
	if cpi == cpj {
		return utility.FromIntPtr(hpm[i].HostPort) < utility.FromIntPtr(hpm[j].HostPort)
	}

	return cpi < cpj
}

// Swap swaps the port mappings at indexes i and j.
func (hpm hashablePortMappings) Swap(i, j int) {
	hpm[i], hpm[j] = hpm[j], hpm[i]
}

// hash returns the hash digest of the port mappings.
func (hpm hashablePortMappings) hash() string {
	if !sort.IsSorted(hpm) {
		sort.Sort(hpm)
	}

	h := newSHA1Hasher()

	for _, pm := range hpm {
		h.add(pm.hash())
	}

	return h.sum()
}

// ECSPodExecutionOptions represent options to configure how a pod is started.
type ECSPodExecutionOptions struct {
	// Cluster is the name of the cluster where the pod will run. If none is
	// specified, this will run in the default cluster.
	Cluster *string
	// CapacityProvider is the name of the capacity provider that the pod will
	// use, which in turn determines the infrastructure that the pod will run
	// on. If none is specified, this will run in the default capacity provider.
	CapacityProvider *string
	// PlacementOptions specify options that determine how a pod is assigned to
	// a container instance.
	PlacementOpts *ECSPodPlacementOptions
	// AWSVPCOpts specify additional networking configuration when using
	// NetworkModeAWSVPC.
	AWSVPCOpts *AWSVPCOptions
	// SupportsDebugMode indicates that the ECS pod should support debugging, so
	// you can run exec in the pod's containers. In order for this to work, the
	// pod must have the correct permissions to perform this operation when it's
	// defined. By default, this is false.
	SupportsDebugMode *bool
	// Tags are any tags to apply to the running pods.
	Tags map[string]string
}

// NewECSPodExecutionOptions returns new uninitialized options to run a pod.
func NewECSPodExecutionOptions() *ECSPodExecutionOptions {
	return &ECSPodExecutionOptions{}
}

// SetCluster sets the name of the cluster where the pod will run.
func (o *ECSPodExecutionOptions) SetCluster(cluster string) *ECSPodExecutionOptions {
	o.Cluster = &cluster
	return o
}

// SetCapacityProvider sets the name of the capacity provider that the pod will
// use.
func (o *ECSPodExecutionOptions) SetCapacityProvider(provider string) *ECSPodExecutionOptions {
	o.CapacityProvider = &provider
	return o
}

// SetPlacementOptions sets the options that determine how a pod is assigned to
// a container instance.
func (o *ECSPodExecutionOptions) SetPlacementOptions(opts ECSPodPlacementOptions) *ECSPodExecutionOptions {
	o.PlacementOpts = &opts
	return o
}

// SetAWSVPCOptions sets the options that configure a pod using
// NetworkModeAWSVPC.
func (o *ECSPodExecutionOptions) SetAWSVPCOptions(opts AWSVPCOptions) *ECSPodExecutionOptions {
	o.AWSVPCOpts = &opts
	return o
}

// SetSupportsDebugMode sets whether or not the pod can run with debug mode
// enabled.
func (o *ECSPodExecutionOptions) SetSupportsDebugMode(supported bool) *ECSPodExecutionOptions {
	o.SupportsDebugMode = &supported
	return o
}

// SetTags sets the tags for the pod itself when it is run. This overwrites any
// existing tags.
func (o *ECSPodExecutionOptions) SetTags(tags map[string]string) *ECSPodExecutionOptions {
	o.Tags = tags
	return o
}

// AddTags adds new tags to the existing ones for the pod itself when it is run.
func (o *ECSPodExecutionOptions) AddTags(tags map[string]string) *ECSPodExecutionOptions {
	if o.Tags == nil {
		o.Tags = map[string]string{}
	}
	for k, v := range tags {
		o.Tags[k] = v
	}
	return o
}

// Validate checks that the placement options are valid.
func (o *ECSPodExecutionOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	if o.PlacementOpts != nil {
		catcher.Wrap(o.PlacementOpts.Validate(), "invalid placement options")
	}
	if o.AWSVPCOpts != nil {
		catcher.Wrap(o.AWSVPCOpts.Validate(), "invalid AWSVPC options")
	}
	if catcher.HasErrors() {
		return catcher.Resolve()
	}

	if o.PlacementOpts == nil {
		o.PlacementOpts = NewECSPodPlacementOptions().SetStrategy(StrategyBinpack).SetStrategyParameter(StrategyParamBinpackMemory)
	}

	return nil
}

// MergeECSPodExecutionOptions merges all the given options to run an ECS pod.
// Options are applied in the order that they're specified and conflicting
// options are overwritten.
func MergeECSPodExecutionOptions(opts ...ECSPodExecutionOptions) ECSPodExecutionOptions {
	merged := ECSPodExecutionOptions{}

	for _, opt := range opts {
		if opt.Cluster != nil {
			merged.Cluster = opt.Cluster
		}

		if opt.CapacityProvider != nil {
			merged.CapacityProvider = opt.CapacityProvider
		}

		if opt.PlacementOpts != nil {
			merged.PlacementOpts = opt.PlacementOpts
		}

		if opt.AWSVPCOpts != nil {
			merged.AWSVPCOpts = opt.AWSVPCOpts
		}

		if opt.SupportsDebugMode != nil {
			merged.SupportsDebugMode = opt.SupportsDebugMode
		}

		if opt.Tags != nil {
			merged.Tags = opt.Tags
		}
	}

	return merged
}

// ECSPodPlacementOptions represent options to control how an ECS pod is
// assigned to a container instance.
type ECSPodPlacementOptions struct {
	// Group is the name of a logical collection of ECS pods. Pods within the
	// same group can support additional placement configuration.
	Group *string

	// Strategy is the overall placement strategy. By default, it uses the
	// binpack strategy.
	Strategy *ECSPlacementStrategy

	// StrategyParameter is the parameter that determines how the placement
	// strategy optimizes pod placement. The default value depends on the
	// strategy:
	// If the strategy is spread, it defaults to "host".
	// If the strategy is binpack, it defaults to "memory".
	// If the strategy is random, this does not apply.
	StrategyParameter *ECSStrategyParameter

	// InstanceFilter is a set of query expressions that restrict the placement
	// of the pod to a set of container instances in the cluster that match the
	// query filter. As a special case, if ConstraintDistinctInstance is the
	// specified filter, it will place each pod in the pod's group on a
	// different instance. Otherwise, all filters are assumed to use the ECS
	// cluster query language to filter the candidate set of instances for a
	// pod. Docs:
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/cluster-query-language.html
	InstanceFilters []string
}

// NewECSPodPlacementOptions creates new options to specify how an ECS pod
// should be assigned to a container instance.
func NewECSPodPlacementOptions() *ECSPodPlacementOptions {
	return &ECSPodPlacementOptions{}
}

// SetGroup sets the name of the group that the pod belongs to.
func (o *ECSPodPlacementOptions) SetGroup(group string) *ECSPodPlacementOptions {
	o.Group = &group
	return o
}

// SetStrategy sets the strategy for placing the pod on a container instance.
func (o *ECSPodPlacementOptions) SetStrategy(s ECSPlacementStrategy) *ECSPodPlacementOptions {
	o.Strategy = &s
	return o
}

// SetStrategyParameter sets the parameter to optimize for when placing the pod
// on a container instance.
func (o *ECSPodPlacementOptions) SetStrategyParameter(p ECSStrategyParameter) *ECSPodPlacementOptions {
	o.StrategyParameter = &p
	return o
}

// SetInstanceFilters sets the instance filters to constrain pod placement to
// one in the set of matching container instances.
func (o *ECSPodPlacementOptions) SetInstanceFilters(filters []string) *ECSPodPlacementOptions {
	o.InstanceFilters = filters
	return o
}

// AddInstanceFilters adds new instance filters to the existing ones to
// constrain pod placement to one in the set of matching container instances.
func (o *ECSPodPlacementOptions) AddInstanceFilters(filters ...string) *ECSPodPlacementOptions {
	o.InstanceFilters = append(o.InstanceFilters, filters...)
	return o
}

// Validate checks that the the strategy and its parameter to optimize are a
// valid combination.
func (o *ECSPodPlacementOptions) Validate() error {
	catcher := grip.NewBasicCatcher()

	catcher.ErrorfWhen(o.Group != nil && *o.Group == "", "cannot specify an empty group name")

	if o.Strategy != nil {
		catcher.Add(o.Strategy.Validate())

		if o.StrategyParameter != nil {
			catcher.ErrorfWhen(*o.Strategy == StrategyBinpack && *o.StrategyParameter != StrategyParamBinpackMemory && *o.StrategyParameter != StrategyParamBinpackCPU, "strategy parameter cannot be '%s' when the strategy is '%s'", *o.StrategyParameter, *o.Strategy)
			catcher.ErrorfWhen(*o.Strategy != StrategySpread && *o.StrategyParameter == StrategyParamSpreadHost, "strategy parameter cannot be '%s' when the strategy is not '%s'", *o.StrategyParameter, StrategySpread)
		}
	}

	if catcher.HasErrors() {
		return catcher.Resolve()
	}

	if o.Strategy == nil {
		strategy := StrategyBinpack
		o.Strategy = &strategy
	}

	if o.Strategy != nil && o.StrategyParameter == nil {
		if *o.Strategy == StrategyBinpack {
			o.StrategyParameter = utility.ToStringPtr(StrategyParamBinpackMemory)
		}
		if *o.Strategy == StrategySpread {
			o.StrategyParameter = utility.ToStringPtr(StrategyParamSpreadHost)
		}
	}

	return nil
}

// ECSPlacementStrategy represents a placement strategy for ECS pods.
type ECSPlacementStrategy string

const (
	// StrategySpread indicates that the ECS pod will be assigned in such a way
	// to achieve an even spread based on the given ECSStrategyParameter.
	StrategySpread ECSPlacementStrategy = ecs.PlacementStrategyTypeSpread
	// StrategyRandom indicates that the ECS pod should be assigned to a
	// container instance randomly.
	StrategyRandom ECSPlacementStrategy = ecs.PlacementStrategyTypeRandom
	// StrategyBinpack indicates that the the ECS pod will be placed on a
	// container instance with the least amount of memory or CPU that will be
	// sufficient for the pod's requirements if possible.
	StrategyBinpack ECSPlacementStrategy = ecs.PlacementStrategyTypeBinpack
)

// Validate checks that the ECS pod status is one of the recognized placement
// strategies.
func (s ECSPlacementStrategy) Validate() error {
	switch s {
	case StrategySpread, StrategyRandom, StrategyBinpack:
		return nil
	default:
		return errors.Errorf("unrecognized placement strategy '%s'", s)
	}
}

// ECSStrategyParameter represents the parameter that ECS will use with its
// strategy to schedule pods on container instances.
type ECSStrategyParameter = string

const (
	// StrategyParamBinpackMemory indicates ECS should optimize its binpacking
	// strategy based on memory usage.
	StrategyParamBinpackMemory ECSStrategyParameter = "memory"
	// StrategyParamBinpackCPU indicates ECS should optimize its binpacking
	// strategy based on CPU usage.
	StrategyParamBinpackCPU ECSStrategyParameter = "cpu"
	// StrategyParamSpreadHost indicates the ECS should spread pods evenly
	// across all container instances (i.e. hosts).
	StrategyParamSpreadHost ECSStrategyParameter = "host"
)

const (
	// ConstraintDistinctInstance is a container instance filter indicating that
	// ECS should place all pods in the same group on different container
	// instances.
	ConstraintDistinctInstance = "distinctInstance"
)

// AWSVPCOptions represent options to configure networking when the network mode
// is NetworkModeAWSVPC.
type AWSVPCOptions struct {
	// Subnets are all the subnet IDs associated with the pod. This is required.
	Subnets []string
	// SecurityGroups are all the security group IDs associated with the pod. If
	// this is not specified, the default security group for the VPC will be
	// used.
	SecurityGroups []string
}

// NewAWSVPCOptions returns new uninitialized options for NetworkModeAWSVPC.
func NewAWSVPCOptions() *AWSVPCOptions {
	return &AWSVPCOptions{}
}

// SetSubnets sets the subnets associated with the pod. This overwrites any
// existing subnets.
func (o *AWSVPCOptions) SetSubnets(subnets []string) *AWSVPCOptions {
	o.Subnets = subnets
	return o
}

// AddSubnets adds new subnets to the existing ones for the pod.
func (o *AWSVPCOptions) AddSubnets(subnets ...string) *AWSVPCOptions {
	o.Subnets = append(o.Subnets, subnets...)
	return o
}

// SetSecurityGroups sets the security groups associated with the pod. This
// overwrites any existing security groups.
func (o *AWSVPCOptions) SetSecurityGroups(groups []string) *AWSVPCOptions {
	o.SecurityGroups = groups
	return o
}

// AddSecurityGroups adds new security groups to the existing ones for the pod.
func (o *AWSVPCOptions) AddSecurityGroups(groups ...string) *AWSVPCOptions {
	o.SecurityGroups = append(o.SecurityGroups, groups...)
	return o
}

// Validate checks that subnets are set.
func (o *AWSVPCOptions) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(len(o.Subnets) == 0, "must specify at least one subnet")
	return catcher.Resolve()
}

// ECSNetworkMode represents possible kinds of networking configuration for a
// pod in ECS.
type ECSNetworkMode string

const (
	// NetworkModeNone indicates that networking is disabled entirely. The pod
	// does not allow any external network connectivity and container ports
	// cannot be mapped.
	NetworkModeNone ECSNetworkMode = "none"
	// NetworkModeAWSVPC indicates that the pod will be allocated its own
	// virtual network interface and IPv4 address. This is supported for Linux
	// and Window containers.
	NetworkModeAWSVPC ECSNetworkMode = "awsvpc"
	// NetworkModeBridge indicates that the container will use Docker's built-in
	// virtual network inside the container instance running the pod. This is
	// only supported for Linux containers.
	NetworkModeBridge ECSNetworkMode = "bridge"
	// NetworkModeHost indicates that the container will directly map its ports
	// to the underlying container instance's network interface.
	// This is only supported for Linux containers.
	NetworkModeHost ECSNetworkMode = "host"
)

// Validate checks that the ECS network mode is one of the recognized modes.
func (m ECSNetworkMode) Validate() error {
	switch m {
	case NetworkModeNone, NetworkModeAWSVPC, NetworkModeBridge, NetworkModeHost:
		return nil
	default:
		return errors.Errorf("unrecognized network mode '%s'", m)
	}
}

// ECSTaskDefinition represents options for an existing ECS task definition.
type ECSTaskDefinition struct {
	// ID is the ID of the task definition, which should already exist.
	ID *string
	// Owned determines whether or not the task definition is owned by its pod
	// or not.
	Owned *bool
}

// NewECSTaskDefinition returns a new uninitialized task definition.
func NewECSTaskDefinition() *ECSTaskDefinition {
	return &ECSTaskDefinition{}
}

// SetID sets the task definition ID.
func (d *ECSTaskDefinition) SetID(id string) *ECSTaskDefinition {
	d.ID = &id
	return d
}

// SetOwned sets if the task definition is owned by its pod.
func (d *ECSTaskDefinition) SetOwned(owned bool) *ECSTaskDefinition {
	d.Owned = &owned
	return d
}

// Validate checsk that the task definition ID is given.
func (d *ECSTaskDefinition) Validate() error {
	catcher := grip.NewBasicCatcher()
	catcher.NewWhen(d.ID == nil, "must specify a task definition ID")
	catcher.NewWhen(utility.FromStringPtr(d.ID) == "", "must specify a non-empty task definition ID")
	return catcher.Resolve()
}
