package mock

import (
	"context"
	"strconv"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/ecs"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSPodDefinitionManager(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodDefinitionManager)(nil), &ECSPodDefinitionManager{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range ecsPodDefinitionManagerTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			resetECSAndSecretsManagerCache()

			c := &ECSClient{}
			defer func() {
				assert.NoError(t, c.Close(ctx))
			}()

			sm := &SecretsManagerClient{}
			defer func() {
				assert.NoError(t, sm.Close(tctx))
			}()

			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(sm))
			require.NoError(t, err)
			mv := NewVault(v)

			pdc := NewECSPodDefinitionCache(&testutil.NoopECSPodDefinitionCache{})

			const cacheTag = "cache_tag"

			pdm, err := ecs.NewBasicPodDefinitionManager(*ecs.NewBasicPodDefinitionManagerOptions().
				SetClient(c).
				SetVault(mv).
				SetCache(pdc).
				SetCacheTag(cacheTag))
			require.NoError(t, err)

			m := NewECSPodDefinitionManager(pdm)

			tCase(tctx, t, m, pdc, c, sm, cacheTag)
		})
	}

	for tName, tCase := range testcase.ECSPodDefinitionManagerTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			resetECSAndSecretsManagerCache()

			c := &ECSClient{}
			defer func() {
				assert.NoError(t, c.Close(ctx))
			}()

			opts := ecs.NewBasicPodDefinitionManagerOptions().SetClient(c)

			pdm, err := ecs.NewBasicPodDefinitionManager(*opts)
			require.NoError(t, err)

			mpdm := NewECSPodDefinitionManager(pdm)

			tCase(tctx, t, mpdm)
		})
	}

	for tName, tCase := range testcase.ECSPodDefinitionManagerVaultTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			resetECSAndSecretsManagerCache()

			c := &ECSClient{}
			defer func() {
				assert.NoError(t, c.Close(ctx))
			}()

			smc := &SecretsManagerClient{}
			defer func() {
				assert.NoError(t, smc.Close(ctx))
			}()

			v, err := secret.NewBasicSecretsManager(*secret.NewBasicSecretsManagerOptions().SetClient(smc))
			require.NoError(t, err)
			mv := NewVault(v)

			opts := ecs.NewBasicPodDefinitionManagerOptions().
				SetClient(c).
				SetVault(mv)

			pdm, err := ecs.NewBasicPodDefinitionManager(*opts)
			require.NoError(t, err)

			mpdm := NewECSPodDefinitionManager(pdm)

			tCase(tctx, t, mpdm)
		})
	}
}

