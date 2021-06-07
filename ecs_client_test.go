package cocoa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECSClient(t *testing.T) {
	assert.Implements(t, (*ECSClient)(nil), &BasicECSClient{})
}
