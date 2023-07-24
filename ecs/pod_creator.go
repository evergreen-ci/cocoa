package ecs

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// BasicPodCreator provides an cocoa.ECSPodCreator implementation to create
// AWS ECS pods.
type BasicPodCreator struct {
	client cocoa.ECSClient
	vault  cocoa.Vault
}

// NewBasicPodCreator creates a helper to create pods backed by AWS ECS.
func NewBasicPodCreator(c cocoa.ECSClient, v cocoa.Vault) (*BasicPodCreator, error) {
	if c == nil {
		return nil, errors.New("missing client")
	}
	return &BasicPodCreator{
		client: c,
		vault:  v,
	}, nil
}

// CreatePod creates a new pod backed by AWS ECS.
func (pc *BasicPodCreator) CreatePod(ctx context.Context, opts ...cocoa.ECSPodCreationOptions) (cocoa.ECSPod, error) {
	mergedPodCreationOpts := cocoa.MergeECSPodCreationOptions(opts...)
	var mergedPodExecutionOpts cocoa.ECSPodExecutionOptions
	if mergedPodCreationOpts.ExecutionOpts != nil {
		mergedPodExecutionOpts = *mergedPodCreationOpts.ExecutionOpts
	}

	if err := mergedPodCreationOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod creation options")
	}

	if err := mergedPodExecutionOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod execution options")
	}

	pdm, err := NewBasicPodDefinitionManager(*NewBasicPodDefinitionManagerOptions().
		SetClient(pc.client).
		SetVault(pc.vault))
	if err != nil {
		return nil, errors.Wrap(err, "initializing pod definition manager")
	}

	pdi, err := pdm.CreatePodDefinition(ctx, mergedPodCreationOpts.DefinitionOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating pod definition")
	}
	mergedPodCreationOpts.DefinitionOpts = pdi.DefinitionOpts

	taskDef := cocoa.NewECSTaskDefinition().
		SetID(pdi.ID).
		SetOwned(true)

	task, err := pc.runTask(ctx, mergedPodExecutionOpts, *taskDef)
	if err != nil {
		return nil, errors.Wrap(err, "running task")
	}

	p, err := pc.createPod(utility.FromStringPtr(mergedPodExecutionOpts.Cluster), task, *taskDef, mergedPodCreationOpts.DefinitionOpts.ContainerDefinitions)
	if err != nil {
		return nil, errors.Wrap(err, "creating pod after requesting task")
	}

	return p, nil
}

// CreatePodFromExistingDefinition creates a new pod backed by AWS ECS from an
// existing definition.
func (pc *BasicPodCreator) CreatePodFromExistingDefinition(ctx context.Context, def cocoa.ECSTaskDefinition, opts ...cocoa.ECSPodExecutionOptions) (cocoa.ECSPod, error) {
	if err := def.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid task definition")
	}

	mergedPodExecutionOpts := cocoa.MergeECSPodExecutionOptions(opts...)
	if err := mergedPodExecutionOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod execution options")
	}

	taskDef := cocoa.NewECSTaskDefinition().
		SetID(utility.FromStringPtr(def.ID)).
		SetOwned(utility.FromBoolPtr(def.Owned))

	task, err := pc.runTask(ctx, mergedPodExecutionOpts, *taskDef)
	if err != nil {
		return nil, errors.Wrap(err, "running task")
	}

	p, err := pc.createPod(utility.FromStringPtr(mergedPodExecutionOpts.Cluster), task, *taskDef, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating pod after requesting task")
	}

	return p, nil
}

// createPod creates the basic ECS pod after its ECS task has been requested.
func (pc *BasicPodCreator) createPod(cluster string, task types.Task, def cocoa.ECSTaskDefinition, containerDefs []cocoa.ECSContainerDefinition) (*BasicPod, error) {
	resources := cocoa.NewECSPodResources().
		SetCluster(cluster).
		SetContainers(pc.translateContainerResources(task.Containers, containerDefs)).
		SetTaskDefinition(def).
		SetTaskID(utility.FromStringPtr(task.TaskArn))

	podOpts := NewBasicPodOptions().
		SetClient(pc.client).
		SetVault(pc.vault).
		SetStatusInfo(translatePodStatusInfo(task)).
		SetResources(*resources)

	p, err := NewBasicPod(podOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating basic pod")
	}

	return p, nil
}

