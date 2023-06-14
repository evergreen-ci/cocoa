package testcase

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ECSPodCreatorTestCase represents a test case for a cocoa.ECSPodCreator.
type ECSPodCreatorTestCase func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator)

// ECSPodCreatorTests returns common test cases that a cocoa.ECSPodCreator
// should support.
func ECSPodCreatorTests() map[string]ECSPodCreatorTestCase {
	return map[string]ECSPodCreatorTestCase{
		"CreatePodSucceedsWithoutSecretSettings": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().SetName("name").SetValue("value")
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetWorkingDir("working_dir").
				AddEnvironmentVariables(*envVar).
				SetMemoryMB(128).
				SetCPU(128).
				AddPortMappings(*cocoa.NewPortMapping().SetContainerPort(1337)).
				SetName("container")

			defOpts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(cocoa.NetworkModeBridge)

			opts := cocoa.NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*validECSPodExecutionOptions())
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, *opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			ps := p.StatusInfo()
			assert.Equal(t, cocoa.StatusStarting, ps.Status)
		},
		"CreatePodFailsWithInvalidCreationOptions": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			opts := cocoa.NewECSPodCreationOptions()

			p, err := c.CreatePod(ctx, *opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithNewSecretsButNoVault": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().
				SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName(t)).
					SetNewValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				AddEnvironmentVariables(*envVar).
				SetName("container")

			defOpts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())

			opts := cocoa.NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*validECSPodExecutionOptions())
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, *opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithRepoCredsButNoVault": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			storedCreds := cocoa.NewStoredRepositoryCredentials().
				SetUsername("username").
				SetPassword("password")
			creds := cocoa.NewRepositoryCredentials().
				SetName(testutil.NewSecretName(t)).
				SetNewCredentials(*storedCreds)
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetRepositoryCredentials(*creds).
				SetName("container")

			defOpts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())

			opts := cocoa.NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*validECSPodExecutionOptions())
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, *opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFromExistingDefinitionFailsWithInvalidTaskDefinition": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			def := cocoa.NewECSTaskDefinition()
			require.Error(t, def.Validate())
			p, err := c.CreatePodFromExistingDefinition(ctx, *def)
			require.Error(t, err)
			require.Zero(t, p)
		},
	}
}

// ECSPodCreatorVaultTests returns common test cases that a cocoa.ECSPodCreator
// should support with a cocoa.Vault.
func ECSPodCreatorVaultTests() map[string]ECSPodCreatorTestCase {
	return map[string]ECSPodCreatorTestCase{
		"CreatePodSucceedsWithNewlyCreatedSecrets": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().
				SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName(t)).
					SetNewValue("value").
					SetOwned(true))

			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				AddEnvironmentVariables(*envVar).
				SetMemoryMB(128).
				SetCPU(128).
				SetName("container")

			defOpts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())

			opts := cocoa.NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*validECSPodExecutionOptions())
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, *opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			checkPodStatus(t, p, cocoa.StatusStarting)

			res := p.Resources()
			require.Len(t, res.Containers, 1)
			require.Len(t, res.Containers[0].Secrets, 1)
		},
		"CreatePodSucceedsWithNewlyCreatedRepoCreds": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			storedCreds := cocoa.NewStoredRepositoryCredentials().
				SetUsername("username").
				SetPassword("password")
			creds := cocoa.NewRepositoryCredentials().
				SetName(testutil.NewSecretName(t)).
				SetNewCredentials(*storedCreds).
				SetOwned(true)

			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetRepositoryCredentials(*creds).
				SetMemoryMB(128).
				SetCPU(128).
				SetName("container")

			defOpts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())

			opts := cocoa.NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*validECSPodExecutionOptions())
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, *opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			checkPodStatus(t, p, cocoa.StatusStarting)

			res := p.Resources()
			require.Len(t, res.Containers, 1)
			require.Len(t, res.Containers[0].Secrets, 1)
		},
	}
}

// ECSPodCreatorRegisteredTaskDefinitionTestCase represents a test case for a
// cocoa.ECSPodCreator with a task definition already registered.
type ECSPodCreatorRegisteredTaskDefinitionTestCase func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator)

// ECSPodCreatorRegisteredTaskDefinitionTests returns common test cases that a
// cocoa.ECSPodCreator should support that rely on an already-registered task
// definition.
func ECSPodCreatorRegisteredTaskDefinitionTests() map[string]func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator, def ecs.TaskDefinition) {
	return map[string]func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator, def ecs.TaskDefinition){
		"CreatePodFromExistingDefinitionSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator, def ecs.TaskDefinition) {
			taskDef := cocoa.NewECSTaskDefinition().SetID(utility.FromStringPtr(def.TaskDefinitionArn))

			p, err := c.CreatePodFromExistingDefinition(ctx, *taskDef, *validECSPodExecutionOptions())
			require.NoError(t, err)
			require.NotZero(t, p)

			defer func() {
				assert.NoError(t, p.Delete(ctx))
			}()

			require.NotZero(t, p.Resources().TaskDefinition)
			assert.Equal(t, utility.FromStringPtr(p.Resources().TaskDefinition.ID), utility.FromStringPtr(def.TaskDefinitionArn))
			assert.False(t, utility.FromBoolPtr(p.Resources().TaskDefinition.Owned), def.TaskDefinitionArn)
			assert.Equal(t, testutil.ECSClusterName(), utility.FromStringPtr(p.Resources().Cluster))
			assert.Len(t, p.Resources().Containers, len(def.ContainerDefinitions))
			assert.Len(t, p.StatusInfo().Containers, len(def.ContainerDefinitions))
			checkPodStatus(t, p, cocoa.StatusStarting)
		},
		"CreatePodFromExistingDefinitionFailsWithNonexistentCluster": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator, def ecs.TaskDefinition) {
			taskDef := cocoa.NewECSTaskDefinition().SetID(utility.FromStringPtr(def.TaskDefinitionArn))
			opts := validECSPodExecutionOptions().SetCluster("foo")

			p, err := c.CreatePodFromExistingDefinition(ctx, *taskDef, *opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFromExistingDefinitionFailsWithNonexistentTaskDefinition": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator, def ecs.TaskDefinition) {
			taskDef := cocoa.NewECSTaskDefinition().SetID(testutil.NewTaskDefinitionFamily(t) + ":1")

			p, err := c.CreatePodFromExistingDefinition(ctx, *taskDef, *validECSPodExecutionOptions())
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFromExistingDefinitionFailsWithInvalidExecutionOptions": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator, def ecs.TaskDefinition) {
			taskDef := cocoa.NewECSTaskDefinition().SetID(utility.FromStringPtr(def.TaskDefinitionArn))
			require.NoError(t, taskDef.Validate())
			placementOpts := cocoa.NewECSPodPlacementOptions().SetStrategy("foo")
			require.Error(t, placementOpts.Validate())
			opts := validECSPodExecutionOptions().SetPlacementOptions(*placementOpts)

			p, err := c.CreatePodFromExistingDefinition(ctx, *taskDef, *opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
	}
}

func validECSPodExecutionOptions() *cocoa.ECSPodExecutionOptions {
	return cocoa.NewECSPodExecutionOptions().
		SetCluster(testutil.ECSClusterName()).
		SetCapacityProvider(testutil.ECSCapacityProvider())
}
