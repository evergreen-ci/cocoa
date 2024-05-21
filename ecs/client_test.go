package ecs

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsECS "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultTestTimeout = time.Minute

func TestBasicECSClient(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSClient)(nil), &BasicClient{})

	testutil.CheckAWSEnvVarsForECS(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	awsOpts := testutil.ValidIntegrationAWSOptions(ctx, hc)
	c, err := NewBasicClient(ctx, awsOpts)
	require.NoError(t, err)
	require.NotNil(t, c)

	defer func() {
		testutil.CleanupTaskDefinitions(ctx, t, c)
		testutil.CleanupTasks(ctx, t, c)
	}()

	for tName, tCase := range testcase.ECSClientTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			tCase(tctx, t, c)
		})
	}

	registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
	defer func() {
		_, err := c.DeregisterTaskDefinition(ctx, &awsECS.DeregisterTaskDefinitionInput{
			TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
		})
		assert.NoError(t, err)
	}()

	for tName, tCase := range testcase.ECSClientRegisteredTaskDefinitionTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			tCase(tctx, t, c, *registerOut.TaskDefinition)
		})
	}
}

func TestConvertFailureToError(t *testing.T) {
	t.Run("ConvertsToFormattedError", func(t *testing.T) {
		const (
			arn    = "some_arn"
			reason = "some reason"
			detail = "some detail"
		)
		err := ConvertFailureToError(types.Failure{
			Arn:    aws.String(arn),
			Reason: aws.String(reason),
			Detail: aws.String(detail),
		})
		require.NotZero(t, err)
		assert.Contains(t, err.Error(), arn)
		assert.Contains(t, err.Error(), reason)
		assert.Contains(t, err.Error(), detail)
	})
	t.Run("ConvertsMissingTaskFailureToTaskNotFound", func(t *testing.T) {
		err := ConvertFailureToError(types.Failure{
			Arn:    aws.String("arn"),
			Reason: aws.String(ReasonTaskMissing),
		})
		assert.True(t, cocoa.IsECSTaskNotFoundError(err))
	})
}

func TestIsNonRetryableError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	awsOpts := testutil.ValidNonIntegrationAWSOptions()
	c, err := NewBasicClient(ctx, awsOpts)
	require.NoError(t, err)
	require.NotNil(t, c)

	assert.False(t, c.isNonRetryableError(&types.UpdateInProgressException{}))
	assert.True(t, c.isNonRetryableError(&types.AccessDeniedException{}))
}
