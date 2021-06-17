package secret

import (
	"context"

	"github.com/pkg/errors"

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
	if err := s.Validate(); err != nil {
		return "", errors.Wrap(err, "invalid secret")
	}
	out, err := m.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         s.Name,
		SecretString: s.Value,
	})
	if out != nil && out.ARN != nil {
		return *out.ARN, err
	}
	return "", err
}

// GetValue returns an existing secret's decrypted value.
func (m *BasicSecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	if id == "" {
		return "", errors.New("must specify a non-empty id")
	}

	out, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &id})
	if out != nil && out.SecretString != nil {
		return *out.SecretString, nil
	}
	return "", err
}

// UpdateValue updates an existing secret's value.
func (m *BasicSecretsManager) UpdateValue(ctx context.Context, id, val string) error {
	if id == "" {
		return errors.New("must specify a non-empty id")
	}
	_, err := m.client.UpdateSecretValue(ctx, &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(id),
		SecretString: aws.String(val),
	})
	return err
}

// DeleteSecret deletes an existing secret.
// If the secret does not exist, this will perform no operation.
func (m *BasicSecretsManager) DeleteSecret(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("must specify a non-empty id")
	}
	_, err := m.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		ForceDeleteWithoutRecovery: aws.Bool(true),
		SecretId:                   &id,
	})
	return err
}
