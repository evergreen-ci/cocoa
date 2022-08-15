package mock

import (
	"context"
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestTagClient(t *testing.T) {
	assert.Implements(t, (*cocoa.TagClient)(nil), &TagClient{})

	testutil.CheckAWSEnvVarsForSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := &TagClient{}
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

	smClient := &SecretsManagerClient{}
	defer func() {
		ResetGlobalSecretCache()

		assert.NoError(t, smClient.Close(ctx))
	}()

	for tName, tCase := range testcase.TagClientSecretTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			ResetGlobalSecretCache()

			tCase(tctx, t, c, smClient)
		})
	}
}
