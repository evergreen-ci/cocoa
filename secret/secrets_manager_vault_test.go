package secret

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsManager(t *testing.T) {
	assert.Implements(t, (*Vault)(nil), &BasicSecretsManager{})

}

func TestVaultCreateAndDeleteSecret(t *testing.T) {
	checkAWSEnvVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	c, err := NewBasicSecretsManagerClient(awsutil.ClientOptions{
		Creds:  credentials.NewEnvCredentials(),
		Region: aws.String(os.Getenv("AWS_REGION")),
		Role:   aws.String(os.Getenv("AWS_ROLE")),
		RetryOpts: &utility.RetryOptions{
			MaxAttempts: 5,
		},
		HTTPClient: hc,
	})
	require.NoError(t, err)

	m := NewBasicSecretsManager(c)

	Cleanup := func(out string) {
		if out != "" {
			err := m.DeleteSecret(ctx, out)
			require.NoError(t, err)
		}
	}

	// Temporarily commented out because waiting for input validation
	// t.Run("VaultCreateFailsWithInvalidInput", func(t *testing.T) {
	// 	out, err := m.CreateSecret(ctx, NamedSecret{})
	// 	assert.Error(t, err)
	// 	assert.Zero(t, out)
	// })

	t.Run("VaultDeleteFailsWithInvalidInput", func(t *testing.T) {
		err := m.DeleteSecret(ctx, "")
		assert.Error(t, err)
	})

	t.Run("VaultCreateAndDeleteSucceed", func(t *testing.T) {
		out, err := m.CreateSecret(ctx, NamedSecret{
			Name:  aws.String(os.Getenv("AWS_SECRET_PREFIX") + "hi"),
			Value: aws.String("world")})

		require.NoError(t, err)
		require.NotZero(t, out)

		defer Cleanup(out)
	})

	t.Run("VaultGetFailsWithInvalidInput", func(t *testing.T) {
		out, err := m.GetValue(ctx, "")
		assert.Error(t, err)
		assert.Zero(t, out)
	})

	t.Run("VaultUpdateFailsWithInvalidInput", func(t *testing.T) {
		err := m.UpdateValue(ctx, "", "")
		assert.Error(t, err)
	})

	t.Run("VaultCreateAndGetSucceed", func(t *testing.T) {

		out, err := m.CreateSecret(ctx, NamedSecret{
			Name:  aws.String(os.Getenv("AWS_SECRET_PREFIX") + "ham"),
			Value: aws.String("eggs")})

		require.NoError(t, err)
		require.NotZero(t, out)

		defer Cleanup(out)

		defer func() {
			if out != "" {
				out, err := m.GetValue(ctx, out)
				require.NoError(t, err)
				require.NotZero(t, out)
				assert.Equal(t, "eggs", out)
			}
		}()
	})

	t.Run("VaultUpdateSucceed", func(t *testing.T) {
		out, err := m.CreateSecret(ctx, NamedSecret{
			Name:  aws.String(os.Getenv("AWS_SECRET_PREFIX") + "spam"),
			Value: aws.String("eggs"),
		})
		require.NoError(t, err)
		require.NotZero(t, out)

		defer Cleanup(out)

		defer func() {
			if out != "" {
				out, err := m.GetValue(ctx, out)
				require.NoError(t, err)
				require.NotZero(t, out)
				assert.Equal(t, "ham", out)
			}
		}()

		defer func() {
			if out != "" {
				err := m.UpdateValue(ctx, out, "ham")
				require.NoError(t, err)
			}
		}()

	})
}
