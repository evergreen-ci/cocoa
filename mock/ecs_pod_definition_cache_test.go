package mock

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/stretchr/testify/assert"
)

func TestECSPodDefinitionCache(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodDefinitionCache)(nil), &ECSPodDefinitionCache{})
}
