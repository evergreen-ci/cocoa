package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/cocoa/secret"
	"github.com/stretchr/testify/assert"
)

func TestInterfaces(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodCreator)(nil), &ECSPodCreator{})
	assert.Implements(t, (*cocoa.ECSClient)(nil), &ECSClient{})

	assert.Implements(t, (*secret.Vault)(nil), &Vault{})
	assert.Implements(t, (*secret.SecretsManagerClient)(nil), &SecretsManagerClient{})
}
