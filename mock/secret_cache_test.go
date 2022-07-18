package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/stretchr/testify/assert"
)

func TestSecretCache(t *testing.T) {
	assert.Implements(t, (*cocoa.SecretCache)(nil), &SecretCache{})
}
