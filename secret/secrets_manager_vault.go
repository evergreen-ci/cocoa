package secret

import (
	"context"
	"errors"
)

// SecretsManager provides a Vault implementation backed by Amazon Secrets
// Manager.
type SecretsManager struct {
}

func (m *SecretsManager) CreateSecret(ctx context.Context, opts ...*SecretCreationOptions) (id string, err error) {
	return "", errors.New("TODO: implement")
}

func (m *SecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	return "", errors.New("TODO: implement")
}

func (m *SecretsManager) UpdateValue(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}

func (m *SecretsManager) DeleteSecret(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}