// ecsPodDefinitionManagerTests are mock-specific tests for ECS and Secrets
// Manager with the ECS pod definition manager.
func ecsPodDefinitionManagerTests() map[string]func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager, pdc *ECSPodDefinitionCache, c *ECSClient, sm *SecretsManagerClient, cacheTag string) {
	return map[string]func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager, pdc *ECSPodDefinitionCache, c *ECSClient, sm *SecretsManagerClient, cacheTag string){
		"CreatePodDefinitionRegistersTaskDefinitionAndCachesWithAllFieldsSet": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager, pdc *ECSPodDefinitionCache, c *ECSClient, sm *SecretsManagerClient, cacheTag string) {
			envVar := cocoa.NewEnvironmentVariable().
				SetName("env_var_name").
				SetValue("env_var_value")
			containerDef := cocoa.NewECSContainerDefinition().
				SetName("name").
				SetImage("image").
				SetCommand([]string{"echo", "foo"}).
				SetWorkingDir("working_dir").
				SetMemoryMB(128).
				SetCPU(256).
				AddEnvironmentVariables(*envVar)
			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				SetMemoryMB(512).
				SetCPU(1024).
				SetTaskRole("task_role").
				SetExecutionRole("execution_role").
				SetNetworkMode(cocoa.NetworkModeAWSVPC).
				SetTags(map[string]string{"creation_tag": "creation_val"}).
				AddContainerDefinitions(*containerDef)
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			require.NoError(t, err)
			require.NotZero(t, pdi.ID)
			require.NotZero(t, pdi.DefinitionOpts)

			require.NotZero(t, c.RegisterTaskDefinitionInput, "should have registered a task definition")

			mem, err := strconv.Atoi(utility.FromStringPtr(c.RegisterTaskDefinitionInput.Memory))
			require.NoError(t, err)
			assert.Equal(t, utility.FromIntPtr(opts.MemoryMB), mem)
			cpu, err := strconv.Atoi(utility.FromStringPtr(c.RegisterTaskDefinitionInput.Cpu))
			require.NoError(t, err)
			assert.Equal(t, utility.FromIntPtr(opts.CPU), cpu)
			require.NotZero(t, opts.NetworkMode)
			assert.EqualValues(t, *opts.NetworkMode, utility.FromStringPtr(c.RegisterTaskDefinitionInput.NetworkMode))
			assert.Equal(t, utility.FromStringPtr(opts.TaskRole), utility.FromStringPtr(c.RegisterTaskDefinitionInput.TaskRoleArn))
			assert.Equal(t, utility.FromStringPtr(opts.ExecutionRole), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ExecutionRoleArn))
			assert.Len(t, c.RegisterTaskDefinitionInput.Tags, 2, "should have user-defined tag and cache tracking tag")
			for _, tag := range c.RegisterTaskDefinitionInput.Tags {
				key := utility.FromStringPtr(tag.Key)
				switch key {
				case "creation_tag":
					assert.Equal(t, opts.Tags["creation_tag"], utility.FromStringPtr(tag.Value), "user-defined tag should be defined")
				case cacheTag:
					assert.Equal(t, "false", utility.FromStringPtr(tag.Value), "cache tag should initially mark pod definition as uncached before caching")
				default:
					assert.FailNow(t, "unrecognized tag '%s'", key)
				}
			}
			require.Len(t, c.RegisterTaskDefinitionInput.ContainerDefinitions, 1)
			assert.Equal(t, containerDef.Command, utility.FromStringPtrSlice(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Command))
			assert.Equal(t, utility.FromStringPtr(containerDef.WorkingDir), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].WorkingDirectory))
			assert.EqualValues(t, utility.FromIntPtr(containerDef.MemoryMB), utility.FromInt64Ptr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Memory))
			assert.EqualValues(t, utility.FromIntPtr(containerDef.CPU), utility.FromInt64Ptr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Cpu))
			require.Len(t, c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Environment, 1)
			assert.Equal(t, utility.FromStringPtr(envVar.Name), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Environment[0].Name))
			assert.Equal(t, utility.FromStringPtr(envVar.Value), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Environment[0].Value))

			require.NotZero(t, pdc.PutInput, "should have cached the pod definition")
			assert.Equal(t, *pdi, *pdc.PutInput)

			require.NotZero(t, c.TagResourceInput, "should have re-tagged resource to indicate that it's cached")
			assert.Equal(t, pdi.ID, utility.FromStringPtr(c.TagResourceInput.ResourceArn))
			require.Len(t, c.TagResourceInput.Tags, 1)
			assert.Equal(t, cacheTag, utility.FromStringPtr(c.TagResourceInput.Tags[0].Key))
			assert.Equal(t, "true", utility.FromStringPtr(c.TagResourceInput.Tags[0].Value), "cache tag should be marked as cached")
		},
		"CreatePodDefinitionTagsStrandedPodDefinitionAsUncachedWhenCachingFails": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager, pdc *ECSPodDefinitionCache, c *ECSClient, sm *SecretsManagerClient, cacheTag string) {
			pdc.PutError = errors.New("fake error")

			envVar := cocoa.NewEnvironmentVariable().
				SetName("env_var_name").
				SetValue("env_var_value")
			containerDef := cocoa.NewECSContainerDefinition().
				SetName("name").
				SetImage("image").
				SetCommand([]string{"echo", "foo"}).
				SetWorkingDir("working_dir").
				SetMemoryMB(128).
				SetCPU(256).
				AddEnvironmentVariables(*envVar)
			opts := cocoa.NewECSPodDefinitionOptions().
				SetName(testutil.NewTaskDefinitionFamily(t)).
				SetMemoryMB(512).
				SetCPU(1024).
				SetTaskRole("task_role").
				SetExecutionRole("execution_role").
				SetNetworkMode(cocoa.NetworkModeAWSVPC).
				SetTags(map[string]string{"creation_tag": "creation_val"}).
				AddContainerDefinitions(*containerDef)
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			assert.Error(t, err, "should have failed to put into cache")
			assert.Zero(t, pdi)

			require.NotZero(t, c.RegisterTaskDefinitionInput, "should have registered a task definition")

			mem, err := strconv.Atoi(utility.FromStringPtr(c.RegisterTaskDefinitionInput.Memory))
			require.NoError(t, err)
			assert.Equal(t, utility.FromIntPtr(opts.MemoryMB), mem)
			cpu, err := strconv.Atoi(utility.FromStringPtr(c.RegisterTaskDefinitionInput.Cpu))
			require.NoError(t, err)
			assert.Equal(t, utility.FromIntPtr(opts.CPU), cpu)
			require.NotZero(t, opts.NetworkMode)
			assert.EqualValues(t, *opts.NetworkMode, utility.FromStringPtr(c.RegisterTaskDefinitionInput.NetworkMode))
			assert.Equal(t, utility.FromStringPtr(opts.TaskRole), utility.FromStringPtr(c.RegisterTaskDefinitionInput.TaskRoleArn))
			assert.Equal(t, utility.FromStringPtr(opts.ExecutionRole), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ExecutionRoleArn))
			assert.Len(t, c.RegisterTaskDefinitionInput.Tags, 2, "should have user-defined tag and cache tracking tag")
			for _, tag := range c.RegisterTaskDefinitionInput.Tags {
				key := utility.FromStringPtr(tag.Key)
				switch key {
				case "creation_tag":
					assert.Equal(t, opts.Tags["creation_tag"], utility.FromStringPtr(tag.Value), "user-defined tag should be defined")
				case cacheTag:
					assert.Equal(t, "false", utility.FromStringPtr(tag.Value), "cache tag should initially mark pod definition as uncached")
				default:
					assert.FailNow(t, "unrecognized tag '%s'", key)
				}
			}
			require.Len(t, c.RegisterTaskDefinitionInput.ContainerDefinitions, 1)
			assert.Equal(t, containerDef.Command, utility.FromStringPtrSlice(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Command))
			assert.Equal(t, utility.FromStringPtr(containerDef.WorkingDir), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].WorkingDirectory))
			assert.EqualValues(t, utility.FromIntPtr(containerDef.MemoryMB), utility.FromInt64Ptr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Memory))
			assert.EqualValues(t, utility.FromIntPtr(containerDef.CPU), utility.FromInt64Ptr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Cpu))
			require.Len(t, c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Environment, 1)
			assert.Equal(t, utility.FromStringPtr(envVar.Name), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Environment[0].Name))
			assert.Equal(t, utility.FromStringPtr(envVar.Value), utility.FromStringPtr(c.RegisterTaskDefinitionInput.ContainerDefinitions[0].Environment[0].Value))

			assert.NotZero(t, pdc.PutInput, "should have attempted to cache the pod definition")

			assert.Zero(t, c.TagResourceInput, "should not have re-tagged resource because it is not cached")
		},
		"CreatePodDefinitionDoesNotCacheWhenRegisteringTaskDefinitionFails": func(ctx context.Context, t *testing.T, pdm cocoa.ECSPodDefinitionManager, pdc *ECSPodDefinitionCache, c *ECSClient, sm *SecretsManagerClient, cacheTag string) {
			c.RegisterTaskDefinitionError = errors.New("fake error")

			envVar := cocoa.NewEnvironmentVariable().
				SetName("env_var_name").
				SetValue("env_var_value")
			containerDef := cocoa.NewECSContainerDefinition().
				SetName("name").
				SetImage("image").
				SetCommand([]string{"echo", "foo"}).
				SetWorkingDir("working_dir").
				SetMemoryMB(128).
				SetCPU(256).
				AddEnvironmentVariables(*envVar)
			opts := cocoa.NewECSPodDefinitionOptions().
				SetMemoryMB(512).
				SetCPU(1024).
				SetTaskRole("task_role").
				SetExecutionRole("execution_role").
				SetNetworkMode(cocoa.NetworkModeAWSVPC).
				SetTags(map[string]string{"creation_tag": "creation_val"}).
				AddContainerDefinitions(*containerDef)
			assert.NoError(t, opts.Validate())

			pdi, err := pdm.CreatePodDefinition(ctx, *opts)
			assert.Error(t, err, "should have failed to register task definition")
			assert.Zero(t, pdi)

			assert.NotZero(t, c.RegisterTaskDefinitionInput, "should have attempted to register a task definition")

			assert.Zero(t, pdc.PutInput, "should not have attempted to cache the pod definition after registration failed")
		},
	}
}
