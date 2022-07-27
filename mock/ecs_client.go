package mock

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsECS "github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/ecs"
	"github.com/evergreen-ci/utility"
)

// ECSTaskDefinition represents a mock ECS task definition in the global ECS service.
type ECSTaskDefinition struct {
	ARN           *string
	Family        *string
	Revision      *int64
	ContainerDefs []ECSContainerDefinition
	MemoryMB      *string
	CPU           *string
	TaskRole      *string
	ExecutionRole *string
	Tags          map[string]string
	Status        *string
	Registered    *time.Time
	Deregistered  *time.Time
}

func newECSTaskDefinition(def *awsECS.RegisterTaskDefinitionInput, rev int) ECSTaskDefinition {
	id := arn.ARN{
		Partition: "aws",
		Service:   "ecs",
		Resource:  fmt.Sprintf("task-definition:%s/%s", utility.FromStringPtr(def.Family), strconv.Itoa(rev)),
	}

	taskDef := ECSTaskDefinition{
		ARN:           utility.ToStringPtr(id.String()),
		Family:        def.Family,
		Revision:      utility.ToInt64Ptr(int64(rev)),
		CPU:           def.Cpu,
		MemoryMB:      def.Memory,
		TaskRole:      def.TaskRoleArn,
		ExecutionRole: def.ExecutionRoleArn,
		Status:        utility.ToStringPtr(awsECS.TaskDefinitionStatusActive),
		Registered:    utility.ToTimePtr(time.Now()),
	}

	taskDef.Tags = newECSTags(def.Tags)

	for _, containerDef := range def.ContainerDefinitions {
		if containerDef == nil {
			continue
		}
		taskDef.ContainerDefs = append(taskDef.ContainerDefs, newECSContainerDefinition(containerDef))
	}

	return taskDef
}

