package mock

import (
	"context"
	"testing"
	"time"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// defaultTestTimeout is the default test timeout for mock tests.
const defaultTestTimeout = time.Second

func TestECSClient(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSClient)(nil), &ECSClient{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := &ECSClient{}
	defer func() {
		resetECSAndSecretsManagerCache()

		assert.NoError(t, c.Close(ctx))
	}()

	for tName, tCase := range testcase.ECSClientTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			resetECSAndSecretsManagerCache()

			tCase(tctx, t, c)
		})
	}

	for tName, tCase := range testcase.ECSClientRegisteredTaskDefinitionTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, defaultTestTimeout)
			defer tcancel()

			registerOut := testutil.RegisterTaskDefinition(ctx, t, c, testutil.ValidRegisterTaskDefinitionInput(t))
			tCase(tctx, t, c, *registerOut.TaskDefinition)
		})
	}
}
