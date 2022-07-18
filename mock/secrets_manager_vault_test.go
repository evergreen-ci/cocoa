package mock

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultWithSecretsManager(t *testing.T) {
	assert.Implements(t, (*cocoa.Vault)(nil), &Vault{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range secretsManagerVaultTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			resetECSAndSecretsManagerCache()

			c := &SecretsManagerClient{}
			defer func() {
				assert.NoError(t, c.Close(tctx))
			}()

			sc := NewSecretCache(&testutil.NoopSecretCache{})

			const cacheTag = "cache_tag"

			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().
				SetClient(c).
				SetCache(sc).
				SetCacheTag(cacheTag))
			require.NoError(t, err)
			mv := NewVault(v)

			tCase(tctx, t, mv, sc, c, cacheTag)
		})
	}

	cleanupSecret := func(ctx context.Context, t *testing.T, v cocoa.Vault, id string) {
		if id != "" {
			require.NoError(t, v.DeleteSecret(ctx, id))
		}
	}

	defer resetECSAndSecretsManagerCache()

	for tName, tCase := range testcase.VaultTests(cleanupSecret) {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			resetECSAndSecretsManagerCache()

			c := &SecretsManagerClient{}
			defer func() {
				assert.NoError(t, c.Close(tctx))
			}()
			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(c))
			require.NoError(t, err)
			mv := NewVault(v)

			tCase(tctx, t, mv)
		})
	}
}

// secretsManagerVaultTests are mock-specific tests for the Secrets Manager
// vault with a cache.
func secretsManagerVaultTests() map[string]func(ctx context.Context, t *testing.T, v cocoa.Vault, sc *SecretCache, c *SecretsManagerClient, cacheTag string) {
	return map[string]func(ctx context.Context, t *testing.T, v cocoa.Vault, sc *SecretCache, c *SecretsManagerClient, cacheTag string){
		"CreateSecretSucceedsAndCaches": func(ctx context.Context, t *testing.T, v cocoa.Vault, sc *SecretCache, c *SecretsManagerClient, cacheTag string) {
			ns := cocoa.NewNamedSecret().
				SetName(testutil.NewSecretName(t)).
				SetValue("value")
			id, err := v.CreateSecret(ctx, *ns)
			require.NoError(t, err)
			require.NotZero(t, id)

			require.NotZero(t, c.CreateSecretInput, "should have created a secret")

			assert.Equal(t, utility.FromStringPtr(ns.Name), utility.FromStringPtr(c.CreateSecretInput.Name))
			assert.Equal(t, utility.FromStringPtr(ns.Value), utility.FromStringPtr(c.CreateSecretInput.SecretString))
			require.Len(t, c.CreateSecretInput.Tags, 1, "should have a cache tracking tag")
			assert.Equal(t, cacheTag, utility.FromStringPtr(c.CreateSecretInput.Tags[0].Key))
			assert.Equal(t, "false", utility.FromStringPtr(c.CreateSecretInput.Tags[0].Value), "cache tag should initially mark secret as uncached before caching")

			require.NotZero(t, sc.PutInput, "should have cache the secret")
			assert.Equal(t, id, sc.PutInput.ID)
			assert.Equal(t, utility.FromStringPtr(ns.Name), sc.PutInput.Name)

			require.NotZero(t, c.TagResourceInput, "should have re-tagged resource to indicate that it's cached")
			assert.Equal(t, id, utility.FromStringPtr(c.TagResourceInput.SecretId))
			require.Len(t, c.TagResourceInput.Tags, 1)
			assert.Equal(t, cacheTag, utility.FromStringPtr(c.TagResourceInput.Tags[0].Key))
			assert.Equal(t, "true", utility.FromStringPtr(c.TagResourceInput.Tags[0].Value), "cache tag should be marked as cached")
		},
		"CreateSecretTagsStrandedSecretAsUncachedWhenCachingFails": func(ctx context.Context, t *testing.T, v cocoa.Vault, sc *SecretCache, c *SecretsManagerClient, cacheTag string) {
			sc.PutError = errors.New("fake error")

			ns := cocoa.NewNamedSecret().
				SetName(testutil.NewSecretName(t)).
				SetValue("value")
			id, err := v.CreateSecret(ctx, *ns)
			assert.Error(t, err, "should have failed to cache the secret")
			assert.Zero(t, id)

			require.NotZero(t, c.CreateSecretInput, "should have created a secret")

			assert.Equal(t, utility.FromStringPtr(ns.Name), utility.FromStringPtr(c.CreateSecretInput.Name))
			assert.Equal(t, utility.FromStringPtr(ns.Value), utility.FromStringPtr(c.CreateSecretInput.SecretString))
			require.Len(t, c.CreateSecretInput.Tags, 1, "should have cache tracking tag")
			assert.Equal(t, cacheTag, utility.FromStringPtr(c.CreateSecretInput.Tags[0].Key))
			assert.Equal(t, "false", utility.FromStringPtr(c.CreateSecretInput.Tags[0].Value), "cache tag should initially mark secret as uncached")

			assert.NotZero(t, sc.PutInput, "should have attempted to cache the secret")
			assert.Zero(t, c.TagResourceInput, "should not have re-tagged secret because it is not cached")
		},
		"CreateSecretDoesNotCacheWhenCreatingSecretFails": func(ctx context.Context, t *testing.T, v cocoa.Vault, sc *SecretCache, c *SecretsManagerClient, cacheTag string) {
			c.CreateSecretError = errors.New("fake error")

			ns := cocoa.NewNamedSecret().
				SetName(testutil.NewSecretName(t)).
				SetValue("value")

			id, err := v.CreateSecret(ctx, *ns)
			assert.Error(t, err, "shoud have failed to register task definition")
			assert.Zero(t, id)

			assert.NotZero(t, c.CreateSecretInput, "should have attempted to create a secret")

			assert.Zero(t, sc.PutInput, "should not have attempted to cache the secret after secret creation failed")
		},
	}
}
