package testcase

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ECSPodTestCase represents a test case for a cocoa.ECSPod.
type ECSPodTestCase func(ctx context.Context, t *testing.T, v cocoa.Vault, pc cocoa.ECSPodCreator)

// ECSPodTests returns common test cases that a cocoa.ECSPod should support.
func ECSPodTests() map[string]ECSPodTestCase {
	envVar := cocoa.NewEnvironmentVariable().SetName("name").SetValue("value")
	secret := cocoa.NewEnvironmentVariable().SetName("secret1").
		SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName("name1")).SetValue("value1"))
	secretOwned := cocoa.NewEnvironmentVariable().SetName("secret2").
		SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName("name2")).SetValue("value2"))
	secretOwned.SecretOpts.SetOwned(true)

	containerDef := cocoa.NewECSContainerDefinition().SetImage("image").
		SetEnvironmentVariables([]cocoa.EnvironmentVariable{*envVar}).
		SetName("container").
		SetMemoryMB(128).
		SetCPU(128)

	execOpts := cocoa.NewECSPodExecutionOptions().
		SetCluster(testutil.ECSClusterName()).
		SetExecutionRole(testutil.ExecutionRole())

	opts := cocoa.NewECSPodCreationOptions().
		AddContainerDefinitions(*containerDef).
		SetMemoryMB(128).
		SetCPU(128).
		SetTaskRole(testutil.TaskRole()).
		AddTags("tag").
		SetExecutionOptions(*execOpts)

	optsSecret := cocoa.NewECSPodCreationOptions().
		AddContainerDefinitions(*containerDef.AddEnvironmentVariables(*secret, *secretOwned)).
		SetMemoryMB(128).
		SetCPU(128).
		SetTaskRole(testutil.TaskRole()).
		AddTags("tag").
		SetExecutionOptions(*execOpts)

	return map[string]ECSPodTestCase{
		"StopSucceeds": func(ctx context.Context, t *testing.T, v cocoa.Vault, pc cocoa.ECSPodCreator) {
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotZero(t, p)

			err = p.Stop(ctx)
			require.NoError(t, err)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)

			assert.Equal(t, cocoa.Stopped, info.Status)
		},
		"StopSucceedsWithSecrets": func(ctx context.Context, t *testing.T, v cocoa.Vault, pc cocoa.ECSPodCreator) {
			p, err := pc.CreatePod(ctx, optsSecret)
			require.NoError(t, err)
			require.NotZero(t, p)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			require.NotNil(t, info.Resources)
			assert.Equal(t, cocoa.Running, info.Status)
			assert.Equal(t, 2, len(info.Resources.Secrets))

			err = p.Stop(ctx)
			require.NoError(t, err)

			info, err = p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.Stopped, info.Status)
			require.Equal(t, 2, len(info.Resources.Secrets))

			arn := info.Resources.Secrets[0].Name
			id, err := v.GetValue(ctx, *arn)
			require.NoError(t, err)
			require.NotNil(t, id)
		},
		"StopFailsOnIncorrectPodStatus": func(ctx context.Context, t *testing.T, v cocoa.Vault, pc cocoa.ECSPodCreator) {
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotZero(t, p)

			err = p.Stop(ctx)
			require.NoError(t, err)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotZero(t, info)
			assert.Equal(t, cocoa.Stopped, info.Status)

			err = p.Stop(ctx)
			require.Error(t, err)
		},
		"DeleteSucceeds": func(ctx context.Context, t *testing.T, v cocoa.Vault, pc cocoa.ECSPodCreator) {
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)
			require.NotZero(t, p)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.Running, info.Status)

			err = p.Delete(ctx)
			require.NoError(t, err)

			info, err = p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.Deleted, info.Status)
		},
		"DeleteSucceedsWithSecrets": func(ctx context.Context, t *testing.T, v cocoa.Vault, pc cocoa.ECSPodCreator) {
			p, err := pc.CreatePod(ctx, optsSecret)
			require.NoError(t, err)
			require.NotZero(t, p)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			require.NotNil(t, info.Resources)
			require.NotNil(t, info.Resources.Secrets)
			assert.Equal(t, cocoa.Running, info.Status)
			require.Equal(t, 2, len(info.Resources.Secrets))

			err = p.Delete(ctx)
			require.NoError(t, err)

			info, err = p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.Deleted, info.Status)
			require.Equal(t, 2, len(info.Resources.Secrets))

			arn0 := info.Resources.Secrets[0].Name
			id, err := v.GetValue(ctx, *arn0)
			require.NoError(t, err)
			require.NotZero(t, id)

			arn1 := info.Resources.Secrets[1].Name
			_, err = v.GetValue(ctx, *arn1)
			require.Error(t, err)

		},
	}
}
