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
		mergedPodExecutionOpts = cocoa.MergeECSPodExecutionOptions(*mergedPodCreationOpts.ExecutionOpts)
	}

	if err := mergedPodCreationOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod creation options")
	}

	if err := mergedPodExecutionOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod execution options")
	}

	secrets := pc.getSecrets(mergedPodCreationOpts)

	repoCreds, err := pc.getRepoCreds(mergedPodCreationOpts)
	if err != nil {
		return nil, errors.Wrap(err, "getting secret repository credentials")
	}

	mergedSecrets := pc.mergeSecrets(secrets, repoCreds)

	resolvedSecrets, err := pc.createSecrets(ctx, mergedSecrets)
	if err != nil {
		return nil, errors.Wrap(err, "creating secret environment variables")
	}

	updatedPodCreationOpts := pc.updateSecrets(mergedPodCreationOpts, resolvedSecrets)

	taskDefinition := pc.exportPodCreationOptions(updatedPodCreationOpts)

	registerOut, err := pc.client.RegisterTaskDefinition(ctx, taskDefinition)
	if err != nil {
		return nil, errors.Wrap(err, "registering task definition")
	}

	if registerOut.TaskDefinition == nil || registerOut.TaskDefinition.TaskDefinitionArn == nil {
		return nil, errors.New("expected a task definition from ECS, but none was returned")
	}

	taskDef := cocoa.NewECSTaskDefinition().
		SetID(utility.FromStringPtr(registerOut.TaskDefinition.TaskDefinitionArn)).
		SetOwned(true)

	runTask := pc.exportTaskExecutionOptions(mergedPodExecutionOpts, *taskDef)

	runOut, err := pc.client.RunTask(ctx, runTask)
	if err != nil {
		return nil, errors.Wrapf(err, "running task for definition '%s' in cluster '%s'", utility.FromStringPtr(runTask.TaskDefinition), utility.FromStringPtr(runTask.Cluster))
	}

	if len(runOut.Failures) > 0 {
		catcher := grip.NewBasicCatcher()
		for _, failure := range runOut.Failures {
			catcher.Errorf("task '%s': %s: %s\n", utility.FromStringPtr(failure.Arn), utility.FromStringPtr(failure.Detail), utility.FromStringPtr(failure.Reason))
		}
		return nil, errors.Wrap(catcher.Resolve(), "running task")
	}

	if len(runOut.Tasks) == 0 || runOut.Tasks[0].TaskArn == nil {
		return nil, errors.New("expected a task to be running in ECS, but none was returned")
	}

	resources := cocoa.NewECSPodResources().
		SetCluster(utility.FromStringPtr(mergedPodExecutionOpts.Cluster)).
		SetContainers(pc.translateContainerResources(runOut.Tasks[0].Containers, resolvedSecrets)).
		SetTaskDefinition(*taskDef).
		SetTaskID(utility.FromStringPtr(runOut.Tasks[0].TaskArn))

	podOpts := NewBasicECSPodOptions().
		SetClient(pc.client).
		SetVault(pc.vault).
		SetStatusInfo(pc.translatePodStatusInfo(runOut.Tasks[0])).
		SetResources(*resources)

	p, err := NewBasicECSPod(podOpts)
	if err != nil {
		return nil, errors.Wrap(err, "creating pod")
	}

	return p, nil
}

// CreatePodFromExistingDefinition creates a new pod backed by AWS ECS from an
// existing definition.
func (pc *BasicECSPodCreator) CreatePodFromExistingDefinition(ctx context.Context, def cocoa.ECSTaskDefinition, opts ...cocoa.ECSPodExecutionOptions) (cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}

// containerSecrets is an intermediate struct containing all relevant secrets
// associated with a container.
type containerSecrets struct {
	envVars   map[string]secretOptions
	repoCreds *secretOptions
}

// initEnvVars initializes the environment variables for the container secrets
// if it is not already initialized.
func (cs *containerSecrets) initEnvVars() {
	if cs.envVars != nil {
		return
	}
	cs.envVars = map[string]secretOptions{}
}

// secretOptions is an intermediate representation of a secret associated with a
// container that may or may not have been created yet.
type secretOptions struct {
	cocoa.SecretOptions
	id string
}

