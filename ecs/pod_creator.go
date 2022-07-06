package ecs

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/grip"
	"github.com/pkg/errors"
)

// BasicECSPodCreator provides an cocoa.ECSPodCreator implementation to create
// AWS ECS pods.
type BasicECSPodCreator struct {
	client cocoa.ECSClient
	vault  cocoa.Vault
}

// NewBasicECSPodCreator creates a helper to create pods backed by AWS ECS.
func NewBasicECSPodCreator(c cocoa.ECSClient, v cocoa.Vault) (*BasicECSPodCreator, error) {
	if c == nil {
		return nil, errors.New("missing client")
	}
	return &BasicECSPodCreator{
		client: c,
		vault:  v,
	}, nil
}

// CreatePod creates a new pod backed by AWS ECS.
func (pc *BasicECSPodCreator) CreatePod(ctx context.Context, opts ...cocoa.ECSPodCreationOptions) (cocoa.ECSPod, error) {
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

	p, err := pc.createPod(utility.FromStringPtr(mergedPodExecutionOpts.Cluster), *task, *taskDef, mergedPodCreationOpts.DefinitionOpts.ContainerDefinitions)
	if err != nil {
		return nil, errors.Wrap(err, "creating pod after requesting task")
	}

	return p, nil
}

// CreatePodFromExistingDefinition creates a new pod backed by AWS ECS from an
// existing definition.
func (pc *BasicECSPodCreator) CreatePodFromExistingDefinition(ctx context.Context, def cocoa.ECSTaskDefinition, opts ...cocoa.ECSPodExecutionOptions) (cocoa.ECSPod, error) {
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

	p, err := pc.createPod(utility.FromStringPtr(mergedPodExecutionOpts.Cluster), *task, *taskDef, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating pod after requesting task")
	}

	return p, nil
}

// createPod creates the basic ECS pod after its ECS task has been requested.
func (pc *BasicECSPodCreator) createPod(cluster string, task ecs.Task, def cocoa.ECSTaskDefinition, containerDefs []cocoa.ECSContainerDefinition) (*BasicECSPod, error) {
	resources := cocoa.NewECSPodResources().
		SetCluster(cluster).
		SetContainers(pc.translateContainerResources(task.Containers, containerDefs)).
		SetTaskDefinition(def).
		SetTaskID(utility.FromStringPtr(task.TaskArn))

	podOpts := NewBasicECSPodOptions().
		SetClient(pc.client).
		SetVault(pc.vault).
		SetStatusInfo(translatePodStatusInfo(task)).
		SetResources(*resources)

	p, err := NewBasicECSPod(podOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating basic pod")
	}

	return p, nil
}

