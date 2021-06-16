package secret

import "context"

// Vault allows you to interact with a secrets storage service.
type Vault interface {
	// CreateSecret creates a new secret and returns the unique identifier for
	// the stored secret.
	CreateSecret(ctx context.Context, s NamedSecret) (id string, err error)
	// GetValue returns the value of the secret identified by ID.
	GetValue(ctx context.Context, id string) (val string, err error)
	// UpdateValue updates an existing secret's value by ID.
	UpdateValue(ctx context.Context, id, val string) error
	// DeleteSecret deletes a secret by ID.
	DeleteSecret(ctx context.Context, id string) error
}

// NamedSecret represents a secret with a name.
type NamedSecret struct {
	// Name is the friendly human-readable name of the secret.
	Name *string
	// Value is the stored value of the secret.
	Value *string
}

// SetName sets the friendly name for the secret.
func (o *NamedSecret) SetName(name string) *NamedSecret {
	o.Name = &name
	return o
}

// SetValue sets the secret value.
func (o *NamedSecret) SetValue(value string) *NamedSecret {
	o.Value = &value
	return o
}
