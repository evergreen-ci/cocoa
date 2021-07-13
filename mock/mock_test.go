package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/stretchr/testify/assert"
)

func TestInterfaces(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPod)(nil), &ECSPod{})
}
