package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretsManagerClient(t *testing.T) {
	assert.Implements(t, (*SecretsManagerClient)(nil), &BasicSecretsManagerClient{})
}
