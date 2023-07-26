package ecs

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicPod(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPod)(nil), &BasicPod{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range map[string]func(ctx context.Context, t *testing.T, c cocoa.ECSClient){
		"InvalidPodOptions": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			opts := NewBasicPodOptions()
			p, err := NewBasicPod(opts)
			require.Error(t, err)
			require.Zero(t, p)
		},
		"InfoIsPopulated": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			res := cocoa.NewECSPodResources().SetTaskID("task_id")
			ps := cocoa.NewECSPodStatusInfo().
				SetStatus(cocoa.StatusRunning).
				AddContainers(*cocoa.NewECSContainerStatusInfo().
					SetContainerID("container_id").
					SetName("name").
					SetStatus(cocoa.StatusRunning))
			opts := NewBasicPodOptions().SetClient(c).SetResources(*res).SetStatusInfo(*ps)

			p, err := NewBasicPod(opts)
			require.NoError(t, err)

			podRes := p.Resources()
			assert.Equal(t, *res, podRes)
			assert.Equal(t, *ps, p.StatusInfo())
		},
	} {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			c, err := NewBasicClient(tctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			defer func() {
				assert.NoError(t, c.Close(ctx))
			}()

			tCase(tctx, t, c)
		})
	}
}

func TestECSPod(t *testing.T) {
	testutil.CheckAWSEnvVarsForECSAndSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	awsOpts, err := testutil.ValidIntegrationAWSOptions(ctx, hc)
	require.NoError(t, err)

	c, err := NewBasicClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupTaskDefinitions(ctx, t, c)
		testutil.CleanupTasks(ctx, t, c)

		assert.NoError(t, c.Close(ctx))
	}()

	smc, err := secret.NewBasicSecretsManagerClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupSecrets(ctx, t, smc)

		assert.NoError(t, smc.Close(ctx))
	}()

	for tName, tCase := range testcase.ECSPodTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smc))
			require.NoError(t, err)

			pc, err := NewBasicPodCreator(c, v)
			require.NoError(t, err)

			tCase(tctx, t, pc, c, v)
		})
	}
}

func TestBasicPodOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)
	t.Run("NewBasicPodOptions", func(t *testing.T) {
		opts := NewBasicPodOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetClient", func(t *testing.T) {
		c, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, err)
		}()
		opts := NewBasicPodOptions().SetClient(c)
		assert.Equal(t, c, opts.Client)
	})
	t.Run("SetVault", func(t *testing.T) {
		c, err := secret.NewBasicSecretsManagerClient(ctx, testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(c))
		require.NoError(t, err)
		opts := NewBasicPodOptions().SetVault(v)
		assert.Equal(t, v, opts.Vault)
	})
	t.Run("SetResources", func(t *testing.T) {
		res := cocoa.NewECSPodResources().SetTaskID("id")
		opts := NewBasicPodOptions().SetResources(*res)
		require.NotZero(t, opts.Resources)
		assert.Equal(t, *res, *opts.Resources)
	})
	t.Run("SetStatusInfo", func(t *testing.T) {
		ps := cocoa.NewECSPodStatusInfo().SetStatus(cocoa.StatusRunning)
		opts := NewBasicPodOptions().SetStatusInfo(*ps)
		require.NotNil(t, opts.StatusInfo)
		assert.Equal(t, *ps, *opts.StatusInfo)
	})
	t.Run("Validate", func(t *testing.T) {
		validResources := func() cocoa.ECSPodResources {
			return *cocoa.NewECSPodResources().
				SetTaskID("task_id").
				SetCluster("cluster").
				AddContainers(*cocoa.NewECSContainerResources().
					SetContainerID("container_id").
					SetName("container_name"))
		}
		validStatusInfo := func() cocoa.ECSPodStatusInfo {
			return *cocoa.NewECSPodStatusInfo().
				SetStatus(cocoa.StatusRunning).
				AddContainers(*cocoa.NewECSContainerStatusInfo().
					SetContainerID("container_id").
					SetName("name").
					SetStatus(cocoa.StatusRunning))
		}
		t.Run("FailsWithEmpty", func(t *testing.T) {
			opts := NewBasicPodOptions()
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			smClient, err := secret.NewBasicSecretsManagerClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smClient))
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetClient(ecsClient).
				SetVault(v).
				SetResources(validResources()).
				SetStatusInfo(validStatusInfo())
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithoutClient", func(t *testing.T) {
			smClient, err := secret.NewBasicSecretsManagerClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smClient))
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetVault(v).
				SetResources(validResources()).
				SetStatusInfo(validStatusInfo())
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithoutVault", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetClient(ecsClient).
				SetResources(validResources()).
				SetStatusInfo(validStatusInfo())
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithoutResources", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetClient(ecsClient).
				SetStatusInfo(validStatusInfo())
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithBadResources", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetClient(ecsClient).
				SetResources(*cocoa.NewECSPodResources()).
				SetStatusInfo(validStatusInfo())
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithoutStatus", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetClient(ecsClient).
				SetResources(validResources())
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithBadStatus", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicPodOptions().
				SetClient(ecsClient).
				SetResources(validResources()).
				SetStatusInfo(*cocoa.NewECSPodStatusInfo())
			assert.Error(t, opts.Validate())
		})
	})
}