func (d *ECSTaskDefinition) export() *awsECS.TaskDefinition {
	var containerDefs []*awsECS.ContainerDefinition
	for _, def := range d.ContainerDefs {
		containerDefs = append(containerDefs, def.export())
	}

	return &awsECS.TaskDefinition{
		TaskDefinitionArn:    d.ARN,
		Family:               d.Family,
		Revision:             d.Revision,
		Cpu:                  d.CPU,
		Memory:               d.MemoryMB,
		TaskRoleArn:          d.TaskRole,
		ExecutionRoleArn:     d.ExecutionRole,
		Status:               d.Status,
		ContainerDefinitions: containerDefs,
		RegisteredAt:         d.Registered,
		DeregisteredAt:       d.Deregistered,
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
}

func newECSContainerDefinition(def *awsECS.ContainerDefinition) ECSContainerDefinition {
	return ECSContainerDefinition{
		Name:     def.Name,
		Image:    def.Image,
		Command:  utility.FromStringPtrSlice(def.Command),
		MemoryMB: def.Memory,
		CPU:      def.Cpu,
		EnvVars:  newEnvVars(def.Environment),
		Secrets:  newSecrets(def.Secrets),
	}
}

func (d *ECSContainerDefinition) export() *awsECS.ContainerDefinition {
	return &awsECS.ContainerDefinition{
		Name:        d.Name,
		Image:       d.Image,
		Command:     utility.ToStringPtrSlice(d.Command),
		Memory:      d.MemoryMB,
		Cpu:         d.CPU,
		Environment: exportEnvVars(d.EnvVars),
		Secrets:     exportSecrets(d.Secrets),
	}
}

// ECSCluster represents a mock ECS cluster running tasks in the global ECS
// service.
type ECSCluster map[string]ECSTask

// ECSTask represents a mock running ECS task within a cluster.
type ECSTask struct {
	ARN               *string
	TaskDef           ECSTaskDefinition
	Cluster           *string
	CapacityProvider  *string
	ContainerInstance *string
	Containers        []ECSContainer
	Group             *string
	ExecEnabled       *bool
	Status            *string
	GoalStatus        *string
	Created           *time.Time
	StopCode          *string
	StopReason        *string
	Stopped           *time.Time
	Tags              map[string]string
}

func newECSTask(in *awsECS.RunTaskInput, taskDef ECSTaskDefinition) ECSTask {
	id := arn.ARN{
		Partition: "aws",
		Service:   "ecs",
		Resource:  fmt.Sprintf("task:%s/%s", utility.FromStringPtr(taskDef.Family), strconv.Itoa(int(utility.FromInt64Ptr(taskDef.Revision)))),
	}

	t := ECSTask{
		ARN:              utility.ToStringPtr(id.String()),
		Cluster:          in.Cluster,
		CapacityProvider: newCapacityProvider(in.CapacityProviderStrategy),
		ExecEnabled:      in.EnableExecuteCommand,
		Group:            in.Group,
		Status:           utility.ToStringPtr(awsECS.DesiredStatusPending),
		GoalStatus:       utility.ToStringPtr(awsECS.DesiredStatusRunning),
		Created:          utility.ToTimePtr(time.Now()),
		TaskDef:          taskDef,
		Tags:             newECSTags(in.Tags),
	}

	for _, containerDef := range taskDef.ContainerDefs {
		t.Containers = append(t.Containers, newECSContainer(containerDef, t))
	}

	return t
}

func (t *ECSTask) export(includeTags bool) *awsECS.Task {
	exported := awsECS.Task{
		TaskArn:              t.ARN,
		ClusterArn:           t.Cluster,
		CapacityProviderName: t.CapacityProvider,
		EnableExecuteCommand: t.ExecEnabled,
		Group:                t.Group,
		TaskDefinitionArn:    t.TaskDef.ARN,
		Cpu:                  t.TaskDef.CPU,
		Memory:               t.TaskDef.MemoryMB,
		LastStatus:           t.Status,
		DesiredStatus:        t.GoalStatus,
		CreatedAt:            t.Created,
		StopCode:             t.StopCode,
		StoppedReason:        t.StopReason,
		StoppedAt:            t.Stopped,
	}
	if includeTags {
		exported.Tags = ecs.ExportTags(t.Tags)
	}

	for _, container := range t.Containers {
		exported.Containers = append(exported.Containers, container.export())
	}

	return &exported
}

// ECSContainer represents a mock running ECS container within a task.
type ECSContainer struct {
	ARN        *string
	TaskARN    *string
	Name       *string
	Image      *string
	CPU        *int64
	MemoryMB   *int64
	Status     *string
	GoalStatus *string
}

func newECSContainer(def ECSContainerDefinition, task ECSTask) ECSContainer {
	name := utility.FromStringPtr(def.Name)
	if name == "" {
		name = utility.RandomString()
	}
	id := arn.ARN{
		Partition: "aws",
		Service:   "ecs",
		Resource:  fmt.Sprintf("container-definition:%s-%s/%s", utility.FromStringPtr(task.TaskDef.Family), name, strconv.Itoa(int(utility.FromInt64Ptr(task.TaskDef.Revision)))),
	}

	return ECSContainer{
		ARN:        utility.ToStringPtr(id.String()),
		TaskARN:    task.ARN,
		Name:       def.Name,
		Image:      def.Image,
		CPU:        def.CPU,
		MemoryMB:   def.MemoryMB,
		Status:     utility.ToStringPtr(awsECS.DesiredStatusPending),
		GoalStatus: utility.ToStringPtr(awsECS.DesiredStatusRunning),
	}
}

func (c *ECSContainer) export() *awsECS.Container {
	exported := &awsECS.Container{
		ContainerArn: c.ARN,
		TaskArn:      c.TaskARN,
		Name:         c.Name,
		Image:        c.Image,
		LastStatus:   c.Status,
	}

	if c.CPU != nil {
		exported.Cpu = utility.ToStringPtr(strconv.Itoa(int(utility.FromInt64Ptr(c.CPU))))
	}
	if c.MemoryMB != nil {
		exported.Memory = utility.ToStringPtr(strconv.Itoa(int(utility.FromInt64Ptr(c.MemoryMB))))
	}

	return exported
}

func newECSTags(tags []*awsECS.Tag) map[string]string {
	converted := map[string]string{}
	for _, t := range tags {
		if t == nil {
			continue
		}
		converted[utility.FromStringPtr(t.Key)] = utility.FromStringPtr(t.Value)
	}
	return converted
}

func newCapacityProvider(providers []*awsECS.CapacityProviderStrategyItem) *string {
	if len(providers) == 0 {
		return nil
	}
	// This is just a fake ECS, so it's okay to arbitrarily choose the first
	// capacity provider for convenience.
	return providers[0].CapacityProvider
}

func newEnvVars(envVars []*awsECS.KeyValuePair) map[string]string {
	converted := map[string]string{}
	for _, envVar := range envVars {
		if envVar == nil {
			continue
		}
		converted[utility.FromStringPtr(envVar.Name)] = utility.FromStringPtr(envVar.Value)
	}
	return converted
}

func exportEnvVars(envVars map[string]string) []*awsECS.KeyValuePair {
	var exported []*awsECS.KeyValuePair
	for k, v := range envVars {
		exported = append(exported, &awsECS.KeyValuePair{
			Name:  utility.ToStringPtr(k),
			Value: utility.ToStringPtr(v),
		})
	}
	return exported
}

func newSecrets(secrets []*awsECS.Secret) map[string]string {
	converted := map[string]string{}
	for _, secret := range secrets {
		if secret == nil {
			continue
		}
		converted[utility.FromStringPtr(secret.Name)] = utility.FromStringPtr(secret.ValueFrom)
	}
	return converted
}

func exportSecrets(secrets map[string]string) []*awsECS.Secret {
	var exported []*awsECS.Secret
	for k, v := range secrets {
		exported = append(exported, &awsECS.Secret{
			Name:      utility.ToStringPtr(k),
			ValueFrom: utility.ToStringPtr(v),
		})
	}
	return exported
}

// ECSService is a global implementation of ECS that provides a simplified
// in-memory implementation of the service that only stores metadata and does
// not orchestrate real containers or container instances. This can be used
// indirectly with the ECSClient to access or modify ECS resources, or used
// directly.
type ECSService struct {
	Clusters map[string]ECSCluster
	TaskDefs map[string][]ECSTaskDefinition
}

// GlobalECSService represents the global fake ECS service state.
var GlobalECSService ECSService

func init() {
	ResetGlobalECSService()
}

// ResetGlobalECSService resets the global fake ECS service back to an
// initialized but clean state.
func ResetGlobalECSService() {
	GlobalECSService = ECSService{
		Clusters: map[string]ECSCluster{},
		TaskDefs: map[string][]ECSTaskDefinition{},
	}
}

// getLatestTaskDefinition is the same as getTaskDefinition, but it can also
// interpret the identifier as just a family name if it's neither an ARN or a
// family and revision. If it matches a family name, the latest active revision
// is returned.
func (s *ECSService) getLatestTaskDefinition(id string) (*ECSTaskDefinition, error) {
	if def, err := s.getTaskDefinition(id); err == nil {
		return def, nil
	}

	// Use the latest active revision in the family if no revision is given.
	family := id
	revisions, ok := GlobalECSService.TaskDefs[family]
	if !ok {
		return nil, errors.New("task definition family not found")
	}

	for i := len(revisions) - 1; i >= 0; i-- {
		if utility.FromStringPtr(revisions[i].Status) == awsECS.TaskDefinitionStatusActive {
			return &revisions[i], nil
		}
	}

	return nil, errors.New("task definition family has no active revisions")
}

// getTaskDefinition gets a task definition by the identifier. The identifier is
// either the task definition's ARN or its family and revision.
func (s *ECSService) getTaskDefinition(id string) (*ECSTaskDefinition, error) {
	if arn.IsARN(id) {
		family, revNum, found := s.taskDefIndexFromARN(id)
		if !found {
			return nil, errors.New("task definition not found")
		}
		return &GlobalECSService.TaskDefs[family][revNum-1], nil
	}

	family, revNum, err := parseFamilyAndRevision(id)
	if err == nil {
		revisions, ok := GlobalECSService.TaskDefs[family]
		if !ok {
			return nil, errors.New("task definition family not found")
		}
		if revNum > len(revisions) {
			return nil, errors.New("task definition revision not found")
		}

		return &revisions[revNum-1], nil
	}

	return nil, errors.New("task definition not found")
}

// parseFamilyAndRevision parses a task definition in the format
// "family:revision".
func parseFamilyAndRevision(taskDef string) (family string, revNum int, err error) {
	partition := strings.LastIndex(taskDef, ":")
	if partition == -1 {
		return "", -1, errors.New("task definition is not in family:revision format")
	}

	family = taskDef[:partition]

	revNum, err = strconv.Atoi(taskDef[partition+1:])
	if err != nil {
		return "", -1, errors.Wrap(err, "parsing revision")
	}
	if revNum <= 0 {
		return "", -1, errors.New("revision cannot be less than 1")
	}

	return family, revNum, nil
}

func (s *ECSService) taskDefIndexFromARN(arn string) (family string, revNum int, found bool) {
	for family, revisions := range GlobalECSService.TaskDefs {
		for revIdx, def := range revisions {
			if utility.FromStringPtr(def.ARN) == arn {
				return family, revIdx + 1, true
			}
		}
	}
	return "", -1, false
}

// ECSClient provides a mock implementation of a cocoa.ECSClient. This makes
// it possible to introspect on inputs to the client and control the client's
// output. It provides some default implementations where possible. For unmocked
// methods, it will issue the API calls to the fake GlobalECSService.
type ECSClient struct {
	RegisterTaskDefinitionInput  *awsECS.RegisterTaskDefinitionInput
	RegisterTaskDefinitionOutput *awsECS.RegisterTaskDefinitionOutput
	RegisterTaskDefinitionError  error

	DescribeTaskDefinitionInput  *awsECS.DescribeTaskDefinitionInput
	DescribeTaskDefinitionOutput *awsECS.DescribeTaskDefinitionOutput
	DescribeTaskDefinitionError  error

	ListTaskDefinitionsInput  *awsECS.ListTaskDefinitionsInput
	ListTaskDefinitionsOutput *awsECS.ListTaskDefinitionsOutput
	ListTaskDefinitionsError  error

	DeregisterTaskDefinitionInput  *awsECS.DeregisterTaskDefinitionInput
	DeregisterTaskDefinitionOutput *awsECS.DeregisterTaskDefinitionOutput
	DeregisterTaskDefinitionError  error

	RunTaskInput  *awsECS.RunTaskInput
	RunTaskOutput *awsECS.RunTaskOutput
	RunTaskError  error

	DescribeTasksInput  *awsECS.DescribeTasksInput
	DescribeTasksOutput *awsECS.DescribeTasksOutput
	DescribeTasksError  error

	ListTasksInput  *awsECS.ListTasksInput
	ListTasksOutput *awsECS.ListTasksOutput
	ListTasksError  error

	StopTaskInput  *awsECS.StopTaskInput
	StopTaskOutput *awsECS.StopTaskOutput
	StopTaskError  error

	TagResourceInput  *awsECS.TagResourceInput
	TagResourceOutput *awsECS.TagResourceOutput
	TagResourceError  error

	CloseError error
}

// RegisterTaskDefinition saves the input and returns a new mock task
// definition. The mock output can be customized. By default, it will create a
// cached task definition based on the input.
func (c *ECSClient) RegisterTaskDefinition(ctx context.Context, in *awsECS.RegisterTaskDefinitionInput) (*awsECS.RegisterTaskDefinitionOutput, error) {
	c.RegisterTaskDefinitionInput = in

	if c.RegisterTaskDefinitionOutput != nil || c.RegisterTaskDefinitionError != nil {
		return c.RegisterTaskDefinitionOutput, c.RegisterTaskDefinitionError
	}

	if in.Family == nil {
		return nil, awserr.New(awsECS.ErrCodeInvalidParameterException, "missing family", nil)
	}

	revisions := GlobalECSService.TaskDefs[utility.FromStringPtr(in.Family)]
	rev := len(revisions) + 1

	taskDef := newECSTaskDefinition(in, rev)

	GlobalECSService.TaskDefs[utility.FromStringPtr(in.Family)] = append(revisions, taskDef)

	return &awsECS.RegisterTaskDefinitionOutput{
		TaskDefinition: taskDef.export(),
		Tags:           in.Tags,
	}, nil
}

// DescribeTaskDefinition saves the input and returns information about the
// matching task definition. The mock output can be customized. By default, it
// will return the task definition information if it exists.
func (c *ECSClient) DescribeTaskDefinition(ctx context.Context, in *awsECS.DescribeTaskDefinitionInput) (*awsECS.DescribeTaskDefinitionOutput, error) {
	c.DescribeTaskDefinitionInput = in

	if c.DescribeTaskDefinitionOutput != nil || c.DescribeTaskDefinitionError != nil {
		return c.DescribeTaskDefinitionOutput, c.DescribeTaskDefinitionError
	}

	id := utility.FromStringPtr(in.TaskDefinition)

	def, err := GlobalECSService.getLatestTaskDefinition(id)
	if err != nil {
		return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "task definition not found", err)
	}

	resp := awsECS.DescribeTaskDefinitionOutput{
		TaskDefinition: def.export(),
	}
	if shouldIncludeTags(in.Include) {
		resp.Tags = ecs.ExportTags(def.Tags)
	}

	return &resp, nil
}

