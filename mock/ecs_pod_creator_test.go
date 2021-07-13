package mock

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/ecs"
	"github.com/evergreen-ci/cocoa/internal/awsutil"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECSPodCreator(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodCreator)(nil), &ECSPodCreator{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range testcase.ECSPodCreatorNoVaultTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, time.Second)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			awsOpts := awsutil.NewClientOptions().
				SetHTTPClient(hc).
				SetCredentials(credentials.NewEnvCredentials()).
				SetRole(testutil.AWSRole()).
				SetRegion(testutil.AWSRegion())

			c, err := ecs.NewBasicECSClient(*awsOpts)
			require.NoError(t, err)

			defer func() {
				assert.NoError(t, c.Close(ctx))
			}()

			pc, err := ecs.NewBasicECSPodCreator(c, nil)
			require.NoError(t, err)

			podCreator := NewECSPodCreator(pc)

			tCase(tctx, t, podCreator)
		})
	}

	for tName, tCase := range testcase.ECSPodCreatorTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, time.Second)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			awsOpts := awsutil.NewClientOptions().
				SetHTTPClient(hc).
				SetCredentials(credentials.NewEnvCredentials()).
				SetRole(testutil.AWSRole()).
				SetRegion(testutil.AWSRegion())

			c, err := ecs.NewBasicECSClient(*awsOpts)
			require.NoError(t, err)

			defer func() {
				assert.NoError(t, c.Close(ctx))
			}()

			sm := &SecretsManagerClient{}
			defer func() {
				assert.NoError(t, sm.Close(ctx))
			}()

			v := NewVault(secret.NewBasicSecretsManager(sm))

			pc, err := ecs.NewBasicECSPodCreator(c, v)
			require.NoError(t, err)

			podCreator := NewECSPodCreator(pc)

			tCase(tctx, t, podCreator)
		})
	}

}
