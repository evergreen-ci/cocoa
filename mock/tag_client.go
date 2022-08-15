package mock

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	awsECS "github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/evergreen-ci/utility"
)

// taggedResource represents an arbitrary AWS resource with its tags.
type taggedResource struct {
	ID   string
	Tags map[string]string
}

func exportTagMapping(res taggedResource) *resourcegroupstaggingapi.ResourceTagMapping {
	return &resourcegroupstaggingapi.ResourceTagMapping{
		ResourceARN: utility.ToStringPtr(res.ID),
		Tags:        exportResourceTags(res.Tags),
	}
}

func exportResourceTags(tags map[string]string) []*resourcegroupstaggingapi.Tag {
	var exported []*resourcegroupstaggingapi.Tag
	for k, v := range tags {
		exported = append(exported, &resourcegroupstaggingapi.Tag{
			Key:   utility.ToStringPtr(k),
			Value: utility.ToStringPtr(v),
		})
	}
	return exported
}

// TagClient provides a mock implementation of a cocoa.TagClient. This makes
// it possible to introspect on inputs to the client and control the client's
// output. It provides some default implementations where possible. By default,
// it will issue the API calls to either the fake GlobalECSService for ECS or
// fake GlobalSecretCache for Secrets Manager.
type TagClient struct {
	GetResourcesInput  *resourcegroupstaggingapi.GetResourcesInput
	GetResourcesOutput *resourcegroupstaggingapi.GetResourcesOutput
	GetResourcesError  error

	CloseError error
}

// RegisterTaskDefinition saves the input and filters for the resources matching
// the input filters. The mock output can be customized. By default, it will
// search for matching resources in ECS or Secrets Manager.
func (c *TagClient) GetResources(ctx context.Context, in *resourcegroupstaggingapi.GetResourcesInput) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	c.GetResourcesInput = in

	if c.GetResourcesOutput != nil || c.GetResourcesError != nil {
		return c.GetResourcesOutput, c.GetResourcesError
	}

	serviceToResourceType := map[string]string{
		"secretsmanager": "secretsmanager:secret",
		"ecs":            "ecs:task-definition",
	}

	// kim; TODO: logical AND across all tag filter, logical OR each individual
	// tag filter key-values.
	resourceTypes := map[string][]taggedResource{}
	if len(in.ResourceTypeFilters) != 0 {
		for _, f := range in.ResourceTypeFilters {
			if f == nil {
				continue
			}

			filter := utility.FromStringPtr(f)
			var resourceType string
			if !strings.Contains(filter, ":") {
				var ok bool
				resourceType, ok = serviceToResourceType[filter]
				if !ok {
					return nil, awserr.New(resourcegroupstaggingapi.ErrCodeInvalidParameterException, "unsupported service", nil)
				}
			} else {
				resourceType = filter
			}

			resourceTypes[resourceType] = []taggedResource{}
		}
	} else {
		// If no resource types are filtered, search all of them.
		for _, resourceType := range serviceToResourceType {
			resourceTypes[resourceType] = []taggedResource{}
		}
	}

	for resourceType := range resourceTypes {
		var matchingAllTags map[string]taggedResource

		switch resourceType {
		case "secretsmanager:secret":
			if len(in.TagFilters) != 0 {
				for _, f := range in.TagFilters {
					if f == nil {
						continue
					}

					matchingTag := c.secretsMatchingTag(utility.FromStringPtr(f.Key), utility.FromStringPtrSlice(f.Values))

					if matchingAllTags == nil {
						// Initialize the candidate set of matching secrets.
						matchingAllTags = matchingTag
					} else {
						// Each matching secret must match all the given tag
						// filters.
						matchingAllTags = c.getSetIntersection(matchingAllTags, matchingTag)
					}
				}
			} else {
				matchingAllTags = map[string]taggedResource{}
				for _, s := range GlobalSecretCache {
					matchingAllTags[s.Name] = c.exportSecretTaggedResource(s)
				}
			}
		case "ecs:task-definition":
			if len(in.TagFilters) != 0 {
				for _, f := range in.TagFilters {
					if f == nil {
						continue
					}

					matchingTag := c.taskDefsMatchingTag(utility.FromStringPtr(f.Key), utility.FromStringPtrSlice(f.Values))

					if matchingAllTags == nil {
						matchingAllTags = matchingTag
					} else {
						matchingAllTags = c.getSetIntersection(matchingAllTags, matchingTag)
					}
				}
			} else {
				matchingAllTags = map[string]taggedResource{}
				for _, family := range GlobalECSService.TaskDefs {
					for _, revision := range family {
						matchingAllTags[revision.ARN] = c.exportTaskDefinitionTaggedResource(revision)
					}
				}
			}
		}

		for _, res := range matchingAllTags {
			resourceTypes[resourceType] = append(resourceTypes[resourceType], res)
		}
	}

	var converted []*resourcegroupstaggingapi.ResourceTagMapping
	for _, res := range resourceTypes {
		for _, tags := range res {
			converted = append(converted, exportTagMapping(tags))
		}
	}

	return &resourcegroupstaggingapi.GetResourcesOutput{
		ResourceTagMappingList: converted,
	}, nil
}

func (c *TagClient) getSetIntersection(a, b map[string]taggedResource) map[string]taggedResource {
	intersection := map[string]taggedResource{}
	for k, v := range a {
		if _, ok := b[k]; ok {
			intersection[k] = v
		}
	}
	return intersection
}

// secretsMatchingTag returns the tagged resources for all secrets containing a
// matching tag key and matching one of the tag values.
func (c *TagClient) secretsMatchingTag(key string, values []string) map[string]taggedResource {
	res := map[string]taggedResource{}
	for _, s := range GlobalSecretCache {
		if s.IsDeleted {
			continue
		}

		v, ok := s.Tags[key]
		if !ok {
			continue
		}

		if len(values) != 0 && !utility.StringSliceContains(values, v) {
			continue
		}

		res[s.Name] = c.exportSecretTaggedResource(s)
	}
	return res
}

func (c *TagClient) exportSecretTaggedResource(s StoredSecret) taggedResource {
	return taggedResource{
		ID:   s.Name,
		Tags: s.Tags,
	}
}

// taskDefsMatchingTag returns the tagged resources for all secrets containing a
// matching tag key and matching one of the tag values.
func (c *TagClient) taskDefsMatchingTag(key string, values []string) map[string]taggedResource {
	res := map[string]taggedResource{}
	for _, family := range GlobalECSService.TaskDefs {
		for _, def := range family {
			if utility.FromStringPtr(def.Status) == awsECS.TaskDefinitionStatusInactive {
				continue
			}

			v, ok := def.Tags[key]
			if !ok {
				continue
			}

			if len(values) != 0 && !utility.StringSliceContains(values, v) {
				continue
			}

			res[def.ARN] = c.exportTaskDefinitionTaggedResource(def)
		}
	}
	return res
}

func (c *TagClient) exportTaskDefinitionTaggedResource(def ECSTaskDefinition) taggedResource {
	return taggedResource{
		ID:   def.ARN,
		Tags: def.Tags,
	}
}

// Close closes the mock client. The mock output can be customized. By default,
// it is a no-op that returns no error.
func (c *TagClient) Close(ctx context.Context) error {
	if c.CloseError != nil {
		return c.CloseError
	}

	return nil
}