// registerTaskDefinition makes the request to register an ECS task definition
// from the options and checks that it returns a valid task definition.
func registerTaskDefinition(ctx context.Context, c cocoa.ECSClient, opts cocoa.ECSPodDefinitionOptions) (*types.TaskDefinition, error) {
	in := exportPodDefinitionOptions(opts)
	out, err := c.RegisterTaskDefinition(ctx, in)
	if err != nil {
		return nil, errors.Wrap(err, "registering task definition")
	}

	if err := validateRegisterTaskDefinitionOutput(out); err != nil {
		return nil, errors.Wrap(err, "validating response from registering task definition")
	}

	return out.TaskDefinition, nil
}

// validateRegisterTaskDefinitionOutput checks that the output from registering
// a task definition is a valid task definition.
func validateRegisterTaskDefinitionOutput(out *ecs.RegisterTaskDefinitionOutput) error {
	if out.TaskDefinition == nil {
		return errors.New("expected a task definition from ECS, but none was returned")
	}
	if utility.FromStringPtr(out.TaskDefinition.TaskDefinitionArn) == "" {
		return errors.New("received a task definition, but it is missing an ARN")
	}
	return nil
}

// runTask makes the request to run an ECS task from the execution options and
// task definition and checks that it returns a valid task.
func (pc *BasicPodCreator) runTask(ctx context.Context, opts cocoa.ECSPodExecutionOptions, def cocoa.ECSTaskDefinition) (types.Task, error) {
	in := pc.exportTaskExecutionOptions(opts, def)
	out, err := pc.client.RunTask(ctx, in)
	if err != nil {
		return types.Task{}, errors.Wrapf(err, "running task for definition '%s' in cluster '%s'", utility.FromStringPtr(in.TaskDefinition), utility.FromStringPtr(in.Cluster))
	}

	if err := pc.validateRunTaskOutput(out); err != nil {
		return types.Task{}, errors.Wrap(err, "validating response from running task")
	}

	return out.Tasks[0], nil
}

// validateRunTaskOutput checks that the output from running a task contains no
// errors and includes the necessary information for the expected tasks.
func (pc *BasicPodCreator) validateRunTaskOutput(out *ecs.RunTaskOutput) error {
	if len(out.Failures) > 0 {
		catcher := grip.NewBasicCatcher()
		for _, f := range out.Failures {
			catcher.Add(ConvertFailureToError(f))
		}
		return errors.Wrap(catcher.Resolve(), "running task")
	}

	if len(out.Tasks) == 0 {
		return errors.New("expected a task to be running in ECS, but none was returned")
	}
	if out.Tasks[0].TaskArn == nil {
		return errors.New("received a task, but it is missing an ARN")
	}

	return nil
}

