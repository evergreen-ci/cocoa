package cocoa

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/evergreen-ci/cocoa/awsutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSPod(t *testing.T) {
	assert.Implements(t, (*ECSPod)(nil), &BasicECSPod{})
}

func TestBasicECSPodOptions(t *testing.T) {
	t.Run("NewBasicECSPodOptions", func(t *testing.T) {
		opts := NewBasicECSPodOptions()
		require.NotZero(t, opts)
		assert.Zero(t, *opts)
	})
	t.Run("SetClient", func(t *testing.T) {
		c, err := NewBasicECSClient(*awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1"))
		require.NoError(t, err)
		opts := NewBasicECSPodOptions().SetClient(c)
		assert.Equal(t, c, opts.Client)
	})
	t.Run("SetVault", func(t *testing.T) {
		c, err := secret.NewBasicSecretsManagerClient(*awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1"))
		require.NoError(t, err)
		v := secret.NewBasicSecretsManager(c)
		opts := NewBasicECSPodOptions().SetVault(v)
		assert.Equal(t, v, opts.Vault)
	})
	t.Run("SetResources", func(t *testing.T) {
		res := NewECSPodResources().SetTaskID("id")
		opts := NewBasicECSPodOptions().SetResources(*res)
		require.NotZero(t, opts.Resources)
		assert.Equal(t, *res, *opts.Resources)
	})
	t.Run("SetStatus", func(t *testing.T) {
		stat := Starting
		opts := NewBasicECSPodOptions().SetStatus(stat)
		assert.Equal(t, stat, utility.FromStringPtr(opts.Status))
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("EmptyIsInvalid", func(t *testing.T) {
			opts := NewBasicECSPodOptions()
			assert.Error(t, opts.Validate())
		})
		t.Run("AllFieldsPopulatedIsValid", func(t *testing.T) {
			awsOpts := awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1")
			ecsClient, err := NewBasicECSClient(*awsOpts)
			require.NoError(t, err)
			smClient, err := secret.NewBasicSecretsManagerClient(*awsOpts)
			require.NoError(t, err)
			v := secret.NewBasicSecretsManager(smClient)
			res := NewECSPodResources().SetTaskID("id")
			opts := NewBasicECSPodOptions().
				SetClient(ecsClient).
				SetVault(v).
				SetResources(*res).
				SetStatus(Starting)
			assert.NoError(t, opts.Validate())
		})
		t.Run("MissingClientIsInvalid", func(t *testing.T) {
			awsOpts := awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1")
			smClient, err := secret.NewBasicSecretsManagerClient(*awsOpts)
			require.NoError(t, err)
			v := secret.NewBasicSecretsManager(smClient)
			res := NewECSPodResources().SetTaskID("id")
			opts := NewBasicECSPodOptions().
				SetVault(v).
				SetResources(*res).
				SetStatus(Starting)
			assert.Error(t, opts.Validate())
		})
		t.Run("MissingVaultIsValid", func(t *testing.T) {
			awsOpts := awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1")
			ecsClient, err := NewBasicECSClient(*awsOpts)
			require.NoError(t, err)
			res := NewECSPodResources().SetTaskID("id")
			opts := NewBasicECSPodOptions().
				SetClient(ecsClient).
				SetResources(*res).
				SetStatus(Starting)
			assert.NoError(t, opts.Validate())
		})
		t.Run("MissingResourcesIsInvalid", func(t *testing.T) {
			awsOpts := awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1")
			smClient, err := secret.NewBasicSecretsManagerClient(*awsOpts)
			require.NoError(t, err)
			v := secret.NewBasicSecretsManager(smClient)
			res := NewECSPodResources()
			opts := NewBasicECSPodOptions().
				SetVault(v).
				SetResources(*res).
				SetStatus(Starting)
			assert.Error(t, opts.Validate())
		})
		t.Run("BadResourcesIsInvalid", func(t *testing.T) {
			awsOpts := awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1")
			smClient, err := secret.NewBasicSecretsManagerClient(*awsOpts)
			require.NoError(t, err)
			v := secret.NewBasicSecretsManager(smClient)
			opts := NewBasicECSPodOptions().
				SetVault(v).
				SetStatus(Starting)
			assert.Error(t, opts.Validate())
		})
		t.Run("MissingStatusIsInvalid", func(t *testing.T) {
			awsOpts := awsutil.NewClientOptions().SetCredentials(credentials.NewEnvCredentials()).SetRegion("us-east-1")
			ecsClient, err := NewBasicECSClient(*awsOpts)
			require.NoError(t, err)
			smClient, err := secret.NewBasicSecretsManagerClient(*awsOpts)
			require.NoError(t, err)
			v := secret.NewBasicSecretsManager(smClient)
			res := NewECSPodResources().SetTaskID("id")
			opts := NewBasicECSPodOptions().
				SetClient(ecsClient).
				SetVault(v).
				SetResources(*res)
			assert.Error(t, opts.Validate())
		})
	})
}

func TestPodSecret(t *testing.T) {
	t.Run("NewPodSecret", func(t *testing.T) {
		s := NewPodSecret()
		require.NotZero(t, s)
		assert.Zero(t, *s)
	})
	t.Run("SetName", func(t *testing.T) {
		name := "name"
		s := NewPodSecret().SetName(name)
		assert.Equal(t, name, utility.FromStringPtr(s.Name))
	})
	t.Run("SetValue", func(t *testing.T) {
		val := "val"
		s := NewPodSecret().SetValue(val)
		assert.Equal(t, val, utility.FromStringPtr(s.Value))
	})
}

func TestECSPodResources(t *testing.T) {
	t.Run("NewECSPodResources", func(t *testing.T) {
		res := NewECSPodResources()
		require.NotZero(t, res)
		assert.Zero(t, *res)
	})
	t.Run("SetTaskID", func(t *testing.T) {
		id := "id"
		res := NewECSPodResources().SetTaskID(id)
		assert.Equal(t, id, utility.FromStringPtr(res.TaskID))
	})
	t.Run("SetTaskDefinition", func(t *testing.T) {
		def := NewECSTaskDefinition().SetID("id")
		res := NewECSPodResources().SetTaskDefinition(*def)
		require.NotZero(t, res.TaskDefinition)
		assert.Equal(t, *def, *res.TaskDefinition)
	})
	t.Run("SetSecrets", func(t *testing.T) {
		s := NewPodSecret().SetName("name").SetValue("value")
		res := NewECSPodResources().SetSecrets([]PodSecret{*s})
		require.Len(t, res.Secrets, 1)
		assert.Equal(t, *s, res.Secrets[0])
	})
	t.Run("AddSecrets", func(t *testing.T) {
		s0 := NewPodSecret().SetName("name0").SetValue("value0")
		s1 := NewPodSecret().SetName("name1").SetValue("value1")
		res := NewECSPodResources().AddSecrets(*s0, *s1)
		require.Len(t, res.Secrets, 2)
		assert.Equal(t, *s0, res.Secrets[0])
		assert.Equal(t, *s1, res.Secrets[1])
	})
}
