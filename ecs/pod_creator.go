package ecs

import (
	"context"
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
func (m *BasicECSPodCreator) CreatePod(ctx context.Context, opts ...*cocoa.ECSPodCreationOptions) (cocoa.ECSPod, error) {

	mergedPodCreationOpts := cocoa.MergeECSPodCreationOptions(opts...)
	mergedPodExecutionOpts := cocoa.MergeECSPodExecutionOptions(mergedPodCreationOpts.ExecutionOpts)

	err := mergedPodCreationOpts.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid pod creation options")
	}

	err = mergedPodExecutionOpts.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid pod execution options")
	}

	allEnvVars, allSecrets, allSecretsToCreate, err := m.exportSecrets(ctx, mergedPodCreationOpts)
	if err != nil {
		return nil, errors.Wrap(err, "exporting secrets")
	}

	err = m.createSecrets(ctx, allSecretsToCreate)
	if err != nil {
		return nil, errors.Wrap(err, "creating secrets")
	}

	taskDefinition, err := m.exportPodCreationOptions(ctx, mergedPodCreationOpts, allEnvVars, allSecrets)
	if err != nil {
		return nil, errors.Wrap(err, "translating the register task definition input to the correct type")
	}

	registerOut, err := m.client.RegisterTaskDefinition(ctx, taskDefinition)
	if err != nil {
		return nil, errors.Wrap(err, "registering task definition")
	}

	if registerOut.TaskDefinition == nil || registerOut.TaskDefinition.TaskDefinitionArn == nil {
		return nil, errors.New("missing task definition")
	}

	taskDef := cocoa.NewECSTaskDefinition()
	taskDef = taskDef.SetID(*registerOut.TaskDefinition.TaskDefinitionArn)
	taskDef = taskDef.SetOwned(*utility.TruePtr())

	runTask := &ecs.RunTaskInput{}
	runTask.SetCluster(*mergedPodExecutionOpts.Cluster).
		SetTaskDefinition(*taskDef.ID).
		SetTags(registerOut.Tags).
		SetEnableExecuteCommand(*mergedPodExecutionOpts.SupportsDebugMode).
		SetPlacementStrategy(translateStrategy(mergedPodExecutionOpts.PlacementOpts.Strategy, mergedPodExecutionOpts.PlacementOpts.StrategyParameter))

	runOut, err := m.client.RunTask(ctx, runTask)
	if err != nil {
		return nil, errors.Wrap(err, "running task")
	}

	if runOut.Failures != nil {
		catcher := grip.NewBasicCatcher()
		for _, failure := range runOut.Failures {
			err = errors.Errorf("task '%s': %s: %s\n", *failure.Arn, *failure.Detail, *failure.Reason)
			catcher.Add(err)
		}
		return nil, errors.Wrap(catcher.Resolve(), "running task")
	}

	if len(runOut.Tasks) == 0 || runOut.Tasks[0].TaskArn == nil {
		return nil, errors.New("missing running task")
	}

	resources := cocoa.NewECSPodResources().
		SetCluster(*mergedPodExecutionOpts.Cluster).
		SetSecrets(allSecrets).
		SetTaskDefinition(*taskDef).
		SetTaskID(*runOut.Tasks[0].TaskArn)

	options := NewBasicECSPodOptions().
		SetClient(m.client).
		SetVault(m.vault).
		SetStatus(cocoa.Running).
		SetResources(*resources)

	p, err := NewBasicECSPod(options)

	if err != nil {
		return nil, errors.Wrap(err, "creating pod")
	}

	return p, nil
}

// CreatePodFromExistingDefinition creates a new pod backed by AWS ECS from an
// existing definition.
func (m *BasicECSPodCreator) CreatePodFromExistingDefinition(ctx context.Context, def cocoa.ECSTaskDefinition, opts ...*cocoa.ECSPodExecutionOptions) (cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}

// translateStringArrayToECSTagArray translates the strings into ECS tags.
func translateStringArrayToECSTagArray(tags []string) []*ecs.Tag {
	ecsTags := []*ecs.Tag{}
	for _, tag := range tags {
		ecsTag := &ecs.Tag{}
		ecsTag.SetKey(tag)
		ecsTags = append(ecsTags, ecsTag)
	}
	return ecsTags
}

// translateStrategy translates the strategy and parameter into ECS placement strategy.
func translateStrategy(strategy *cocoa.ECSPlacementStrategy, param *cocoa.ECSStrategyParameter) []*ecs.PlacementStrategy {
	placementStrat := ecs.PlacementStrategy{}
	placementStrat.SetType(string(*strategy)).SetField(*param)
	return []*ecs.PlacementStrategy{&placementStrat}
}