// ListTaskDefinitions saves the input and lists all matching task definitions.
// The mock output can be customized. By default, it will list all cached task
// definitions that match the input filters.
func (c *ECSClient) ListTaskDefinitions(ctx context.Context, in *awsECS.ListTaskDefinitionsInput) (*awsECS.ListTaskDefinitionsOutput, error) {
	c.ListTaskDefinitionsInput = in

	if c.ListTaskDefinitionsOutput != nil || c.ListTaskDefinitionsError != nil {
		return c.ListTaskDefinitionsOutput, c.ListTaskDefinitionsError
	}

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

	return &awsECS.ListTaskDefinitionsOutput{
		TaskDefinitionArns: arns,
	}, nil
}

// DeregisterTaskDefinition saves the input and deletes an existing mock task
// definition. The mock output can be customized. By default, it will delete a
// cached task definition if it exists.
func (c *ECSClient) DeregisterTaskDefinition(ctx context.Context, in *awsECS.DeregisterTaskDefinitionInput) (*awsECS.DeregisterTaskDefinitionOutput, error) {
	c.DeregisterTaskDefinitionInput = in

	if c.DeregisterTaskDefinitionOutput != nil || c.DeregisterTaskDefinitionError != nil {
		return c.DeregisterTaskDefinitionOutput, c.DeregisterTaskDefinitionError
	}

	if in.TaskDefinition == nil {
		return nil, awserr.New(awsECS.ErrCodeInvalidParameterException, "missing task definition", nil)
	}

	id := utility.FromStringPtr(in.TaskDefinition)

	def, err := GlobalECSService.getTaskDefinition(id)
	if err != nil {
		return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "task definition not found", err)
	}

	def.Status = utility.ToStringPtr(awsECS.TaskDefinitionStatusInactive)
	def.Deregistered = utility.ToTimePtr(time.Now())
	GlobalECSService.TaskDefs[utility.FromStringPtr(def.Family)][utility.FromInt64Ptr(def.Revision)-1] = *def

	return &awsECS.DeregisterTaskDefinitionOutput{
		TaskDefinition: def.export(),
	}, nil
}

