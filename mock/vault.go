package mock

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa/secret"
)

// MockVault provides a mock implementation of a secret.Vault. This makes it
// possible to introspect on inputs to the vault and control the vault's output.
// It provides some default implementations where possible.
type MockVault struct {
}

func (m *MockVault) CreateSecret(ctx context.Context, opts ...*secret.SecretCreationOptions) (id string, err error) {
	return "", errors.New("TODO: implement")
}

func (m *MockVault) GetValue(ctx context.Context, id string) (val string, err error) {
	return "", errors.New("TODO: implement")
}

func (m *MockVault) UpdateValue(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}

func (m *MockVault) DeleteSecret(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}
