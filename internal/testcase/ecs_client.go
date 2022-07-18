package testcase

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awsECS "github.com/aws/aws-sdk-go/service/ecs"
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
			out, err := c.RegisterTaskDefinition(ctx, &awsECS.RegisterTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeregisteringExistingTaskDefinitionMultipleTimesIsIdempotent": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))

			for i := 0; i < 3; i++ {
				deregisterOut, err := c.DeregisterTaskDefinition(ctx, &awsECS.DeregisterTaskDefinitionInput{
					TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
				})
				require.NoError(t, err)
				require.NotZero(t, deregisterOut)
				require.NotZero(t, deregisterOut.TaskDefinition)
				require.Equal(t, registerOut.TaskDefinition.TaskDefinitionArn, deregisterOut.TaskDefinition.TaskDefinitionArn)
			}
		},
		"DeregisterTaskDefinitionFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DeregisterTaskDefinition(ctx, &awsECS.DeregisterTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeregisterTaskDefinitionFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DeregisterTaskDefinition(ctx, &awsECS.DeregisterTaskDefinitionInput{
				TaskDefinition: aws.String(testutil.NewTaskDefinitionFamily(t) + ":1"),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RegisterAndDeregisterTaskDefinitionSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
			defer cleanupTaskDefinition(ctx, t, c, &registerOut)
			assert.Equal(t, awsECS.TaskDefinitionStatusActive, *registerOut.TaskDefinition.Status)
			assert.NotZero(t, utility.FromTimePtr(registerOut.TaskDefinition.RegisteredAt))

			deregisterOut, err := c.DeregisterTaskDefinition(ctx, &awsECS.DeregisterTaskDefinitionInput{
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, deregisterOut)
		},
		"RunTaskFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.RunTask(ctx, &awsECS.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: aws.String(testutil.NewTaskDefinitionFamily(t) + ":1"),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"RunTaskFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.RunTask(ctx, &awsECS.RunTaskInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"StopTaskFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.StopTask(ctx, &awsECS.StopTaskInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"StopTaskFailsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.StopTask(ctx, &awsECS.StopTaskInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Task:    aws.String(utility.RandomString()),
			})
			assert.Error(t, err)
			assert.True(t, cocoa.IsECSTaskNotFoundError(err))
			assert.Zero(t, out)
		},
		"DescribeTasksFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTasks(ctx, &awsECS.DescribeTasksInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTaskDefinitionFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTaskDefinition(ctx, &awsECS.DescribeTaskDefinitionInput{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DescribeTaskDefinitionFailsWithNonexistentTaskDefinition": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.DescribeTaskDefinition(ctx, &awsECS.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String(testutil.NewTaskDefinitionFamily(t)),
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"ListTasksSucceedsWithNoResultWithZeroInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.ListTasks(ctx, &awsECS.ListTasksInput{})
			assert.NoError(t, err)
			if out != nil {
				assert.Empty(t, out.TaskArns)
			}
		},
		"ListTasksSucceedsWithNoResultsWithValidButNonexistentInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			out, err := c.ListTasks(ctx, &awsECS.ListTasksInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Family:  aws.String(testutil.NewTaskDefinitionFamily(t)),
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			assert.Empty(t, out.TaskArns)
		},
		"TagResourceSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
			defer cleanupTaskDefinition(ctx, t, c, &registerOut)
			tags := []*awsECS.Tag{
				{
					Key:   aws.String("some_key"),
					Value: aws.String("some_value"),
				},
			}
			_, err := c.TagResource(ctx, &awsECS.TagResourceInput{
				ResourceArn: registerOut.TaskDefinition.TaskDefinitionArn,
				Tags:        tags,
			})
			require.NoError(t, err)

			describeOut, err := c.DescribeTaskDefinition(ctx, &awsECS.DescribeTaskDefinitionInput{
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
				Include:        []*string{aws.String("TAGS")},
			})
			require.NoError(t, err)
			require.NotZero(t, describeOut)
			require.NotZero(t, describeOut.TaskDefinition)
			require.Len(t, describeOut.Tags, 1)
			assert.Equal(t, utility.FromStringPtr(describeOut.Tags[0].Key), utility.FromStringPtr(tags[0].Key))
			assert.Equal(t, utility.FromStringPtr(describeOut.Tags[0].Value), utility.FromStringPtr(tags[0].Value))
		},
		"TagResourceIsIdempotent": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
			defer cleanupTaskDefinition(ctx, t, c, &registerOut)

			tags := []*awsECS.Tag{
				{
					Key:   aws.String("some_key"),
					Value: aws.String("some_value"),
				},
			}
			for i := 0; i < 3; i++ {
				_, err := c.TagResource(ctx, &awsECS.TagResourceInput{
					ResourceArn: registerOut.TaskDefinition.TaskDefinitionArn,
					Tags:        tags,
				})
				require.NoError(t, err)
			}

			describeOut, err := c.DescribeTaskDefinition(ctx, &awsECS.DescribeTaskDefinitionInput{
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
				Include:        []*string{aws.String("TAGS")},
			})
			require.NoError(t, err)
			require.NotZero(t, describeOut)
			require.NotZero(t, describeOut.TaskDefinition)
			require.Len(t, describeOut.Tags, 1)
			assert.Equal(t, utility.FromStringPtr(describeOut.Tags[0].Key), utility.FromStringPtr(tags[0].Key))
			assert.Equal(t, utility.FromStringPtr(describeOut.Tags[0].Value), utility.FromStringPtr(tags[0].Value))
		},
		"TagResourceOverwritesExistingTagValue": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
			defer cleanupTaskDefinition(ctx, t, c, &registerOut)

			oldTags := []*awsECS.Tag{
				{
					Key:   aws.String("mango"),
					Value: aws.String("is second best fruit"),
				},
				{
					Key:   aws.String("fish"),
					Value: aws.String("are friends and food"),
				},
			}

			_, err := c.TagResource(ctx, &awsECS.TagResourceInput{
				ResourceArn: registerOut.TaskDefinition.TaskDefinitionArn,
				Tags:        oldTags,
			})
			require.NoError(t, err)

			newTags := []*awsECS.Tag{
				{
					Key:   aws.String("mango"),
					Value: aws.String("is best fruit"),
				},
			}
			_, err = c.TagResource(ctx, &awsECS.TagResourceInput{
				ResourceArn: registerOut.TaskDefinition.TaskDefinitionArn,
				Tags:        newTags,
			})
			require.NoError(t, err)

			describeOut, err := c.DescribeTaskDefinition(ctx, &awsECS.DescribeTaskDefinitionInput{
				TaskDefinition: registerOut.TaskDefinition.TaskDefinitionArn,
				Include:        []*string{aws.String("TAGS")},
			})
			require.NoError(t, err)
			require.NotZero(t, describeOut)
			require.NotZero(t, describeOut.TaskDefinition)
			require.Len(t, describeOut.Tags, 2)
			for _, tag := range describeOut.Tags {
				k := utility.FromStringPtr(tag.Key)
				switch k {
				case utility.FromStringPtr(oldTags[0].Key):
					assert.Equal(t, utility.FromStringPtr(newTags[0].Value), utility.FromStringPtr(tag.Value), "first tag should have new value")
				case utility.FromStringPtr(oldTags[1].Key):
					assert.Equal(t, utility.FromStringPtr(oldTags[1].Value), utility.FromStringPtr(tag.Value), "second tag should have unmodified value")
				default:
					assert.Fail(t, "unexpected tag key '%s'", k)
				}
			}
		},
		"TagResourceFailsWithZeroInput": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			_, err := c.TagResource(ctx, &awsECS.TagResourceInput{})
			assert.Error(t, err)
		},
		"TagResourceFailsWithNonexistentResource": func(ctx context.Context, t *testing.T, c cocoa.ECSClient) {
			_, err := c.TagResource(ctx, &awsECS.TagResourceInput{ResourceArn: aws.String("foo")})
			assert.Error(t, err)
		},
	}
}