// RunTask saves the input options and returns the mock result of running a task
// definition. The mock output can be customized. By default, it will create
// mock output based on the input.
func (c *ECSClient) RunTask(ctx context.Context, in *awsECS.RunTaskInput) (*awsECS.RunTaskOutput, error) {
	c.RunTaskInput = in

	if c.RunTaskOutput != nil || c.RunTaskError != nil {
		return c.RunTaskOutput, c.RunTaskError
	}

	if in.TaskDefinition == nil {
		return nil, awserr.New(awsECS.ErrCodeInvalidParameterException, "missing task definition", nil)
	}

	clusterName := c.getOrDefaultCluster(in.Cluster)
	cluster, ok := GlobalECSService.Clusters[clusterName]
	if !ok {
		return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "cluster not found", nil)
	}

	taskDefID := utility.FromStringPtr(in.TaskDefinition)

	def, err := GlobalECSService.getLatestTaskDefinition(taskDefID)
	if err != nil {
		return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "task definition not found", err)
	}

	task := newECSTask(in, *def)

	cluster[utility.FromStringPtr(task.ARN)] = task

	return &awsECS.RunTaskOutput{
		Tasks: []*awsECS.Task{task.export(true)},
	}, nil
}

func (c *ECSClient) getOrDefaultCluster(name *string) string {
	if name == nil {
		return "default"
	}
	return *name
}

