package secret

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// BasicSecretsManager provides a Vault implementation backed by Amazon Secrets
// Manager.
type BasicSecretsManager struct {
	client SecretsManagerClient
}

// NewBasicSecretsManager creates a Vault backed by Secrets Manager.
func NewBasicSecretsManager(c SecretsManagerClient) *BasicSecretsManager {
	return &BasicSecretsManager{
		client: c,
	}
}

// CreateSecret creates a new secret.
func (m *BasicSecretsManager) CreateSecret(ctx context.Context, s NamedSecret) (id string, err error) {
	newManager := NewBasicSecretsManager(m.client)
	if s.Name == nil || s.Value == nil {
		return "", errors.New("Invalid input")
	}
	out, err := newManager.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: s.Name, SecretString: s.Value})
	return *out.ARN, err
}

// GetValue returns an existing secret's decrypted value.
func (m *BasicSecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	newManager := NewBasicSecretsManager(m.client)
	out, err := newManager.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &id})
	return *out.SecretString, err
}

// UpdateValue updates an existing secret's value.
func (m *BasicSecretsManager) UpdateValue(ctx context.Context, id string) error {
	newManager := NewBasicSecretsManager(m.client)
	return newManager.UpdateValue(ctx, id)
}

// DeleteSecret deletes an existing secret.
func (m *BasicSecretsManager) DeleteSecret(ctx context.Context, id string) error {
	newManager := NewBasicSecretsManager(m.client)
	_, err := newManager.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{SecretId: &id})
	return err
}
