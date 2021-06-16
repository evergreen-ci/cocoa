package mock

import (
	"context"
	"errors"

	"github.com/evergreen-ci/cocoa/secret"
)

// Vault provides a mock implementation of a secret.Vault. This makes it
// possible to introspect on inputs to the vault and control the vault's output.
// It provides some default implementations where possible.
type Vault struct{}

// CreateSecret saves the input options and returns a mock secret ID. The mock
// output can be customized. By default, it will create a cached mock secret
// based on the input.
func (m *Vault) CreateSecret(ctx context.Context, s secret.NamedSecret) (id string, err error) {
	return "", errors.New("TODO: implement")
}

// GetValue saves the input options and returns an existing mock secret's value.
// The mock output can be customized. By default, it will return a cached mock
// secret's value if it exists.
func (m *Vault) GetValue(ctx context.Context, id string) (val string, err error) {
	return "", errors.New("TODO: implement")
}

// UpdateValue saves the input options and updates an existing mock secret. The
// mock output can be customized. By default, it will update a cached mock
// secret if it exists.
func (m *Vault) UpdateValue(ctx context.Context, id, val string) error {
	return errors.New("TODO: implement")
}

// DeleteSecret saves the input options and deletes an existing mock secret. The
// mock output can be customized. By default, it will delete a cached mock
// secret if it exists.
func (m *Vault) DeleteSecret(ctx context.Context, id string) error {
	return errors.New("TODO: implement")
}