// DescribeTasks saves the input and returns information about the existing
// tasks. The mock output can be customized. By default, it will describe all
// cached tasks that match.
func (c *ECSClient) DescribeTasks(ctx context.Context, in *awsECS.DescribeTasksInput) (*awsECS.DescribeTasksOutput, error) {
	c.DescribeTasksInput = in

	if c.DescribeTasksOutput != nil || c.DescribeTasksError != nil {
		return c.DescribeTasksOutput, c.DescribeTasksError
	}

	cluster, ok := GlobalECSService.Clusters[c.getOrDefaultCluster(in.Cluster)]
	if !ok {
		return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "cluster not found", nil)
	}

	includeTags := shouldIncludeTags(in.Include)
	ids := utility.FromStringPtrSlice(in.Tasks)

	var tasks []*awsECS.Task
	var failures []*awsECS.Failure
	for _, id := range ids {
		task, ok := cluster[id]
		if !ok {
			failures = append(failures, &awsECS.Failure{
				Arn: utility.ToStringPtr(id),
				// This reason specifically matches the one returned by ECS when
				// it cannot find the task.
				Reason: utility.ToStringPtr("MISSING"),
			})
			continue
		}

		tasks = append(tasks, task.export(includeTags))
	}

	return &awsECS.DescribeTasksOutput{
		Tasks:    tasks,
		Failures: failures,
	}, nil
}

