package ecs

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"
)

// BasicECSClient provides a cocoa.ECSClient implementation that wraps the AWS
// ECS API. It supports retrying requests using exponential backoff and jitter.
type BasicECSClient struct {
	ecs     *ecs.ECS
	opts    *awsutil.ClientOptions
	session *session.Session
}

// NewBasicECSClient creates a new AWS ECS client from the given options.
func NewBasicECSClient(opts awsutil.ClientOptions) (*BasicECSClient, error) {
	c := &BasicECSClient{
		opts: &opts,
	}
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	return c, nil
}

func (c *BasicECSClient) setup() error {
	if err := c.opts.Validate(); err != nil {
		return errors.Wrap(err, "invalid options")
	}

	if c.ecs != nil {
		return nil
	}

	if err := c.setupSession(); err != nil {
		return errors.Wrap(err, "setting up session")
	}

	c.ecs = ecs.New(c.session)

	return nil
}

func (c *BasicECSClient) setupSession() error {
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

// RegisterTaskDefinition registers a new task definition.
func (c *BasicECSClient) RegisterTaskDefinition(ctx context.Context, in *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.RegisterTaskDefinitionOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("RegisterTaskDefinition", in)
			out, err = c.ecs.RegisterTaskDefinitionWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}

	return out, nil
}

// DescribeTaskDefinition describes an existing task definition.
func (c *BasicECSClient) DescribeTaskDefinition(ctx context.Context, in *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.DescribeTaskDefinitionOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("DescribeTaskDefinition", in)
			out, err = c.ecs.DescribeTaskDefinitionWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTaskDefinitions returns the ARNs for the task definitions that match the
// input filters.
func (c *BasicECSClient) ListTaskDefinitions(ctx context.Context, in *ecs.ListTaskDefinitionsInput) (*ecs.ListTaskDefinitionsOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.ListTaskDefinitionsOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("ListTaskDefinitions", in)
			out, err = c.ecs.ListTaskDefinitionsWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// DeregisterTaskDefinition deregisters an existing task definition.
func (c *BasicECSClient) DeregisterTaskDefinition(ctx context.Context, in *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.DeregisterTaskDefinitionOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("DeregisterTaskDefinition", in)
			out, err = c.ecs.DeregisterTaskDefinitionWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}

	return out, nil
}

// RunTask runs a new task.
func (c *BasicECSClient) RunTask(ctx context.Context, in *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.RunTaskOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("RunTask", in)
			out, err = c.ecs.RunTaskWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// DescribeTasks describes one or more existing tasks.
func (c *BasicECSClient) DescribeTasks(ctx context.Context, in *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.DescribeTasksOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("DescribeTasks", in)
			out, err = c.ecs.DescribeTasksWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTasks returns the ARNs for the task that match the input filters.
func (c *BasicECSClient) ListTasks(ctx context.Context, in *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.ListTasksOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("ListTasks", in)
			out, err = c.ecs.ListTasksWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// StopTask stops a running task.
func (c *BasicECSClient) StopTask(ctx context.Context, in *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.StopTaskOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("StopTask", in)
			out, err = c.ecs.StopTaskWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if isTaskNotFoundError(awsErr) {
					return false, cocoa.NewECSTaskNotFoundError(utility.FromStringPtr(in.Task))
				}
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// TagResource adds tags to an existing resource in ECS.
func (c *BasicECSClient) TagResource(ctx context.Context, in *ecs.TagResourceInput) (*ecs.TagResourceOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *ecs.TagResourceOutput
	var err error
	if err := utility.Retry(ctx,
		func() (bool, error) {
			msg := awsutil.MakeAPILogMessage("TagResource", in)
			out, err = c.ecs.TagResourceWithContext(ctx, in)
			if awsErr, ok := err.(awserr.Error); ok {
				grip.Debug(message.WrapError(awsErr, msg))
				if c.isNonRetryableErrorCode(awsErr.Code()) {
					return false, err
				}
			}
			return true, err
		}, *c.opts.RetryOpts); err != nil {
		return nil, err
	}
	return out, nil
}

// Close closes the client and cleans up its resources.
func (c *BasicECSClient) Close(ctx context.Context) error {
	c.opts.Close()
	return nil
}

// isNonRetryableErrorCode returns whether or not the error code from ECS is
// known to be not retryable.
func (c *BasicECSClient) isNonRetryableErrorCode(code string) bool {
	switch code {
	case ecs.ErrCodeAccessDeniedException,
		ecs.ErrCodeClientException,
		ecs.ErrCodeInvalidParameterException,
		ecs.ErrCodeClusterNotFoundException,
		request.InvalidParameterErrCode,
		request.ParamRequiredErrCode:
		return true
	default:
		return false
	}
}

// isTaskNotFoundError returns whether or not the error returned from ECS is
// because the task cannot be found.
func isTaskNotFoundError(err error) bool {
	awsErr, ok := err.(awserr.Error)
	if !ok {
		return false
	}
	return awsErr.Code() == ecs.ErrCodeInvalidParameterException && strings.Contains(awsErr.Message(), "The referenced task was not found")
}

// ConvertFailureToError converts an ECS failure message into a formatted error.
// If the failure is due to being unable to find the task, it will return a
// cocoa.ECSTaskNotFound error.
// Docs: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/api_failures_messages.html
func ConvertFailureToError(f ecs.Failure) error {
	if isTaskNotFoundFailure(f) {
		return cocoa.NewECSTaskNotFoundError(utility.FromStringPtr(f.Arn))
	}
	var parts []string
	if arn := utility.FromStringPtr(f.Arn); arn != "" {
		parts = append(parts, fmt.Sprintf("task '%s'", arn))
	}
	if reason := utility.FromStringPtr(f.Reason); reason != "" {
		parts = append(parts, fmt.Sprintf("(reason) %s", reason))
	}
	if detail := utility.FromStringPtr(f.Detail); detail != "" {
		parts = append(parts, fmt.Sprintf("(detail) %s", detail))
	}
	if len(parts) == 0 {
		return errors.New("ECS failure did not contain any additional failure information")
	}
	return errors.New(strings.Join(parts, ": "))
}

// isTaskNotFoundFailure returns whether or not the failure reason returned from
// ECS is because the task cannot be found.
func isTaskNotFoundFailure(f ecs.Failure) bool {
	return f.Arn != nil && utility.FromStringPtr(f.Reason) == "MISSING"
}
