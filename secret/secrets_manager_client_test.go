package secret

import (
	"context"
	"testing"
	"time"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultTestTimeout is the standard timeout for integration tests against
// Secrets Manager.
const defaultTestTimeout = time.Minute

func TestBasicSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*cocoa.SecretsManagerClient)(nil), &BasicSecretsManagerClient{})

	testutil.CheckAWSEnvVarsForSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	awsOpts := testutil.ValidIntegrationAWSOptions()
	c, err := NewBasicSecretsManagerClient(ctx, awsOpts)
	require.NoError(t, err)
	defer func() {
		testutil.CleanupSecrets(ctx, t, c)
	}()

	for tName, tCase := range testcase.SecretsManagerClientTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			tCase(tctx, t, c)
		})
	}

}
