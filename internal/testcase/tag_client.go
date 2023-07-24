package testcase

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsManagerTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TagClientTestCase represents a test case for a cocoa.TagClient.
type TagClientTestCase func(ctx context.Context, t *testing.T, c cocoa.TagClient)

// TagClientTests returns common test cases that a cocoa.TagClient should
// support.
func TagClientTests() map[string]TagClientTestCase {
	return map[string]TagClientTestCase{
		"GetResourcesFailsWithInvalidInput": func(ctx context.Context, t *testing.T, c cocoa.TagClient) {
			out, err := c.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				TagFilters: []types.TagFilter{
					{
						Values: []string{""},
					},
				},
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"GetResourcesFailsWithInvalidResourceType": func(ctx context.Context, t *testing.T, c cocoa.TagClient) {
			out, err := c.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"nonexistent"},
			})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"GetResourcesSucceedsWithNoResults": func(ctx context.Context, t *testing.T, c cocoa.TagClient) {
			out, err := c.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager"},
				TagFilters: []types.TagFilter{
					{
						Key:    aws.String("nonexistent"),
						Values: []string{"nonexistent"},
					},
				},
			})
			require.NoError(t, err)
			require.NotZero(t, out)
			assert.Empty(t, out.ResourceTagMappingList)
		},
	}
}

// TagClientSecretTestCase represents a test case for a cocoa.TagClient with a
// cocoa.SecretsManagerClient.
type TagClientSecretTestCase func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient)

// TagClientSecretTests returns common test cases that rely on Secrets Manager
// that a cocoa.TagClient should support.
func TagClientSecretTests() map[string]TagClientSecretTestCase {
	checkResources := func(t *testing.T, out resourcegroupstaggingapi.GetResourcesOutput, expected []string) {
		require.Len(t, out.ResourceTagMappingList, len(expected), "number of results should match expected")
		for _, res := range out.ResourceTagMappingList {
			arn := utility.FromStringPtr(res.ResourceARN)
			assert.True(t, utility.StringSliceContains(expected, arn), "unexpected resource '%s' in results", arn)
		}
	}
	return map[string]TagClientSecretTestCase{
		"GetResourcesMatchesSingleTagKeyAndValueForSingleResource": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{utility.FromStringPtr(inputTags[0].Value)},
					},
				},
			})
			require.NoError(t, err)

			checkResources(t, *getResourcesOut, []string{utility.FromStringPtr(createSecretOut.ARN)})
		},
		"GetResourcesMatchesSingleKeyAndValueForMultipleResources": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			var arns []string
			for i := 0; i < 3; i++ {
				createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
					Name:         aws.String(testutil.NewSecretName(t)),
					SecretString: aws.String(utility.RandomString()),
					Tags:         inputTags,
				})
				defer cleanupSecret(ctx, t, smClient, &createSecretOut)
				arns = append(arns, utility.FromStringPtr(createSecretOut.ARN))
			}

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{utility.FromStringPtr(inputTags[0].Value)},
					},
				},
			})
			require.NoError(t, err)

			checkResources(t, *getResourcesOut, arns)
		},
		"GetResourcesMatchesSingleTagKeyAndOneOfMultipleValues": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{"foo", "bar", utility.FromStringPtr(inputTags[0].Value), "baz"},
					},
				},
			})
			require.NoError(t, err)

			checkResources(t, *getResourcesOut, []string{utility.FromStringPtr(createSecretOut.ARN)})
		},
		"GetResourcesMatchesMultipleTagKeys": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key: inputTags[0].Key,
					},
					{
						Key: inputTags[1].Key,
					},
				},
			})
			require.NoError(t, err)

			checkResources(t, *getResourcesOut, []string{utility.FromStringPtr(createSecretOut.ARN)})
		},
		"GetResourcesMatchesMultipleTagKeysAndValues": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{"foo", utility.FromStringPtr(inputTags[0].Value), "baz"},
					},
					{
						Key:    inputTags[1].Key,
						Values: []string{"qux", utility.FromStringPtr(inputTags[1].Value), "quux"},
					},
				},
			})
			require.NoError(t, err)

			checkResources(t, *getResourcesOut, []string{utility.FromStringPtr(createSecretOut.ARN)})
		},
		"GetResourcesReturnsNoResultsForUnmatchedResourceType": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"ecs:task-definition"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{utility.FromStringPtr(inputTags[0].Value)},
					},
				},
			})
			require.NoError(t, err)
			require.NotZero(t, getResourcesOut)
			assert.Empty(t, getResourcesOut.ResourceTagMappingList)
		},
		"GetResourcesOmitsResultForAnyUnmatchedTagKey": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{utility.FromStringPtr(inputTags[0].Value)},
					},
					{
						Key: aws.String("nonexistent"),
					},
				},
			})
			require.NoError(t, err)
			require.NotZero(t, getResourcesOut)
			assert.Empty(t, getResourcesOut.ResourceTagMappingList)
		},
		"GetResourcesOmitsResultsForAnyUnmatchedTagValues": func(ctx context.Context, t *testing.T, tagClient cocoa.TagClient, smClient cocoa.SecretsManagerClient) {
			inputTags := []secretsManagerTypes.Tag{
				{
					Key:   aws.String(utility.RandomString()),
					Value: aws.String(utility.RandomString()),
				},
			}
			createSecretOut := testutil.CreateSecret(ctx, t, smClient, secretsmanager.CreateSecretInput{
				Name:         aws.String(testutil.NewSecretName(t)),
				SecretString: aws.String(utility.RandomString()),
				Tags:         inputTags,
			})
			defer cleanupSecret(ctx, t, smClient, &createSecretOut)

			getResourcesOut, err := tagClient.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
				ResourceTypeFilters: []string{"secretsmanager:secret"},
				TagFilters: []types.TagFilter{
					{
						Key:    inputTags[0].Key,
						Values: []string{utility.FromStringPtr(inputTags[0].Value)},
					},
					{
						Key:    inputTags[0].Key,
						Values: []string{"nonexistent"},
					},
				},
			})
			require.NoError(t, err)
			require.NotZero(t, getResourcesOut)
			assert.Empty(t, getResourcesOut.ResourceTagMappingList)
		},
	}
}
