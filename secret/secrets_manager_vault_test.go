package secret

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicSecretsManager(t *testing.T) {
	assert.Implements(t, (*cocoa.Vault)(nil), &BasicSecretsManager{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	t.Run("NewBasicSecretsManager", func(t *testing.T) {
		c, err := NewBasicSecretsManagerClient(testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, c.Close(ctx))
		}()
		t.Run("FailsWithZeroOptions", func(t *testing.T) {
			sm, err := NewBasicSecretsManager(*NewBasicSecretsManagerOptions())
			assert.Error(t, err)
			assert.Zero(t, sm)
		})
		t.Run("SucceedsWithValidOptions", func(t *testing.T) {
			sm, err := NewBasicSecretsManager(*NewBasicSecretsManagerOptions().SetClient(c))
			assert.NoError(t, err)
			assert.NotZero(t, sm)
		})
	})
}

func TestSecretsManager(t *testing.T) {
	testutil.CheckAWSEnvVarsForSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupSecret := func(ctx context.Context, t *testing.T, v cocoa.Vault, id string) {
		if id != "" {
			require.NoError(t, v.DeleteSecret(ctx, id))
		}
	}

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	c, err := NewBasicSecretsManagerClient(testutil.ValidIntegrationAWSOptions(hc))
	require.NoError(t, err)
	defer func() {
		testutil.CleanupSecrets(ctx, t, c)

		assert.NoError(t, c.Close(ctx))
	}()

	for tName, tCase := range testcase.VaultTests(cleanupSecret) {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			m, err := NewBasicSecretsManager(*NewBasicSecretsManagerOptions().SetClient(c))
			require.NoError(t, err)
			require.NotNil(t, m)

			tCase(tctx, t, m)
		})
	}
}

func TestBasicSecretsManagerOptions(t *testing.T) {
	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	t.Run("NewBasicSecretsManagerOptions", func(t *testing.T) {
		opts := NewBasicSecretsManagerOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetClient", func(t *testing.T) {
		c, err := NewBasicSecretsManagerClient(testutil.ValidNonIntegrationAWSOptions())
		require.NoError(t, err)
		opts := NewBasicSecretsManagerOptions().SetClient(c)
		assert.Equal(t, c, opts.Client)
	})
	t.Run("SetCache", func(t *testing.T) {
		sc := &testutil.NoopSecretCache{}
		opts := NewBasicSecretsManagerOptions().SetCache(sc)
		require.NotZero(t, opts.Cache)
		assert.Equal(t, sc, opts.Cache)
	})
	t.Run("SetCacheTag", func(t *testing.T) {
		tag := "tag"
		opts := NewBasicSecretsManagerOptions().SetCacheTag(tag)
		assert.Equal(t, tag, utility.FromStringPtr(opts.CacheTag))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("FailsWithEmpty", func(t *testing.T) {
			opts := NewBasicSecretsManagerOptions()
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			smClient, err := NewBasicSecretsManagerClient(testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicSecretsManagerOptions().
				SetClient(smClient).
				SetCache(&testutil.NoopSecretCache{}).
				SetCacheTag("tag")
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithoutClient", func(t *testing.T) {
			opts := NewBasicSecretsManagerOptions().
				SetCache(&testutil.NoopSecretCache{}).
				SetCacheTag("tag")
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithCacheTagButNoCache", func(t *testing.T) {
			c, err := NewBasicSecretsManagerClient(testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicSecretsManagerOptions().
				SetClient(c).
				SetCacheTag("tag")
			assert.Error(t, opts.Validate())
		})
		t.Run("DefaultsCacheTagWithCache", func(t *testing.T) {
			c, err := NewBasicSecretsManagerClient(testutil.ValidNonIntegrationAWSOptions())
			require.NoError(t, err)
			opts := NewBasicSecretsManagerOptions().
				SetClient(c).
				SetCache(&testutil.NoopSecretCache{})
			assert.NoError(t, opts.Validate())
			assert.Equal(t, defaultCacheTrackingTag, utility.FromStringPtr(opts.CacheTag))
		})
	})
}
