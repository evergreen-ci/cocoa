package testcase

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ECSPodTestCase represents a test case for a cocoa.ECSPod.
type ECSPodTestCase func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault)

// ECSPodTests returns common test cases that a cocoa.ECSPod should support.
func ECSPodTests() map[string]ECSPodTestCase {
	makeEnvVar := func(t *testing.T) *cocoa.EnvironmentVariable {
		return cocoa.NewEnvironmentVariable().
			SetName(t.Name()).
			SetValue("value")
	}

	makeSecretEnvVar := func(t *testing.T) *cocoa.EnvironmentVariable {
		return cocoa.NewEnvironmentVariable().
			SetName(t.Name()).
			SetSecretOptions(*cocoa.NewSecretOptions().
				SetName(testutil.NewSecretName(t.Name())).
				SetValue(utility.RandomString()).
				SetOwned(true))
	}

	makeContainerDef := func(t *testing.T) *cocoa.ECSContainerDefinition {
		return cocoa.NewECSContainerDefinition().
			SetImage("image").
			SetMemoryMB(128).
			SetCPU(128).
			SetName("container")
	}

	makePodCreationOpts := func(t *testing.T) *cocoa.ECSPodCreationOptions {
		return cocoa.NewECSPodCreationOptions().
			SetName(testutil.NewTaskDefinitionFamily(t.Name())).
			SetMemoryMB(128).
			SetCPU(128).
			SetTaskRole(testutil.TaskRole()).
			SetExecutionRole(testutil.ExecutionRole()).
			SetExecutionOptions(*cocoa.NewECSPodExecutionOptions().
				SetCluster(testutil.ECSClusterName()))
	}

	return map[string]ECSPodTestCase{
		"InfoIsPopulated": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			opts := makePodCreationOpts(t).AddContainerDefinitions(
				*makeContainerDef(t).AddEnvironmentVariables(
					*makeSecretEnvVar(t),
				),
			)
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)

			defer cleanupPod(ctx, t, p, c, v)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			assert.Equal(t, cocoa.StatusRunning, info.Status)
			assert.NotZero(t, info.Resources.TaskID)
			assert.NotZero(t, info.Resources.TaskDefinition)
			assert.Equal(t, opts.ExecutionOpts.Cluster, info.Resources.Cluster)

			require.Len(t, info.Resources.Secrets, len(opts.ContainerDefinitions[0].EnvVars))
			for _, s := range info.Resources.Secrets {
				val, err := v.GetValue(ctx, utility.FromStringPtr(s.Name))
				require.NoError(t, err)
				assert.Equal(t, utility.FromStringPtr(s.Value), val)
				assert.True(t, utility.FromBoolPtr(s.Owned))
			}

			require.True(t, utility.FromBoolPtr(info.Resources.TaskDefinition.Owned))
			def, err := c.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: info.Resources.TaskDefinition.ID,
			})
			require.NoError(t, err)
			require.NotZero(t, def.TaskDefinition)
			assert.Equal(t, utility.FromStringPtr(opts.Name), utility.FromStringPtr(def.TaskDefinition.Family))

			task, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: info.Resources.Cluster,
				Tasks:   []*string{info.Resources.TaskID},
			})
			require.NoError(t, err)
			require.Len(t, task.Tasks, 1)
			require.Len(t, task.Tasks[0].Containers, 1)
			assert.Equal(t, utility.FromStringPtr(opts.ContainerDefinitions[0].Image), utility.FromStringPtr(task.Tasks[0].Containers[0].Image))
		},
		"StopSucceeds": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			opts := makePodCreationOpts(t).AddContainerDefinitions(
				*makeContainerDef(t).AddEnvironmentVariables(
					*makeSecretEnvVar(t),
				),
			)
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)

			defer cleanupPod(ctx, t, p, c, v)

			require.NoError(t, p.Stop(ctx))

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusStopped, info.Status)
		},
		"StopSucceedsWithSecrets": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			secret := cocoa.NewEnvironmentVariable().
				SetName(utility.RandomString()).
				SetSecretOptions(*cocoa.NewSecretOptions().
					SetName(testutil.NewSecretName(t.Name())).
					SetValue(utility.RandomString()))
			ownedSecret := makeSecretEnvVar(t)

			secretOpts := makePodCreationOpts(t).
				AddContainerDefinitions(*makeContainerDef(t).
					AddEnvironmentVariables(*secret, *ownedSecret))

			p, err := pc.CreatePod(ctx, secretOpts)
			require.NoError(t, err)

			defer cleanupPod(ctx, t, p, c, v)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusRunning, info.Status)
			assert.Len(t, info.Resources.Secrets, 2)

			require.NoError(t, p.Stop(ctx))

			info, err = p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusStopped, info.Status)
			require.Len(t, info.Resources.Secrets, 2)

			arn := info.Resources.Secrets[0].Name
			id, err := v.GetValue(ctx, *arn)
			require.NoError(t, err)
			require.NotNil(t, id)
		},
		"StopIsIdempotent": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			opts := makePodCreationOpts(t).AddContainerDefinitions(
				*makeContainerDef(t).AddEnvironmentVariables(
					*makeSecretEnvVar(t),
				),
			)
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)

			defer cleanupPod(ctx, t, p, c, v)

			require.NoError(t, p.Stop(ctx))

			info, err := p.Info(ctx)
			require.NoError(t, err)
			assert.Equal(t, cocoa.StatusStopped, info.Status)

			require.NoError(t, p.Stop(ctx))
		},
		"DeleteSucceeds": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			opts := makePodCreationOpts(t).AddContainerDefinitions(
				*makeContainerDef(t).AddEnvironmentVariables(
					*makeSecretEnvVar(t),
				),
			)
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)

			defer cleanupPod(ctx, t, p, c, v)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusRunning, info.Status)

			require.NoError(t, p.Delete(ctx))

			info, err = p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusDeleted, info.Status)
		},
		"DeleteSucceedsWithSecrets": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			secret := cocoa.NewEnvironmentVariable().SetName(t.Name()).
				SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName(t.Name())).SetValue("value1"))
			ownedSecret := cocoa.NewEnvironmentVariable().SetName("secret2").
				SetSecretOptions(*cocoa.NewSecretOptions().SetName(testutil.NewSecretName(t.Name())).SetValue("value2").SetOwned(true))

			secretOpts := makePodCreationOpts(t).AddContainerDefinitions(*makeContainerDef(t).AddEnvironmentVariables(*secret, *ownedSecret))

			p, err := pc.CreatePod(ctx, secretOpts)
			require.NoError(t, err)

			defer cleanupPod(ctx, t, p, c, v)

			info, err := p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusRunning, info.Status)
			require.Len(t, info.Resources.Secrets, 2)

			require.NoError(t, p.Delete(ctx))

			info, err = p.Info(ctx)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, cocoa.StatusDeleted, info.Status)
			require.Len(t, info.Resources.Secrets, 2)

			arn0 := info.Resources.Secrets[0].Name
			val, err := v.GetValue(ctx, *arn0)
			require.NoError(t, err)
			require.NotZero(t, val)
			assert.Equal(t, *info.Resources.Secrets[0].NamedSecret.Value, *secret.SecretOpts.NamedSecret.Value)

			arn1 := info.Resources.Secrets[1].Name
			_, err = v.GetValue(ctx, *arn1)
			require.Error(t, err)
		},
		"DeleteIsIdempotent": func(ctx context.Context, t *testing.T, pc cocoa.ECSPodCreator, c cocoa.ECSClient, v cocoa.Vault) {
			opts := makePodCreationOpts(t).AddContainerDefinitions(
				*makeContainerDef(t).AddEnvironmentVariables(
					*makeEnvVar(t),
				),
			)
			p, err := pc.CreatePod(ctx, opts)
			require.NoError(t, err)

			require.NoError(t, p.Delete(ctx))
			require.NoError(t, p.Delete(ctx))
		},
	}
}

// cleanupPod cleans up all resources regardless of whether they're owned by the
// pod or not.
func cleanupPod(ctx context.Context, t *testing.T, p cocoa.ECSPod, c cocoa.ECSClient, v cocoa.Vault) {
	info, err := p.Info(ctx)
	require.NoError(t, err)

	_, err = c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: info.Resources.TaskDefinition.ID,
	})
	assert.NoError(t, err)

	_, err = c.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: info.Resources.Cluster,
		Task:    info.Resources.TaskID,
	})
	assert.NoError(t, err)

	for _, s := range info.Resources.Secrets {
		assert.NoError(t, v.DeleteSecret(ctx, utility.FromStringPtr(s.Name)))
	}
}
