package cocoa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECSPod(t *testing.T) {
	assert.Implements(t, (*ECSPod)(nil), &BasicECSPod{})
}
