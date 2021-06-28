package mock

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/utility"
)

// ECSCluster represents a mock ECS cluster running tasks in the global ECS
// service.
type ECSCluster map[string]ECSTask

// ECSTask represents a mock ECS task.
type ECSTask struct {
	ARN         *string
	Cluster     *string
	ExecEnabled *bool
	Tags        []string
}

// ECSTaskDefinition represents a mock ECS task definition in the global ECS service.
type ECSTaskDefinition struct {
	ARN           *string
	Family        *string
	Revision      *int64
	ContainerDefs []ECSContainerDefinition
	MemoryMB      *string
	CPU           *string
	TaskRole      *string
	Tags          map[string]string
	Status        *string
	Registered    *time.Time
}

func (d *ECSTaskDefinition) export() *ecs.TaskDefinition {
	var containerDefs []*ecs.ContainerDefinition
	for _, def := range d.ContainerDefs {
		containerDefs = append(containerDefs, def.export())
	}
	return &ecs.TaskDefinition{
		TaskDefinitionArn:    d.ARN,
		Family:               d.Family,
		Revision:             d.Revision,
		Cpu:                  d.CPU,
		Memory:               d.MemoryMB,
		TaskRoleArn:          d.TaskRole,
		Status:               d.Status,
		ContainerDefinitions: containerDefs,
	}
}

// ECSContainerDefinition represents a mock ECS container definition in a mock
// ECS task definition.
type ECSContainerDefinition struct {
	Name     *string
	Image    *string
	Command  []string
	MemoryMB *int64
	CPU      *int64
	EnvVars  map[string]string
	Secrets  map[string]string
	Tags     map[string]string
}

func (d *ECSContainerDefinition) export() *ecs.ContainerDefinition {
	var env []*ecs.KeyValuePair
	for k, v := range d.EnvVars {
		env = append(env, &ecs.KeyValuePair{
			Name:  utility.ToStringPtr(k),
			Value: utility.ToStringPtr(v),
		})
	}
	var tags []*ecs.Tag
	for k, v := range d.Tags {
		tags = append(tags, &ecs.Tag{
			Key:   utility.ToStringPtr(k),
			Value: utility.ToStringPtr(v),
		})
	}
	return &ecs.ContainerDefinition{
		Name:        d.Name,
		Image:       d.Image,
		Command:     utility.ToStringPtrSlice(d.Command),
		Memory:      d.MemoryMB,
		Cpu:         d.CPU,
		Environment: env,
	}
}

// ECSService is a global implementation of ECS that provides a simplified
// in-memory implementation of the service that only stores metadata and does
// not orchestrate real containers or container instances. This can be used
// indirectly with the ECSClient to access or modify ECS resources, or used
// directly.
type ECSService struct {
	// kim: TODO: need mutexes for multi-threaded?
	Clusters map[string]ECSCluster
	TaskDefs map[string][]ECSTaskDefinition
}

// GlobalECSService represents the global fake ECS service state.
var GlobalECSService ECSService

func init() {
	GlobalECSService = ECSService{
		Clusters: map[string]ECSCluster{},
		TaskDefs: map[string][]ECSTaskDefinition{},
	}
}

// ECSClient provides a mock implementation of a cocoa.ECSClient. This makes
// it possible to introspect on inputs to the client and control the client's
// output. It provides some default implementations where possible.
type ECSClient struct {
	RegisterTaskDefinitionInput  *ecs.RegisterTaskDefinitionInput
	RegisterTaskDefinitionOutput *ecs.RegisterTaskDefinitionOutput

	DeregisterTaskDefinitionInput  *ecs.DeregisterTaskDefinitionInput
	DeregisterTaskDefinitionOutput *ecs.DeregisterTaskDefinitionOutput
}

// RegisterTaskDefinition saves the input and returns a new mock task
// definition. The mock output can be customized. By default, it will create a
// cached task definition based on the input.
func (c *ECSClient) RegisterTaskDefinition(ctx context.Context, in *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	c.RegisterTaskDefinitionInput = in

	if c.RegisterTaskDefinitionOutput != nil {
		return c.RegisterTaskDefinitionOutput, nil
	}

	revisions := GlobalECSService.TaskDefs[utility.FromStringPtr(in.Family)]
	rev := len(revisions) + 1

	id := arn.ARN{
		Partition: "aws",
		Service:   "ecs",
		Resource:  fmt.Sprintf("%s:%s", utility.FromStringPtr(in.Family), strconv.Itoa(rev)),
	}

	taskDef := ECSTaskDefinition{
		ARN:        utility.ToStringPtr(id.String()),
		Family:     in.Family,
		Revision:   utility.ToInt64Ptr(int64(rev)),
		CPU:        in.Cpu,
		MemoryMB:   in.Memory,
		TaskRole:   in.TaskRoleArn,
		Tags:       map[string]string{},
		Status:     utility.ToStringPtr(ecs.TaskDefinitionStatusActive),
		Registered: utility.ToTimePtr(time.Now()),
	}

	for _, t := range in.Tags {
		if t == nil {
			continue
		}
		taskDef.Tags[utility.FromStringPtr(t.Key)] = utility.FromStringPtr(t.Value)
	}
	for _, def := range in.ContainerDefinitions {
		containerDef := ECSContainerDefinition{
			Name:     def.Name,
			Image:    def.Image,
			Command:  utility.FromStringPtrSlice(def.Command),
			MemoryMB: def.Memory,
			CPU:      def.Cpu,
			EnvVars:  map[string]string{},
			Secrets:  map[string]string{},
		}
		for _, s := range def.Secrets {
			containerDef.EnvVars[utility.FromStringPtr(s.Name)] = utility.FromStringPtr(s.ValueFrom)
		}
		taskDef.ContainerDefs = append(taskDef.ContainerDefs, containerDef)
	}

	GlobalECSService.TaskDefs[utility.FromStringPtr(in.Family)] = append(revisions, taskDef)

	return &ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: taskDef.export(),
		Tags:           in.Tags,
	}, nil
}

