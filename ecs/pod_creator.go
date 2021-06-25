package ecs

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/utility"
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
	var out cocoa.ECSPod
	var err error

	mergedPodOptions := cocoa.MergeECSPodCreationOptions(opts...)

	translatedIn, secrets, err := m.TranslateRegisterTaskDefinitionInput(ctx, mergedPodOptions)
	if err != nil {
		return nil, errors.Wrap(err, "translating the register task definition input to the correct type")
	}

	registerOut, err := m.client.RegisterTaskDefinition(ctx, translatedIn)
	if err != nil {
		return nil, errors.Wrap(err, "registering task definition")
	}

	taskDef := cocoa.ECSTaskDefinition{
		ID:    registerOut.TaskDefinition.TaskDefinitionArn,
		Owned: utility.TruePtr(),
	}

	runOut, err := m.client.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        cocoa.MergeECSPodExecutionOptions(mergedPodOptions.ExecutionOpts).Cluster,
		TaskDefinition: taskDef.ID,
	})
	if err != nil || runOut.Failures != nil || runOut.Tasks == nil {
		return nil, errors.Wrap(err, "running task")
	}

	out = &BasicECSPod{
		client: m.client,
		vault:  m.vault,
		resources: cocoa.ECSPodResources{
			TaskID:         runOut.Tasks[0].TaskArn,
			TaskDefinition: &taskDef,
			Secrets:        secrets,
		},
		status: cocoa.Starting,
	}
	if err != nil {
		return nil, errors.Wrap(err, "creating pod")
	}

	return out, nil
}

// CreatePodFromExistingDefinition creates a new pod backed by AWS ECS from an
// existing definition.
func (m *BasicECSPodCreator) CreatePodFromExistingDefinition(ctx context.Context, def cocoa.ECSTaskDefinition, opts ...*cocoa.ECSPodExecutionOptions) (cocoa.ECSPod, error) {
	return nil, errors.New("TODO: implement")
}

// EnvVarArrayToKeyValuePairsAndSecrets translates the Environment Variables Array to an ECS KeyValuePair array and PodSecret array
func (m *BasicECSPodCreator) EnvVarArrayToKeyValuePairsAndSecrets(ctx context.Context, variables []cocoa.EnvironmentVariable) ([]*ecs.KeyValuePair, []cocoa.PodSecret, error) {
	variablesPtr := []*ecs.KeyValuePair{}
	secrets := []cocoa.PodSecret{}

	for _, variable := range variables {
		if variable.SecretOpts == nil {
			variablesPtr = append(variablesPtr, &ecs.KeyValuePair{
				Name:  variable.Name,
				Value: variable.Value,
			})
		} else {
			if !*variable.SecretOpts.Exists {
				_, err := m.vault.CreateSecret(ctx, variable.SecretOpts.PodSecret.NamedSecret)
				if err != nil {
					return nil, nil, errors.Wrap(err, "creating secret in conversion from env var array to key value pair and secrets")
				}
			}
			secrets = append(secrets, variable.SecretOpts.PodSecret)
		}
	}

	return variablesPtr, secrets, nil
}

// TranslateRegisterTaskDefinitionInput translates the custom types to respective ECS formats for the RegisterTaskDefinitionInput
func (m *BasicECSPodCreator) TranslateRegisterTaskDefinitionInput(ctx context.Context, merged cocoa.ECSPodCreationOptions) (*ecs.RegisterTaskDefinitionInput, []cocoa.PodSecret, error) {
	StringArrayToPtr := func(command []string) []*string {
		ptrArray := []*string{}
		for _, str := range command {
			ptrArray = append(ptrArray, &str)
		}
		return ptrArray
	}

	TranslateStringArrayToECSTagArray := func(tags []string) []*ecs.Tag {
		ecsTags := []*ecs.Tag{}
		for _, tag := range tags {
			ecsTags = append(ecsTags, &ecs.Tag{
				Key: &tag,
			})
		}
		return ecsTags
	}

	PtrIntToPtrStr := func(in *int) *string {
		return utility.ToStringPtr(strconv.Itoa(*in))
	}

	registerContainerDefinitions := []*ecs.ContainerDefinition{}
	var secrets []cocoa.PodSecret

	for _, def := range merged.ContainerDefinitions {
		cpu64 := int64(*def.CPU)
		mem64 := int64(*def.MemoryMB)
		envVars, secretsArray, err := m.EnvVarArrayToKeyValuePairsAndSecrets(ctx, def.EnvVars)
		if err != nil {
			return nil, nil, errors.Wrap(err, "converting env var array to key value pair and secrets")
		}
		containerDef := &ecs.ContainerDefinition{
			Command:     StringArrayToPtr(def.Command),
			Cpu:         &cpu64,
			Image:       def.Image,
			Name:        def.Name,
			Memory:      &mem64,
			Environment: envVars,
		}
		registerContainerDefinitions = append(registerContainerDefinitions, containerDef)
		secrets = append(secrets, secretsArray...)
	}

	return &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: registerContainerDefinitions,
		Memory:               PtrIntToPtrStr(merged.MemoryMB),
		Cpu:                  PtrIntToPtrStr(merged.CPU),
		TaskRoleArn:          merged.TaskRole,
		Tags:                 TranslateStringArrayToECSTagArray(merged.ExecutionOpts.Tags),
		Family:               merged.Name,
	}, secrets, nil
}