// getSecrets retrieves the secrets from the secret environment variables for
// each container. This returns a map of container names to the secret
// environment variables associated with each container.
func (pc *BasicECSPodCreator) getSecrets(opts cocoa.ECSPodCreationOptions) map[string]containerSecrets {
	secrets := map[string]containerSecrets{}

	for _, def := range opts.ContainerDefinitions {
		containerName := utility.FromStringPtr(def.Name)

		for _, envVar := range def.EnvVars {
			if envVar.SecretOpts == nil {
				continue
			}

			container := secrets[containerName]
			container.initEnvVars()
			container.envVars[utility.FromStringPtr(envVar.Name)] = secretOptions{
				SecretOptions: *envVar.SecretOpts,
			}
			secrets[containerName] = container
		}
	}

	return secrets
}

// getRepoCreds retrieves the secret repository credentials for each container.
// This returns a map of container names to the repository credentials
// associated with each container.
func (pc *BasicECSPodCreator) getRepoCreds(opts cocoa.ECSPodCreationOptions) (map[string]containerSecrets, error) {
	creds := map[string]containerSecrets{}

	for _, def := range opts.ContainerDefinitions {
		if def.RepoCreds == nil {
			continue
		}

		opts := cocoa.NewSecretOptions().
			SetName(utility.FromStringPtr(def.RepoCreds.SecretName)).
			SetOwned(utility.FromBoolPtr(def.RepoCreds.Owned))

		if def.RepoCreds.NewCreds != nil {
			val, err := json.Marshal(def.RepoCreds.NewCreds)
			if err != nil {
				return nil, errors.Wrap(err, "formatting new credentials to create")
			}
			opts.SetValue(string(val)).SetExists(false)
		} else {
			opts.SetExists(true)
		}

		containerName := utility.FromStringPtr(def.Name)
		container := creds[containerName]
		container.repoCreds = &secretOptions{SecretOptions: *opts}
		creds[containerName] = container
	}

	return creds, nil
}

// createSecrets creates secrets that do not already exist for each container
// and returns the map of container names to all secrets associated with each
// container.
func (pc *BasicECSPodCreator) createSecrets(ctx context.Context, secrets map[string]containerSecrets) (map[string]containerSecrets, error) {
	resolved := map[string]containerSecrets{}

	for containerName, secret := range secrets {
		for envVarName, opts := range secret.envVars {
			resolvedOpts, err := pc.createSecret(ctx, opts)
			if err != nil {
				return nil, errors.Wrapf(err, "creating secret environment variable '%s' for container '%s'", utility.FromStringPtr(opts.Name), containerName)
			}

			updated := secrets[containerName]
			updated.envVars[envVarName] = *resolvedOpts
			resolved[containerName] = updated
		}

		if secret.repoCreds != nil {
			resolvedOpts, err := pc.createSecret(ctx, *secret.repoCreds)
			if err != nil {
				return nil, errors.Wrapf(err, "creating repository credentials for container '%s'", containerName)
			}

			updated := secrets[containerName]
			updated.repoCreds = resolvedOpts
			resolved[containerName] = updated
		}
	}

	return resolved, nil
}

// createSecret creates a single secret if it does not exist yet. It returns the
// resolved container secret since the secret identifier is known after the
// secret is created.
func (pc *BasicECSPodCreator) createSecret(ctx context.Context, secret secretOptions) (*secretOptions, error) {
	if utility.FromBoolPtr(secret.Exists) {
		secret.id = utility.FromStringPtr(secret.Name)
		return &secret, nil
	}
	if pc.vault == nil {
		return nil, errors.New("no vault was specified")
	}
	arn, err := pc.vault.CreateSecret(ctx, secret.NamedSecret)
	if err != nil {
		return nil, err
	}

	secret.id = arn

	return &secret, nil
}

// mergeSecrets merges the secrets for each container with the secret repository
// credentials for each container.
func (pc *BasicECSPodCreator) mergeSecrets(toMerge ...map[string]containerSecrets) map[string]containerSecrets {
	merged := map[string]containerSecrets{}

	for _, secrets := range toMerge {
		for containerName, container := range secrets {
			mergedContainer := merged[containerName]

			mergedContainer.initEnvVars()
			for envVarName, secretOpts := range container.envVars {
				mergedContainer.envVars[envVarName] = secretOpts
			}

			if container.repoCreds != nil {
				mergedContainer.repoCreds = container.repoCreds
			}

			merged[containerName] = mergedContainer
		}
	}

	return merged
}