// DeregisterTaskDefinition saves the input and deletes an existing mock task
// definition. The mock output can be customized. By default, it will delete a
// cached task definition if it exists.
func (c *ECSClient) DeregisterTaskDefinition(ctx context.Context, in *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	c.DeregisterTaskDefinitionInput = in

	if c.DeregisterTaskDefinitionOutput != nil {
		return c.DeregisterTaskDefinitionOutput, nil
	}

	id := utility.FromStringPtr(in.TaskDefinition)

	var taskDef ECSTaskDefinition
	if arn.IsARN(id) {
		var found bool
		for family, revisions := range GlobalECSService.TaskDefs {
			for revNum, def := range revisions {
				if utility.FromStringPtr(def.ARN) == id {
					taskDef = def
					taskDef.Status = utility.ToStringPtr(ecs.TaskDefinitionStatusInactive)
					GlobalECSService.TaskDefs[family][revNum] = taskDef
					break
				}
			}
		}
		if !found {
			return nil, errors.New("resource not found")
		}
	} else {
		splitIdx := strings.LastIndex(id, ":")
		if splitIdx == -1 {
			return nil, errors.New("malformed task definition input")
		}

		family := id[:splitIdx]

		revNum, err := strconv.Atoi(id[splitIdx+1:])
		if err != nil {
			return nil, errors.Wrap(err, "parsing revision")
		}

		revisions, ok := GlobalECSService.TaskDefs[family]
		if !ok {
			return nil, errors.New("family not found")
		}
		if len(revisions) < revNum {
			return nil, errors.New("revision not found")
		}

		taskDef = revisions[revNum]
		taskDef.Status = utility.ToStringPtr(ecs.TaskDefinitionStatusInactive)
		GlobalECSService.TaskDefs[family][revNum] = taskDef
	}

	return &ecs.DeregisterTaskDefinitionOutput{
		TaskDefinition: taskDef.export(),
	}, nil
}

// ListTaskDefinitions saves the input and lists all matching task definitions.
// The mock output can be customized. By default, it will list all cached task
// definitions that match the input filters.
func (c *ECSClient) ListTaskDefinitions(ctx context.Context, in *ecs.ListTaskDefinitionsInput) (*ecs.ListTaskDefinitionsOutput, error) {
	var arns []*string
	for _, revisions := range GlobalECSService.TaskDefs {
		for _, def := range revisions {
			if in.FamilyPrefix != nil && utility.FromStringPtr(def.Family) != *in.FamilyPrefix {
				continue
			}
			if in.Status != nil && utility.FromStringPtr(def.Status) != *in.Status {
				continue
			}

			arns = append(arns, def.ARN)
		}
	}
	return &ecs.ListTaskDefinitionsOutput{
		TaskDefinitionArns: arns,
	}, nil
}

// RunTask saves the input options and returns the mock result of running a task
// definition. The mock output can be customized. By default, it will create
// mock output based on the input.
func (c *ECSClient) RunTask(ctx context.Context, in *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

// DescribeTasks saves the input and returns information about the existing
// tasks. The mock output can be customized. By default, it will describe all
// cached tasks that match.
func (c *ECSClient) DescribeTasks(ctx context.Context, in *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	return nil, errors.New("TODO: implement")
}

// ListTasks saves the input and lists all matching tasks. The mock output can
// be customized. By default, it will list all cached task definitions that
// match the input filters.
func (c *ECSClient) ListTasks(ctx context.Context, in *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	return nil, errors.New("TODO: implement")
}

// StopTask saves the input and stops a mock task. The mock output can be
// customized. By default, it will mark a cached task as stopped if it exists
// and is running.
func (c *ECSClient) StopTask(ctx context.Context, in *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	return nil, errors.New("TODO: implement")
}

// Close closes the mock client. The mock output can be customized. By default,
// it is a no-op that returns no error.
func (c *ECSClient) Close(ctx context.Context) error {
	return nil
}
