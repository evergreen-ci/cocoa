package secret

import "context"

// Vault allows you to interact with a secrets storage service.
type Vault interface {
	// CreateSecret creates a new secret. Options are applied in the order
	// they're specified and conflicting options are overwritten.
	CreateSecret(ctx context.Context, opts ...*CreationOptions) (id string, err error)
	// GetValue returns the value of the secret identified by ID.
	GetValue(ctx context.Context, id string) (val string, err error)
	// UpdateValue updates an existing secret's value by ID.
	UpdateValue(ctx context.Context, id string) error
	// DeleteSecret deletes a secret by ID.
	DeleteSecret(ctx context.Context, id string) error
}

// CreationOptions provide options to create a secret.
type CreationOptions struct {
	Name  *string
	Value *string
}

// SetName sets the friendly name for the secret.
func (o *CreationOptions) SetName(name string) *CreationOptions {
	o.Name = &name
	return o
}

// SetValue sets the secret value.
func (o *CreationOptions) SetValue(value string) *CreationOptions {
	o.Value = &value
	return o
}

//nolint:deadcode
func mergeCreationOptions(opts ...*CreationOptions) *CreationOptions {
	merged := CreationOptions{}

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
