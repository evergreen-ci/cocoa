package ecs

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/evergreen-ci/utility"
	"github.com/stretchr/testify/assert"
)

func TestTranslateECSStatus(t *testing.T) {
	t.Run("ReturnsUnknownForNilStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusUnknown, TranslateECSStatus(nil))
	})
	t.Run("ReturnsUnknownForUnrecognizedStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusUnknown, TranslateECSStatus(utility.ToStringPtr("foo")))
	})
	t.Run("ReturnsUnknownForEmptyStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusUnknown, TranslateECSStatus(utility.ToStringPtr("")))
	})
	t.Run("ReturnsStartingForStatusesBeforeRunning", func(t *testing.T) {
		for _, status := range []string{TaskStatusProvisioning, TaskStatusPending, TaskStatusActivating} {
			assert.Equal(t, cocoa.StatusStarting, TranslateECSStatus(utility.ToStringPtr(status)))
		}
	})
	t.Run("ReturnsStartingForStatusesInBetweenRunningAndStopped", func(t *testing.T) {
		for _, status := range []string{TaskStatusDeactivating, TaskStatusStopping, TaskStatusDeprovisioning} {
			assert.Equal(t, cocoa.StatusStopping, TranslateECSStatus(utility.ToStringPtr(status)))
		}
	})
	t.Run("ReturnsStartingForStatusesInBetweenRunningAndStopped", func(t *testing.T) {
		for _, status := range []string{TaskStatusDeactivating, TaskStatusStopping, TaskStatusDeprovisioning} {
			assert.Equal(t, cocoa.StatusStopping, TranslateECSStatus(utility.ToStringPtr(status)))
		}
	})
	t.Run("ReturnsStartingForStoppedStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusStopped, TranslateECSStatus(utility.ToStringPtr(TaskStatusStopped)))
	})
}