// exportEnvVars translates the environment variables into ECS environment variables and secrets.
func (m *BasicECSPodCreator) exportEnvVars(ctx context.Context, variables []cocoa.EnvironmentVariable) ([]*ecs.KeyValuePair, []cocoa.PodSecret, []cocoa.NamedSecret, error) {
	keyValuePairs := []*ecs.KeyValuePair{}
	secrets := []cocoa.PodSecret{}
	createSecrets := []cocoa.NamedSecret{}

	for _, variable := range variables {
		if variable.SecretOpts == nil {
			keyValue := ecs.KeyValuePair{}
			keyValue.SetName(*variable.Name).SetValue(*variable.Value)
			keyValuePairs = append(keyValuePairs, &keyValue)
		} else {
			if !*variable.SecretOpts.Exists {
				createSecrets = append(createSecrets, variable.SecretOpts.PodSecret.NamedSecret)
			}
			secrets = append(secrets, variable.SecretOpts.PodSecret)
		}
	}

	return keyValuePairs, secrets, createSecrets, nil
}

// exportSecrets extracts all secrets and environment variables from ECS creation options
func (m *BasicECSPodCreator) exportSecrets(ctx context.Context, merged cocoa.ECSPodCreationOptions) ([]*ecs.KeyValuePair, []cocoa.PodSecret, []cocoa.NamedSecret, error) {
	allEnvVars := []*ecs.KeyValuePair{}
	allSecrets := []cocoa.PodSecret{}
	allSecretsToCreate := []cocoa.NamedSecret{}

	for _, def := range merged.ContainerDefinitions {
		envVars, secrets, secretsToCreate, err := m.exportEnvVars(ctx, def.EnvVars)
		if err != nil {
			return nil, nil, nil, err
		}
		allEnvVars = append(allEnvVars, envVars...)
		allSecrets = append(allSecrets, secrets...)
		allSecretsToCreate = append(allSecretsToCreate, secretsToCreate...)
	}

	return allEnvVars, allSecrets, allSecretsToCreate, nil
}

// createSecrets creates secrets that do not already exist
func (m *BasicECSPodCreator) createSecrets(ctx context.Context, secrets []cocoa.NamedSecret) error {
	for _, secret := range secrets {
		_, err := m.vault.CreateSecret(ctx, secret)
		if err != nil {
			return err
		}
	}

	return nil
}

// translateSecrets translates a PodSecret to an ECS Secret
func translateSecrets(ctx context.Context, secrets []cocoa.PodSecret) []*ecs.Secret {
	ecsSecrets := []*ecs.Secret{}

	for _, secret := range secrets {
		ecsSecret := ecs.Secret{}
		ecsSecret.SetName(*secret.Name)
		ecsSecret.SetValueFrom(*secret.Value)
		ecsSecrets = append(ecsSecrets, &ecsSecret)
	}

	return ecsSecrets
}

// exportPodCreationOptions converts options to create a pod into its equivalent ECS task definition.
func (m *BasicECSPodCreator) exportPodCreationOptions(ctx context.Context, merged cocoa.ECSPodCreationOptions, envVars []*ecs.KeyValuePair, secrets []cocoa.PodSecret) (*ecs.RegisterTaskDefinitionInput, error) {
	var containerDefs []*ecs.ContainerDefinition

	for _, def := range merged.ContainerDefinitions {
		if def.CPU == nil {
			return nil, errors.New("missing CPU in container definition")
		}
		cpu64 := int64(*def.CPU)

		if def.MemoryMB == nil {
			return nil, errors.New("missing MemoryMB in container definition")
		}
		mem64 := int64(*def.MemoryMB)

		containerDef := ecs.ContainerDefinition{}
		containerDef.SetCommand(utility.ToStringPtrSlice(def.Command)).
			SetCpu(cpu64).
			SetImage(*def.Image).
			SetName(*def.Name).
			SetMemory(mem64).
			SetEnvironment(envVars).
			SetSecrets(translateSecrets(ctx, secrets))

		containerDefs = append(containerDefs, &containerDef)
	}

	if merged.MemoryMB == nil {
		return nil, errors.New("missing MemoryMB definition")
	}

	if merged.CPU == nil {
		return nil, errors.New("missing CPU definition")
	}

	taskDef := &ecs.RegisterTaskDefinitionInput{}
	taskDef.SetContainerDefinitions(containerDefs).
		SetMemory(strconv.Itoa(*merged.MemoryMB)).
		SetCpu(strconv.Itoa(*merged.CPU)).
		SetTaskRoleArn(*merged.TaskRole).
		SetTags(translateStringArrayToECSTagArray(merged.ExecutionOpts.Tags)).
		SetFamily(*merged.Name)

	return taskDef, nil
}
