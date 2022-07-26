package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/stretchr/testify/assert"
)

func TestVault(t *testing.T) {
	assert.Implements(t, (*cocoa.Vault)(nil), &Vault{})
}
