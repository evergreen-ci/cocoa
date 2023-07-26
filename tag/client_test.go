package tag

import (
	"context"
	"testing"
	"time"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultTestTimeout is the standard timeout for integration tests against
// the Resource Groups Tagging API.
const defaultTestTimeout = time.Minute

func TestBasicTagClient(t *testing.T) {
	assert.Implements(t, (*cocoa.TagClient)(nil), &BasicTagClient{})

	testutil.CheckAWSEnvVarsForSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := utility.GetHTTPClient()
	defer utility.PutHTTPClient(hc)

	awsOpts, err := testutil.ValidIntegrationAWSOptions(ctx, hc)
	require.NoError(t, err)

	c, err := NewBasicTagClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, c.Close(ctx))
	}()

	for tName, tCase := range testcase.TagClientTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			tCase(tctx, t, c)
		})
	}

	smClient, err := secret.NewBasicSecretsManagerClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupSecrets(ctx, t, smClient)

		assert.NoError(t, smClient.Close(ctx))
	}()

	for tName, tCase := range testcase.TagClientSecretTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			tCase(tctx, t, c, smClient)
		})
	}
}
