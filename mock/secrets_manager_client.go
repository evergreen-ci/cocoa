package mock

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// SecretsManagerClient provides a mock implementation of a
// secret.SecretsManagerClient. This makes it possible to introspect on inputs
// to the client and control the client's output. It provides some default
// implementations where possible.
type SecretsManagerClient struct{}

// CreateSecret saves the input options and returns a new mock secret. The mock
// output can be customized. By default, it will create a cached mock secret
// based on the input.
func (c *SecretsManagerClient) CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

// GetSecretValue saves the input options and returns an existing mock secret's
// value. The mock output can be customized. By default, it will return a cached
// mock secret if it exists.
func (c *SecretsManagerClient) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return nil, errors.New("TODO: implement")
}

// UpdateSecret saves the input options and returns an updated mock secret
// value. The mock output can be customized. By default, it will update a cached
// mock secret if it exists.
func (c *SecretsManagerClient) UpdateSecret(ctx context.Context, in *secretsmanager.UpdateSecretInput) (*secretsmanager.UpdateSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

// DeleteSecret saves the input options and deletes an existing mock secret. The
// mock output can be customized. By default, it will delete a cached mock
// secret if it exists.
func (c *SecretsManagerClient) DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

// Close closes the mock client. The mock output can be customized. By default,
// it is a no-op that returns no error.
func (c *SecretsManagerClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
