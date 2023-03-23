package cocoa

import (
	"fmt"
	"testing"

	awsECS "github.com/aws/aws-sdk-go/service/ecs"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSPodCreationOptions(t *testing.T) {
	t.Run("NewECSPodCreationOptions", func(t *testing.T) {
		opts := NewECSPodCreationOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetDefinitionOptions", func(t *testing.T) {
		defOpts := NewECSPodDefinitionOptions().SetName("name")
		opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
		assert.Equal(t, *defOpts, opts.DefinitionOpts)
	})
	t.Run("SetExecutionOptions", func(t *testing.T) {
		execOpts := NewECSPodExecutionOptions().SetCluster("cluster")
		opts := NewECSPodCreationOptions().SetExecutionOptions(*execOpts)
		assert.Equal(t, *execOpts, *opts.ExecutionOpts)
	})
	t.Run("Validate", func(t *testing.T) {
		getValidPodDefOpts := func() *ECSPodDefinitionOptions {
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				SetCommand([]string{"echo"})
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128)
			return defOpts
		}

		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			assert.Error(t, NewECSPodCreationOptions().Validate())
		})
		t.Run("SucceedsWithValidFieldsPopulated", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			defOpts := NewECSPodDefinitionOptions().
				SetName("name").
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				AddTags(map[string]string{"key": "val"})
			placementOpts := NewECSPodPlacementOptions().SetStrategy(StrategyRandom)
			execOpts := NewECSPodExecutionOptions().
				SetCluster("cluster").
				SetCapacityProvider("provider").
				SetPlacementOptions(*placementOpts).
				SetSupportsDebugMode(true).
				SetTags(map[string]string{"other_key": "other_val"})
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithBadExecutionOptions", func(t *testing.T) {
			defOpts := getValidPodDefOpts()
			assert.NoError(t, defOpts.Validate())
			placementOpts := NewECSPodPlacementOptions().SetStrategy("foo")
			execOpts := NewECSPodExecutionOptions().SetPlacementOptions(*placementOpts)
			assert.Error(t, execOpts.Validate())
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*getValidPodDefOpts()).
				SetExecutionOptions(*execOpts)
			assert.Error(t, opts.Validate())
		})
		t.Run("AWSVPCOptionsWithNetworkModeAWSVPCIsValid", func(t *testing.T) {
			defOpts := getValidPodDefOpts().SetNetworkMode(NetworkModeAWSVPC)
			awsvpcOpts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			execOpts := NewECSPodExecutionOptions().SetAWSVPCOptions(*awsvpcOpts)
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("MissingExecutionOptionsWithNetworkModeAWSVPCIsInvalid", func(t *testing.T) {
			defOpts := getValidPodDefOpts().SetNetworkMode(NetworkModeAWSVPC)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.Error(t, opts.Validate())
		})
		t.Run("MissingAWSVPCOptionsWithNetworkModeAWSVPCIsInvalid", func(t *testing.T) {
			defOpts := getValidPodDefOpts().SetNetworkMode(NetworkModeAWSVPC)
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*NewECSPodExecutionOptions())
			assert.Error(t, opts.Validate())
		})
		t.Run("AWSVPCOptionsWithoutNetworkModeAWSVPCIsInvalid", func(t *testing.T) {
			defOpts := getValidPodDefOpts()
			awsvpcOpts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			execOpts := NewECSPodExecutionOptions().SetAWSVPCOptions(*awsvpcOpts)
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*execOpts)
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithNetworkModeAWSVPCAndPortMappingToIdenticalPortAndAWSVPCOptions", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337).SetHostPort(1337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeAWSVPC)
			awsvpcOpts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			execOpts := NewECSPodExecutionOptions().SetAWSVPCOptions(*awsvpcOpts)
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithNetworkModeAWSVPCAndPortMappingToUnspecifiedHostPortAndAWSVPCOptions", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeAWSVPC)
			awsvpcOpts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			execOpts := NewECSPodExecutionOptions().SetAWSVPCOptions(*awsvpcOpts)
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*execOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithNetworkModeAWSVPCANdPortMappingsToDifferentHostPort", func(t *testing.T) {
			pm := NewPortMapping().
				SetContainerPort(1337).
				SetHostPort(9001)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeAWSVPC)
			awsvpcOpts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			execOpts := NewECSPodExecutionOptions().SetAWSVPCOptions(*awsvpcOpts)
			opts := NewECSPodCreationOptions().
				SetDefinitionOptions(*defOpts).
				SetExecutionOptions(*execOpts)
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithNetworkModeHostAndPortMappingToIdenticalHostPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337).SetHostPort(1337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeHost)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithNetworkModeHostAndPortMappingToUnspecifiedHostPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeHost)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithNetworkModeHostAndPortMappingsToDifferentHostPort", func(t *testing.T) {
			pm := NewPortMapping().
				SetContainerPort(1337).
				SetHostPort(9001)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeBridge)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithNetworkModeBridgeAndPortMappingToIdenticalHostPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337).SetHostPort(1337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeHost)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithNetworkModeBridgeAndPortMappingToUnspecifiedHostPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeBridge)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithNetworkModeBridgeAndPortMappingsToDifferentHostPort", func(t *testing.T) {
			pm := NewPortMapping().
				SetContainerPort(1337).
				SetHostPort(13337)
			containerDef := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*pm)
			defOpts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetNetworkMode(NetworkModeBridge)
			opts := NewECSPodCreationOptions().SetDefinitionOptions(*defOpts)
			assert.NoError(t, opts.Validate())
		})
	})
}

