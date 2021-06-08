package secret

import (
	"context"
	"errors"
)

// BasicSecretsManager provides a Vault implementation backed by Amazon Secrets
// Manager.
type BasicSecretsManager struct {
}

func (m *BasicSecretsManager) CreateSecret(ctx context.Context, opts ...*SecretCreationOptions) (id string, err error) {
	return "", errors.New("TODO: implement")
}

func (m *BasicSecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	return "", errors.New("TODO: implement")
}

func (m *BasicSecretsManager) UpdateValue(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}

func (m *BasicSecretsManager) DeleteSecret(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}
