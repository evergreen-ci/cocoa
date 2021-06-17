package secret

import (
	"context"
	"fmt"
	"os"
	"path"
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

	checkAWSEnvVars(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupSecret := func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient, out *secretsmanager.CreateSecretOutput) {
		if out != nil && out.ARN != nil {
			out, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
				ForceDeleteWithoutRecovery: aws.Bool(true),
				SecretId:                   out.ARN,
			})
			require.NoError(t, err)
			require.NotZero(t, out)
		}
	}

	for tName, tCase := range map[string]func(context.Context, *testing.T, *BasicSecretsManagerClient){
		"CreateSecretFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			out, err := c.CreateSecret(ctx, &secretsmanager.CreateSecretInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeleteSecretFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			out, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"CreateAndDeleteSecretSucceed": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			createSecretOut, err := c.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         aws.String(makeTestSecret(t.Name())),
				SecretString: aws.String("hello"),
			})
			require.NoError(t, err)
			require.NotZero(t, createSecretOut)
			require.NotZero(t, createSecretOut.ARN)
			out, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
				ForceDeleteWithoutRecovery: aws.Bool(true),
				SecretId:                   createSecretOut.ARN,
			})
			require.NoError(t, err)
			require.NotZero(t, out)

		},
		"GetValueFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			out, err := c.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"GetValueFailsWithValidNonexistentInput": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			out, err := c.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
				SecretId: aws.String(utility.RandomString()),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"UpdateValueFailsWithValidNonexistentInput": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			out, err := c.UpdateSecretValue(ctx, &secretsmanager.UpdateSecretInput{
				SecretId:     aws.String(utility.RandomString()),
				SecretString: aws.String("hello"),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"CreateAndGetSucceed": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			secretName := makeTestSecret(t.Name())
			outCreate, err := c.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         aws.String(secretName),
				SecretString: aws.String("foo"),
			})
			require.NoError(t, err)
			require.NotZero(t, outCreate)
			require.NotZero(t, &outCreate)

			defer cleanupSecret(ctx, t, c, outCreate)

			require.NotZero(t, outCreate.ARN)
			require.NotZero(t, &outCreate.ARN)

			// if out != nil && out.ARN != nil {
			out, err := c.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
				SecretId: outCreate.ARN,
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			assert.Equal(t, "foo", *out.SecretString)
			assert.Equal(t, secretName, *out.Name)
			// }
		},
		"UpdateSecretModifiesValue": func(ctx context.Context, t *testing.T, c *BasicSecretsManagerClient) {
			secretName := makeTestSecret(t.Name())
			out, err := c.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         aws.String(secretName),
				SecretString: aws.String("bar"),
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotZero(t, &out)

			defer cleanupSecret(ctx, t, c, out)

			require.NotZero(t, out.ARN)

			if out != nil && out.ARN != nil {
				out, err := c.UpdateSecretValue(ctx, &secretsmanager.UpdateSecretInput{
					SecretId:     out.ARN,
					SecretString: aws.String("leaf"),
				})
				require.NoError(t, err)
				require.NotZero(t, out)
			}

			if out != nil && out.ARN != nil {
				out, err := c.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
					SecretId: out.ARN,
				})
				require.NoError(t, err)
				require.NotZero(t, out)
				assert.Equal(t, "leaf", *out.SecretString)
				assert.Equal(t, secretName, *out.Name)
			}
		},
	} {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

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
			require.NotNil(t, c)

			defer c.Close(tctx)

			tCase(tctx, t, c)
		})
	}

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

func makeTestSecret(name string) string {
	return fmt.Sprint(path.Join(os.Getenv("AWS_SECRET_PREFIX"), "cocoa", name, utility.RandomString()))
}