func TestECSPodDefinition(t *testing.T) {
	t.Run("NewECSPodDefinitionOptions", func(t *testing.T) {
		opts := NewECSPodDefinitionOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetName", func(t *testing.T) {
		name := "name"
		def := NewECSPodDefinitionOptions().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(def.Name))
	})
	t.Run("SetContainerDefinitions", func(t *testing.T) {
		containerDef := NewECSContainerDefinition().SetImage("image")

		opts := NewECSPodDefinitionOptions().SetContainerDefinitions([]ECSContainerDefinition{*containerDef})
		require.Len(t, opts.ContainerDefinitions, 1)
		assert.Equal(t, *containerDef, opts.ContainerDefinitions[0])

		opts.SetContainerDefinitions(nil)
		assert.Empty(t, opts.ContainerDefinitions)
	})
	t.Run("AddContainerDefinitions", func(t *testing.T) {
		containerDefs := []ECSContainerDefinition{
			*NewECSContainerDefinition().SetImage("image0"),
			*NewECSContainerDefinition().SetImage("image1"),
		}
		opts := NewECSPodDefinitionOptions().AddContainerDefinitions(containerDefs...)
		assert.ElementsMatch(t, containerDefs, opts.ContainerDefinitions)

		opts.AddContainerDefinitions()
		assert.ElementsMatch(t, containerDefs, opts.ContainerDefinitions)
	})
	t.Run("SetMemoryMB", func(t *testing.T) {
		mem := 128
		opts := NewECSPodDefinitionOptions().SetMemoryMB(mem)
		assert.Equal(t, mem, utility.FromIntPtr(opts.MemoryMB))
	})
	t.Run("SetCPU", func(t *testing.T) {
		cpu := 128
		opts := NewECSPodDefinitionOptions().SetCPU(cpu)
		assert.Equal(t, cpu, utility.FromIntPtr(opts.CPU))
	})
	t.Run("SetNetworkMode", func(t *testing.T) {
		mode := NetworkModeAWSVPC
		opts := NewECSPodDefinitionOptions().SetNetworkMode(mode)
		require.NotZero(t, opts.NetworkMode)
		assert.Equal(t, mode, *opts.NetworkMode)
	})
	t.Run("SetTaskRole", func(t *testing.T) {
		r := "task_role"
		opts := NewECSPodDefinitionOptions().SetTaskRole(r)
		assert.Equal(t, r, utility.FromStringPtr(opts.TaskRole))
	})
	t.Run("SetExecutionRole", func(t *testing.T) {
		r := "execution_role"
		opts := NewECSPodDefinitionOptions().SetExecutionRole(r)
		assert.Equal(t, r, utility.FromStringPtr(opts.ExecutionRole))
	})
	t.Run("SetTags", func(t *testing.T) {
		tags := map[string]string{"key": "value"}

		opts := NewECSPodDefinitionOptions().SetTags(tags)
		require.Len(t, opts.Tags, len(tags))
		for k, v := range tags {
			assert.Equal(t, v, opts.Tags[k])
		}

		opts.SetTags(nil)
		assert.Empty(t, opts.Tags)
	})
	t.Run("AddTags", func(t *testing.T) {
		tags := map[string]string{"key0": "val0", "key1": "val1"}
		opts := NewECSPodDefinitionOptions().AddTags(tags)
		assert.Equal(t, tags, opts.Tags)
		opts.AddTags(map[string]string{})
		assert.Equal(t, tags, opts.Tags)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithMemoryCPUAndContainerDefinition", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithoutContainerDefinition", func(t *testing.T) {
			opts := NewECSPodDefinitionOptions().
				SetMemoryMB(128).
				SetCPU(128)
			assert.Error(t, opts.Validate())
		})
		t.Run("NameIsGenerated", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128)
			assert.NoError(t, opts.Validate())
			assert.NotZero(t, utility.FromStringPtr(opts.Name))
		})
		t.Run("FailsWithBadContainerDefinition", func(t *testing.T) {
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*NewECSContainerDefinition()).
				SetMemoryMB(128).
				SetCPU(128)
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				SetName("name").
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128).
				SetTaskRole("role").
				AddTags(map[string]string{"key": "val"})
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithMissingCPU", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128)
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedsWithoutPodCPUWhenContainerCPUIsGiven", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image").SetCPU(128)
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWhenTotalContainerCPUExceedsPodCPU", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image").SetCPU(256)
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(1024).
				SetCPU(128)
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithZeroCPU", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(0)
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithoutMemory", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetCPU(128)
			assert.Error(t, opts.Validate())
		})
		t.Run("SucceedWithoutPodMemoryWhenContainerMemoryIsGiven", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image").SetMemoryMB(128)
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetCPU(128)
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithTotalContainerMemoryExceedingPodMemory", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image").SetMemoryMB(256)
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(1024)
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithZeroMemory", func(t *testing.T) {
			containerDef := NewECSContainerDefinition().SetImage("image")
			opts := NewECSPodDefinitionOptions().
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(0).
				SetCPU(128)
			assert.Error(t, opts.Validate())
		})
	})
	t.Run("Hash", func(t *testing.T) {
		getValidPodDefOpts := func() *ECSPodDefinitionOptions {
			containerDef := NewECSContainerDefinition().
				SetName("container_name").
				SetImage("image")
			defOpts := NewECSPodDefinitionOptions().
				SetName("pod_name").
				AddContainerDefinitions(*containerDef).
				SetMemoryMB(128).
				SetCPU(128)
			return defOpts
		}
		baseHash := getValidPodDefOpts().Hash()

		t.Run("ReturnsSameValueForSameInput", func(t *testing.T) {
			assert.Equal(t, baseHash, getValidPodDefOpts().Hash())
		})
		t.Run("ChangesForName", func(t *testing.T) {
			opts := getValidPodDefOpts().SetName("new_name")
			assert.NotEqual(t, baseHash, opts.Hash(), "name should affect hash")
		})
		t.Run("ChangesForMemory", func(t *testing.T) {
			opts := getValidPodDefOpts().SetMemoryMB(1024)
			assert.NotEqual(t, baseHash, opts.Hash(), "memory should affect hash")
		})
		t.Run("ChangesForCPU", func(t *testing.T) {
			opts := getValidPodDefOpts().SetCPU(1024)
			assert.NotEqual(t, baseHash, opts.Hash(), "CPU should affect hash")
		})
		t.Run("ChangesForNetworkMode", func(t *testing.T) {
			opts := getValidPodDefOpts().SetNetworkMode(NetworkModeHost)
			assert.NotEqual(t, baseHash, opts.Hash(), "network mode should affect hash")
		})
		t.Run("ChangesForTaskRole", func(t *testing.T) {
			opts := getValidPodDefOpts().SetTaskRole("task_role")
			assert.NotEqual(t, baseHash, opts.Hash(), "task role should affect hash")
		})
		t.Run("ChangesForExecutionRole", func(t *testing.T) {
			opts := getValidPodDefOpts().SetExecutionRole("execution_role")
			assert.NotEqual(t, baseHash, opts.Hash(), "execution role should affect hash")
		})
		t.Run("ChangesForTags", func(t *testing.T) {
			opts := getValidPodDefOpts().SetTags(map[string]string{
				"key": "value",
			})
			assert.NotEqual(t, baseHash, opts.Hash(), "tags should affect hash")
		})
		t.Run("ReturnsSameValueForSameUnorderedTags", func(t *testing.T) {
			opts := getValidPodDefOpts()
			for i := 0; i < 10; i++ {
				opts.AddTags(map[string]string{
					utility.RandomString(): utility.RandomString(),
				})
			}
			h0 := opts.Hash()
			h1 := opts.Hash()
			assert.Equal(t, h0, h1, "order of tags should not affect hash")
		})
		t.Run("ReturnsSameValueForDifferentContainerDefinitionOrder", func(t *testing.T) {
			opts := getValidPodDefOpts()
			cd0 := NewECSContainerDefinition().SetName("container0").SetImage("debian")
			cd1 := NewECSContainerDefinition().SetName("container1").SetImage("ubuntu")

			opts.SetContainerDefinitions([]ECSContainerDefinition{*cd0, *cd1})
			h0 := opts.Hash()

			opts.SetContainerDefinitions([]ECSContainerDefinition{*cd1, *cd0})
			h1 := opts.Hash()

			assert.Equal(t, h0, h1, "order of container definitions should not affect hash")
		})
		t.Run("ChangesForContainerName", func(t *testing.T) {
			opts := getValidPodDefOpts()
			opts.ContainerDefinitions[0].SetName("new_name")
			assert.NotEqual(t, baseHash, opts.Hash(), "container name should affect hash")
		})
		t.Run("ChangesForContainerImage", func(t *testing.T) {
			opts := getValidPodDefOpts()
			opts.ContainerDefinitions[0].SetImage("alpine")
			assert.NotEqual(t, baseHash, opts.Hash(), "container image should affect hash")
		})
		t.Run("ChangesForDifferentContainerCommand", func(t *testing.T) {
			opts := getValidPodDefOpts()
			opts.ContainerDefinitions[0].SetCommand([]string{"echo", "foo", "bar"})
			assert.NotEqual(t, baseHash, opts.Hash(), "container command should affect hash")
		})
		t.Run("ChangesForDifferentContainerCommandArgOrder", func(t *testing.T) {
			opts := getValidPodDefOpts()

			opts.ContainerDefinitions[0].SetCommand([]string{"echo", "foo", "bar"})
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetCommand([]string{"echo", "bar", "foo"})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "order of container command args should affect hash")
		})
		t.Run("ChangesForDifferentContainerWorkingDir", func(t *testing.T) {
			opts := getValidPodDefOpts()
			opts.ContainerDefinitions[0].SetWorkingDir("/var/run")
			assert.NotEqual(t, baseHash, opts.Hash(), "container working directory should affect hash")
		})
		t.Run("ChangesForDifferentContainerMemoryMB", func(t *testing.T) {
			opts := getValidPodDefOpts()
			opts.ContainerDefinitions[0].SetMemoryMB(64)
			assert.NotEqual(t, baseHash, opts.Hash(), "container memory should affect hash")
		})
		t.Run("ChangesForDifferentContainerCPU", func(t *testing.T) {
			opts := getValidPodDefOpts()
			opts.ContainerDefinitions[0].SetCPU(64)
			assert.NotEqual(t, baseHash, opts.Hash(), "container CPU should affect hash")
		})
		t.Run("ChangesForDifferentEnvVars", func(t *testing.T) {
			opts := getValidPodDefOpts()
			ev := NewEnvironmentVariable().SetName("ENV_VAR").SetValue("value")
			opts.ContainerDefinitions[0].AddEnvironmentVariables(*ev)
			assert.NotEqual(t, baseHash, opts.Hash(), "container environment variables should affect hash")
		})
		t.Run("ChangesForDifferentEnvVarName", func(t *testing.T) {
			opts := getValidPodDefOpts()
			ev := NewEnvironmentVariable().SetName("ENV_VAR")

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h0 := opts.Hash()

			ev.SetName("NEW_ENV_VAR")
			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container environment variable name should affect hash")
		})
		t.Run("ChangesForDifferentEnvVarValue", func(t *testing.T) {
			opts := getValidPodDefOpts()
			ev := NewEnvironmentVariable().SetValue("value")

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h0 := opts.Hash()

			ev.SetValue("new_value")
			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container environment variable value should affect hash")
		})
		t.Run("ChangesForDifferentSecretID", func(t *testing.T) {
			opts := getValidPodDefOpts()
			secretOpts := NewSecretOptions()
			ev := NewEnvironmentVariable().SetSecretOptions(*secretOpts)

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h0 := opts.Hash()

			secretOpts.SetID("id")
			ev.SetSecretOptions(*secretOpts)
			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container secret ID should affect hash")
		})
		t.Run("ChangesForDifferentSecretName", func(t *testing.T) {
			opts := getValidPodDefOpts()
			secretOpts := NewSecretOptions()
			ev := NewEnvironmentVariable().SetSecretOptions(*secretOpts)

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h0 := opts.Hash()

			secretOpts.SetName("secret_name")
			ev.SetSecretOptions(*secretOpts)
			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container secret name should affect hash")
		})
		t.Run("ChangesForDifferentSecretValue", func(t *testing.T) {
			opts := getValidPodDefOpts()
			secretOpts := NewSecretOptions()
			ev := NewEnvironmentVariable().SetSecretOptions(*secretOpts)

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h0 := opts.Hash()

			secretOpts.SetNewValue("new_value")
			ev.SetSecretOptions(*secretOpts)
			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container secret value should affect hash")
		})
		t.Run("ChangesForDifferentSecretOwnership", func(t *testing.T) {
			opts := getValidPodDefOpts()
			secretOpts := NewSecretOptions()
			ev := NewEnvironmentVariable().SetSecretOptions(*secretOpts)

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h0 := opts.Hash()

			secretOpts.SetOwned(true)
			ev.SetSecretOptions(*secretOpts)
			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container secret value should affect hash")
		})
		t.Run("ReturnsSameValueForDifferentEnvVarOrder", func(t *testing.T) {
			opts := getValidPodDefOpts()
			ev0 := NewEnvironmentVariable().SetName("ENV_VAR0").SetValue("value0")
			ev1 := NewEnvironmentVariable().SetName("ENV_VAR1").SetValue("value1")

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev0, *ev1})
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetEnvironmentVariables([]EnvironmentVariable{*ev1, *ev0})
			h1 := opts.Hash()

			assert.Equal(t, h0, h1, "order of container environment variables should not affect hash")
		})
		t.Run("ChangesForDifferentRepoCreds", func(t *testing.T) {
			opts := getValidPodDefOpts()
			creds := NewRepositoryCredentials().SetID("id")
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			assert.NotEqual(t, baseHash, opts.Hash(), "container repo creds should affect hash")
		})
		t.Run("ChangesForDifferentRepoCredsID", func(t *testing.T) {
			opts := getValidPodDefOpts()

			creds := NewRepositoryCredentials()
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds.SetID("id"))
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container repo creds ID should affect hash")
		})
		t.Run("ChangesForDifferentRepoCredsName", func(t *testing.T) {
			opts := getValidPodDefOpts()

			creds := NewRepositoryCredentials()
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds.SetName("name"))
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container repo creds name should affect hash")
		})
		t.Run("ChangesForNewRepoCredsUsername", func(t *testing.T) {
			opts := getValidPodDefOpts()

			newCreds := NewStoredRepositoryCredentials()
			creds := NewRepositoryCredentials().SetNewCredentials(*newCreds)
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds.SetNewCredentials(*newCreds.SetUsername("username")))
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container new repo creds should affect hash")
		})
		t.Run("ChangesForNewRepoCredsPassword", func(t *testing.T) {
			opts := getValidPodDefOpts()

			newCreds := NewStoredRepositoryCredentials()
			creds := NewRepositoryCredentials().SetNewCredentials(*newCreds)
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds.SetNewCredentials(*newCreds.SetPassword("password")))
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container new repo creds should affect hash")
		})
		t.Run("ChangesForDifferentRepoCredsOwnership", func(t *testing.T) {
			opts := getValidPodDefOpts()

			creds := NewRepositoryCredentials()
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds.SetOwned(true))
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container repo creds name should affect hash")
		})
		t.Run("ChangesForDifferentLogConfiguration", func(t *testing.T) {
			opts := getValidPodDefOpts()
			creds := NewRepositoryCredentials().SetID("id")
			opts.ContainerDefinitions[0].SetRepositoryCredentials(*creds)
			assert.NotEqual(t, baseHash, opts.Hash(), "container repo creds should affect hash")
		})
		t.Run("ChangesForDifferentLogConfigurationDriver", func(t *testing.T) {
			opts := getValidPodDefOpts()

			logConf := NewLogConfiguration()
			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf.SetLogDriver(awsECS.LogDriverAwslogs))
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "log configuration driver should affect hash")
		})
		t.Run("ChangesForDifferentLogConfigurationOptions", func(t *testing.T) {
			opts := getValidPodDefOpts()

			logConf := NewLogConfiguration()
			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf)
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf.SetOptions(map[string]string{"key": "value"}))
			h1 := opts.Hash()

			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf.SetOptions(map[string]string{"key": "value", "key2": "value"}))
			h2 := opts.Hash()

			assert.NotEqual(t, h0, h1, h2, "log configuration options should affect hash")
		})
		t.Run("ReturnsSameValueForSameUnorderedLogConfigurationOptions", func(t *testing.T) {
			opts := getValidPodDefOpts()

			logConf := NewLogConfiguration()
			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf)
			logConfOptions := map[string]string{}

			for i := 0; i < 10; i++ {
				logConfOptions[utility.RandomString()] = utility.RandomString()
			}
			opts.ContainerDefinitions[0].SetLogConfiguration(*logConf.SetOptions(logConfOptions))
			h0 := opts.Hash()
			h1 := opts.Hash()
			assert.Equal(t, h0, h1, "order of log configuration options should not affect hash")
		})
		t.Run("ChangesForDifferentPortMappings", func(t *testing.T) {
			opts := getValidPodDefOpts()

			pm := NewPortMapping().SetContainerPort(12345)
			opts.ContainerDefinitions[0].AddPortMappings(*pm)

			assert.NotEqual(t, baseHash, opts.Hash(), "port mapping should affect hash")
		})
		t.Run("ChangesForDifferentContainerPortMapping", func(t *testing.T) {
			opts := getValidPodDefOpts()

			pm := NewPortMapping()
			opts.ContainerDefinitions[0].SetPortMappings([]PortMapping{*pm})
			h0 := opts.Hash()

			pm.SetContainerPort(12345)
			opts.ContainerDefinitions[0].SetPortMappings([]PortMapping{*pm})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "container port mapping should affect hash")
		})
		t.Run("ChangesForDifferentHostPortMapping", func(t *testing.T) {
			opts := getValidPodDefOpts()

			pm := NewPortMapping()
			opts.ContainerDefinitions[0].SetPortMappings([]PortMapping{*pm})
			h0 := opts.Hash()

			pm.SetHostPort(12345)
			opts.ContainerDefinitions[0].SetPortMappings([]PortMapping{*pm})
			h1 := opts.Hash()

			assert.NotEqual(t, h0, h1, "host port mapping should affect hash")
		})
		t.Run("ReturnsSameValueForDifferentPortMappingOrder", func(t *testing.T) {
			opts := getValidPodDefOpts()
			pm0 := NewPortMapping().SetContainerPort(1234).SetHostPort(5678)
			pm1 := NewPortMapping().SetContainerPort(1337).SetHostPort(9001)

			opts.ContainerDefinitions[0].SetPortMappings([]PortMapping{*pm0, *pm1})
			h0 := opts.Hash()

			opts.ContainerDefinitions[0].SetPortMappings([]PortMapping{*pm1, *pm0})
			h1 := opts.Hash()

			assert.Equal(t, h0, h1, "order of port mappings should not affect hash")
		})
	})
}