// shouldIncludeTags returns whether or not the ECS response should include
// resource tags.
func shouldIncludeTags(includes []*string) bool {
	for _, include := range includes {
		// "TAGS" is a magic string in the ECS API that indicates that the
		// response should include resource tags.
		if utility.FromStringPtr(include) == "TAGS" {
			return true
		}
	}
	return false
}

// ListTasks saves the input and lists all matching tasks. The mock output can
// be customized. By default, it will list all cached task definitions that
// match the input filters.
func (c *ECSClient) ListTasks(ctx context.Context, in *awsECS.ListTasksInput) (*awsECS.ListTasksOutput, error) {
	c.ListTasksInput = in

	if c.ListTasksOutput != nil || c.ListTasksError != nil {
		return c.ListTasksOutput, c.ListTasksError
	}

	cluster, ok := GlobalECSService.Clusters[c.getOrDefaultCluster(in.Cluster)]
	if !ok {
		return &awsECS.ListTasksOutput{}, nil
	}

	var arns []string
	for arn, task := range cluster {
		if in.DesiredStatus != nil && utility.FromStringPtr(task.GoalStatus) != *in.DesiredStatus {
			continue
		}

		if in.ContainerInstance != nil && utility.FromStringPtr(task.ContainerInstance) != *in.ContainerInstance {
			continue
		}

		if in.Family != nil && utility.FromStringPtr(task.TaskDef.Family) != *in.Family {
			continue
		}

		arns = append(arns, arn)
	}

	return &awsECS.ListTasksOutput{
		TaskArns: utility.ToStringPtrSlice(arns),
	}, nil
}