// createSecrets creates any necessary secrets from the secret environment
// variables for each container. Once the secrets are created, their IDs are
// set.
func createSecrets(ctx context.Context, v cocoa.Vault, opts *cocoa.ECSPodDefinitionOptions) error {
	var defs []cocoa.ECSContainerDefinition
	for i, def := range opts.ContainerDefinitions {
		defs = append(defs, def)
		containerName := utility.FromStringPtr(def.Name)

		var envVars []cocoa.EnvironmentVariable
		for _, envVar := range def.EnvVars {
			if envVar.SecretOpts == nil || envVar.SecretOpts.NewValue == nil {
				envVars = append(envVars, envVar)
				defs[i].EnvVars = append(defs[i].EnvVars, envVar)
				continue
			}

			id, err := createSecret(ctx, v, *envVar.SecretOpts)
			if err != nil {
				return errors.Wrapf(err, "creating secret environment variable '%s' for container '%s'", utility.FromStringPtr(opts.Name), containerName)
			}

			updated := *envVar.SecretOpts
			updated.SetID(id)
			envVar.SecretOpts = &updated
			envVars = append(envVars, envVar)
		}

		defs[i].EnvVars = envVars

		repoCreds := def.RepoCreds
		if def.RepoCreds != nil && def.RepoCreds.NewCreds != nil {
			val, err := json.Marshal(def.RepoCreds.NewCreds)
			if err != nil {
				return errors.Wrap(err, "formatting new repository credentials to create")
			}
			secretOpts := cocoa.NewSecretOptions().
				SetName(utility.FromStringPtr(def.RepoCreds.Name)).
				SetNewValue(string(val))
			id, err := createSecret(ctx, v, *secretOpts)
			if err != nil {
				return errors.Wrapf(err, "creating repository credentials for container '%s'", utility.FromStringPtr(def.Name))
			}

			updated := *def.RepoCreds
			updated.SetID(id)
			repoCreds = &updated
		}

		defs[i].RepoCreds = repoCreds
	}

	// Since the options format makes extensive use of pointers and pointers may
	// be shared between the input and the options used during pod creation, we
	// have to avoid mutating the original input. Therefore, replace the
	// entire slice of container definitions to create a separate slice in
	// memory and avoid mutating the original input's container definitions.
	opts.ContainerDefinitions = defs

	return nil
}

// createSecret creates a single secret. It returns the newly-created secret's
// ID.
func createSecret(ctx context.Context, v cocoa.Vault, secret cocoa.SecretOptions) (id string, err error) {
	if v == nil {
		return "", errors.New("no vault was specified")
	}
	return v.CreateSecret(ctx, *cocoa.NewNamedSecret().
		SetName(utility.FromStringPtr(secret.Name)).
		SetValue(utility.FromStringPtr(secret.NewValue)))
}

// ExportTags converts a mapping of tag names to values into ECS tags.
func ExportTags(tags map[string]string) []types.Tag {
	var ecsTags []types.Tag

	for k, v := range tags {
		var tag types.Tag
		tag.Key = aws.String(k)
		tag.Value = aws.String(v)
		ecsTags = append(ecsTags, tag)
	}

	return ecsTags
}

// exportOverrides converts options to override the pod definition into its
// equivalent ECS task override options.
func (pc *BasicPodCreator) exportOverrides(opts *cocoa.ECSOverridePodDefinitionOptions) *types.TaskOverride {
	if opts == nil {
		return nil
	}

	var overrides types.TaskOverride

	overrides.ContainerOverrides = pc.exportOverrideContainerDefinitions(opts.ContainerDefinitions)

	if opts.MemoryMB != nil {
		overrides.Memory = aws.String(strconv.Itoa(*opts.MemoryMB))
	}
	if opts.CPU != nil {
		overrides.Cpu = aws.String(strconv.Itoa(*opts.CPU))
	}
	if opts.TaskRole != nil {
		overrides.TaskRoleArn = opts.TaskRole
	}
	if opts.ExecutionRole != nil {
		overrides.ExecutionRoleArn = opts.ExecutionRole
	}

	return &overrides
}

// exportOverrideContainerDefinitions converts options to override container
// definitions into equivalent ECS container overrides.
func (pc *BasicPodCreator) exportOverrideContainerDefinitions(defs []cocoa.ECSOverrideContainerDefinition) []types.ContainerOverride {
	var containerOverrides []types.ContainerOverride

	for _, def := range defs {
		var containerOverride types.ContainerOverride
		if def.Command != nil {
			containerOverride.Command = def.Command
		}
		if def.MemoryMB != nil {
			containerOverride.Memory = aws.Int32(int32(*def.MemoryMB))
		}
		if def.CPU != nil {
			containerOverride.Cpu = aws.Int32(int32(*def.CPU))
		}

		var envVars []types.KeyValuePair
		for _, envVar := range def.EnvVars {
			var pair types.KeyValuePair
			pair.Name = envVar.Name
			pair.Value = envVar.Value
			envVars = append(envVars, pair)
		}
		containerOverride.Environment = envVars

		containerOverride.Name = def.Name
		containerOverrides = append(containerOverrides, containerOverride)
	}

	return containerOverrides
}

