package secret

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/internal/awsutil"
	"github.com/evergreen-ci/cocoa/internal/testcase"
	"github.com/evergreen-ci/cocoa/internal/testutil"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*cocoa.SecretsManagerClient)(nil), &BasicSecretsManagerClient{})

	testutil.CheckAWSEnvVarsForSecretsManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for tName, tCase := range testcase.SecretsManagerClientTests() {
		t.Run(tName, func(t *testing.T) {
			tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
			defer tcancel()

			hc := utility.GetHTTPClient()
			defer utility.PutHTTPClient(hc)

			c, err := NewBasicSecretsManagerClient(awsutil.ClientOptions{
				Creds:  credentials.NewEnvCredentials(),
				Region: aws.String(os.Getenv("AWS_REGION")),
				Role:   aws.String(os.Getenv("AWS_ROLE")),
				RetryOpts: &utility.RetryOptions{
					MaxAttempts: 5,
				},
				HTTPClient: hc,
			})
			require.NoError(t, err)
			require.NotNil(t, c)

			defer c.Close(tctx)

			tCase(tctx, t, c)
		})
	}

}

// func checkAWSEnvVars(t *testing.T) {
//     missing := []string{}
//
//     for _, envVar := range []string{
//         "AWS_ACCESS_KEY",
//         "AWS_SECRET_ACCESS_KEY",
//         "AWS_SECRET_PREFIX",
//         "AWS_ROLE",
//         "AWS_REGION",
//     } {
//         if os.Getenv(envVar) == "" {
//             missing = append(missing, envVar)
//         }
//     }
//
//     if len(missing) > 0 {
//         assert.FailNow(t, fmt.Sprintf("missing required AWS environment variables: %s", missing))
//     }
// }