func TestECSNetworkMode(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		for _, m := range []ECSNetworkMode{
			NetworkModeNone,
			NetworkModeAWSVPC,
			NetworkModeBridge,
			NetworkModeHost,
		} {
			t.Run(fmt.Sprintf("SucceedsForStatus=%s", m), func(t *testing.T) {
				assert.NoError(t, m.Validate())
			})
		}
		t.Run("FailsForEmptyStatus", func(t *testing.T) {
			assert.Error(t, ECSNetworkMode("").Validate())
		})
		t.Run("FailsForInvalidStatus", func(t *testing.T) {
			assert.Error(t, ECSNetworkMode("invalid").Validate())
		})
	})
}

func TestECSContainerDefinition(t *testing.T) {
	t.Run("NewECSContainerDefinition", func(t *testing.T) {
		def := NewECSContainerDefinition()
		require.NotZero(t, def)
		assert.Zero(t, *def)
	})
	t.Run("SetName", func(t *testing.T) {
		name := "name"
		def := NewECSContainerDefinition().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(def.Name))
	})
	t.Run("SetImage", func(t *testing.T) {
		image := "image"
		def := NewECSContainerDefinition().SetImage(image)
		assert.Equal(t, image, utility.FromStringPtr(def.Image))
	})
	t.Run("SetCommand", func(t *testing.T) {
		cmd := []string{"echo", "hello"}
		def := NewECSContainerDefinition().SetCommand(cmd)
		assert.Equal(t, cmd, def.Command)
	})
	t.Run("SetWorkingDir", func(t *testing.T) {
		dir := "working_dir"
		def := NewECSContainerDefinition().SetWorkingDir(dir)
		assert.Equal(t, dir, utility.FromStringPtr(def.WorkingDir))
	})
	t.Run("SetMemoryMB", func(t *testing.T) {
		mem := 128
		def := NewECSContainerDefinition().SetMemoryMB(mem)
		assert.Equal(t, mem, utility.FromIntPtr(def.MemoryMB))
	})
	t.Run("SetCPU", func(t *testing.T) {
		cpu := 128
		def := NewECSContainerDefinition().SetCPU(cpu)
		assert.Equal(t, cpu, utility.FromIntPtr(def.CPU))
	})
	t.Run("SetEnvironmentVariables", func(t *testing.T) {
		ev := NewEnvironmentVariable().SetName("name").SetValue("value")

		def := NewECSContainerDefinition().SetEnvironmentVariables([]EnvironmentVariable{*ev})
		require.Len(t, def.EnvVars, 1)
		assert.Equal(t, *ev, def.EnvVars[0])

		def.SetEnvironmentVariables(nil)
		assert.Empty(t, def.EnvVars)
	})
	t.Run("AddEnvironmentVariables", func(t *testing.T) {
		envVars := []EnvironmentVariable{
			*NewEnvironmentVariable().SetName("name0").SetValue("value0"),
			*NewEnvironmentVariable().SetName("name1").SetValue("value1"),
		}
		def := NewECSContainerDefinition().AddEnvironmentVariables(envVars...)
		assert.ElementsMatch(t, envVars, def.EnvVars)

		def.AddEnvironmentVariables()
		assert.ElementsMatch(t, envVars, def.EnvVars)
	})
	t.Run("SetRepositoryCredentials", func(t *testing.T) {
		creds := NewRepositoryCredentials().SetName("name")
		def := NewECSContainerDefinition().SetRepositoryCredentials(*creds)
		require.NotZero(t, def.RepoCreds)
		assert.Equal(t, utility.FromStringPtr(creds.Name), utility.FromStringPtr(def.RepoCreds.Name))
	})
	t.Run("SetPortMappings", func(t *testing.T) {
		pm := NewPortMapping().SetContainerPort(1337)

		def := NewECSContainerDefinition().SetPortMappings([]PortMapping{*pm})
		require.Len(t, def.PortMappings, 1)
		assert.Equal(t, *pm, def.PortMappings[0])

		def = NewECSContainerDefinition().SetPortMappings(nil)
		assert.Empty(t, def.PortMappings)
	})
	t.Run("AddPortMappings", func(t *testing.T) {
		pms := []PortMapping{
			*NewPortMapping().SetContainerPort(1337),
			*NewPortMapping().SetContainerPort(23456),
		}
		def := NewECSContainerDefinition().AddPortMappings(pms...)
		assert.ElementsMatch(t, pms, def.PortMappings)

		def.AddPortMappings()
		assert.ElementsMatch(t, pms, def.PortMappings)
	})
	t.Run("SetLogConfiguration", func(t *testing.T) {
		lc := NewLogConfiguration().
			SetLogDriver(awsECS.LogDriverAwslogs).
			SetOptions(map[string]string{
				"awslogs-group":  "group",
				"awslogs-region": "region",
			})

		def := NewECSContainerDefinition().SetLogConfiguration(*lc)
		assert.Len(t, def.LogConfiguration.Options, 2)
		assert.Equal(t, awsECS.LogDriverAwslogs, utility.FromStringPtr(def.LogConfiguration.LogDriver))

		def = NewECSContainerDefinition().SetLogConfiguration(LogConfiguration{})
		assert.Empty(t, def.LogConfiguration)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			assert.Error(t, NewECSContainerDefinition().Validate())
		})
		t.Run("SucceedsWithJustImage", func(t *testing.T) {
			def := NewECSContainerDefinition().SetImage("image")
			assert.NoError(t, def.Validate())
		})
		t.Run("FailsWIthoutImage", func(t *testing.T) {
			def := NewECSContainerDefinition().
				SetMemoryMB(128).
				SetCPU(128)
			assert.Error(t, def.Validate())
		})
		t.Run("NameIsGenerated", func(t *testing.T) {
			def := NewECSContainerDefinition().SetImage("image")
			assert.NoError(t, def.Validate())
			assert.NotZero(t, utility.FromStringPtr(def.Name))
		})
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			ev := NewEnvironmentVariable().SetName("name").SetValue("value")
			def := NewECSContainerDefinition().
				SetImage("image").
				SetMemoryMB(128).
				SetCPU(128).
				SetCommand([]string{"echo"}).
				AddEnvironmentVariables(*ev)
			assert.NoError(t, def.Validate())
		})
		t.Run("FailsWithZeroCPU", func(t *testing.T) {
			def := NewECSContainerDefinition().
				SetImage("image").
				SetCPU(0)
			assert.Error(t, def.Validate())
		})
		t.Run("FailsWIthZeroMemory", func(t *testing.T) {
			def := NewECSContainerDefinition().
				SetImage("image").
				SetMemoryMB(0)
			assert.Error(t, def.Validate())
		})
		t.Run("FailsWithBadEnvironmentVariables", func(t *testing.T) {
			def := NewECSContainerDefinition().
				SetImage("image").
				AddEnvironmentVariables(*NewEnvironmentVariable())
			assert.Error(t, def.Validate())
		})
		t.Run("FailsWithBadRepositoryCredentials", func(t *testing.T) {
			def := NewECSContainerDefinition().
				SetImage("image").
				SetRepositoryCredentials(*NewRepositoryCredentials())
			assert.Error(t, def.Validate())
		})
		t.Run("FailsWithBadPortMapping", func(t *testing.T) {
			def := NewECSContainerDefinition().
				SetImage("image").
				AddPortMappings(*NewPortMapping())
			assert.Error(t, def.Validate())
		})
	})
}