// updateSecrets updates the container definitions with the resolved container
// secrets.
func (pc *BasicECSPodCreator) updateSecrets(opts cocoa.ECSPodCreationOptions, resolved map[string]containerSecrets) cocoa.ECSPodCreationOptions {
	for i, c := range opts.ContainerDefinitions {
		containerName := utility.FromStringPtr(c.Name)
		secrets, ok := resolved[containerName]
		if !ok {
			continue
		}

		for j, envVar := range c.EnvVars {
			if envVar.SecretOpts == nil {
				continue
			}

			envVarName := utility.FromStringPtr(envVar.Name)
			secretOpts, ok := secrets.envVars[envVarName]
			if !ok {
				continue
			}

			envVar.SecretOpts = &secretOpts.SecretOptions
			// Resolve the secret name (which could be its friendly name if the
			// secret was just created) to its ID.
			envVar.SecretOpts.Name = &secretOpts.id
			opts.ContainerDefinitions[i].EnvVars[j] = envVar
		}

		if secrets.repoCreds != nil {
			// Resolve the secret name (which could be its friendly name if the
			// repo credentials were just created) to its ID.
			c.RepoCreds.SecretName = &secrets.repoCreds.id
		}
	}
	return opts
}

// exportTags converts a mapping of tag names to values into ECS tags.
func (pc *BasicECSPodCreator) exportTags(tags map[string]string) []*ecs.Tag {
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
		constraint.SetType("memberOf").SetExpression(filter)
		constraints = append(constraints, &constraint)
	}

	return constraints
}

