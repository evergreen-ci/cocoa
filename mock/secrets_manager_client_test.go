package mock

import (
	"context"
	"testing"
	"time"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
)

func TestSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*cocoa.SecretsManagerClient)(nil), &SecretsManagerClient{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range testcase.SecretsManagerClientTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			c := &SecretsManagerClient{}
			defer c.Close(tctx)

			tCase(tctx, t, c)
		})
	}
}