func TestEnvironmentVariable(t *testing.T) {
	t.Run("NewEnvironmentVariable", func(t *testing.T) {
		ev := NewEnvironmentVariable()
		require.NotZero(t, ev)
		assert.Zero(t, *ev)
	})
	t.Run("SetName", func(t *testing.T) {
		name := "name"
		ev := NewEnvironmentVariable().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(ev.Name))
	})
	t.Run("SetValue", func(t *testing.T) {
		val := "value"
		ev := NewEnvironmentVariable().SetValue(val)
		assert.Equal(t, val, utility.FromStringPtr(ev.Value))
	})
	t.Run("SetSecretOptions", func(t *testing.T) {
		opts := NewSecretOptions().SetName("name").SetNewValue("value")
		ev := NewEnvironmentVariable().SetSecretOptions(*opts)
		require.NotNil(t, ev.SecretOpts)
		assert.Equal(t, utility.FromStringPtr(opts.Name), utility.FromStringPtr(ev.SecretOpts.Name))
		assert.Equal(t, utility.FromStringPtr(opts.NewValue), utility.FromStringPtr(ev.SecretOpts.NewValue))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithNameAndValue", func(t *testing.T) {
			ev := NewEnvironmentVariable().SetName("name").SetValue("value")
			assert.NoError(t, ev.Validate())
		})
		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			assert.Error(t, NewEnvironmentVariable().Validate())
		})
		t.Run("SucceedsWithNameAndNewSecretOptions", func(t *testing.T) {
			ev := NewEnvironmentVariable().
				SetName("name").
				SetSecretOptions(*NewSecretOptions().
					SetName("secret_name").
					SetNewValue("secret_value"))
			assert.NoError(t, ev.Validate())
		})
		t.Run("FailsWithNameAndBadNewSecretOptions", func(t *testing.T) {
			ev := NewEnvironmentVariable().SetName("name").SetSecretOptions(*NewSecretOptions())
			assert.Error(t, ev.Validate())
		})
		t.Run("FailsWithoutName", func(t *testing.T) {
			ev := NewEnvironmentVariable().SetValue("value")
			assert.Error(t, ev.Validate())
		})
		t.Run("FailsWithEmptyName", func(t *testing.T) {
			ev := NewEnvironmentVariable().SetName("").SetValue("value")
			assert.Error(t, ev.Validate())
		})
		t.Run("FailsWithValueAndSecretOptions", func(t *testing.T) {
			ev := NewEnvironmentVariable().
				SetName("name").
				SetValue("value").
				SetSecretOptions(*NewSecretOptions().
					SetName("secret_name").
					SetNewValue("secret_value"))
			assert.Error(t, ev.Validate())
		})
		t.Run("FailsWithoutValueOrSecretOptions", func(t *testing.T) {
			ev := NewEnvironmentVariable().SetName("name")
			assert.Error(t, ev.Validate())
		})
	})
}

