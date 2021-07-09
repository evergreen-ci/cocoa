package ecs

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/awsutil"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSPodCreatorInterface(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodCreator)(nil), &BasicECSPodCreator{})
}

func TestECSPodCreator(t *testing.T) {
	testutil.CheckAWSEnvVarsForECSAndSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range map[string]func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault){
		"NewPodCreatorFailsWithMissingClient": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			podCreator, err := NewBasicECSPodCreator(nil, nil)
			require.Error(t, err)
			require.Zero(t, podCreator)
		},
		"CreatePodFailsWithInvalidCreationOpts": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			opts := cocoa.NewECSPodCreationOptions()
			require.NotZero(t, opts)
			assert.Zero(t, *opts)

			podCreator, err := NewBasicECSPodCreator(c, nil)
			require.NoError(t, err)

			p, err := podCreator.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithSecretsNoVault": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			envVar := cocoa.NewEnvironmentVariable().SetName("evName").
				SetSecretOptions(*cocoa.NewSecretOptions().SetName("name").SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar})
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().SetName("name").
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				AddTags("tag").
				SetExecutionOptions(*execOpts)
			require.NotZero(t, opts)
			assert.NotZero(t, *opts)
			assert.NoError(t, opts.Validate())

			podCreator, err := NewBasicECSPodCreator(c, nil)
			require.NoError(t, err)

			p, err := podCreator.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithIncompleteContainerDef": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			envVar := cocoa.NewEnvironmentVariable().SetName("ev").
				SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName("name")).SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetName("container")
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().SetName("name").
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				AddTags("tag").
				SetExecutionOptions(*execOpts)
			require.NotZero(t, opts)
			assert.NotZero(t, *opts)
			assert.NoError(t, opts.Validate())

			podCreator, err := NewBasicECSPodCreator(c, v)
			require.NoError(t, err)

			p, err := podCreator.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithInvalidSecretName": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			envVar := cocoa.NewEnvironmentVariable().SetName("ev").
				SetSecretOptions(*cocoa.NewSecretOptions().SetName("name").SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetName("container").
				SetMemoryMB(128).
				SetCPU(128)
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().SetName("name").
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				AddTags("tag").
				SetExecutionOptions(*execOpts)
			require.NotZero(t, opts)
			assert.NotZero(t, *opts)
			assert.NoError(t, opts.Validate())

			podCreator, err := NewBasicECSPodCreator(c, v)
			require.NoError(t, err)

			p, err := podCreator.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodFailsWithSecretsButNoExecutionRole": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			envVar := cocoa.NewEnvironmentVariable().SetName("ev").
				SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName("name")).SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetName("container").
				SetMemoryMB(128).
				SetCPU(128)
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().
				SetCluster(testutil.ECSClusterName())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.TaskRole()).
				AddTags("tag").
				SetExecutionOptions(*execOpts)
			require.NotZero(t, opts)
			assert.NotZero(t, *opts)
			assert.NoError(t, opts.Validate())

			podCreator, err := NewBasicECSPodCreator(c, v)
			require.NoError(t, err)

			p, err := podCreator.CreatePod(ctx, opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"CreatePodSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			envVar := cocoa.NewEnvironmentVariable().SetName("ev").
				SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName("name")).SetValue("value"))
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetName("container").
				SetMemoryMB(128).
				SetCPU(128)
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().
				SetCluster(testutil.ECSClusterName()).
				SetExecutionRole(testutil.ExecutionRole())
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.TaskRole()).
				AddTags("tag").
				SetExecutionOptions(*execOpts)
			require.NotZero(t, opts)
			assert.NotZero(t, *opts)
			assert.NoError(t, opts.Validate())

			podCreator, err := NewBasicECSPodCreator(c, v)
			require.NoError(t, err)
			require.NotNil(t, podCreator)

			p, err := podCreator.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			info, err := p.Info(ctx)
			require.NoError(t, err)
			assert.Equal(t, cocoa.Running, info.Status)
		},
		"CreatePodSucceedsWithEVNoVault": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault) {
			envVar := cocoa.NewEnvironmentVariable().SetName("name").SetValue("value")
			containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
				SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
				SetName("container").
				SetMemoryMB(128).
				SetCPU(128)
			require.NotNil(t, containerDef.EnvVars)

			execOpts := cocoa.NewECSPodExecutionOptions().
				SetCluster(testutil.ECSClusterName()).
				AddTags(map[string]string{"k0": "v0", "k1": "v1"})
			assert.NoError(t, execOpts.Validate())

			opts := cocoa.NewECSPodCreationOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole(testutil.TaskRole()).
				SetExecutionOptions(*execOpts)
			require.NotZero(t, opts)
			assert.NotZero(t, *opts)
			assert.NoError(t, opts.Validate())

			podCreator, err := NewBasicECSPodCreator(c, v)
			require.NoError(t, err)
			require.NotNil(t, podCreator)

			p, err := podCreator.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotNil(t, p)

			defer func() {
				require.NoError(t, p.Delete(ctx))
			}()

			info, err := p.Info(ctx)
			require.NoError(t, err)
			assert.Equal(t, cocoa.Running, info.Status)
		},
	} {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			awsOpts := awsutil.NewClientOptions().
				SetHTTPClient(hc).
				SetCredentials(credentials.NewEnvCredentials()).
				SetRole(testutil.AWSRole()).
				SetRegion(testutil.AWSRegion())

			c, err := NewBasicECSClient(*awsOpts)
			require.NoError(t, err)
			defer c.Close(ctx)

			secretsClient, err := secret.NewBasicSecretsManagerClient(awsutil.ClientOptions{
				Creds:  credentials.NewEnvCredentials(),
				Region: aws.String(testutil.AWSRegion()),
				Role:   aws.String(testutil.AWSRole()),
				RetryOpts: &utility.RetryOptions{
					MaxAttempts: 5,
				},
				HTTPClient: hc,
			})
			require.NoError(t, err)
			require.NotNil(t, c)
			defer secretsClient.Close(ctx)

			m := secret.NewBasicSecretsManager(secretsClient)
			require.NotNil(t, m)

			tCase(tctx, t, c, m)
		})
	}
}