// StopTask saves the input and stops a mock task. The mock output can be
// customized. By default, it will mark a cached task as stopped if it exists
// and is running.
func (c *ECSClient) StopTask(ctx context.Context, in *awsECS.StopTaskInput) (*awsECS.StopTaskOutput, error) {
	c.StopTaskInput = in

	if c.StopTaskOutput != nil || c.StopTaskError != nil {
		return c.StopTaskOutput, c.StopTaskError
	}

	cluster, ok := GlobalECSService.Clusters[c.getOrDefaultCluster(in.Cluster)]
	if !ok {
		return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "cluster not found", nil)
	}

	task, ok := cluster[utility.FromStringPtr(in.Task)]
	if !ok {
		return nil, cocoa.NewECSTaskNotFoundError(utility.FromStringPtr(in.Task))
	}

	task.Status = utility.ToStringPtr(awsECS.DesiredStatusStopped)
	task.GoalStatus = utility.ToStringPtr(awsECS.DesiredStatusStopped)
	task.StopCode = utility.ToStringPtr(awsECS.TaskStopCodeUserInitiated)
	task.StopReason = in.Reason
	task.Stopped = utility.ToTimePtr(time.Now())
	for i := range task.Containers {
		task.Containers[i].Status = utility.ToStringPtr(awsECS.DesiredStatusStopped)
	}

	cluster[utility.FromStringPtr(in.Task)] = task

	return &awsECS.StopTaskOutput{
		Task: task.export(true),
	}, nil
}

// TagResource saves the input and tags a mock task or task definition. The mock
// output can be customized. By default, it will add the tag to the resource if
// it exists.
func (c *ECSClient) TagResource(ctx context.Context, in *awsECS.TagResourceInput) (*awsECS.TagResourceOutput, error) {
	c.TagResourceInput = in

	if c.TagResourceOutput != nil || c.TagResourceError != nil {
		return c.TagResourceOutput, c.TagResourceError
	}

	id := utility.FromStringPtr(in.ResourceArn)

	taskDef, err := GlobalECSService.getTaskDefinition(id)
	if err == nil {
		for k, v := range newECSTags(in.Tags) {
			taskDef.Tags[k] = v
		}
		return &awsECS.TagResourceOutput{}, nil
	}

	for _, cluster := range GlobalECSService.Clusters {
		task, ok := cluster[id]
		if !ok {
			continue
		}
		for k, v := range newECSTags(in.Tags) {
			task.Tags[k] = v
		}
		cluster[id] = task
		return &awsECS.TagResourceOutput{}, nil
	}

	return nil, awserr.New(awsECS.ErrCodeResourceNotFoundException, "task or task definition not found", nil)
}

// Close closes the mock client. The mock output can be customized. By default,
// it is a no-op that returns no error.
func (c *ECSClient) Close(ctx context.Context) error {
	if c.CloseError != nil {
		return c.CloseError
	}

	return nil
}
