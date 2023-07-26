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

func TestBasicPodDefinitionManager(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodDefinitionManager)(nil), &BasicPodDefinitionManager{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	t.Run("NewPodDefinitionManager", func(t *testing.T) {
		c, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, c.Close(ctx))
		}()
		t.Run("FailsWithZeroOptions", func(t *testing.T) {
			pdm, err := NewBasicPodDefinitionManager(*NewBasicPodDefinitionManagerOptions())
			assert.Error(t, err)
			assert.Zero(t, pdm)
		})
		t.Run("SucceedsWithValidOptions", func(t *testing.T) {
			pdm, err := NewBasicPodDefinitionManager(*NewBasicPodDefinitionManagerOptions().SetClient(c))
			assert.NoError(t, err)
			assert.NotZero(t, pdm)
		})
	})
}

func TestECSPodDefinitionManager(t *testing.T) {
	testutil.CheckAWSEnvVarsForECSAndSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	awsOpts, err := testutil.ValidIntegrationAWSOptions(ctx, hc)
	require.NoError(t, err)

	c, err := NewBasicClient(ctx, awsOpts)
	require.NoError(t, err)
	require.NotZero(t, c)
	defer func() {
		testutil.CleanupTaskDefinitions(ctx, t, c)
		assert.NoError(t, c.Close(ctx))
	}()

	for tName, tCase := range testcase.ECSPodDefinitionManagerTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			opts := NewBasicPodDefinitionManagerOptions().SetClient(c)

			pdm, err := NewBasicPodDefinitionManager(*opts)
			require.NoError(t, err)

			tCase(tctx, t, pdm)
		})
	}

	smc, err := secret.NewBasicSecretsManagerClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupSecrets(ctx, t, smc)

		assert.NoError(t, smc.Close(ctx))
	}()

	for tName, tCase := range testcase.ECSPodDefinitionManagerVaultTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smc))
			require.NoError(t, err)
			require.NotNil(t, v)

			opts := NewBasicPodDefinitionManagerOptions().
				SetClient(c).
				SetVault(v)

			pdm, err := NewBasicPodDefinitionManager(*opts)
			require.NoError(t, err)

			tCase(tctx, t, pdm)
		})
	}
}

func TestBasicPodDefinitionManagerOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	t.Run("NewBasicPodDefinitionManagerOptions", func(t *testing.T) {
		opts := NewBasicPodDefinitionManagerOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetClient", func(t *testing.T) {
		c, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		opts := NewBasicPodDefinitionManagerOptions().SetClient(c)
		assert.Equal(t, c, opts.Client)
	})
	t.Run("SetVault", func(t *testing.T) {
		c, err := secret.NewBasicSecretsManagerClient(ctx, testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(c))
		require.NoError(t, err)
		opts := NewBasicPodDefinitionManagerOptions().SetVault(v)
		assert.Equal(t, v, opts.Vault)
	})
	t.Run("SetCache", func(t *testing.T) {
		pdc := &testutil.NoopECSPodDefinitionCache{}
		opts := NewBasicPodDefinitionManagerOptions().SetCache(pdc)
		require.NotZero(t, opts.Cache)
		assert.Equal(t, pdc, opts.Cache)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("FailsWithEmpty", func(t *testing.T) {
			opts := NewBasicPodDefinitionManagerOptions()
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			ecsClient, err := NewBasicClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			smClient, err := secret.NewBasicSecretsManagerClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smClient))
			require.NoError(t, err)
			opts := NewBasicPodDefinitionManagerOptions().
				SetClient(ecsClient).
				SetVault(v).
				SetCache(&testutil.NoopECSPodDefinitionCache{Tag: "tag"})
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithoutClient", func(t *testing.T) {
			smClient, err := secret.NewBasicSecretsManagerClient(ctx, testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smClient))
			require.NoError(t, err)
			opts := NewBasicPodDefinitionManagerOptions().
				SetVault(v).
				SetCache(&testutil.NoopECSPodDefinitionCache{Tag: "tag"})
			assert.Error(t, opts.Validate())
		})
	})
}
