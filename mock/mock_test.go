package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/stretchr/testify/assert"
)

func TestMockSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSClient)(nil), &MockECSClient{})
	assert.Implements(t, (*secret.SecretsManagerClient)(nil), &MockSecretsManagerClient{})
}
