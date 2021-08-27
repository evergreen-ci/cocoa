package testutil

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/stretchr/testify/assert"
)

const projectName = "cocoa"

// NewSecretName creates a new test secret name with a common prefix, the given
// name, and a random string.
func NewSecretName(name string) string {
	return fmt.Sprint(path.Join(strings.TrimSuffix(SecretPrefix(), "/"), projectName, name, utility.RandomString()))
}

// SecretPrefix returns the prefix name for secrets from the environment
// variable.
func SecretPrefix() string {
	return os.Getenv("AWS_SECRET_PREFIX")
}

// CleanupSecrets cleans up all existing secrets used in Cocoa tests.
func CleanupSecrets(ctx context.Context, t *testing.T, c cocoa.SecretsManagerClient) {
	for token := cleanupSecretsWithToken(ctx, t, c, nil); token != nil; token = cleanupSecretsWithToken(ctx, t, c, token) {
	}
}

// cleanupSecretsWithToken cleans up all existing secrets used in Cocoa tests
// based on the results from the pagination token.
func cleanupSecretsWithToken(ctx context.Context, t *testing.T, c cocoa.SecretsManagerClient, token *string) (nextToken *string) {
	out, err := c.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		NextToken: token,
	})
	if !assert.NoError(t, err) {
		return nil
	}
	if !assert.NotZero(t, out) {
		return nil
	}

	for _, secret := range out.SecretList {
		if secret == nil {
			continue
		}
		if secret.ARN == nil {
			continue
		}

		arn := *secret.ARN

		// Ignore secrets that were not generated within Cocoa.
		name := path.Join(strings.TrimSuffix(SecretPrefix(), "/"), "cocoa")
		if !strings.Contains(arn, name) {
			continue
		}

		_, err := c.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			ForceDeleteWithoutRecovery: utility.TruePtr(),
			SecretId:                   &arn,
		})
		if assert.NoError(t, err) {
			grip.Info(message.Fields{
				"message": "cleaned up leftover secret",
				"arn":     arn,
				"test":    t.Name(),
			})
		}
	}

	return out.NextToken
}
