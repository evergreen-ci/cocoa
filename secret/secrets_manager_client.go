package secret

import (
	"context"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
)

// SecretsManagerClient provides a common interface to interact with a Secrets
// Manager client and its mock implementation for testing. Implementations must
// handle retrying and backoff.
type SecretsManagerClient interface {
	// CreateSecret creates a new secret.
	CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error)
	// GetSecretValue gets the decrypted value of a secret.
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error)
	// DeleteSecret deletes an existing secret.
	DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error)
	// Close closes the client and cleans up its resources. Implementations
	// should ensure that this is idempotent.
	Close(ctx context.Context) error
}

// BasicSecretsManagerClient provides a SecretsManagerClient implementation that
// wraps the Secrets Manager API. It supports retrying requests using
// exponential backoff and jitter.
type BasicSecretsManagerClient struct {
	sm      *secretsmanager.SecretsManager
	opts    awsutil.ClientOptions
	session *session.Session
}

// NewBasicSecretsManagerClient creates a new Secrets Manager client from the
// given options.
func NewBasicSecretsManagerClient(opts awsutil.ClientOptions) (*BasicSecretsManagerClient, error) {
	c := &BasicSecretsManagerClient{
		opts: opts,
	}
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	return c, nil
}

func (c *BasicSecretsManagerClient) setup() error {
	if err := c.opts.Validate(); err != nil {
		return errors.Wrap(err, "invalid options")
	}

	if c.sm != nil {
		return nil
	}

	if err := c.setupSession(); err != nil {
		return errors.Wrap(err, "setting up session")
	}

	c.sm = secretsmanager.New(c.session)

	return nil
}

func (c *BasicSecretsManagerClient) setupSession() error {
	if c.session != nil {
		return nil
	}

	creds, err := c.opts.GetCredentials()
	if err != nil {
		return errors.Wrap(err, "getting credentials")
	}
	sess, err := session.NewSession(&aws.Config{
		HTTPClient:  c.opts.HTTPClient,
		Region:      c.opts.Region,
		Credentials: creds,
	})
	if err != nil {
		return errors.Wrap(err, "creating session")
	}

	c.session = sess

	return nil
}

// CreateSecret creates a new secret.
func (c *BasicSecretsManagerClient) CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *secretsmanager.CreateSecretOutput
	var err error
	msg := awsutil.MakeAPILogMessage("CreateSecret", in)
	if err := utility.Retry(
		ctx,
		func() (bool, error) {
			out, err = c.sm.CreateSecretWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				switch awsErr.Code() {
				case request.InvalidParameterErrCode, request.ParamRequiredErrCode:
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, err
}

// GetSecretValue gets the decrypted value of an existing secret.
func (c *BasicSecretsManagerClient) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput) (*secretsmanager.GetSecretValueOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *secretsmanager.GetSecretValueOutput
	var err error
	msg := awsutil.MakeAPILogMessage("GetSecret", in)
	if err := utility.Retry(
		ctx,
		func() (bool, error) {
			out, err = c.sm.GetSecretValueWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				switch awsErr.Code() {
				case request.InvalidParameterErrCode, request.ParamRequiredErrCode:
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, err
}

// UpdateSecret updates the decrypted value of an existing secret.
func (c *BasicSecretsManagerClient) UpdateSecret(ctx context.Context, in *secretsmanager.UpdateSecretInput) (*secretsmanager.UpdateSecretOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *secretsmanager.UpdateSecretOutput
	var err error
	msg := awsutil.MakeAPILogMessage("UpdateSecret", in)
	if err := utility.Retry(
		ctx,
		func() (bool, error) {
			out, err = c.sm.UpdateSecretWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				switch awsErr.Code() {
				case request.InvalidParameterErrCode, request.ParamRequiredErrCode:
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, err
}

// DeleteSecret deletes an existing secret.
func (c *BasicSecretsManagerClient) DeleteSecret(ctx context.Context, in *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *secretsmanager.DeleteSecretOutput
	var err error
	msg := awsutil.MakeAPILogMessage("DeleteSecret", in)
	if err := utility.Retry(
		ctx,
		func() (bool, error) {
			out, err = c.sm.DeleteSecretWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				switch awsErr.Code() {
				case request.InvalidParameterErrCode, request.ParamRequiredErrCode:
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// Close closes the client.
func (c *BasicSecretsManagerClient) Close(ctx context.Context) error {
	c.opts.Close()
	return nil
}
