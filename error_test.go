package cocoa

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func TestECSTaskNotFoundError(t *testing.T) {
	assert.Implements(t, (*error)(nil), new(ECSTaskNotFoundError))
	t.Run("IsECSTaskNotFoundError", func(t *testing.T) {
		err := NewECSTaskNotFoundError("arn")
		assert.Error(t, err)
		assert.True(t, IsECSTaskNotFoundError(err))
	})
	t.Run("OtherErrorsAreNotECSTaskNotFound", func(t *testing.T) {
		err := errors.New("some error")
		assert.False(t, IsECSTaskNotFoundError(err))
	})
	t.Run("WrappedECSTaskNotFoundError", func(t *testing.T) {
		err := errors.Wrap(NewECSTaskNotFoundError("arn"), "wrapping message")
		assert.True(t, IsECSTaskNotFoundError(err))
	})
}
