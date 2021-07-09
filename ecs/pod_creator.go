package ecs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
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

	if err := mergedPodCreationOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod creation options")
	}

	if err := mergedPodExecutionOpts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid pod execution options")
	}

	secrets := m.getSecrets(mergedPodCreationOpts)

	if err := m.createSecrets(ctx, secrets); err != nil {
		return nil, errors.Wrap(err, "creating secrets")
	}

	taskDefinition := m.exportPodCreationOptions(ctx, mergedPodCreationOpts)

	registerOut, err := m.client.RegisterTaskDefinition(ctx, taskDefinition)
	if err != nil {
		return nil, errors.Wrap(err, "registering task definition")
	}

	if registerOut.TaskDefinition == nil || registerOut.TaskDefinition.TaskDefinitionArn == nil {
		return nil, errors.New("expected a task definition from ECS, but none was returned")
	}

	taskDef := cocoa.NewECSTaskDefinition().SetID(utility.FromStringPtr(registerOut.TaskDefinition.TaskDefinitionArn)).SetOwned(true)

	runTask := m.exportTaskExecution(mergedPodExecutionOpts, *taskDef)

	runOut, err := m.client.RunTask(ctx, runTask)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("running task-- Cluster: %s, Task: %s", utility.FromStringPtr(runTask.Cluster), utility.FromStringPtr(runTask.TaskDefinition)))
	}

	if len(runOut.Failures) > 0 {
		catcher := grip.NewBasicCatcher()
		for _, failure := range runOut.Failures {
			catcher.Errorf("task '%s': %s: %s\n", *failure.Arn, *failure.Detail, *failure.Reason)
		}
		return nil, errors.Wrap(catcher.Resolve(), "running task")
	}

	if len(runOut.Tasks) == 0 || runOut.Tasks[0].TaskArn == nil {
		return nil, errors.New("expected a task to be running in ECS, but none was returned")
	}

	resources := cocoa.NewECSPodResources().
		SetCluster(utility.FromStringPtr(mergedPodExecutionOpts.Cluster)).
		SetSecrets(translatePodSecrets(secrets)).
		SetTaskDefinition(*taskDef).
		SetTaskID(utility.FromStringPtr(runOut.Tasks[0].TaskArn))

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

// getSecrets retrieves the secrets from the secret environment variables for the pod.
func (m *BasicECSPodCreator) getSecrets(merged cocoa.ECSPodCreationOptions) []cocoa.SecretOptions {
	var secrets []cocoa.SecretOptions

	for _, def := range merged.ContainerDefinitions {
		for _, variable := range def.EnvVars {
			if variable.SecretOpts != nil {
				secrets = append(secrets, *variable.SecretOpts)
			}
		}
	}

	return secrets
}

// createSecrets creates secrets that do not already exist.
func (m *BasicECSPodCreator) createSecrets(ctx context.Context, secrets []cocoa.SecretOptions) error {
	for _, secret := range secrets {
		if !utility.FromBoolPtr(secret.Exists) {
			if m.vault == nil {
				return errors.New("no vault was specified")
			}
			arn, err := m.vault.CreateSecret(ctx, secret.PodSecret.NamedSecret)
			if err != nil {
				return err
			}
			secret.SetName(arn)
			secret.PodSecret.SetName(arn)
		}
	}

	return nil
}

// exportTags converts a mapping of string-string into ECS tags.
func exportTags(tags map[string]string) []*ecs.Tag {
	var ecsTags []*ecs.Tag

	for k, v := range tags {
		ecsTag := &ecs.Tag{}
		ecsTag.SetKey(k)
		ecsTag.SetValue(v)
		ecsTags = append(ecsTags, ecsTag)
	}

	return ecsTags
}

// exportStrategy converts the strategy and parameter into an ECS placement strategy.
func exportStrategy(strategy *cocoa.ECSPlacementStrategy, param *cocoa.ECSStrategyParameter) []*ecs.PlacementStrategy {
	placementStrat := ecs.PlacementStrategy{}
	placementStrat.SetType(string(*strategy)).SetField(utility.FromStringPtr(param))

	return []*ecs.PlacementStrategy{&placementStrat}
}

