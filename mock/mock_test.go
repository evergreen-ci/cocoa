package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/stretchr/testify/assert"
)

func TestMockSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodManager)(nil), &MockECSPodManager{})
	assert.Implements(t, (*cocoa.ECSPod)(nil), &MockECSPod{})
	assert.Implements(t, (*cocoa.ECSClient)(nil), &MockECSClient{})

	assert.Implements(t, (*secret.Vault)(nil), &MockVault{})
	assert.Implements(t, (*secret.SecretsManagerClient)(nil), &MockSecretsManagerClient{})
}