func TestRepositoryCredentials(t *testing.T) {
	t.Run("NewRepositoryCredentials", func(t *testing.T) {
		creds := NewRepositoryCredentials()
		require.NotZero(t, creds)
		assert.Zero(t, *creds)
	})
	t.Run("SetName", func(t *testing.T) {
		name := "secret_name"
		creds := NewRepositoryCredentials().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(creds.Name))
	})
	t.Run("SetOwned", func(t *testing.T) {
		creds := NewRepositoryCredentials().SetOwned(true)
		assert.True(t, utility.FromBoolPtr(creds.Owned))
	})
	t.Run("SetNewCredentials", func(t *testing.T) {
		storedCreds := NewStoredRepositoryCredentials().
			SetUsername("username").
			SetPassword("password")
		creds := NewRepositoryCredentials().SetNewCredentials(*storedCreds)
		require.NotZero(t, creds.NewCreds)
		assert.Equal(t, *storedCreds, *creds.NewCreds)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithNewCredsAndName", func(t *testing.T) {
			storedCreds := NewStoredRepositoryCredentials().
				SetUsername("username").
				SetPassword("password")
			creds := NewRepositoryCredentials().
				SetName("name").
				SetNewCredentials(*storedCreds)
			assert.NoError(t, creds.Validate())
		})
		t.Run("SucceedsWithJustID", func(t *testing.T) {
			creds := NewRepositoryCredentials().SetID("id")
			assert.NoError(t, creds.Validate())
		})
		t.Run("FailsWithEmpty", func(t *testing.T) {
			creds := NewRepositoryCredentials()
			assert.Error(t, creds.Validate())
		})
		t.Run("FailsWithEmptyID", func(t *testing.T) {
			creds := NewRepositoryCredentials().SetID("")
			assert.Error(t, creds.Validate())
		})
		t.Run("FailsWithJustNewCreds", func(t *testing.T) {
			storedCreds := NewStoredRepositoryCredentials().
				SetUsername("username").
				SetPassword("password")
			creds := NewRepositoryCredentials().SetNewCredentials(*storedCreds)
			assert.Error(t, creds.Validate())
		})
		t.Run("FailsWithJustName", func(t *testing.T) {
			creds := NewRepositoryCredentials().SetName("name")
			assert.Error(t, creds.Validate())
		})
		t.Run("FailsWithBadNewCredentials", func(t *testing.T) {
			storedCreds := NewStoredRepositoryCredentials()
			creds := NewRepositoryCredentials().SetName("name").SetNewCredentials(*storedCreds)
			assert.Error(t, creds.Validate())
		})
		t.Run("FailsWithIDAndNewCreds", func(t *testing.T) {
			storedCreds := NewStoredRepositoryCredentials().
				SetUsername("username").
				SetPassword("password")
			creds := NewRepositoryCredentials().SetID("id").SetNewCredentials(*storedCreds)
			assert.Error(t, creds.Validate())
		})
	})
}

func TestStoredRepositoryCredentials(t *testing.T) {
	t.Run("SetUsername", func(t *testing.T) {
		username := "username"
		creds := NewStoredRepositoryCredentials().SetUsername(username)
		assert.Equal(t, username, utility.FromStringPtr(creds.Username))
	})
	t.Run("SetPassword", func(t *testing.T) {
		password := "password"
		creds := NewStoredRepositoryCredentials().SetPassword(password)
		assert.Equal(t, password, utility.FromStringPtr(creds.Password))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithUsernameAndPassword", func(t *testing.T) {
			creds := NewStoredRepositoryCredentials().
				SetUsername("username").
				SetPassword("password")
			assert.NoError(t, creds.Validate())
		})
		t.Run("FailsWithoutUsername", func(t *testing.T) {
			creds := NewStoredRepositoryCredentials().SetPassword("password")
			assert.Error(t, creds.Validate())
		})
		t.Run("FailsWithoutPassword", func(t *testing.T) {
			creds := NewStoredRepositoryCredentials().SetPassword("password")
			assert.Error(t, creds.Validate())
		})
	})
}

func TestKeyValue(t *testing.T) {
	t.Run("NewKeyValue", func(t *testing.T) {
		kv := NewKeyValue()
		require.NotZero(t, kv)
		assert.Zero(t, *kv)
	})
	t.Run("SetName", func(t *testing.T) {
		const name = "name"
		kv := NewKeyValue().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(kv.Name))
	})
	t.Run("SetValue", func(t *testing.T) {
		const value = "value"
		kv := NewKeyValue().SetValue(value)
		assert.Equal(t, value, utility.FromStringPtr(kv.Value))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithNameAndValue", func(t *testing.T) {
			kv := NewKeyValue().SetName("name").SetValue("value")
			assert.NoError(t, kv.Validate())
		})
		t.Run("SucceedsWithJustName", func(t *testing.T) {
			kv := NewKeyValue().SetName("name")
			assert.NoError(t, kv.Validate())
		})
		t.Run("SucceedsWithEmptyValue", func(t *testing.T) {
			kv := NewKeyValue().SetName("name").SetValue("")
			assert.NoError(t, kv.Validate())
		})
		t.Run("FailsWithJustValue", func(t *testing.T) {
			kv := NewKeyValue().SetValue("value")
			assert.Error(t, kv.Validate())
		})
		t.Run("FailsWithZero", func(t *testing.T) {
			assert.Error(t, NewKeyValue().Validate())
		})
		t.Run("FailsWithEmptyName", func(t *testing.T) {
			kv := NewKeyValue().SetName("").SetValue("value")
			assert.Error(t, kv.Validate())
		})
	})
}

func TestSecretOptions(t *testing.T) {
	t.Run("NewSecretOptions", func(t *testing.T) {
		opts := NewSecretOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetID", func(t *testing.T) {
		id := "id"
		opts := NewSecretOptions().SetID(id)
		assert.Equal(t, id, utility.FromStringPtr(opts.ID))
	})
	t.Run("SetName", func(t *testing.T) {
		name := "name"
		opts := NewSecretOptions().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(opts.Name))
	})
	t.Run("SetNewValue", func(t *testing.T) {
		val := "value"
		opts := NewSecretOptions().SetNewValue(val)
		assert.Equal(t, val, utility.FromStringPtr(opts.NewValue))
	})
	t.Run("SetOwned", func(t *testing.T) {
		opts := NewSecretOptions().SetOwned(true)
		assert.True(t, utility.FromBoolPtr(opts.Owned))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithNameAndNewValue", func(t *testing.T) {
			s := NewSecretOptions().SetName("name").SetNewValue("value")
			assert.NoError(t, s.Validate())
		})
		t.Run("SucceedsWithID", func(t *testing.T) {
			s := NewSecretOptions().SetID("id")
			assert.NoError(t, s.Validate())
		})
		t.Run("SucceedsWithIDAndName", func(t *testing.T) {
			s := NewSecretOptions().SetID("id").SetName("name")
			assert.NoError(t, s.Validate())
		})
		t.Run("FailsWithEmpty", func(t *testing.T) {
			s := NewSecretOptions()
			assert.Error(t, s.Validate())
		})
		t.Run("FailsWithEmptyID", func(t *testing.T) {
			s := NewSecretOptions().SetID("")
			assert.Error(t, s.Validate())
		})
		t.Run("FailsWithJustName", func(t *testing.T) {
			s := NewSecretOptions().SetName("name")
			assert.Error(t, s.Validate())
		})
		t.Run("FailsWithJustNewValue", func(t *testing.T) {
			s := NewSecretOptions().SetNewValue("value")
			assert.Error(t, s.Validate())
		})
		t.Run("FailsWithIDAndNewValue", func(t *testing.T) {
			s := NewSecretOptions().SetID("id").SetNewValue("value")
			assert.Error(t, s.Validate())
		})
	})
}

