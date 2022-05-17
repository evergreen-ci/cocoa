package ecs

import (
	"testing"

	"github.com/evergreen-ci/cocoa"
	"github.com/stretchr/testify/assert"
)

func TestTaskStatusToCocoaStatus(t *testing.T) {
	t.Run("ReturnsStartingForStatusesBeforeRunning", func(t *testing.T) {
		for _, s := range []TaskStatus{TaskStatusProvisioning, TaskStatusPending, TaskStatusActivating} {
			assert.Equal(t, cocoa.StatusStarting, s.ToCocoaStatus())
		}
	})
	t.Run("ReturnsRunningForRunningStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusRunning, TaskStatusRunning.ToCocoaStatus())
	})
	t.Run("ReturnsStartingForStatusesBetweenRunningAndStopped", func(t *testing.T) {
		for _, s := range []TaskStatus{TaskStatusDeactivating, TaskStatusStopping, TaskStatusDeprovisioning} {
			assert.Equal(t, cocoa.StatusStopping, s.ToCocoaStatus())
		}
	})
	t.Run("ReturnsStoppedForStoppedStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusStopped, TaskStatusStopped.ToCocoaStatus())
	})
	t.Run("ReturnsUnknownForEmptyStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusUnknown, TaskStatus("").ToCocoaStatus())
	})
	t.Run("ReturnsUnknownForUnrecognizedStatus", func(t *testing.T) {
		assert.Equal(t, cocoa.StatusUnknown, TaskStatus("foo").ToCocoaStatus())
	})
}

func TestTaskStatusBefore(t *testing.T) {
	t.Run("ReturnsTrueWhenComparedAgainstLaterStatus", func(t *testing.T) {
		assert.True(t, TaskStatusRunning.Before(TaskStatusDeprovisioning))
	})
	t.Run("ReturnsFalseWhenComparedAgainstEarlierStatus", func(t *testing.T) {
		assert.False(t, TaskStatusActivating.Before(TaskStatusProvisioning))
	})
	t.Run("IsNotSelfInclusive", func(t *testing.T) {
		assert.False(t, TaskStatusStopping.Before(TaskStatusStopping))
	})
}

func TestTaskStatusAfter(t *testing.T) {
	t.Run("ReturnsFalseWhenComparedAgainstLaterStatus", func(t *testing.T) {
		assert.False(t, TaskStatusDeprovisioning.After(TaskStatusStopped))
	})
	t.Run("ReturnsTrueWhenComparedAgainstEarlierStatus", func(t *testing.T) {
		assert.True(t, TaskStatusDeactivating.After(TaskStatusPending))
	})
	t.Run("IsNotSelfInclusive", func(t *testing.T) {
		assert.False(t, TaskStatusPending.After(TaskStatusPending))
	})
}
