package cocoa

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSClientInterface(t *testing.T) {
	assert.Implements(t, (*ECSClient)(nil), &BasicECSClient{})
}

func TestECSClientRegisterAndDeregisterTaskDefinition(t *testing.T) {
	checkAWSEnvVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	t.Run("RegisterFailsWithInvalidInput", func(t *testing.T) {
		out, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{})
		assert.Error(t, err)
		assert.Zero(t, out)
	})
	t.Run("DeregisterFailsWithInvalidInput", func(t *testing.T) {
		out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{})
		assert.Error(t, err)
		assert.Zero(t, out)
	})
	t.Run("RegisterAndDeregisterSucceeds", func(t *testing.T) {
		out, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Command: []*string{aws.String("echo"), aws.String("hello")},
					Image:   aws.String("ubuntu"),
					Name:    aws.String("hello_world"),
				},
			},
			Cpu:    aws.String("128"),
			Memory: aws.String("4"),
			Family: aws.String("foo"),
		})
		require.NoError(t, err)
		require.NotZero(t, out)

		defer func() {
			if out != nil && out.TaskDefinition != nil && out.TaskDefinition.TaskDefinitionArn != nil {
				out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
					TaskDefinition: out.TaskDefinition.TaskDefinitionArn,
				})
				require.NoError(t, err)
				require.NotZero(t, out)
			}
		}()

		require.NotZero(t, out.TaskDefinition)
		require.NotZero(t, out.TaskDefinition.Status)
		assert.Equal(t, "ACTIVE", *out.TaskDefinition.Status)
		require.NotZero(t, out.TaskDefinition.RegisteredAt)
		assert.NotZero(t, *out.TaskDefinition.RegisteredAt)
	})
}

func checkAWSEnvVars(t *testing.T) {
	missing := []string{}

	for _, envVar := range []string{
		"AWS_ACCESS_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_ROLE",
		"AWS_REGION",
		"AWS_ECS_CLUSTER",
	} {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		assert.FailNow(t, fmt.Sprintf("missing required AWS environment variables: %s", missing))
	}
}