func TestPortMappings(t *testing.T) {
	t.Run("NewPortMapping", func(t *testing.T) {
		pm := NewPortMapping()
		require.NotZero(t, pm)
		assert.Zero(t, *pm)
	})
	t.Run("SetContainerPort", func(t *testing.T) {
		port := 1337
		pm := NewPortMapping().SetContainerPort(1337)
		assert.Equal(t, port, utility.FromIntPtr(pm.ContainerPort))
	})
	t.Run("SetHostPort", func(t *testing.T) {
		port := 1337
		pm := NewPortMapping().SetHostPort(1337)
		assert.Equal(t, port, utility.FromIntPtr(pm.HostPort))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			pm := NewPortMapping()
			assert.Error(t, pm.Validate())
		})
		t.Run("SucceedsWithJustContainerPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337)
			assert.NoError(t, pm.Validate())
		})
		t.Run("SucceedsWithContainerAndHostPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(1337).SetHostPort(1337)
			assert.NoError(t, pm.Validate())
		})
		t.Run("FailsWithNegativeContainerPort", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(-100)
			assert.Error(t, pm.Validate())
		})
		t.Run("FailsWithContainerPortAboveMax", func(t *testing.T) {
			pm := NewPortMapping().SetContainerPort(100000)
			assert.Error(t, pm.Validate())
		})
		t.Run("FailsWIthNegativeHostPort", func(t *testing.T) {
			pm := NewPortMapping().
				SetContainerPort(1337).
				SetHostPort(-100)
			assert.Error(t, pm.Validate())
		})
		t.Run("FailsWithHostPortAboveMax", func(t *testing.T) {
			pm := NewPortMapping().
				SetContainerPort(1337).
				SetHostPort(100000)
			assert.Error(t, pm.Validate())
		})
	})
}

func TestLogConfiguration(t *testing.T) {
	t.Run("NewLogConfiguration", func(t *testing.T) {
		lc := NewLogConfiguration()
		require.NotZero(t, lc)
		assert.Zero(t, *lc)
	})
	t.Run("SetLogDriver", func(t *testing.T) {
		driver := awsECS.LogDriverAwslogs
		lc := NewLogConfiguration().SetLogDriver(awsECS.LogDriverAwslogs)
		assert.Equal(t, driver, utility.FromStringPtr(lc.LogDriver))
	})
	t.Run("SetOptions", func(t *testing.T) {
		options := map[string]string{
			"awslogs-group":  "group",
			"awslogs-region": "region",
		}
		lc := NewLogConfiguration().SetOptions(map[string]string{
			"awslogs-group":  "group",
			"awslogs-region": "region",
		})
		assert.Equal(t, options, lc.Options)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			pm := NewLogConfiguration()
			assert.Error(t, pm.Validate())
		})
		t.Run("FailsWithNoDriverPopulated", func(t *testing.T) {
			lc := NewLogConfiguration().
				SetOptions(map[string]string{
					"awslogs-group":  "group",
					"awslogs-region": "region",
				})
			assert.Error(t, lc.Validate())
		})
		t.Run("FailsWithNoOptionsPopulated", func(t *testing.T) {
			lc := NewLogConfiguration().SetLogDriver(awsECS.LogDriverAwslogs)
			assert.Error(t, lc.Validate())
		})
		t.Run("FailsWithNoLogGroupOption", func(t *testing.T) {
			lc := NewLogConfiguration().
				SetLogDriver(awsECS.LogDriverAwslogs).
				SetOptions(map[string]string{
					"awslogs-region": "region",
				})
			assert.Error(t, lc.Validate())
		})
		t.Run("FailsWithNoRegionOption", func(t *testing.T) {
			lc := NewLogConfiguration().
				SetLogDriver(awsECS.LogDriverAwslogs).
				SetOptions(map[string]string{
					"awslogs-group": "group",
				})
			assert.Error(t, lc.Validate())
		})
		t.Run("SucceedsWithDriverAndOptions", func(t *testing.T) {
			lc := NewLogConfiguration().
				SetLogDriver(awsECS.LogDriverAwslogs).
				SetOptions(map[string]string{
					"awslogs-group":  "group",
					"awslogs-region": "region",
				})
			assert.NoError(t, lc.Validate())
		})
	})
}

func TestECSPodExecutionOptions(t *testing.T) {
	t.Run("NewECSPodExecutionOptions", func(t *testing.T) {
		opts := NewECSPodExecutionOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetCluster", func(t *testing.T) {
		const cluster = "cluster"
		opts := NewECSPodExecutionOptions().SetCluster(cluster)
		assert.Equal(t, cluster, utility.FromStringPtr(opts.Cluster))
	})
	t.Run("SetCapacityProvider", func(t *testing.T) {
		const provider = "capacity_provider"
		opts := NewECSPodExecutionOptions().SetCapacityProvider(provider)
		assert.Equal(t, provider, utility.FromStringPtr(opts.CapacityProvider))
	})
	t.Run("SetOverrideOptions", func(t *testing.T) {
		overrideOpts := NewECSOverridePodDefinitionOptions().
			AddContainerDefinitions(*NewECSOverrideContainerDefinition().SetCPU(512)).
			SetMemoryMB(1024).
			SetCPU(2048).
			SetTaskRole("task_role").
			SetExecutionRole("execution_role")
		opts := NewECSPodExecutionOptions().SetOverrideOptions(*overrideOpts)
		assert.Equal(t, *overrideOpts, *opts.OverrideOpts)
	})
	t.Run("SetPlacementOptions", func(t *testing.T) {
		placementOpts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack)
		opts := NewECSPodExecutionOptions().SetPlacementOptions(*placementOpts)
		require.NotZero(t, opts.PlacementOpts)
		assert.Equal(t, *placementOpts, *opts.PlacementOpts)
	})
	t.Run("SetAWSVPCOptions", func(t *testing.T) {
		awsvpcOpts := NewAWSVPCOptions().
			AddSubnets("subnet-12345").
			AddSecurityGroups("sg-12345")
		opts := NewECSPodExecutionOptions().SetAWSVPCOptions(*awsvpcOpts)
		require.NotZero(t, opts.AWSVPCOpts)
		assert.Equal(t, *awsvpcOpts, *opts.AWSVPCOpts)
	})
	t.Run("SetSupportsDebugMode", func(t *testing.T) {
		opts := NewECSPodExecutionOptions().SetSupportsDebugMode(true)
		assert.True(t, utility.FromBoolPtr(opts.SupportsDebugMode))
	})
	t.Run("SetTags", func(t *testing.T) {
		tags := map[string]string{
			"key0": "val0",
			"key1": "val1",
		}
		opts := NewECSPodExecutionOptions().SetTags(tags)
		assert.Equal(t, tags, opts.Tags)
		opts.SetTags(nil)
		assert.Empty(t, opts.Tags)
	})
	t.Run("AddTags", func(t *testing.T) {
		tags := map[string]string{
			"key0": "val0",
			"key1": "val1",
		}
		opts := NewECSPodExecutionOptions().AddTags(tags)
		assert.Equal(t, tags, opts.Tags)
		opts.AddTags(map[string]string{})
		assert.Equal(t, tags, opts.Tags)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithNoFieldsPopulated", func(t *testing.T) {
			opts := NewECSPodExecutionOptions()
			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			awsvpcOpts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			opts := NewECSPodExecutionOptions().
				SetCluster("cluster").
				SetAWSVPCOptions(*awsvpcOpts)
			assert.NoError(t, opts.Validate())
		})
		t.Run("NoPlacementOptionsAreDefaultedToBinpackMemory", func(t *testing.T) {
			opts := NewECSPodExecutionOptions()
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.PlacementOpts)
			require.NotZero(t, opts.PlacementOpts.Strategy)
			assert.Equal(t, StrategyBinpack, *opts.PlacementOpts.Strategy)
			assert.Equal(t, StrategyParamBinpackMemory, utility.FromStringPtr(opts.PlacementOpts.StrategyParameter))
		})
		t.Run("FailsWithBadPlacementOptions", func(t *testing.T) {
			placementOpts := NewECSPodPlacementOptions().SetStrategy("foo")
			opts := NewECSPodExecutionOptions().SetPlacementOptions(*placementOpts)
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithBadAWSVPCOptions", func(t *testing.T) {
			opts := NewECSPodExecutionOptions().SetAWSVPCOptions(*NewAWSVPCOptions())
			assert.Error(t, opts.Validate())
		})
	})
}

