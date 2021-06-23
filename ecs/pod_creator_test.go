package ecs

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/stretchr/testify/assert"
)

func TestECSPodCreatorInterface(t *testing.T) {
	assert.Implements(t, (*cocoa.ECSPodCreator)(nil), &BasicECSPodCreator{})
}
