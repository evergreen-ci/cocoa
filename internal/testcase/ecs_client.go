package testcase

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ECSClientTestCase represents a test case for a cocoa.ECSClient.
type ECSClientTestCase func(ctx context.Context, t *testing.T, c cocoa.ECSClient)

// ECSClientTests returns common test cases that a cocoa.ECSClient should
// support.
func ECSClientTests() map[string]ECSClientTestCase {
	return map[string]ECSClientTestCase{
		"RegisterTaskDefinitionFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeregisteringExistingTaskDefinitionMultipleTimesIsIdempotent": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Command: []*string{aws.String("echo"), aws.String("hello")},
						Image:   aws.String("busybox"),
						Name:    aws.String("hello_world"),
					},
				},
				Cpu:    aws.String("128"),
				Memory: aws.String("4"),
				Family: aws.String(testutil.NewTaskDefinitionFamily(t)),
			})
			require.NoError(t, err)
			require.NotNil(t, registerOut)
			require.NotNil(t, registerOut.TaskDefinition)
			require.NotNil(t, registerOut.TaskDefinition.TaskDefinitionArn)

			for i := 0; i < 3; i++ {
				deregisterOut, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
					TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
				})
				require.NoError(t, err)
				require.NotZero(t, deregisterOut)
				require.NotZero(t, deregisterOut.TaskDefinition)
				require.Equal(t, registerOut.TaskDefinition.TaskDefinitionArn, deregisterOut.TaskDefinition.TaskDefinitionArn)
			}
		},
		"DeregisterTaskDefinitionFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeregisterTaskDefinitionFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: aws.String(testutil.NewTaskDefinitionFamily(t) + ":1"),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RegisterAndDeregisterTaskDefinitionSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut, err := c.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Command: []*string{aws.String("echo"), aws.String("hello")},
						Image:   aws.String("busybox"),
						Name:    aws.String("hello_world"),
					},
				},
				Cpu:    aws.String("128"),
				Memory: aws.String("4"),
				Family: aws.String(testutil.NewTaskDefinitionFamily(t)),
			})
			require.NoError(t, err)
			require.NotNil(t, registerOut)
			require.NotNil(t, registerOut.TaskDefinition)
			require.NotNil(t, registerOut.TaskDefinition.TaskDefinitionArn)
			require.NotZero(t, registerOut.TaskDefinition.Status)
			assert.Equal(t, ecs.TaskDefinitionStatusActive, *registerOut.TaskDefinition.Status)
			require.NotZero(t, registerOut.TaskDefinition.RegisteredAt)
			assert.NotZero(t, *registerOut.TaskDefinition.RegisteredAt)

			out, err := c.DeregisterTaskDefinition(ctx, &ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, out)
		},
		"RunTaskFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: aws.String(testutil.NewTaskDefinitionFamily(t) + ":1"),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RunTaskFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.RunTask(ctx, &ecs.RunTaskInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"StopTaskFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.StopTask(ctx, &ecs.StopTaskInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"StopTaskFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.StopTask(ctx, &ecs.StopTaskInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Task:    aws.String(utility.RandomString()),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTasksFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTasksFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Tasks:   []*string{aws.String(utility.RandomString())},
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			assert.NotZero(t, out.Failures)
			assert.Empty(t, out.Tasks)
		},
		"DescribeTaskDefinitionFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTaskDefinitionFailsWithNonexistentTaskDefinition": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String(testutil.NewTaskDefinitionFamily(t)),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"ListTasksFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.ListTasks(ctx, &ecs.ListTasksInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"ListTasksSucceedsWithNoResultsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.ListTasks(ctx, &ecs.ListTasksInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Family:  aws.String(testutil.NewTaskDefinitionFamily(t)),
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			assert.Empty(t, out.TaskArns)
		},
	}
}

// ECSClientRegisteredTaskDefinitionTestCase represents a test case for a
// cocoa.ECSClient with a task definition already registered.
type ECSClientRegisteredTaskDefinitionTestCase func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def ecs.TaskDefinition)

