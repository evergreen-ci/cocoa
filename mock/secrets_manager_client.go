package mock

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// MockSecretsManagerClient proivdes a mock implementation of a
// secret.SecretsManagerClient. This makes it possible to introspect on inputs
// to the client and control the client's output. It provides some default
// implementations where possible.
type MockSecretsManagerClient struct{}

func (c *MockSecretsManagerClient) CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockSecretsManagerClient) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockSecretsManagerClient) DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	return nil, errors.New("TODO: implement")
}

func (c *MockSecretsManagerClient) Close(ctx context.Context) error {
	return errors.New("TODO: implement")
}
