package ecs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/awsutil"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSClientInterface(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSClient)(nil), &BasicECSClient{})
}

func TestECSClient(t *testing.T) {
	testutil.CheckAWSEnvVarsForECS(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	c, err := NewBasicECSClient(awsutil.ClientOptions{
		Creds:  credentials.NewEnvCredentials(),
		Region: aws.String(os.Getenv("AWS_REGION")),
		Role:   aws.String(os.Getenv("AWS_ROLE")),
		RetryOpts: &utility.RetryOptions{
			MaxAttempts: 5,
		},
		HTTPClient: hc,
	})
	require.NoError(t, err)
	require.NotNil(t, c)

	for tName, tCase := range testcase.ECSClientTaskDefinitionTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			defer c.Close(tctx)

			tCase(tctx, t, c)
		})
	}

	registerIn := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Command: []*string{aws.String("echo"), aws.String("foo")},
				Image:   aws.String("busybox"),
				Name:    aws.String("print_foo"),
			},
		},
		Cpu:    aws.String("128"),
		Memory: aws.String("4"),
		Family: aws.String(testutil.NewTaskDefinitionFamily(t.Name())),
	}

	registerOut, err := c.RegisterTaskDefinition(ctx, registerIn)
	require.NoError(t, err)
	require.NotZero(t, registerOut)
	require.NotZero(t, registerOut.TaskDefinition)

	defer func() {
		cleanupTaskDefinition(ctx, t, c, registerOut)
		require.NoError(t, c.Close(ctx))
	}()

	for tName, tCase := range testcase.ECSClientRegisteredTaskDefinitionTests(*registerIn, *registerOut) {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			tCase(tctx, t, c)
		})
	}
}

// cleanupTaskDefinition cleans up an existing task definition.
func cleanupTaskDefinition(ctx context.Context, t *testing.T, c cocoa.ECSClient, out *ecs.RegisterTaskDefinitionOutput) {
	if out != nil && out.TaskDefinition != nil && out.TaskDefinition.TaskDefinitionArn != nil {
		out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
			TaskDefinition: out.TaskDefinition.TaskDefinitionArn,
		})
		require.NoError(t, err)
		require.NotZero(t, out)
	}
}
