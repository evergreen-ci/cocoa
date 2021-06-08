package cocoa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECSPodManager(t *testing.T) {
	assert.Implements(t, (*ECSPodManager)(nil), &BasicECSPodManager{})
}