// exportEnvVars converts the non-secret environment variables into ECS environment variables.
func exportEnvVars(variables []cocoa.EnvironmentVariable) []*ecs.KeyValuePair {
	var keyValuePairs []*ecs.KeyValuePair

	for _, variable := range variables {
		if variable.SecretOpts == nil {
			keyValue := ecs.KeyValuePair{}
			keyValue.SetName(utility.FromStringPtr(variable.Name)).SetValue(utility.FromStringPtr(variable.Value))
			keyValuePairs = append(keyValuePairs, &keyValue)
		}
	}

	return keyValuePairs
}

// exportSecrets converts environment variables backed by secrets into ECS Secrets.
func exportSecrets(envVars []cocoa.EnvironmentVariable) []*ecs.Secret {
	var ecsSecrets []*ecs.Secret

	for _, envVar := range envVars {
		if envVar.SecretOpts != nil {
			ecsSecret := ecs.Secret{}
			ecsSecret.SetName(utility.FromStringPtr(envVar.Name))
			ecsSecret.SetValueFrom(utility.FromStringPtr(envVar.SecretOpts.Name))
			ecsSecrets = append(ecsSecrets, &ecsSecret)
		}
	}

	return ecsSecrets
}

// translatePodSecrets translates secret options into pod secrets.
func translatePodSecrets(secrets []cocoa.SecretOptions) []cocoa.PodSecret {
	var podSecrets []cocoa.PodSecret

	for _, secret := range secrets {
		podSecrets = append(podSecrets, secret.PodSecret)
	}

	return podSecrets
}

// exportPodCreationOptions converts options to create a pod into its equivalent ECS task definition.
func (m *BasicECSPodCreator) exportPodCreationOptions(ctx context.Context, merged cocoa.ECSPodCreationOptions) *ecs.RegisterTaskDefinitionInput {
	var containerDefs []*ecs.ContainerDefinition

	for _, def := range merged.ContainerDefinitions {
		containerDef := ecs.ContainerDefinition{}
		containerDef.SetCommand(utility.ToStringPtrSlice(def.Command)).
			SetCpu(int64(utility.FromIntPtr(def.CPU))).
			SetImage(utility.FromStringPtr(def.Image)).
			SetName(utility.FromStringPtr(def.Name)).
			SetMemory(int64(utility.FromIntPtr(def.MemoryMB))).
			SetEnvironment(exportEnvVars(def.EnvVars)).
			SetSecrets(exportSecrets(def.EnvVars))

		containerDefs = append(containerDefs, &containerDef)
	}

	taskDef := &ecs.RegisterTaskDefinitionInput{}
	taskDef.SetContainerDefinitions(containerDefs).
		SetMemory(strconv.Itoa(utility.FromIntPtr(merged.MemoryMB))).
		SetCpu(strconv.Itoa(utility.FromIntPtr(merged.CPU))).
		SetTaskRoleArn(utility.FromStringPtr(merged.TaskRole)).
		SetTags(exportTags(merged.ExecutionOpts.Tags)).
		SetFamily(strings.Join([]string{testutil.TaskDefinitionPrefix(), "cocoa", utility.FromStringPtr(merged.Name)}, "-")).
		SetExecutionRoleArn(utility.FromStringPtr(merged.ExecutionOpts.ExecutionRole))

	return taskDef
}

// exportTaskExecution converts execution options and a task definition into an ECS task execution input.
func (m *BasicECSPodCreator) exportTaskExecution(merged cocoa.ECSPodExecutionOptions, taskDef cocoa.ECSTaskDefinition) *ecs.RunTaskInput {
	runTask := &ecs.RunTaskInput{}
	runTask.SetCluster(utility.FromStringPtr(merged.Cluster)).
		SetTaskDefinition(utility.FromStringPtr(taskDef.ID)).
		SetTags(exportTags(merged.Tags)).
		SetEnableExecuteCommand(utility.FromBoolPtr(merged.SupportsDebugMode)).
		SetPlacementStrategy(exportStrategy(merged.PlacementOpts.Strategy, merged.PlacementOpts.StrategyParameter))

	return runTask
}
