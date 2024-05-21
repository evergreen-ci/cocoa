package ecs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicPodCreator(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodCreator)(nil), &BasicPodCreator{})

	testutil.CheckAWSEnvVarsForECSAndSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range map[string]func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache){
		"NewPodCreatorFailsWithMissingClientAndVault": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache) {
			podCreator, err := NewBasicPodCreator(*NewBasicPodCreatorOptions())
			require.Error(t, err)
			require.Zero(t, podCreator)
		},
		"NewPodCreatorFailsWithMissingClient": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache) {
			podCreator, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetVault(v))
			require.Error(t, err)
			require.Zero(t, podCreator)
		},
		"NewPodCreatorSucceedsWithNoVault": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache) {
			podCreator, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c))
			require.NoError(t, err)
			require.NotZero(t, podCreator)
		},
		"NewPodCreatorSucceedsWithClientAndVault": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache) {
			podCreator, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c).SetVault(v))
			require.NoError(t, err)
			require.NotZero(t, podCreator)
		},
		"NewPodCreatorSucceedsWithClientVaultAndCache": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache) {
			podCreator, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c).SetVault(v).SetCache(pdc))
			require.NoError(t, err)
			require.NotZero(t, podCreator)
		},
		"NewPodCreatorSucceedsWithCacheAndClient": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, v cocoa.Vault, pdc cocoa.ECSPodDefinitionCache) {
			podCreator, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c).SetCache(pdc))
			require.NoError(t, err)
			require.NotZero(t, podCreator)
		},
	} {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			awsOpts := testutil.ValidNonIntegrationAWSOptions()

			c, err := NewBasicClient(ctx, awsOpts)
			require.NoError(t, err)

			smc, err := secret.NewBasicSecretsManagerClient(ctx, awsOpts)
			require.NoError(t, err)
			require.NotNil(t, c)

			m, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smc))
			require.NoError(t, err)
			require.NotNil(t, m)

			pdc := &testutil.NoopECSPodDefinitionCache{Tag: "cache-tag"}

			tCase(tctx, t, c, m, pdc)
		})
	}
}

func TestECSPodCreator(t *testing.T) {
	testutil.CheckAWSEnvVarsForECSAndSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	awsOpts := testutil.ValidIntegrationAWSOptions(ctx, hc)
	c, err := NewBasicClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupTaskDefinitions(ctx, t, c)
		testutil.CleanupTasks(ctx, t, c)
	}()

	for tName, tCase := range testcase.ECSPodCreatorTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			pc, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c))
			require.NoError(t, err)

			tCase(tctx, t, pc)
		})
	}

	smc, err := secret.NewBasicSecretsManagerClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupSecrets(ctx, t, smc)
	}()

	for tName, tCase := range testcase.ECSPodCreatorVaultTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			m, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smc))
			require.NoError(t, err)
			require.NotNil(t, m)

			pc, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c).SetVault(m))
			require.NoError(t, err)

			tCase(tctx, t, pc)
		})
	}

	registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
	defer func() {
		_, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
			TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
		})
		assert.NoError(t, err)
	}()

	for tName, tCase := range testcase.ECSPodCreatorRegisteredTaskDefinitionTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			pc, err := NewBasicPodCreator(*NewBasicPodCreatorOptions().SetClient(c))
			require.NoError(t, err)

			tCase(tctx, t, pc, *registerOut.TaskDefinition)
		})
	}
}