func TestECSOverridePodDefinitionOptions(t *testing.T) {
	t.Run("NewECSOverridePodDefinitionOptions", func(t *testing.T) {
		opts := NewECSOverridePodDefinitionOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetContainerDefinitions", func(t *testing.T) {
		containerDef := NewECSOverrideContainerDefinition().SetName("name")

		opts := NewECSOverridePodDefinitionOptions().SetContainerDefinitions([]ECSOverrideContainerDefinition{*containerDef})
		require.Len(t, opts.ContainerDefinitions, 1)
		assert.Equal(t, *containerDef, opts.ContainerDefinitions[0])

		opts.SetContainerDefinitions(nil)
		assert.Empty(t, opts.ContainerDefinitions)
	})
	t.Run("AddContainerDefinitions", func(t *testing.T) {
		containerDefs := []ECSOverrideContainerDefinition{
			*NewECSOverrideContainerDefinition().SetName("name0"),
			*NewECSOverrideContainerDefinition().SetName("name1"),
		}

		opts := NewECSOverridePodDefinitionOptions().AddContainerDefinitions(containerDefs...)
		require.Len(t, opts.ContainerDefinitions, 2)
		assert.ElementsMatch(t, containerDefs, opts.ContainerDefinitions)

		opts.AddContainerDefinitions()
		assert.ElementsMatch(t, containerDefs, opts.ContainerDefinitions)
	})
	t.Run("SetMemoryMB", func(t *testing.T) {
		const mem = 128
		opts := NewECSOverridePodDefinitionOptions().SetMemoryMB(mem)
		assert.Equal(t, mem, utility.FromIntPtr(opts.MemoryMB))
	})
	t.Run("SetCPU", func(t *testing.T) {
		const cpu = 128
		opts := NewECSOverridePodDefinitionOptions().SetCPU(cpu)
		assert.Equal(t, cpu, utility.FromIntPtr(opts.CPU))
	})
	t.Run("SetTaskRole", func(t *testing.T) {
		const r = "task_role"
		opts := NewECSPodDefinitionOptions().SetTaskRole(r)
		assert.Equal(t, r, utility.FromStringPtr(opts.TaskRole))
	})
	t.Run("SetExecutionRole", func(t *testing.T) {
		const r = "execution_role"
		opts := NewECSPodDefinitionOptions().SetExecutionRole(r)
		assert.Equal(t, r, utility.FromStringPtr(opts.ExecutionRole))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithZero", func(t *testing.T) {
			assert.NoError(t, NewECSOverridePodDefinitionOptions().Validate())
		})
		t.Run("SucceedsWithValidMemoryMB", func(t *testing.T) {
			assert.NoError(t, NewECSOverridePodDefinitionOptions().SetMemoryMB(1024).Validate())
		})
		t.Run("SucceedsWithValidCPU", func(t *testing.T) {
			assert.NoError(t, NewECSOverridePodDefinitionOptions().SetCPU(1024).Validate())
		})
		t.Run("FailsWithInvalidMemoryMB", func(t *testing.T) {
			assert.Error(t, NewECSOverridePodDefinitionOptions().SetMemoryMB(-30).Validate())
		})
		t.Run("FailsWithInvalidCPU", func(t *testing.T) {
			assert.Error(t, NewECSOverridePodDefinitionOptions().SetCPU(-30).Validate())
		})
		t.Run("FailsWithInvalidOverrideContainerDefinition", func(t *testing.T) {
			assert.Error(t, NewECSOverridePodDefinitionOptions().AddContainerDefinitions(*NewECSOverrideContainerDefinition()).Validate())
		})
	})
}

func TestECSOverrideContainerDefinition(t *testing.T) {
	t.Run("NewECSOverrideContainerDefinition", func(t *testing.T) {
		def := NewECSOverrideContainerDefinition()
		require.NotZero(t, def)
		assert.Zero(t, *def)
	})
	t.Run("SetName", func(t *testing.T) {
		const name = "name"
		def := NewECSOverrideContainerDefinition().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(def.Name))
	})
	t.Run("SetCommand", func(t *testing.T) {
		cmd := []string{"echo", "hello"}
		def := NewECSOverrideContainerDefinition().SetCommand(cmd)
		assert.Equal(t, cmd, def.Command)
	})
	t.Run("SetMemoryMB", func(t *testing.T) {
		const mem = 128
		def := NewECSOverrideContainerDefinition().SetMemoryMB(mem)
		assert.Equal(t, mem, utility.FromIntPtr(def.MemoryMB))
	})
	t.Run("SetCPU", func(t *testing.T) {
		const mem = 128
		def := NewECSOverrideContainerDefinition().SetCPU(mem)
		assert.Equal(t, mem, utility.FromIntPtr(def.CPU))
	})
	t.Run("SetEnvironmentVariables", func(t *testing.T) {
		envVar := NewKeyValue().SetName("name").SetValue("value")
		def := NewECSOverrideContainerDefinition().SetEnvironmentVariables([]KeyValue{*envVar})
		require.Len(t, def.EnvVars, 1)
		assert.Equal(t, *envVar, def.EnvVars[0])

		def.SetEnvironmentVariables(nil)
		assert.Empty(t, def.EnvVars)
	})
	t.Run("AddEnvironmentVariables", func(t *testing.T) {
		envVars := []KeyValue{
			*NewKeyValue().SetName("name0").SetValue("value0"),
			*NewKeyValue().SetName("name1").SetValue("value1"),
		}
		def := NewECSOverrideContainerDefinition().AddEnvironmentVariables(envVars...)
		assert.ElementsMatch(t, envVars, def.EnvVars)

		def.AddEnvironmentVariables()
		assert.ElementsMatch(t, envVars, def.EnvVars)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithJustName", func(t *testing.T) {
			assert.NoError(t, NewECSOverrideContainerDefinition().SetName("name").Validate())
		})
		t.Run("SucceedsWithValidMemoryMB", func(t *testing.T) {
			def := NewECSOverrideContainerDefinition().
				SetName("name").
				SetMemoryMB(1024)
			assert.NoError(t, def.Validate())
		})
		t.Run("SucceedsWithValidCPU", func(t *testing.T) {
			def := NewECSOverrideContainerDefinition().
				SetName("name").
				SetCPU(1024)
			assert.NoError(t, def.Validate())
		})
		t.Run("SucceedsWithValidEnvVars", func(t *testing.T) {
			def := NewECSOverrideContainerDefinition().
				SetName("name").
				AddEnvironmentVariables(*NewKeyValue().
					SetName("env_var_name").
					SetValue("env_var_value"))
			assert.NoError(t, def.Validate())
		})
		t.Run("FailsWithZero", func(t *testing.T) {
			assert.Error(t, NewECSOverrideContainerDefinition().Validate())
		})
		t.Run("FailsWithEmptyName", func(t *testing.T) {
			assert.Error(t, NewECSOverrideContainerDefinition().SetName("").Validate())
		})
		t.Run("FailsWithInvalidMemoryMB", func(t *testing.T) {
			assert.Error(t, NewECSOverrideContainerDefinition().SetName("name").SetMemoryMB(-30).Validate())
		})
		t.Run("FailsWithInvalidCPU", func(t *testing.T) {
			assert.Error(t, NewECSOverrideContainerDefinition().SetName("name").SetCPU(-30).Validate())
		})
		t.Run("FailsWithInvalidEnvVars", func(t *testing.T) {
			def := NewECSOverrideContainerDefinition().
				SetName("name").
				AddEnvironmentVariables(*NewKeyValue())
			assert.Error(t, def.Validate())
		})
	})
}

