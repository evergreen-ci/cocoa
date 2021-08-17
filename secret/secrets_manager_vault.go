package secret

import (
	"context"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/utility"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// BasicSecretsManager provides a cocoa.Vault implementation backed by AWS
// Secrets Manager.
type BasicSecretsManager struct {
	client cocoa.SecretsManagerClient
}

// NewBasicSecretsManager creates a Vault backed by AWS Secrets Manager.
func NewBasicSecretsManager(c cocoa.SecretsManagerClient) *BasicSecretsManager {
	return &BasicSecretsManager{
		client: c,
	}
}

// CreateSecret creates a new secret.
func (m *BasicSecretsManager) CreateSecret(ctx context.Context, s cocoa.NamedSecret) (id string, err error) {
	if err := s.Validate(); err != nil {
		return "", errors.Wrap(err, "invalid secret")
	}
	out, err := m.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         s.Name,
		SecretString: s.Value,
	})
	if err != nil {
		return "", err
	}
	if out == nil || out.ARN == nil {
		return "", errors.New("expected an ID, but none was returned from Secrets Manager")
	}
	return *out.ARN, nil
}

// UpsertSecret creates a new secret if it doesn't exist, or updates the secret
// if it already exists.
// kim: TODO: test
func (m *BasicSecretsManager) UpsertSecret(ctx context.Context, s cocoa.NamedSecret) (id string, err error) {
	id, err = m.CreateSecret(ctx, s)
	if err == nil {
		return id, nil
	}
	if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() != secretsmanager.ErrCodeResourceExistsException {
		return "", err
	}

	describeSecret, err := m.client.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: s.Name,
	})
	if err != nil {
		return "", err
	}
	if describeSecret == nil || describeSecret.ARN == nil {
		return "", errors.New("expected an ID for an existing secret, but not was returned from Secrets Manager")
	}

	arn := utility.FromStringPtr(describeSecret.ARN)
	updated := cocoa.NewNamedSecret().SetName(arn).SetValue(utility.FromStringPtr(s.Value))
	if err := m.UpdateValue(ctx, *updated); err != nil {
		return "", err
	}

	return arn, nil
}

// GetValue returns an existing secret's decrypted value.
func (m *BasicSecretsManager) GetValue(ctx context.Context, id string) (val string, err error) {
	if id == "" {
		return "", errors.New("must specify a non-empty id")
	}

	out, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &id})
	if err != nil {
		return "", err
	}
	if out == nil || out.SecretString == nil {
		return "", errors.New("expected a value, but none was returned from Secrets Manager")
	}
	return *out.SecretString, nil
}

// UpdateValue updates an existing secret's value.
func (m *BasicSecretsManager) UpdateValue(ctx context.Context, s cocoa.NamedSecret) error {
	if err := s.Validate(); err != nil {
		return errors.Wrap(err, "invalid secret")
	}
	_, err := m.client.UpdateSecretValue(ctx, &secretsmanager.UpdateSecretInput{
		SecretId:     s.Name,
		SecretString: s.Value,
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