// registerTaskDefinition makes the request to register an ECS task definition
// from the options and checks that it returns a valid task definition.
func registerTaskDefinition(ctx context.Context, c cocoa.ECSClient, opts cocoa.ECSPodDefinitionOptions) (*ecs.TaskDefinition, error) {
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
func (pc *BasicECSPodCreator) runTask(ctx context.Context, opts cocoa.ECSPodExecutionOptions, def cocoa.ECSTaskDefinition) (*ecs.Task, error) {
	in := pc.exportTaskExecutionOptions(opts, def)
	out, err := pc.client.RunTask(ctx, in)
	if err != nil {
		return nil, errors.Wrapf(err, "running task for definition '%s' in cluster '%s'", utility.FromStringPtr(in.TaskDefinition), utility.FromStringPtr(in.Cluster))
	}

	if err := pc.validateRunTaskOutput(out); err != nil {
		return nil, errors.Wrap(err, "validating response from running task")
	}

	return out.Tasks[0], nil
}

// validateRunTaskOutput checks that the output from running a task contains no
// errors and includes the necessary information for the expected tasks.
func (pc *BasicECSPodCreator) validateRunTaskOutput(out *ecs.RunTaskOutput) error {
	if len(out.Failures) > 0 {
		catcher := grip.NewBasicCatcher()
		for _, f := range out.Failures {
			if f == nil {
				continue
			}
			catcher.Add(ConvertFailureToError(*f))
		}
		return errors.Wrap(catcher.Resolve(), "running task")
	}

	if len(out.Tasks) == 0 {
		return errors.New("expected a task to be running in ECS, but none was returned")
	}
	if out.Tasks[0] == nil {
		return errors.New("received a task, but it was nil")
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

// exportTags converts a mapping of tag names to values into ECS tags.
func exportTags(tags map[string]string) []*ecs.Tag {
	var ecsTags []*ecs.Tag

	for k, v := range tags {
		var tag ecs.Tag
		tag.SetKey(k).SetValue(v)
		ecsTags = append(ecsTags, &tag)
	}

	return ecsTags
}

// exportStrategy converts the strategy and parameter into an ECS placement
// strategy.
func (pc *BasicECSPodCreator) exportStrategy(opts *cocoa.ECSPodPlacementOptions) []*ecs.PlacementStrategy {
	var placementStrat ecs.PlacementStrategy
	placementStrat.SetType(string(*opts.Strategy)).SetField(utility.FromStringPtr(opts.StrategyParameter))
	return []*ecs.PlacementStrategy{&placementStrat}
}

// exportPlacementConstraints converts the placement options into ECS placement
// constraints.
func (pc *BasicECSPodCreator) exportPlacementConstraints(opts *cocoa.ECSPodPlacementOptions) []*ecs.PlacementConstraint {
	var constraints []*ecs.PlacementConstraint

	for _, filter := range opts.InstanceFilters {
		var constraint ecs.PlacementConstraint
		if filter == cocoa.ConstraintDistinctInstance {
			constraint.SetType(filter)
		} else {
			constraint.SetType("memberOf").SetExpression(filter)
		}
		constraints = append(constraints, &constraint)
	}

	return constraints
}

// exportEnvVars converts the non-secret environment variables into ECS
// environment variables.
func exportEnvVars(envVars []cocoa.EnvironmentVariable) []*ecs.KeyValuePair {
	var converted []*ecs.KeyValuePair

	for _, envVar := range envVars {
		if envVar.SecretOpts != nil {
			continue
		}
		var pair ecs.KeyValuePair
		pair.SetName(utility.FromStringPtr(envVar.Name)).SetValue(utility.FromStringPtr(envVar.Value))
		converted = append(converted, &pair)
	}

	return converted
}

// exportSecrets converts environment variables backed by secrets into ECS
// Secrets.
func exportSecrets(envVars []cocoa.EnvironmentVariable) []*ecs.Secret {
	var secrets []*ecs.Secret

	for _, envVar := range envVars {
		if envVar.SecretOpts == nil {
			continue
		}

		var secret ecs.Secret
		secret.SetName(utility.FromStringPtr(envVar.Name))
		secret.SetValueFrom(utility.FromStringPtr(envVar.SecretOpts.ID))
		secrets = append(secrets, &secret)
	}

	return secrets
}

// translateContainerResources translates the containers and stored secrets
// into the resources associated with each container.
func (pc *BasicECSPodCreator) translateContainerResources(containers []*ecs.Container, defs []cocoa.ECSContainerDefinition) []cocoa.ECSContainerResources {
	var resources []cocoa.ECSContainerResources

	for _, container := range containers {
		if container == nil {
			continue
		}

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
func (pc *BasicECSPodCreator) translateContainerSecrets(defs []cocoa.ECSContainerDefinition) []cocoa.ContainerSecret {
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
func translatePodStatusInfo(task ecs.Task) cocoa.ECSPodStatusInfo {
	lastStatus := TaskStatus(utility.FromStringPtr(task.LastStatus)).ToCocoaStatus()
	return *cocoa.NewECSPodStatusInfo().
		SetStatus(lastStatus).
		SetContainers(translateContainerStatusInfo(task.Containers))
}

// translateContainerStatusInfo translates an ECS container to its equivalent
// cocoa container status information.
func translateContainerStatusInfo(containers []*ecs.Container) []cocoa.ECSContainerStatusInfo {
	var statuses []cocoa.ECSContainerStatusInfo

	for _, container := range containers {
		if container == nil {
			continue
		}
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

	var containerDefs []*ecs.ContainerDefinition
	for _, def := range opts.ContainerDefinitions {
		containerDefs = append(containerDefs, exportContainerDefinition(def))
	}
	taskDef.SetContainerDefinitions(containerDefs)

	if mem := utility.FromIntPtr(opts.MemoryMB); mem != 0 {
		taskDef.SetMemory(strconv.Itoa(mem))
	}

	if cpu := utility.FromIntPtr(opts.CPU); cpu != 0 {
		taskDef.SetCpu(strconv.Itoa(cpu))
	}

	if opts.NetworkMode != nil {
		taskDef.SetNetworkMode(string(*opts.NetworkMode))
	}

	taskDef.SetFamily(utility.FromStringPtr(opts.Name)).
		SetTaskRoleArn(utility.FromStringPtr(opts.TaskRole)).
		SetExecutionRoleArn(utility.FromStringPtr(opts.ExecutionRole)).
		SetTags(exportTags(opts.Tags))

	return &taskDef
}

// exportContainerDefinition converts a container definition into an ECS
// container definition input.
func exportContainerDefinition(def cocoa.ECSContainerDefinition) *ecs.ContainerDefinition {
	var containerDef ecs.ContainerDefinition
	if mem := utility.FromIntPtr(def.MemoryMB); mem != 0 {
		containerDef.SetMemory(int64(mem))
	}
	if cpu := utility.FromIntPtr(def.CPU); cpu != 0 {
		containerDef.SetCpu(int64(cpu))
	}
	if dir := utility.FromStringPtr(def.WorkingDir); dir != "" {
		containerDef.SetWorkingDirectory(dir)
	}
	containerDef.SetCommand(utility.ToStringPtrSlice(def.Command)).
		SetImage(utility.FromStringPtr(def.Image)).
		SetName(utility.FromStringPtr(def.Name)).
		SetEnvironment(exportEnvVars(def.EnvVars)).
		SetSecrets(exportSecrets(def.EnvVars)).
		SetRepositoryCredentials(exportRepoCreds(def.RepoCreds)).
		SetPortMappings(exportPortMappings(def.PortMappings))
	return &containerDef
}

// exportRepoCreds exports the repository credentials into ECS repository
// credentials.
func exportRepoCreds(creds *cocoa.RepositoryCredentials) *ecs.RepositoryCredentials {
	if creds == nil {
		return nil
	}
	var converted ecs.RepositoryCredentials
	converted.SetCredentialsParameter(utility.FromStringPtr(creds.ID))
	return &converted
}

// exportTaskExecutionOptions converts execution options and a task definition
// into an ECS task execution input.
func (pc *BasicECSPodCreator) exportTaskExecutionOptions(opts cocoa.ECSPodExecutionOptions, taskDef cocoa.ECSTaskDefinition) *ecs.RunTaskInput {
	var runTask ecs.RunTaskInput
	runTask.SetCluster(utility.FromStringPtr(opts.Cluster)).
		SetCapacityProviderStrategy(pc.exportCapacityProvider(opts.CapacityProvider)).
		SetTaskDefinition(utility.FromStringPtr(taskDef.ID)).
		SetTags(exportTags(opts.Tags)).
		SetEnableExecuteCommand(utility.FromBoolPtr(opts.SupportsDebugMode)).
		SetPlacementStrategy(pc.exportStrategy(opts.PlacementOpts)).
		SetPlacementConstraints(pc.exportPlacementConstraints(opts.PlacementOpts)).
		SetNetworkConfiguration(pc.exportAWSVPCOptions(opts.AWSVPCOpts))
	if opts.PlacementOpts != nil && opts.PlacementOpts.Group != nil {
		runTask.SetGroup(utility.FromStringPtr(opts.PlacementOpts.Group))
	}
	return &runTask
}

// exportCapacityProvider converts the capacity provider name into an ECS
// capacity provider strategy.
func (pc *BasicECSPodCreator) exportCapacityProvider(provider *string) []*ecs.CapacityProviderStrategyItem {
	if provider == nil {
		return nil
	}
	var converted ecs.CapacityProviderStrategyItem
	converted.SetCapacityProvider(utility.FromStringPtr(provider))
	return []*ecs.CapacityProviderStrategyItem{&converted}
}

// exportPortMappings converts port mappings into ECS port mappings.
func exportPortMappings(mappings []cocoa.PortMapping) []*ecs.PortMapping {
	var converted []*ecs.PortMapping
	for _, pm := range mappings {
		mapping := &ecs.PortMapping{}
		if pm.ContainerPort != nil {
			mapping.SetContainerPort(int64(utility.FromIntPtr(pm.ContainerPort)))
		}
		if pm.HostPort != nil {
			mapping.SetHostPort(int64(utility.FromIntPtr(pm.HostPort)))
		}
		converted = append(converted, mapping)
	}
	return converted
}

// exportAWSVPCOptions converts AWSVPC options into ECS AWSVPC options.
func (pc *BasicECSPodCreator) exportAWSVPCOptions(opts *cocoa.AWSVPCOptions) *ecs.NetworkConfiguration {
	if opts == nil {
		return nil
	}

	var converted ecs.AwsVpcConfiguration
	if len(opts.Subnets) != 0 {
		converted.SetSubnets(utility.ToStringPtrSlice(opts.Subnets))
	}
	if len(opts.SecurityGroups) != 0 {
		converted.SetSecurityGroups(utility.ToStringPtrSlice(opts.SecurityGroups))
	}

	var networkConf ecs.NetworkConfiguration
	networkConf.SetAwsvpcConfiguration(&converted)

	return &networkConf
}
