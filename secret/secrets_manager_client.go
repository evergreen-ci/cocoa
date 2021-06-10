package secret

import (
	"context"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/evergreen-ci/cocoa/awsutil"
)

// SecretsManagerClient provides a common interface to interact with a Secrets
// Manager client and its mock implementation for testing. Implementations must
// handle retrying and backoff.
type SecretsManagerClient interface {
	// CreateSecret creates a new secret.
	CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error)
	// GetSecretValue gets the decrypted value of a secret.
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
	// DeleteSecret deletes an existing secret.
	DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error)
	// Close closes the client and cleans up its resources. Implementations
	// should ensure that this is idempotent.
	Close(ctx context.Context) error
}

// BasicSecretsManagerClient provides a SecretsManagerClient implementation that
// wraps the Secrets Manager API. It supports retrying requests using
// exponential backoff and jitter.
type BasicSecretsManagerClient struct {
	opts awsutil.ClientOptions
}

// NewBasicSecretsManagerClient creates a new Secrets Manager client from the
// given options.
func NewBasicSecretsManagerClient(opts awsutil.ClientOptions) (*BasicSecretsManagerClient, error) {
	if err := opts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid options")
	}

	return &BasicSecretsManagerClient{
		opts: opts,
	}, nil
}

// CreateSecret creates a new secret.
func (c *BasicSecretsManagerClient) CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

// GetSecretValue gets the decrypted value of an existing secret.
func (c *BasicSecretsManagerClient) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return nil, errors.New("TODO: implement")
}

// DeleteSecret deletes an existing secret.
func (c *BasicSecretsManagerClient) DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

// Close closes the client.
func (c *BasicSecretsManagerClient) Close(ctx context.Context) error {
	c.opts.Close()
	return nil
}
