package secret

import (
	"context"
	"errors"
)

// BasicSecretsManager provides a Vault implementation backed by Amazon Secrets
// Manager.
type BasicSecretsManager struct {
	v    *Vault
	opts *CreationOptions
}

// CreateSecret creates a new secret.
func (m *BasicSecretsManager) CreateSecret(ctx context.Context, opts ...*CreationOptions) (id string, err error) {
	return "", errors.New("TODO: implement")
}

// GetValue returns an existing secret's decrypted value.
func (m *BasicSecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	return "", errors.New("TODO: implement")
}

// UpdateValue updates an existing secret's value.
func (m *BasicSecretsManager) UpdateValue(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}

// DeleteSecret deletes an existing secret.
func (m *BasicSecretsManager) DeleteSecret(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}