// exportEnvVars converts the non-secret environment variables into ECS
// environment variables.
func (pc *BasicECSPodCreator) exportEnvVars(envVars []cocoa.EnvironmentVariable) []*ecs.KeyValuePair {
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
func (pc *BasicECSPodCreator) exportSecrets(envVars []cocoa.EnvironmentVariable) []*ecs.Secret {
	var secrets []*ecs.Secret

	for _, envVar := range envVars {
		if envVar.SecretOpts == nil {
			continue
		}

		var secret ecs.Secret
		secret.SetName(utility.FromStringPtr(envVar.Name))
		secret.SetValueFrom(utility.FromStringPtr(envVar.SecretOpts.Name))
		secrets = append(secrets, &secret)
	}

	return secrets
}

// translateContainerResources translates the containers and  stored secrets
// into the resources associated with each container.
func (pc *BasicECSPodCreator) translateContainerResources(containers []*ecs.Container, resolvedSecrets map[string]containerSecrets) []cocoa.ECSContainerResources {
	var resources []cocoa.ECSContainerResources

	for _, container := range containers {
		if container == nil {
			continue
		}

		name := utility.FromStringPtr(container.Name)
		res := cocoa.NewECSContainerResources().
			SetContainerID(utility.FromStringPtr(container.ContainerArn)).
			SetName(name).
			SetSecrets(pc.translateContainerSecrets(resolvedSecrets[name]))
		resources = append(resources, *res)
	}

	return resources
}

// translateContainerSecrets translates the given secrets for a container into
// a slice of container secrets.
func (pc *BasicECSPodCreator) translateContainerSecrets(cs containerSecrets) []cocoa.ContainerSecret {
	var translated []cocoa.ContainerSecret

	for _, secretOpts := range cs.envVars {
		translated = append(translated, pc.translateSecretOptions(secretOpts))
	}

	if cs.repoCreds != nil {
		translated = append(translated, pc.translateSecretOptions(*cs.repoCreds))
	}

	return translated
}

// translateSecretOptions translates the given secret options into the resolved
// container secret.
func (pc *BasicECSPodCreator) translateSecretOptions(opts secretOptions) cocoa.ContainerSecret {
	translated := *cocoa.NewContainerSecret().
		SetID(opts.id).
		SetName(utility.FromStringPtr(opts.Name)).
		SetOwned(utility.FromBoolPtr(opts.Owned))
	return translated
}

func (pc *BasicECSPodCreator) translatePodStatusInfo(task *ecs.Task) cocoa.ECSPodStatusInfo {
	return *cocoa.NewECSPodStatusInfo().
		SetStatus(pc.translateECSStatus(task.LastStatus)).
		SetContainers(pc.translateContainerStatusInfo(task.Containers))
}

func (pc *BasicECSPodCreator) translateContainerStatusInfo(containers []*ecs.Container) []cocoa.ECSContainerStatusInfo {
	var statuses []cocoa.ECSContainerStatusInfo

	for _, container := range containers {
		if container == nil {
			continue
		}
		status := cocoa.NewECSContainerStatusInfo().
			SetContainerID(utility.FromStringPtr(container.ContainerArn)).
			SetName(utility.FromStringPtr(container.Name)).
			SetStatus(pc.translateECSStatus(container.LastStatus))
		statuses = append(statuses, *status)
	}

	return statuses
}

// translateECSStatus translate the ECS status into its equivalent cocoa
// status.
func (pc *BasicECSPodCreator) translateECSStatus(status *string) cocoa.ECSStatus {
	if status == nil {
		return cocoa.StatusUnknown
	}
	switch *status {
	case "PROVISIONING", "PENDING", "ACTIVATING":
		return cocoa.StatusStarting
	case "RUNNING":
		return cocoa.StatusRunning
	case "DEACTIVATING", "STOPPING", "DEPROVISIONING":
		return cocoa.StatusStopped
	default:
		return cocoa.StatusUnknown
	}
}

// exportPodCreationOptions converts options to create a pod into its equivalent
// ECS task definition.
func (pc *BasicECSPodCreator) exportPodCreationOptions(opts cocoa.ECSPodCreationOptions) *ecs.RegisterTaskDefinitionInput {
	var taskDef ecs.RegisterTaskDefinitionInput

	var containerDefs []*ecs.ContainerDefinition
	for _, def := range opts.ContainerDefinitions {
		containerDefs = append(containerDefs, pc.exportContainerDefinition(def))
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
		SetTags(pc.exportTags(opts.Tags))

	return &taskDef
}

// exportContainerDefinition converts a container definition into an ECS
// container definition input.
func (pc *BasicECSPodCreator) exportContainerDefinition(def cocoa.ECSContainerDefinition) *ecs.ContainerDefinition {
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
		SetEnvironment(pc.exportEnvVars(def.EnvVars)).
		SetSecrets(pc.exportSecrets(def.EnvVars)).
		SetRepositoryCredentials(pc.exportRepoCreds(def.RepoCreds)).
		SetPortMappings(pc.exportPortMappings(def.PortMappings))
	return &containerDef
}

// exportRepoCreds exports the repository credentials into ECS repository
// credentials.
func (pc *BasicECSPodCreator) exportRepoCreds(creds *cocoa.RepositoryCredentials) *ecs.RepositoryCredentials {
	if creds == nil {
		return nil
	}
	var converted ecs.RepositoryCredentials
	converted.SetCredentialsParameter(utility.FromStringPtr(creds.SecretName))
	return &converted
}

// exportTaskExecutionOptions converts execution options and a task definition
// into an ECS task execution input.
func (pc *BasicECSPodCreator) exportTaskExecutionOptions(opts cocoa.ECSPodExecutionOptions, taskDef cocoa.ECSTaskDefinition) *ecs.RunTaskInput {
	var runTask ecs.RunTaskInput
	runTask.SetCluster(utility.FromStringPtr(opts.Cluster)).
		SetTaskDefinition(utility.FromStringPtr(taskDef.ID)).
		SetTags(pc.exportTags(opts.Tags)).
		SetEnableExecuteCommand(utility.FromBoolPtr(opts.SupportsDebugMode)).
		SetPlacementStrategy(pc.exportStrategy(opts.PlacementOpts)).
		SetPlacementConstraints(pc.exportPlacementConstraints(opts.PlacementOpts)).
		SetNetworkConfiguration(pc.exportAWSVPCOptions(opts.AWSVPCOpts))
	return &runTask
}

// exportPortMappings converts port mappings into ECS port mappings.
func (pc *BasicECSPodCreator) exportPortMappings(mappings []cocoa.PortMapping) []*ecs.PortMapping {
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