// exportStrategy converts the strategy and parameter into an ECS placement
// strategy.
func (pc *BasicPodCreator) exportStrategy(opts *cocoa.ECSPodPlacementOptions) []types.PlacementStrategy {
	var placementStrat types.PlacementStrategy
	placementStrat.Type = types.PlacementStrategyType(*opts.Strategy)
	placementStrat.Field = opts.StrategyParameter
	return []types.PlacementStrategy{placementStrat}
}

// exportPlacementConstraints converts the placement options into ECS placement
// constraints.
func (pc *BasicPodCreator) exportPlacementConstraints(opts *cocoa.ECSPodPlacementOptions) []types.PlacementConstraint {
	var constraints []types.PlacementConstraint

	for _, filter := range opts.InstanceFilters {
		var constraint types.PlacementConstraint
		if filter == cocoa.ConstraintDistinctInstance {
			constraint.Type = types.PlacementConstraintType(filter)
		} else {
			constraint.Type = "memberOf"
			constraint.Expression = aws.String(filter)
		}
		constraints = append(constraints, constraint)
	}

	return constraints
}

// exportEnvVars converts the non-secret environment variables into ECS
// environment variables.
func exportEnvVars(envVars []cocoa.EnvironmentVariable) []types.KeyValuePair {
	var converted []types.KeyValuePair

	for _, envVar := range envVars {
		if envVar.SecretOpts != nil {
			continue
		}
		var pair types.KeyValuePair
		pair.Name = envVar.Name
		pair.Value = envVar.Value
		converted = append(converted, pair)
	}

	return converted
}

// exportSecrets converts environment variables backed by secrets into ECS
// Secrets.
func exportSecrets(envVars []cocoa.EnvironmentVariable) []types.Secret {
	var secrets []types.Secret

	for _, envVar := range envVars {
		if envVar.SecretOpts == nil {
			continue
		}

		var secret types.Secret
		secret.Name = envVar.Name
		secret.ValueFrom = envVar.SecretOpts.ID
		secrets = append(secrets, secret)
	}

	return secrets
}

// translateContainerResources translates the containers and stored secrets
// into the resources associated with each container.
func (pc *BasicPodCreator) translateContainerResources(containers []types.Container, defs []cocoa.ECSContainerDefinition) []cocoa.ECSContainerResources {
	var resources []cocoa.ECSContainerResources

	for _, container := range containers {
		name := utility.FromStringPtr(container.Name)
		res := cocoa.NewECSContainerResources().
			SetContainerID(utility.FromStringPtr(container.ContainerArn)).
			SetName(name).
			SetSecrets(pc.translateContainerSecrets(defs))
		resources = append(resources, *res)
	}

	return resources
}

// translateContainerSecrets translates the given secrets for a container into
// a slice of container secrets.
func (pc *BasicPodCreator) translateContainerSecrets(defs []cocoa.ECSContainerDefinition) []cocoa.ContainerSecret {
	var translated []cocoa.ContainerSecret

	for _, def := range defs {
		for _, envVar := range def.EnvVars {
			if envVar.SecretOpts == nil {
				continue
			}

			cs := cocoa.NewContainerSecret().
				SetID(utility.FromStringPtr(envVar.SecretOpts.ID)).
				SetOwned(utility.FromBoolPtr(envVar.SecretOpts.Owned))
			if name := utility.FromStringPtr(envVar.SecretOpts.Name); name != "" {
				cs.SetName(name)
			}
			translated = append(translated, *cs)
		}

		if def.RepoCreds != nil {
			cs := cocoa.NewContainerSecret().
				SetID(utility.FromStringPtr(def.RepoCreds.ID)).
				SetOwned(utility.FromBoolPtr(def.RepoCreds.Owned))
			if name := utility.FromStringPtr(def.RepoCreds.Name); name != "" {
				cs.SetName(name)
			}
			translated = append(translated, *cs)
		}

	}

	return translated
}

