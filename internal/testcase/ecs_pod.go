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

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStarting, stat.Status)

			res := p.Resources()
			require.NoError(t, err)
			assert.NotZero(t, res.TaskID)
			assert.NotZero(t, res.TaskDefinition)
			assert.Equal(t, opts.ExecutionOpts.Cluster, res.Cluster)

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStarting, stat.Status)

			require.Len(t, res.Secrets, len(opts.ContainerDefinitions[0].EnvVars))
			for _, s := range res.Secrets {
				val, err := v.GetValue(ctx, utility.FromStringPtr(s.Name))
				require.NoError(t, err)
				assert.Equal(t, utility.FromStringPtr(s.Value), val)
				assert.True(t, utility.FromBoolPtr(s.Owned))
			}

			require.True(t, utility.FromBoolPtr(res.TaskDefinition.Owned))
			def, err := c.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: res.TaskDefinition.ID,
			})
			require.NoError(t, err)
			require.NotZero(t, def.TaskDefinition)
			assert.Equal(t, utility.FromStringPtr(opts.Name), utility.FromStringPtr(def.TaskDefinition.Family))

			task, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: res.Cluster,
				Tasks:   []*string{res.TaskID},
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

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStopped, stat.Status)
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

			res := p.Resources()
			assert.Len(t, res.Secrets, 2)

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStarting, stat.Status)

			require.NoError(t, p.Stop(ctx))

			res = p.Resources()
			require.Len(t, res.Secrets, 2)
			val, err := v.GetValue(ctx, utility.FromStringPtr(res.Secrets[0].Name))
			require.NoError(t, err)
			assert.Equal(t, utility.FromStringPtr(secret.SecretOpts.Value), val)

			stat = p.Status()
			assert.Equal(t, cocoa.StatusStopped, stat.Status)
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

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStopped, stat.Status)

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

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStarting, stat.Status)

			require.NoError(t, p.Delete(ctx))

			stat = p.Status()
			assert.Equal(t, cocoa.StatusDeleted, stat.Status)
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

			stat := p.Status()
			assert.Equal(t, cocoa.StatusStarting, stat.Status)

			res := p.Resources()
			require.Len(t, res.Secrets, 2)
			val, err := v.GetValue(ctx, utility.FromStringPtr(res.Secrets[0].Name))
			require.NoError(t, err)
			assert.Equal(t, utility.FromStringPtr(secret.SecretOpts.Value), val)

			val, err = v.GetValue(ctx, utility.FromStringPtr(res.Secrets[1].Name))
			require.NoError(t, err)
			assert.Equal(t, utility.FromStringPtr(ownedSecret.SecretOpts.Value), val)

			require.NoError(t, p.Delete(ctx))

			stat = p.Status()
			assert.Equal(t, cocoa.StatusDeleted, stat.Status)

			res = p.Resources()
			require.Len(t, res.Secrets, 2)

			val, err = v.GetValue(ctx, utility.FromStringPtr(res.Secrets[0].Name))
			require.NoError(t, err)
			assert.Equal(t, utility.FromStringPtr(secret.SecretOpts.Value), val)

			_, err = v.GetValue(ctx, utility.FromStringPtr(res.Secrets[1].Name))
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
	res := p.Resources()

	_, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: res.TaskDefinition.ID,
	})
	assert.NoError(t, err)

	_, err = c.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: res.Cluster,
		Task:    res.TaskID,
	})
	assert.NoError(t, err)

	for _, s := range res.Secrets {
		assert.NoError(t, v.DeleteSecret(ctx, utility.FromStringPtr(s.Name)))
	}
}
