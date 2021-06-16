package secret

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
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
	out, err := m.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         s.Name,
		SecretString: s.Value,
	})
	return *out.ARN, err
}

// GetValue returns an existing secret's decrypted value.
func (m *BasicSecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	out, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &id})
	if err != nil {
		return "", err
	}
	return *out.SecretString, err
}

// UpdateValue updates an existing secret's value.
func (m *BasicSecretsManager) UpdateValue(ctx context.Context, id, val string) error {
	_, err := m.client.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(id),
		SecretString: aws.String(val),
	})
	return err
}

// DeleteSecret deletes an existing secret.
func (m *BasicSecretsManager) DeleteSecret(ctx context.Context, id string) error {
	_, err := m.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		ForceDeleteWithoutRecovery: aws.Bool(true),
		SecretId:                   &id,
	})
	return err
}
