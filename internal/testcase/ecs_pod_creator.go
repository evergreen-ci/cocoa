package testcase

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ECSPodCreatorTestCase represents a test case for a cocoa.ECSPodCreator.
type ECSPodCreatorTestCase func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator)

// ECSPodCreatorNoVaultTests returns common test cases that a cocoa.ECSPodCreator should support.
func ECSPodCreatorNoVaultTests() map[string]ECSPodCreatorTestCase {
	return map[string]ECSPodCreatorTestCase{
		"CreatePodFailsWithInvalidCreationOpts": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			opts := cocoa.NewECSPodCreationOptions()

			p, err := c.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithSecretsNoVault": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().
				SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName("name")).
					SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetName("container")
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().SetCluster(testutil.ECSClusterName()).SetExecutionRole(testutil.ExecutionRole())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				SetName(testutil.NewTaskDefinitionFamily("name")).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithIncompleteContainerDef": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image")

			execOpts := cocoa.NewECSPodExecutionOptions().SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				SetName(testutil.NewTaskDefinitionFamily("name")).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodSucceedsWithEnvVars": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().SetName("name").SetValue("value")
			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetMemoryMB(128).
				SetCPU(128).
				SetName("container")
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				SetName(testutil.NewTaskDefinitionFamily("name")).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			info, err := p.Info(ctx)
			require.NoError(t, err)
			assert.Equal(t, cocoa.Running, info.Status)
		},
	}
}

// ECSPodCreatorTests returns common test casese that a cocoa.ECSPodCreator should support that rely on an ECSPodCreator with both an ECSClient and Vault.
func ECSPodCreatorTests() map[string]ECSPodCreatorTestCase {
	return map[string]ECSPodCreatorTestCase{
		"CreatePodFailsWithSecretsButNoExecutionRole": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName("name")).
					SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetMemoryMB(128).
				SetCPU(128).
				SetName("container")
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().
				SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.TaskRole()).
				SetExecutionOptions(*execOpts).
				SetName(testutil.NewTaskDefinitionFamily("name"))
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodSucceedsWithNewlyCreatedSecrets": func(ctx context.Context, t *testing.T, c cocoa.ECSPodCreator) {
			envVar := cocoa.NewEnvironmentVariable().
				SetName("envVar").
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName("name")).
					SetValue("value").
					SetExists(false))
			envVar.SecretOpts.SetOwned(true)

			containerDef := cocoa.NewECSContainerDefinition().
				SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetMemoryMB(128).
				SetCPU(128).
				SetName("container")
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().
				SetCluster(testutil.ECSClusterName()).
				SetExecutionRole(testutil.ExecutionRole())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				SetName(testutil.NewTaskDefinitionFamily("name")).
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.TaskRole()).
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())

			p, err := c.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			info, err := p.Info(ctx)
			require.NoError(t, err)
			assert.Equal(t, cocoa.Running, info.Status)
		},
	}
}
