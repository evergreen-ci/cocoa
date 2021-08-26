package secret

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsManager(t *testing.T) {
	assert.Implements(t, (*cocoa.Vault)(nil), &BasicSecretsManager{})

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

	defer func() {
		c, err := NewBasicSecretsManagerClient(awsutil.ClientOptions{
			Creds:  credentials.NewEnvCredentials(),
			Region: aws.String(testutil.AWSRegion()),
			Role:   aws.String(testutil.AWSRole()),
			RetryOpts: &utility.RetryOptions{
				MaxAttempts: 5,
			},
			HTTPClient: hc,
		})
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, c.Close(ctx))
		}()

		secrets := cleanupSecrets(ctx, t, c)
		grip.InfoWhen(len(secrets) > 0, message.Fields{
			"message": "cleaned up leftover secrets",
			"secrets": secrets,
			"test":    t.Name(),
		})

	}()

	for tName, tCase := range testcase.VaultTests(cleanupSecret) {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			c, err := NewBasicSecretsManagerClient(awsutil.ClientOptions{
				Creds:  credentials.NewEnvCredentials(),
				Region: aws.String(testutil.AWSRegion()),
				Role:   aws.String(testutil.AWSRole()),
				RetryOpts: &utility.RetryOptions{
					MaxAttempts: 5,
				},
				HTTPClient: hc,
			})
			require.NoError(t, err)
			require.NotNil(t, c)

			m := NewBasicSecretsManager(c)
			require.NotNil(t, m)

			tCase(tctx, t, m)
		})
	}
}

func cleanupSecrets(ctx context.Context, t *testing.T, c cocoa.SecretsManagerClient) []string {
	var secrets []string
	var nextToken *string

	secrets, nextToken = cleanupSecretsWithToken(ctx, t, c, nextToken)

	for nextToken != nil {
		var nextSecrets []string
		nextSecrets, nextToken = cleanupSecretsWithToken(ctx, t, c, nextToken)
		secrets = append(secrets, nextSecrets...)
	}

	return secrets
}

func cleanupSecretsWithToken(ctx context.Context, t *testing.T, c cocoa.SecretsManagerClient, nextToken *string) (secrets []string, followingToken *string) {
	out, err := c.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		NextToken: nextToken,
	})
	require.NoError(t, err)
	require.NotZero(t, out)

	for _, secret := range out.SecretList {
		if secret == nil {
			continue
		}
		if secret.ARN == nil {
			continue
		}
		name := strings.Join([]string{strings.TrimSuffix(testutil.SecretPrefix(), "/"), "cocoa"}, "/")
		arn := *secret.ARN
		if !strings.Contains(arn, name) {
			continue
		}
		_, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			ForceDeleteWithoutRecovery: utility.TruePtr(),
			SecretId:                   &arn,
		})
		assert.NoError(t, err)
		secrets = append(secrets, arn)
	}

	return secrets, out.NextToken
}