// ECSClientRegisteredTaskDefinitionTestCase represents a test case for a
// cocoa.ECSClient with a task definition already registered.
type ECSClientRegisteredTaskDefinitionTestCase func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def awsECS.TaskDefinition)

// ECSClientRegisteredTaskDefinitionTests returns common test cases that a
// cocoa.ECSClient should support that rely on an already-registered task
// definition.
func ECSClientRegisteredTaskDefinitionTests() map[string]ECSClientRegisteredTaskDefinitionTestCase {
	return map[string]ECSClientRegisteredTaskDefinitionTestCase{
		"RunAndStopTaskSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def awsECS.TaskDefinition) {
			require.NotZero(t, def.Status)
			assert.Equal(t, awsECS.TaskDefinitionStatusActive, utility.FromStringPtr(def.Status))

			runOut, err := c.RunTask(ctx, &awsECS.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, runOut)
			require.Empty(t, runOut.Failures)
			require.NotEmpty(t, runOut.Tasks)
			assert.Equal(t, runOut.Tasks[0].TaskDefinitionArn, def.TaskDefinitionArn)

			out, err := c.StopTask(ctx, &awsECS.StopTaskInput{
				Cluster: aws.String(testutil.ECSClusterName()),
				Task:    aws.String(*runOut.Tasks[0].TaskArn),
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotZero(t, out.Task)
			assert.Equal(t, runOut.Tasks[0].TaskArn, out.Task.TaskArn)
		},
		"DescribeTaskSucceedsWithRunningTask": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def awsECS.TaskDefinition) {
			require.NotZero(t, def.Status)
			assert.Equal(t, awsECS.TaskDefinitionStatusActive, *def.Status)

			runOut, err := c.RunTask(ctx, &awsECS.RunTaskInput{
				Cluster:        aws.String(testutil.ECSClusterName()),
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, runOut)
			require.NotEmpty(t, runOut.Tasks)

			defer cleanupTask(ctx, t, c, runOut)

			out, err := c.DescribeTasks(ctx, &awsECS.DescribeTasksInput{
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
		"RegisterSucceedsWithDuplicateTaskDefinitionFamily": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def awsECS.TaskDefinition) {
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
		"DescribeTaskDefinitionSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def awsECS.TaskDefinition) {
			out, err := c.DescribeTaskDefinition(ctx, &awsECS.DescribeTaskDefinitionInput{
				TaskDefinition: def.TaskDefinitionArn,
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			require.NotZero(t, out.TaskDefinition)
			assert.Equal(t, def, *out.TaskDefinition)
		},
		"ListTasksSucceeds": func(ctx context.Context, t *testing.T, c cocoa.ECSClient, def awsECS.TaskDefinition) {
			runOut, err := c.RunTask(ctx, &awsECS.RunTaskInput{
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

			out, err := c.ListTasks(ctx, &awsECS.ListTasksInput{
				Cluster:       aws.String(testutil.ECSClusterName()),
				DesiredStatus: aws.String(awsECS.DesiredStatusRunning),
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
func cleanupTaskDefinition(ctx context.Context, t *testing.T, c cocoa.ECSClient, out *awsECS.RegisterTaskDefinitionOutput) {
	if out != nil && out.TaskDefinition != nil && out.TaskDefinition.TaskDefinitionArn != nil {
		out, err := c.DeregisterTaskDefinition(ctx, &awsECS.DeregisterTaskDefinitionInput{
			TaskDefinition: out.TaskDefinition.TaskDefinitionArn,
		})
		require.NoError(t, err)
		require.NotZero(t, out)
	}
}

// cleanupTask cleans up a running task.
func cleanupTask(ctx context.Context, t *testing.T, c cocoa.ECSClient, runOut *awsECS.RunTaskOutput) {
	if runOut != nil && len(runOut.Tasks) > 0 && runOut.Tasks[0].TaskArn != nil {
		out, err := c.StopTask(ctx, &awsECS.StopTaskInput{
			Cluster: aws.String(testutil.ECSClusterName()),
			Task:    aws.String(*runOut.Tasks[0].TaskArn),
		})
		require.NoError(t, err)
		require.NotZero(t, out)
	}
}
