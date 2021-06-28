package secret

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/awsutil"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultInterface(t *testing.T) {
	assert.Implements(t, (*cocoa.Vault)(nil), &BasicSecretsManager{})
}

func TestSecretsManager(t *testing.T) {
	testutil.CheckAWSEnvVarsForSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupSecret := func(ctx context.Context, t *testing.T, m *BasicSecretsManager, id string) {
		if id != "" {
			require.NoError(t, m.DeleteSecret(ctx, id))
		}
	}

	for tName, tCase := range map[string]func(context.Context, *testing.T, *BasicSecretsManager){
		"CreateFailsWithInvalidInput": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			out, err := m.CreateSecret(ctx, cocoa.NamedSecret{})
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"DeleteFailsWithInvalidInput": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			err := m.DeleteSecret(ctx, "")
			assert.Error(t, err)
		},
		"DeleteSecretWithExistingSecretSucceeds": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			out, err := m.CreateSecret(ctx, cocoa.NamedSecret{
				Name:  aws.String(testutil.NewSecretName(t.Name())),
				Value: aws.String("hello")})

			require.NoError(t, err)
			require.NotZero(t, out)

			defer cleanupSecret(ctx, t, m, out)
		},
		"GetValueFailsWithInvalidInput": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			out, err := m.GetValue(ctx, "")
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"UpdateFailsWithInvalidInput": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			err := m.UpdateValue(ctx, "", "")
			assert.Error(t, err)
		},
		"GetValueWithExistingSecretSucceeds": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			out, err := m.CreateSecret(ctx, cocoa.NamedSecret{
				Name:  aws.String(testutil.NewSecretName(t.Name())),
				Value: aws.String("eggs")})

			require.NoError(t, err)
			require.NotZero(t, out)

			defer cleanupSecret(ctx, t, m, out)

			if out != "" {
				out, err := m.GetValue(ctx, out)
				require.NoError(t, err)
				require.NotZero(t, out)
				assert.Equal(t, "eggs", out)
			}
		},
		"UpdateValueSucceeds": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			out, err := m.CreateSecret(ctx, cocoa.NamedSecret{
				Name:  aws.String(testutil.NewSecretName(t.Name())),
				Value: aws.String("eggs"),
			})
			require.NoError(t, err)
			require.NotZero(t, out)

			defer cleanupSecret(ctx, t, m, out)

			if out != "" {
				err := m.UpdateValue(ctx, out, "ham")
				require.NoError(t, err)
			}

			if out != "" {
				out, err := m.GetValue(ctx, out)
				require.NoError(t, err)
				require.NotZero(t, out)
				assert.Equal(t, "ham", out)
			}
		},
		"DeleteSecretWithValidNonexistentInputWillNoop": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			err := m.DeleteSecret(ctx, testutil.NewSecretName(t.Name()))
			assert.NoError(t, err)
		},
		"GetValueWithValidNonexistentInputFails": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			out, err := m.GetValue(ctx, testutil.NewSecretName(t.Name()))
			assert.Error(t, err)
			assert.Zero(t, out)
		},
		"UpdateValueWithValidNonexistentInputFails": func(ctx context.Context, t *testing.T, m *BasicSecretsManager) {
			err := m.UpdateValue(ctx, testutil.NewSecretName(t.Name()), "leaf")
			assert.Error(t, err)
		},
	} {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			c, err := NewBasicSecretsManagerClient(awsutil.ClientOptions{
				Creds:  credentials.NewEnvCredentials(),
				Region: aws.String(testutil.AWSRegion()),
				Role:   aws.String(testutil.AWSRole()),
				RetryOpts: &utility.RetryOptions{
					MaxAttempts: 5,
				},
				HTTPClient: hc,
			})
			require.NoError(t, err)
			require.NotNil(t, c)

			m := NewBasicSecretsManager(c)
			require.NotNil(t, m)

			tCase(tctx, t, m)
		})
	}
}
