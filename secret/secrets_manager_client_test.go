package secret

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*SecretsManagerClient)(nil), &BasicSecretsManagerClient{})
}

func TestSecretsManagerCreateAndDeleteSecret(t *testing.T) {
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

	t.Run("CreateFailsWithInvalidInput", func(t *testing.T) {
		out, err := c.CreateSecret(ctx, &secretsmanager.CreateSecretInput{})
		assert.Error(t, err)
		assert.Zero(t, out)
	})

	t.Run("DeleteFailsWithInvalidInput", func(t *testing.T) {
		out, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{})
		assert.Error(t, err)
		assert.Zero(t, out)
	})

	t.Run("CreateAndDeleteSucceed", func(t *testing.T) {
		out, err := c.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(os.Getenv("AWS_SECRET_PREFIX") + "hello"),
			Description:  aws.String("foo"),
			SecretString: aws.String("bar"),
		})
		require.NoError(t, err)
		require.NotZero(t, out)

		defer func() {
			if out != nil && out.Name != nil && out.ARN != nil {
				out, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
					SecretId: out.ARN,
				})
				require.NoError(t, err)
				require.NotZero(t, out)
			}
		}()

		require.NotZero(t, out.Name)
		require.NotZero(t, out.ARN)

	})
}

func checkAWSEnvVars(t *testing.T) {
	missing := []string{}

	for _, envVar := range []string{
		"AWS_ACCESS_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SECRET_PREFIX",
		"AWS_ROLE",
		"AWS_REGION",
	} {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		assert.FailNow(t, fmt.Sprintf("missing required AWS environment variables: %s", missing))
	}
}
