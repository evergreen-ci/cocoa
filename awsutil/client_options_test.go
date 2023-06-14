package awsutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/credentials"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientOptions(t *testing.T) {
	t.Run("SetCredentials", func(t *testing.T) {
		creds := credentials.NewEnvCredentials()
		opts := NewClientOptions().SetCredentials(creds)
		assert.Equal(t, creds, opts.Creds)
	})
	t.Run("SetRole", func(t *testing.T) {
		role := "role"
		opts := NewClientOptions().SetRole(role)
		require.NotNil(t, opts.Role)
		assert.Equal(t, role, *opts.Role)
	})
	t.Run("SetRegion", func(t *testing.T) {
		region := "region"
		opts := NewClientOptions().SetRegion(region)
		require.NotNil(t, opts.Region)
		assert.Equal(t, region, *opts.Region)
	})
	t.Run("SetRetryOptions", func(t *testing.T) {
		retryOpts := utility.RetryOptions{
			MaxAttempts: 10,
			MinDelay:    100 * time.Millisecond,
			MaxDelay:    time.Second,
		}
		opts := NewClientOptions().SetRetryOptions(retryOpts)
		require.NotNil(t, opts.RetryOpts)
		assert.Equal(t, retryOpts, *opts.RetryOpts)
	})
	t.Run("SetHTTPClient", func(t *testing.T) {
		hc := http.DefaultClient
		opts := NewClientOptions().SetHTTPClient(hc)
		require.NotNil(t, opts.HTTPClient)
		assert.Equal(t, hc, opts.HTTPClient)
		assert.False(t, opts.ownsHTTPClient)
	})
	t.Run("Validate", func(t *testing.T) {
		t.Run("SucceedsWithAllOptionSet", func(t *testing.T) {
			creds := credentials.NewEnvCredentials()
			role := "role"
			region := "region"
			retryOpts := utility.RetryOptions{
				MaxAttempts: 10,
				MinDelay:    100 * time.Millisecond,
				MaxDelay:    time.Second,
			}
			hc := http.DefaultClient
			opts := NewClientOptions().
				SetCredentials(creds).
				SetRole(role).
				SetRegion(region).
				SetRetryOptions(retryOpts).
				SetHTTPClient(hc)

			require.NoError(t, opts.Validate())

			assert.Equal(t, creds, opts.Creds)
			assert.Equal(t, region, *opts.Region)
			assert.Equal(t, role, *opts.Role)
			assert.Equal(t, retryOpts, *opts.RetryOpts)
			assert.Equal(t, hc, opts.HTTPClient)
			assert.False(t, opts.ownsHTTPClient)
		})
		t.Run("SucceedsWithoutCredentialsWhenRoleIsGiven", func(t *testing.T) {
			role := "role"
			region := "region"
			retryOpts := utility.RetryOptions{
				MaxAttempts: 10,
				MinDelay:    100 * time.Millisecond,
				MaxDelay:    time.Second,
			}
			hc := http.DefaultClient
			opts := NewClientOptions().
				SetRole(role).
				SetRegion(region).
				SetRetryOptions(retryOpts).
				SetHTTPClient(hc)

			assert.NoError(t, opts.Validate())
		})
		t.Run("SucceedsWithoutRoleWhenCredentialsAreGiven", func(t *testing.T) {
			creds := credentials.NewEnvCredentials()
			region := "region"
			retryOpts := utility.RetryOptions{
				MaxAttempts: 10,
				MinDelay:    100 * time.Millisecond,
				MaxDelay:    time.Second,
			}
			hc := http.DefaultClient
			opts := NewClientOptions().
				SetCredentials(creds).
				SetRegion(region).
				SetRetryOptions(retryOpts).
				SetHTTPClient(hc)

			assert.NoError(t, opts.Validate())
		})
		t.Run("FailsWithNeitherCredentialsNorRoleAreGiven", func(t *testing.T) {
			region := "region"
			retryOpts := utility.RetryOptions{
				MaxAttempts: 10,
				MinDelay:    100 * time.Millisecond,
				MaxDelay:    time.Second,
			}
			hc := http.DefaultClient
			opts := NewClientOptions().
				SetRegion(region).
				SetRetryOptions(retryOpts).
				SetHTTPClient(hc)

			assert.Error(t, opts.Validate())
		})
		t.Run("FailsWithoutRegion", func(t *testing.T) {
			creds := credentials.NewEnvCredentials()
			role := "role"
			retryOpts := utility.RetryOptions{
				MaxAttempts: 10,
				MinDelay:    100 * time.Millisecond,
				MaxDelay:    time.Second,
			}
			hc := http.DefaultClient
			opts := NewClientOptions().
				SetCredentials(creds).
				SetRole(role).
				SetRetryOptions(retryOpts).
				SetHTTPClient(hc)

			assert.Error(t, opts.Validate())
		})
		t.Run("DefaultsHTTPClient", func(t *testing.T) {
			creds := credentials.NewEnvCredentials()
			role := "role"
			region := "region"
			retryOpts := utility.RetryOptions{
				MaxAttempts: 10,
				MinDelay:    100 * time.Millisecond,
				MaxDelay:    time.Second,
			}
			opts := NewClientOptions().
				SetCredentials(creds).
				SetRole(role).
				SetRegion(region).
				SetRetryOptions(retryOpts)

			require.NoError(t, opts.Validate())
			defer opts.Close()

			assert.Equal(t, creds, opts.Creds)
			assert.Equal(t, region, *opts.Region)
			assert.Equal(t, role, *opts.Role)
			assert.Equal(t, retryOpts, *opts.RetryOpts)
			assert.NotZero(t, opts.HTTPClient)
			assert.True(t, opts.ownsHTTPClient)
		})
	})
}
