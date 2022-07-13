package testcase

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ECSPodDefinitionManagerTestCase represents a test case for a cocoa.ECSPodDefinitionManager.
type ECSPodDefinitionManagerTestCase func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager)

// ECSPodDefinitionManagerTests returns common test cases that a
// cocoa.ECSPodDefinitionManager should support.
func ECSPodDefinitionManagerTests() map[string]ECSPodDefinitionManagerTestCase {
	return map[string]ECSPodDefinitionManagerTestCase{
		"CreatePodDefinitionSucceedsWithoutSecretSettings": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
			envVar := cocoa.NewEnvironmentVariable().SetName("name").SetValue("value")
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetWorkingDir("working_dir").
				AddEnvironmentVariables(*envVar).
				SetMemoryMB(128).
				SetCPU(128).
				AddPortMappings(*cocoa.NewPortMapping().SetContainerPort(1337)).
				SetName("container")

			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(cocoa.NetworkModeBridge)
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			require.NoError(t, err)
			require.NotZero(t, pdi)

			// TODO (EVG-16899): this should defer clean up the pod definition.

			assert.NotZero(t, pdi.ID)
			assert.NotZero(t, pdi.DefinitionOpts)
		},
		"CreatePodDefinitionFailsWithInvalidPodDefinitionOptions": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
			opts := cocoa.NewECSPodDefinitionOptions()
			assert.Error(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			assert.Error(t, err)
			assert.Zero(t, pdi)
		},
		"CreatePodDefinitionFailsWithNewSecretsButNoVault": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
			envVar := cocoa.NewEnvironmentVariable().
				SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName(t)).
					SetNewValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				AddEnvironmentVariables(*envVar).
				SetName("container")

			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())
			assert.NoError(t, opts.Validate())

			p, err := pdm.CreatePodDefinition(ctx, *opts)
			assert.Error(t, err)
			assert.Zero(t, p)
		},
		"CreatePodDefinitionFailsWithNewRepoCredsButNoVault": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
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

			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			assert.Error(t, err)
			assert.Zero(t, pdi)
		},
	}
}

// ECSPodDefinitionManagerVaultTests returns common test cases that a
// cocoa.ECSPodDefinitionManager should support with a cocoa.Vault.
func ECSPodDefinitionManagerVaultTests() map[string]ECSPodDefinitionManagerTestCase {
	return map[string]ECSPodDefinitionManagerTestCase{
		"CreatePodDefinitionSucceedsWithNewlyCreatedSecrets": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
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

			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			require.NoError(t, err)
			require.NotZero(t, pdi)

			// TODO (EVG-16899): this should defer clean up the pod definition.

			assert.NotZero(t, pdi.ID)
			assert.NotZero(t, pdi.DefinitionOpts)
		},
		"CreatePodDefinitionSucceedsWithNewlyCreatedRepoCreds": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
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

			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole()).
				SetExecutionRole(testutil.ECSExecutionRole())
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			require.NoError(t, err)
			require.NotZero(t, pdi)

			// TODO (EVG-16899): this should defer clean up the pod definition.

			assert.NotZero(t, pdi.ID)
			assert.NotZero(t, pdi.DefinitionOpts)
		},
		"CreatePodDefinitionFailsWithNewSecretsButNoExecutionRole": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager) {
			envVar := cocoa.NewEnvironmentVariable().SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName(t)).
					SetNewValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				AddEnvironmentVariables(*envVar).
				SetMemoryMB(128).
				SetCPU(128).
				SetName("container")

			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.ECSTaskRole())
			assert.Error(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			assert.Error(t, err)
			assert.Zero(t, pdi)
		},
	}
}