// ECSClientRegisteredTaskDefinitionTests returns common test cases that a
// cocoa.ECSClient should support that rely on an already-registered task
// definition.
func ECSClientRegisteredTaskDefinitionTests() map[string]ECSClientRegisteredTaskDefinitionTestCase {
	return map[string]ECSClientRegisteredTaskDefinitionTestCase{
		"RunAndStopTaskSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def ecs.TaskDefinition) {
			require.NotZero(t, def.Status)
			assert.Equal(t, ecs.TaskDefinitionStatusActive, utility.FromStringPtr(def.Status))

			runOut, err := c.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, runOut)
			require.Empty(t, runOut.Failures)
			require.NotEmpty(t, runOut.Tasks)
			assert.Equal(t, runOut.Tasks[0].TaskDefinitionArn, def.TaskDefinitionArn)

			out, err := c.StopTask(ctx, &ecs.StopTaskInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Task:    aws.String(*runOut.Tasks[0].TaskArn),
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotZero(t, out.Task)
			assert.Equal(t, runOut.Tasks[0].TaskArn, out.Task.TaskArn)
		},
		"DescribeTaskSucceedsWithRunningTask": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def ecs.TaskDefinition) {
			require.NotZero(t, def.Status)
			assert.Equal(t, ecs.TaskDefinitionStatusActive, *def.Status)

			runOut, err := c.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, runOut)
			require.NotEmpty(t, runOut.Tasks)

			defer cleanupTask(ctx, t, c, runOut)

			out, err := c.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Tasks:   []*string{aws.String(*runOut.Tasks[0].TaskArn)},
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotEmpty(t, out.Tasks)
			assert.Len(t, out.Tasks, 1)
			assert.NotZero(t, out.Tasks[0].TaskDefinitionArn)
			assert.Equal(t, utility.FromStringPtr(out.Tasks[0].TaskDefinitionArn), utility.FromStringPtr(def.TaskDefinitionArn))
			require.NotZero(t, out.Tasks[0].TaskArn)
			require.NotZero(t, runOut.Tasks[0].TaskArn)
			assert.Equal(t, out.Tasks[0].TaskArn, runOut.Tasks[0].TaskArn)
		},
		"RegisterSucceedsWithDuplicateTaskDefinitionFamily": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def ecs.TaskDefinition) {
			duplicateTaskDef := testutil.ValidRegisterTaskDefinitionInput(t)
			duplicateTaskDef.Family = def.Family

			outDuplicate, err := c.RegisterTaskDefinition(ctx, &duplicateTaskDef)
			require.NoError(t, err)
			require.NotZero(t, outDuplicate)
			require.NotZero(t, outDuplicate.TaskDefinition)

			defer cleanupTaskDefinition(ctx, t, c, outDuplicate)

			assert.Equal(t, utility.FromStringPtr(def.Family), utility.FromStringPtr(outDuplicate.TaskDefinition.Family))
			assert.True(t, utility.FromInt64Ptr(outDuplicate.TaskDefinition.Revision) > utility.FromInt64Ptr(def.Revision), "registering a task definition in the same family as another task definition should create a new, separate revision")
		},
		"DescribeTaskDefinitionSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def ecs.TaskDefinition) {
			out, err := c.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotZero(t, out.TaskDefinition)
			assert.Equal(t, def, *out.TaskDefinition)
		},
		"ListTasksSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def ecs.TaskDefinition) {
			runOut, err := c.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, runOut)
			require.Empty(t, runOut.Failures)
			require.NotEmpty(t, runOut.Tasks)
			assert.Equal(t, runOut.Tasks[0].TaskDefinitionArn, def.TaskDefinitionArn)
			taskARN := utility.FromStringPtr(runOut.Tasks[0].TaskArn)
			assert.NotZero(t, taskARN)

			out, err := c.ListTasks(ctx, &ecs.ListTasksInput{
				Cluster:       aws.String(testutil.ECSClusterName()),
				DesiredStatus: aws.String(ecs.DesiredStatusRunning),
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			assert.NotEmpty(t, out.TaskArns)
			var taskARNFound bool
			for _, arn := range out.TaskArns {
				if taskARN == utility.FromStringPtr(arn) {
					taskARNFound = true
					break
				}
			}
			assert.True(t, taskARNFound, "task that was just requested to run should appear in results for tasks trying to run")
		},
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

// cleanupTask cleans up a running task.
func cleanupTask(ctx context.Context, t *testing.T, c cocoa.ECSClient, runOut *ecs.RunTaskOutput) {
	if runOut != nil && len(runOut.Tasks) > 0 && runOut.Tasks[0].TaskArn != nil {
		out, err := c.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: aws.String(testutil.ECSClusterName()),
			Task:    aws.String(*runOut.Tasks[0].TaskArn),
		})
		require.NoError(t, err)
		require.NotZero(t, out)
	}
}
