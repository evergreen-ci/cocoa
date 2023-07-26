package testutil

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
	"github.com/pkg/errors"
)

// runtimeNamespace is a random string generated during testing runtime that
// acts as a namespace for this particular runtime's tests. It is used to
// namespace AWS resources (e.g. secrets, task definitions). This avoids an
// issue where the tests can be running concurrently on different machines and
// may interfere with each other due to the way AWS resources are cleaned up at
// the end of tests. For example, if one machine is running the ECS tests and at
// the same time, another machine is cleaning up the resources for the same ECS
// tests, they should not affect one another.
var runtimeNamespace = utility.RandomString()

// AWSRole returns the AWS IAM role from the environment variable.
func AWSRole() string {
	return os.Getenv("AWS_ROLE")
}

// ValidIntegrationAWSOptions returns valid options to create an AWS client that
// can make actual requests to AWS for integration testing.
func ValidIntegrationAWSOptions(ctx context.Context, hc *http.Client) (awsutil.ClientOptions, error) {
	config, err := config.LoadDefaultConfig(ctx,
		config.WithHTTPClient(hc),
	)
	if err != nil {
		return awsutil.ClientOptions{}, errors.Wrap(err, "loading config")
	}

	role := AWSRole()
	if role == "" {
		return awsutil.ClientOptions{
			Config: &config,
		}, nil
	}

	config.Credentials, err = (&awsutil.ClientOptions{
		HTTPClient:    hc,
		Role:          &role,
		Region:        &config.Region,
		CredsProvider: &config.Credentials,
	}).GetCredentialsProvider(ctx)
	if err != nil {
		return awsutil.ClientOptions{}, errors.Wrap(err, "getting credentials")
	}
	return awsutil.ClientOptions{
		Config: &config,
	}, nil
}

// ValidNonIntegrationAWSOptions returns valid options to create an AWS client
// that doesn't make any actual requests to AWS.
func ValidNonIntegrationAWSOptions() awsutil.ClientOptions {
	return *awsutil.NewClientOptions().
		SetCredentialsProvider(credentials.NewStaticCredentialsProvider("", "", "")).
		SetRegion("us-east-1")
}
