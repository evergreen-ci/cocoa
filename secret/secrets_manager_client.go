package secret

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// SecretsManagerClient provides a common interface to interact with a Secrets
// Manager client and its mock implementation for testing. Implementations must
// handle retrying and backoff.
type SecretsManagerClient interface {
	// CreateSecret creates a new secret.
	CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error)
	// CreateSecret gets the decrypted contents of a secret.
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
	// CreateSecret deletes an existing secret.
	DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error)
	// Close closes the client and cleans up its resources. Implementations
	// should ensure that this is idempotent.
	Close(ctx context.Context) error
}

// BasicSecretsManagerClient provides a SecretsManagerClient implementation that
// wraps the Secrets Manager API. It supports retrying requests using
// exponential backoff and jitter.
type BasicSecretsManagerClient struct{}

func (c *BasicSecretsManagerClient) CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *BasicSecretsManagerClient) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *BasicSecretsManagerClient) DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *BasicSecretsManagerClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
