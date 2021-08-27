package testutil

import (
	"os"
)

// AWSRegion returns the AWS region from the environment variable.
func AWSRegion() string {
	return os.Getenv("AWS_REGION")
}

// AWSRole returns the AWS IAM role from the environment variable.
func AWSRole() string {
	return os.Getenv("AWS_ROLE")
}