// translatePodStatusInfo translates an ECS task to its equivalent cocoa
// status information.
func translatePodStatusInfo(task types.Task) cocoa.ECSPodStatusInfo {
	lastStatus := TaskStatus(utility.FromStringPtr(task.LastStatus)).ToCocoaStatus()
	return *cocoa.NewECSPodStatusInfo().
		SetStatus(lastStatus).
		SetContainers(translateContainerStatusInfo(task.Containers))
}

// translateContainerStatusInfo translates an ECS container to its equivalent
// cocoa container status information.
func translateContainerStatusInfo(containers []types.Container) []cocoa.ECSContainerStatusInfo {
	var statuses []cocoa.ECSContainerStatusInfo

	for _, container := range containers {
		lastStatus := TaskStatus(utility.FromStringPtr(container.LastStatus)).ToCocoaStatus()
		status := cocoa.NewECSContainerStatusInfo().
			SetContainerID(utility.FromStringPtr(container.ContainerArn)).
			SetName(utility.FromStringPtr(container.Name)).
			SetStatus(lastStatus)
		statuses = append(statuses, *status)
	}

	return statuses
}

// exportPodDefinitionOptions converts options to create a pod definition into
// its equivalent ECS task definition.
func exportPodDefinitionOptions(opts cocoa.ECSPodDefinitionOptions) *ecs.RegisterTaskDefinitionInput {
	var taskDef ecs.RegisterTaskDefinitionInput

	taskDef.ContainerDefinitions = exportContainerDefinitions(opts.ContainerDefinitions)
	taskDef.Family = opts.Name
	taskDef.Tags = ExportTags(opts.Tags)

	if mem := utility.FromIntPtr(opts.MemoryMB); mem != 0 {
		taskDef.Memory = aws.String(strconv.Itoa(mem))
	}

	if cpu := utility.FromIntPtr(opts.CPU); cpu != 0 {
		taskDef.Cpu = aws.String(strconv.Itoa(cpu))
	}

	if opts.TaskRole != nil {
		taskDef.TaskRoleArn = opts.TaskRole
	}
	if opts.ExecutionRole != nil {
		taskDef.ExecutionRoleArn = opts.ExecutionRole
	}

	if opts.NetworkMode != nil {
		taskDef.NetworkMode = types.NetworkMode(*opts.NetworkMode)
	}

	return &taskDef
}

// exportContainerDefinition converts container definitions into their
// equivalent ECS container definition.
func exportContainerDefinitions(defs []cocoa.ECSContainerDefinition) []types.ContainerDefinition {
	var containerDefs []types.ContainerDefinition

	for _, def := range defs {
		var containerDef types.ContainerDefinition
		if mem := utility.FromIntPtr(def.MemoryMB); mem != 0 {
			containerDef.Memory = aws.Int32(int32(mem))
		}
		if cpu := utility.FromIntPtr(def.CPU); cpu != 0 {
			containerDef.Cpu = int32(cpu)
		}
		if dir := utility.FromStringPtr(def.WorkingDir); dir != "" {
			containerDef.WorkingDirectory = aws.String(dir)
		}
		containerDef.Command = def.Command
		containerDef.Image = def.Image
		containerDef.Name = def.Name
		containerDef.Environment = exportEnvVars(def.EnvVars)
		containerDef.Secrets = exportSecrets(def.EnvVars)
		containerDef.LogConfiguration = exportLogConfiguration(def.LogConfiguration)
		containerDef.RepositoryCredentials = exportRepoCreds(def.RepoCreds)
		containerDef.PortMappings = exportPortMappings(def.PortMappings)
		containerDefs = append(containerDefs, containerDef)
	}

	return containerDefs
}