func TestECSPodPlacementOptions(t *testing.T) {
	t.Run("NewECSPodPlacementOptions", func(t *testing.T) {
		opts := NewECSPodPlacementOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetGroup", func(t *testing.T) {
		group := "group"
		opts := NewECSPodPlacementOptions().SetGroup(group)
		assert.Equal(t, group, utility.FromStringPtr(opts.Group))
	})
	t.Run("SetStrategy", func(t *testing.T) {
		strategy := StrategyBinpack
		opts := NewECSPodPlacementOptions().SetStrategy(strategy)
		require.NotZero(t, opts.Strategy)
		assert.Equal(t, strategy, *opts.Strategy)
	})
	t.Run("SetStrategyParameter", func(t *testing.T) {
		param := StrategyParamBinpackCPU
		opts := NewECSPodPlacementOptions().SetStrategyParameter(param)
		assert.Equal(t, param, utility.FromStringPtr(opts.StrategyParameter))
	})
	t.Run("SetInstanceFilters", func(t *testing.T) {
		filters := []string{"runningTasksCount == 0"}
		opts := NewECSPodPlacementOptions().SetInstanceFilters(filters)
		assert.ElementsMatch(t, filters, opts.InstanceFilters)
	})
	t.Run("AddInstanceFilters", func(t *testing.T) {
		filter := "runningTasksCount == 0"
		opts := NewECSPodPlacementOptions().AddInstanceFilters(filter)
		require.Len(t, opts.InstanceFilters, 1)
		assert.Equal(t, filter, opts.InstanceFilters[0])

		opts.AddInstanceFilters()
		require.Len(t, opts.InstanceFilters, 1)
		assert.Equal(t, filter, opts.InstanceFilters[0])
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithNoFieldsPopulated", func(t *testing.T) {
			assert.NoError(t, NewECSPodPlacementOptions().Validate())
		})
		t.Run("EmptyDefaultsToBinpackMemory", func(t *testing.T) {
			var opts ECSPodPlacementOptions
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			require.NotZero(t, opts.StrategyParameter)
			assert.Equal(t, StrategyBinpack, *opts.Strategy)
			assert.Equal(t, StrategyParamBinpackMemory, *opts.StrategyParameter)
		})
		t.Run("BinpackWithoutParameterDefaultsToMemoryBinpacking", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack)
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			assert.Equal(t, StrategyBinpack, *opts.Strategy)
			assert.Equal(t, StrategyParamBinpackMemory, utility.FromStringPtr(opts.StrategyParameter))
		})
		t.Run("SucceedsWithBinpackByMemory", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack).SetStrategyParameter(StrategyParamBinpackMemory)
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			assert.Equal(t, StrategyBinpack, *opts.Strategy)
			assert.Equal(t, StrategyParamBinpackMemory, utility.FromStringPtr(opts.StrategyParameter))
		})
		t.Run("SucceedsWithBinpackByCPU", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack).SetStrategyParameter(StrategyParamBinpackCPU)
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			assert.Equal(t, StrategyBinpack, *opts.Strategy)
			assert.Equal(t, StrategyParamBinpackCPU, utility.FromStringPtr(opts.StrategyParameter))
		})
		t.Run("FailsWithBinpackAndSpreadHostParameter", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack).SetStrategyParameter(StrategyParamSpreadHost)
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithBinpackAndInvalidParameter", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategyBinpack).SetStrategyParameter("foo")
			assert.Error(t, opts.Validate())
		})
		t.Run("SpreadWithoutParameterDefaultsToHostSpread", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategySpread)
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			assert.Equal(t, StrategySpread, *opts.Strategy)
			assert.Equal(t, StrategyParamSpreadHost, utility.FromStringPtr(opts.StrategyParameter))
		})
		t.Run("SucceedsWithSpreadingByHost", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategySpread).SetStrategyParameter(StrategyParamSpreadHost)
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			assert.Equal(t, StrategySpread, *opts.Strategy)
			assert.Equal(t, StrategyParamSpreadHost, utility.FromStringPtr(opts.StrategyParameter))
		})
		t.Run("SucceedsWithSpreadingByCustomParameter", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetStrategy(StrategySpread).SetStrategyParameter("custom")
			require.NoError(t, opts.Validate())
			require.NotZero(t, opts.Strategy)
			assert.Equal(t, StrategySpread, *opts.Strategy)
			assert.Equal(t, "custom", utility.FromStringPtr(opts.StrategyParameter))
		})
		t.Run("SucceedsWithNonemptyGroupName", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetGroup("group")
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithEmptyGroupName", func(t *testing.T) {
			opts := NewECSPodPlacementOptions().SetGroup("")
			assert.Error(t, opts.Validate())
		})
	})
}

func TestAWSVPCOptions(t *testing.T) {
	t.Run("NewAWSVPCOptions", func(t *testing.T) {
		opts := NewAWSVPCOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetSubnets", func(t *testing.T) {
		subnets := []string{"subnet-12345", "subnet-67890"}
		opts := NewAWSVPCOptions().SetSubnets(subnets)
		assert.ElementsMatch(t, subnets, opts.Subnets)
		opts.SetSubnets(nil)
		assert.Empty(t, opts.Subnets)
	})
	t.Run("AddSubnets", func(t *testing.T) {
		subnets := []string{"subnet-12345", "subnet-67890"}
		opts := NewAWSVPCOptions().AddSubnets(subnets...)
		assert.ElementsMatch(t, subnets, opts.Subnets)
		opts.AddSubnets()
		assert.ElementsMatch(t, subnets, opts.Subnets)
	})
	t.Run("SetSecurityGroups", func(t *testing.T) {
		groups := []string{"sg-12345", "sg-67890"}
		opts := NewAWSVPCOptions().SetSecurityGroups(groups)
		assert.ElementsMatch(t, groups, opts.SecurityGroups)
		opts.SetSecurityGroups(nil)
		assert.Empty(t, opts.SecurityGroups)
	})
	t.Run("AddSecurityGroups", func(t *testing.T) {
		groups := []string{"sg-12345", "sg-67890"}
		opts := NewAWSVPCOptions().AddSecurityGroups(groups...)
		assert.ElementsMatch(t, groups, opts.SecurityGroups)
		opts.AddSecurityGroups()
		assert.ElementsMatch(t, groups, opts.SecurityGroups)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			opts := NewAWSVPCOptions().
				AddSubnets("subnet-12345").
				AddSecurityGroups("sg-12345")
			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithJustSubnets", func(t *testing.T) {
			opts := NewAWSVPCOptions().AddSubnets("subnet-12345")
			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			opts := NewAWSVPCOptions()
			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithoutSubnets", func(t *testing.T) {
			opts := NewAWSVPCOptions().AddSecurityGroups("sg-12345")
			assert.Error(t, opts.Validate())
		})
	})
}

func TestECSTaskDefinition(t *testing.T) {
	t.Run("NewECSTaskDefinition", func(t *testing.T) {
		def := NewECSTaskDefinition()
		require.NotZero(t, def)
		assert.Zero(t, *def)
	})
	t.Run("SetID", func(t *testing.T) {
		id := "id"
		def := NewECSTaskDefinition().SetID(id)
		assert.Equal(t, id, utility.FromStringPtr(def.ID))
	})
	t.Run("SetOwned", func(t *testing.T) {
		def := NewECSTaskDefinition().SetOwned(true)
		assert.True(t, utility.FromBoolPtr(def.Owned))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithAllFieldsPopulated", func(t *testing.T) {
			def := NewECSTaskDefinition().SetID("id").SetOwned(true)
			assert.NoError(t, def.Validate())
		})
		t.Run("SucceedsWithJustTaskDefinitionID", func(t *testing.T) {
			def := NewECSTaskDefinition().SetID("id")
			assert.NoError(t, def.Validate())
		})
		t.Run("FailsWithNoFieldsPopulated", func(t *testing.T) {
			assert.Error(t, NewECSTaskDefinition().Validate())
		})
		t.Run("FailsWithoutTaskDefinitionID", func(t *testing.T) {
			def := NewECSTaskDefinition().SetOwned(true)
			assert.Error(t, def.Validate())
		})
	})
}
