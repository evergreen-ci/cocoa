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
func TestECSClientTaskDefinition(t *testing.T) {

	cleanupTaskDefinition := func(ctx context.Context, t *testing.T, c *BasicECSClient, out *ecs.RegisterTaskDefinitionOutput) {
		if out != nil && out.TaskDefinition != nil && out.TaskDefinition.TaskDefinitionArn != nil {
			out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: out.TaskDefinition.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, out)
		}
	}

	cleanupTask := func(ctx context.Context, t *testing.T, c *BasicECSClient, runOut *ecs.RunTaskOutput) {
		out, err := c.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: aws.String(os.Getenv("AWS_ECS_CLUSTER")),
			Task:    aws.String(*runOut.Tasks[0].TaskArn),
		})
		require.NoError(t, err)
		require.NotZero(t, out)
		require.NotZero(t, out.Task)
		assert.Equal(t, *runOut.Tasks[0].TaskArn, *out.Task.TaskArn)
	}

func TestECSClientRegisterAndDeregisterTaskDefinition(t *testing.T) {
	checkAWSEnvVarsForECS(t)

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

	registerOut, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Command: []*string{aws.String("echo"), aws.String("foo")},
				Image:   aws.String("ubuntu"),
				Name:    aws.String("print_foo"),
			},
		},
		Cpu:         aws.String("128"),
		Memory:      aws.String("4"),
		Family:      aws.String(os.Getenv("AWS_ECS_TASK_DEFINITION_PREFIX") + "bar"),
		TaskRoleArn: aws.String("arn:aws:iam::579766882180:role/dev.task.role"),
	})
	require.NoError(t, err)
	require.NotZero(t, registerOut)
	require.NotZero(t, registerOut.TaskDefinition)

	defer func() {
		cleanupTaskDefinition(ctx, t, c, registerOut)
		c.Close(ctx)
	}()

	for tName, tCase := range map[string]func(context.Context, *testing.T, *BasicECSClient){
		"RegisterFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeregisterFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RegisterAndDeregisterSucceeds": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Command: []*string{aws.String("echo"), aws.String("hello")},
						Image:   aws.String("ubuntu"),
						Name:    aws.String("hello_world"),
					},
				},
				Cpu:         aws.String("128"),
				Memory:      aws.String("4"),
				Family:      aws.String(os.Getenv("AWS_ECS_TASK_DEFINITION_PREFIX") + "foo"),
				TaskRoleArn: aws.String("arn:aws:iam::579766882180:role/dev.task.role"),
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
		},
		"RunTaskFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.RunTask(ctx, &ecs.RunTaskInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTasksFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"StopTaskFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.StopTask(ctx, &ecs.StopTaskInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RunTaskFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.RunTask(ctx, &ecs.RunTaskInput{
				TaskDefinition: aws.String(utility.RandomString()),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTasksFailsWithNoClusterAndValidButNonexistentInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Tasks: []*string{aws.String(utility.RandomString())},
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTasksReturnsFailureWithClusterAndValidButNonexistentInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(os.Getenv("AWS_ECS_CLUSTER")),
				Tasks:   []*string{aws.String("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
			})

			assert.NoError(t, err)
			assert.NotZero(t, out)
			assert.NotZero(t, out.Failures)
			assert.Empty(t, out.Tasks)

		},
		"StopTaskFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			out, err := c.StopTask(ctx, &ecs.StopTaskInput{
				Task: aws.String(utility.RandomString()),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RegisterSucceedsWithDuplicateTaskDefinition": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			outDuplicate, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Command: []*string{aws.String("echo"), aws.String("foo")},
						Image:   aws.String("ubuntu"),
						Name:    aws.String("print_foo"),
					},
				},
				Cpu:         aws.String("128"),
				Memory:      aws.String("4"),
				Family:      aws.String(os.Getenv("AWS_ECS_TASK_DEFINITION_PREFIX") + "bar"),
				TaskRoleArn: aws.String("arn:aws:iam::579766882180:role/dev.task.role"),
			})

			require.NoError(t, err)
			require.NotZero(t, outDuplicate)
			require.NotZero(t, outDuplicate.TaskDefinition)

			defer cleanupTaskDefinition(ctx, t, c, outDuplicate)

			assert.True(t, *outDuplicate.TaskDefinition.Revision > *registerOut.TaskDefinition.Revision)

		},
		"RunAndStopTaskSucceedsWithRegisteredTaskDefinition": func(ctx context.Context, t *testing.T, c *BasicECSClient) {

			require.NotZero(t, registerOut.TaskDefinition.Status)
			assert.Equal(t, "ACTIVE", *registerOut.TaskDefinition.Status)

			runOut, err := c.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(os.Getenv("AWS_ECS_CLUSTER")),
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
			})

			require.NoError(t, err)
			require.NotZero(t, runOut)

			defer func() {
				out, err := c.StopTask(ctx, &ecs.StopTaskInput{
					Cluster: aws.String(os.Getenv("AWS_ECS_CLUSTER")),
					Task:    aws.String(*runOut.Tasks[0].TaskArn),
				})
				require.NoError(t, err)
				require.NotZero(t, out)
				require.NotZero(t, out.Task)
				assert.Equal(t, runOut.Tasks[0].TaskArn, out.Task.TaskArn)
			}()

			require.Empty(t, runOut.Failures)
			require.NotEmpty(t, runOut.Tasks)
			assert.Equal(t, runOut.Tasks[0].TaskDefinitionArn, registerOut.TaskDefinition.TaskDefinitionArn)
		},
		"DescribeTaskSucceedsWithRunningTask": func(ctx context.Context, t *testing.T, c *BasicECSClient) {
			require.NotZero(t, registerOut.TaskDefinition.Status)
			assert.Equal(t, "ACTIVE", *registerOut.TaskDefinition.Status)

			runOut, err := c.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(os.Getenv("AWS_ECS_CLUSTER")),
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
			})

			require.NoError(t, err)
			require.NotZero(t, runOut)
			require.NotEmpty(t, runOut.Tasks)

			defer cleanupTask(ctx, t, c, runOut)

			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(os.Getenv("AWS_ECS_CLUSTER")),
				Tasks:   []*string{aws.String(*runOut.Tasks[0].TaskArn)},
			})

			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotEmpty(t, out.Tasks)
			assert.Equal(t, *out.Tasks[0].TaskDefinitionArn, *registerOut.TaskDefinition.TaskDefinitionArn)
			assert.Equal(t, out.Tasks[0].TaskArn, runOut.Tasks[0].TaskArn)
		},
	} {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			tCase(tctx, t, c)
		})
	}
}

func checkAWSEnvVarsForECS(t *testing.T) {
	checkEnvVars(t,
		"AWS_ACCESS_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_ROLE",
		"AWS_REGION",
		"AWS_ECS_CLUSTER",
	)
}

func checkAWSEnvVarsForECSAndSecretsManager(t *testing.T) {
	checkEnvVars(t,
		"AWS_ACCESS_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_ROLE",
		"AWS_REGION",
		"AWS_ECS_CLUSTER",
		"AWS_SECRET_PREFIX",
		"AWS_ECS_TASK_DEFINITION_PREFIX",
	)
}

func checkEnvVars(t *testing.T, envVars ...string) {
	var missing []string

	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		assert.FailNow(t, fmt.Sprintf("missing required AWS environment variables: %s", missing))
	}
}
