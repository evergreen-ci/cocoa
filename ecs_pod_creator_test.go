package cocoa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECSPodCreator(t *testing.T) {
	assert.Implements(t, (*ECSPodCreator)(nil), &BasicECSPodCreator{})
}
