package cocoa

import (
	"context"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
)

// ECSClient provides a common interface to interact with an ECS client and its
// mock implementation for testing. Implementations must handle retrying and
// backoff.
type ECSClient interface {
	// RegisterTaskDefinition registers the definition for a new task with ECS.
	RegisterTaskDefinition(context.Context, *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	// DeregisterTaskDefinition deregisters an existing ECS task definition.
	DeregisterTaskDefinition(ctx context.Context, in *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error)
	// RunTask runs a registered task.
	RunTask(ctx context.Context, in *ecs.RunTaskInput) (*ecs.RunTaskOutput, error)
	// Close closes the client and cleans up its resources. Implementations
	// should ensure that this is idempotent.
	Close(ctx context.Context) error
}

// BasicECSClient provides an ECSClient implementation that wraps the ECS API.
// It supports retrying requests using exponential backoff and jitter.
type BasicECSClient struct {
	ecs     *ecs.ECS
	opts    *awsutil.ClientOptions
	session *session.Session
}

// NewBasicECSClient creates a new ECS client from the given options.
func NewBasicECSClient(opts awsutil.ClientOptions) (*BasicECSClient, error) {
	if err := opts.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid options")
	}

	return &BasicECSClient{
		opts: &opts,
	}, nil
}

func (c *BasicECSClient) setup() error {
	if c.ecs != nil {
		return nil
	}

	if c.session == nil {
		creds, err := c.opts.GetCredentials()
		if err != nil {
			return errors.Wrap(err, "getting credentials")
		}
		s, err := session.NewSession(&aws.Config{
			HTTPClient:  c.opts.HTTPClient,
			Region:      c.opts.Region,
			Credentials: creds,
		})
		if err != nil {
			return errors.Wrap(err, "creating session")
		}
		c.session = s
	}

	c.ecs = ecs.New(c.session)

	return nil
}

// RegisterTaskDefinition registers a new task definition.
func (c *BasicECSClient) RegisterTaskDefinition(ctx context.Context, in *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.RegisterTaskDefinitionOutput
	var err error
	msg := makeAWSLogMessage("RegisterTaskDefinition", in)
	if err := utility.Retry(ctx,
		func() (bool, error) {
			out, err = c.ecs.RegisterTaskDefinitionWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, err
}

// DeregisterTaskDefinition deregisters an existing task definition.
func (c *BasicECSClient) DeregisterTaskDefinition(context.Context, *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	return nil, errors.New("TODO: implement")
}

// RunTask runs a new task.
func (c *BasicECSClient) RunTask(context.Context, *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

// Close closes the client and cleans up its resources.
func (c *BasicECSClient) Close(ctx context.Context) error {
	c.opts.Close()
	return nil
}

func makeAWSLogMessage(endpoint string, in interface{}) message.Fields {
	return message.Fields{
		"message":  "AWS API call",
		"endpoint": endpoint,
		"input":    in,
	}
}
