package secret

import "context"

// Vault allows you to interact with a secrets storage service.
type Vault interface {
	// CreateSecret creates a new secret. Options are applied in the order
	// they're specified and conflicting options are overwritten.
	CreateSecret(ctx context.Context, opts ...*SecretCreationOptions) (id string, err error)
	// GetValue returns the value of the secret identified by ID.
	GetValue(ctx context.Context, id string) (val string, err error)
	// UpdateValue updates an existing secret's value by ID.
	UpdateValue(ctx context.Context, id string) error
	// DeleteSecret deletes a secret by ID.
	DeleteSecret(ctx context.Context, id string) error
}

// SecretCreationOptions provide options to create a secret.
type SecretCreationOptions struct {
	Name  *string
	Value *string
}

// SetName sets the friendly name for the secret.
func (o *SecretCreationOptions) SetName(name string) *SecretCreationOptions {
	o.Name = &name
	return o
}

// SetValue sets the secret value.
func (o *SecretCreationOptions) SetValue(value string) *SecretCreationOptions {
	o.Value = &value
	return o
}

func mergeSecretCreationOptions(opts ...*SecretCreationOptions) *SecretCreationOptions {
	merged := SecretCreationOptions{}

	for _, opt := range opts {
		if opt == nil {
			continue
		}

		if opt.Name != nil {
			merged.Name = opt.Name
		}

		if opt.Value != nil {
			merged.Value = opt.Value
		}
	}

	return &merged
}
