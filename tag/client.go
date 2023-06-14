package tag

import (
	"context"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/message"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
)

// BasicTagClient provides a cocoa.TagClient implementation that wraps the AWS
// Resource Groups Tagging API. It supports retrying requests using exponential
// backoff and jitter.
type BasicTagClient struct {
	awsutil.BaseClient
	rgt *resourcegroupstaggingapi.ResourceGroupsTaggingAPI
}

// NewBasicTagClient creates a new AWS Resource Groups Tagging API
// client from the given options.
func NewBasicTagClient(opts awsutil.ClientOptions) (*BasicTagClient, error) {
	c := &BasicTagClient{
		BaseClient: awsutil.NewBaseClient(opts),
	}
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	return c, nil
}

func (c *BasicTagClient) setup() error {
	if c.rgt != nil {
		return nil
	}

	sess, err := c.GetSession()
	if err != nil {
		return errors.Wrap(err, "initializing session")
	}

	c.rgt = resourcegroupstaggingapi.New(sess)

	return nil
}

// GetResources finds arbitrary AWS resources that match the input filters.
func (c *BasicTagClient) GetResources(ctx context.Context, in *resourcegroupstaggingapi.GetResourcesInput) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	if err := c.setup(); err != nil {
		return nil, errors.Wrap(err, "setting up client")
	}

	var out *resourcegroupstaggingapi.GetResourcesOutput
	var err error
	if err := utility.Retry(ctx, func() (bool, error) {
		msg := awsutil.MakeAPILogMessage("GetResources", in)
		out, err = c.rgt.GetResourcesWithContext(ctx, in)
		if awsErr, ok := err.(awserr.Error); ok {
			grip.Debug(message.WrapError(awsErr, msg))
			if c.isNonRetryableErrorCode(awsErr.Code()) {
				return false, err
			}
		}
		return true, err
	}, c.GetRetryOptions()); err != nil {
		return nil, err
	}
	return out, nil
}

// Close cleans up all resources owned by the client.
func (c *BasicTagClient) Close(ctx context.Context) error {
	return c.BaseClient.Close(ctx)
}

func (c *BasicTagClient) isNonRetryableErrorCode(code string) bool {
	switch code {
	case resourcegroupstaggingapi.ErrCodeInvalidParameterException:
		return true
	default:
		return false
	}
}