// exportLogConfiguration exports the log configuration into ECS log configuration.
func exportLogConfiguration(logConfiguration *cocoa.LogConfiguration) *types.LogConfiguration {
	if logConfiguration == nil {
		return nil
	}
	var converted types.LogConfiguration
	converted.LogDriver = types.LogDriver(utility.FromStringPtr(logConfiguration.LogDriver))
	options := map[string]string{}
	for k, v := range logConfiguration.Options {
		options[k] = v
	}
	converted.Options = options
	return &converted
}

// exportRepoCreds exports the repository credentials into ECS repository
// credentials.
func exportRepoCreds(creds *cocoa.RepositoryCredentials) *types.RepositoryCredentials {
	if creds == nil {
		return nil
	}
	var converted types.RepositoryCredentials
	converted.CredentialsParameter = creds.ID
	return &converted
}

// exportTaskExecutionOptions converts execution options and a task definition
// into an ECS task execution input.
func (pc *BasicPodCreator) exportTaskExecutionOptions(opts cocoa.ECSPodExecutionOptions, taskDef cocoa.ECSTaskDefinition) *ecs.RunTaskInput {
	var runTask ecs.RunTaskInput
	runTask.Cluster = opts.Cluster
	runTask.CapacityProviderStrategy = pc.exportCapacityProvider(opts.CapacityProvider)
	runTask.TaskDefinition = taskDef.ID
	runTask.Tags = ExportTags(opts.Tags)
	runTask.EnableExecuteCommand = utility.FromBoolPtr(opts.SupportsDebugMode)
	runTask.Overrides = pc.exportOverrides(opts.OverrideOpts)
	runTask.PlacementStrategy = pc.exportStrategy(opts.PlacementOpts)
	runTask.PlacementConstraints = pc.exportPlacementConstraints(opts.PlacementOpts)
	runTask.NetworkConfiguration = pc.exportAWSVPCOptions(opts.AWSVPCOpts)

	if opts.PlacementOpts != nil && opts.PlacementOpts.Group != nil {
		runTask.Group = opts.PlacementOpts.Group
	}
	return &runTask
}

// exportCapacityProvider converts the capacity provider name into an ECS
// capacity provider strategy.
func (pc *BasicPodCreator) exportCapacityProvider(provider *string) []types.CapacityProviderStrategyItem {
	if provider == nil {
		return nil
	}
	var converted types.CapacityProviderStrategyItem
	converted.CapacityProvider = provider
	return []types.CapacityProviderStrategyItem{converted}
}

// exportPortMappings converts port mappings into ECS port mappings.
func exportPortMappings(mappings []cocoa.PortMapping) []types.PortMapping {
	var converted []types.PortMapping
	for _, pm := range mappings {
		mapping := types.PortMapping{}
		if pm.ContainerPort != nil {
			mapping.ContainerPort = aws.Int32(int32(utility.FromIntPtr(pm.ContainerPort)))
		}
		if pm.HostPort != nil {
			mapping.HostPort = aws.Int32(int32(utility.FromIntPtr(pm.HostPort)))
		}
		converted = append(converted, mapping)
	}
	return converted
}

// exportAWSVPCOptions converts AWSVPC options into ECS AWSVPC options.
func (pc *BasicPodCreator) exportAWSVPCOptions(opts *cocoa.AWSVPCOptions) *types.NetworkConfiguration {
	if opts == nil {
		return nil
	}

	var converted types.AwsVpcConfiguration
	if len(opts.Subnets) != 0 {
		converted.Subnets = opts.Subnets
	}
	if len(opts.SecurityGroups) != 0 {
		converted.SecurityGroups = opts.SecurityGroups
	}

	var networkConf types.NetworkConfiguration
	networkConf.AwsvpcConfiguration = &converted

	return &networkConf
}
