package awsutil

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/evergreen-ci/utility"
	"github.com/pkg/errors"
)

// BaseClient provides various helpers to set up and use AWS clients for various
// services.
type BaseClient struct {
	opts    ClientOptions
	session *session.Session
}

// NewBaseClient creates a new base AWS client from the client options.
func NewBaseClient(opts ClientOptions) BaseClient {
	return BaseClient{opts: opts}
}

// GetSession ensures that the session is initialized and returns it.
func (c *BaseClient) GetSession() (*session.Session, error) {
	if err := c.opts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid options")
	}

	if c.session != nil {
		return c.session, nil
	}

	sess, err := c.opts.GetSession()
	if err != nil {
		return nil, errors.Wrap(err, "creating session")
	}

	c.session = sess

	return c.session, nil
}

// GetRetryOptions returns the retry options for the client.
func (c *BaseClient) GetRetryOptions() utility.RetryOptions {
	if c.opts.RetryOpts == nil {
		c.opts.RetryOpts = &utility.RetryOptions{}
	}
	return *c.opts.RetryOpts
}

// Close closes the client and cleans up its resources.
func (c *BaseClient) Close(ctx context.Context) error {
	c.opts.Close()
	return nil
}
