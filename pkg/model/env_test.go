package model_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestExecutorError(t *testing.T) {
	t.Run("creates error with exit code", func(t *testing.T) {
		original := errors.New("command failed")
		err := model.NewExecutorError(original, 42)

		gt.Equal(t, err.ExitCode(), 42)
		gt.S(t, err.Error()).Contains("42")
	})

	t.Run("unwraps to original error", func(t *testing.T) {
		original := errors.New("command failed")
		err := model.NewExecutorError(original, 1)

		gt.True(t, errors.Is(err, original))
	})

	t.Run("IsExecutorError returns true for ExecutorError", func(t *testing.T) {
		err := model.NewExecutorError(errors.New("fail"), 1)
		gt.True(t, model.IsExecutorError(err))
	})

	t.Run("IsExecutorError returns true for wrapped ExecutorError", func(t *testing.T) {
		execErr := model.NewExecutorError(errors.New("fail"), 1)
		wrapped := fmt.Errorf("outer: %w", execErr)
		gt.True(t, model.IsExecutorError(wrapped))
	})

	t.Run("IsExecutorError returns false for non-ExecutorError", func(t *testing.T) {
		err := errors.New("not an executor error")
		gt.False(t, model.IsExecutorError(err))
	})

	t.Run("IsExecutorError returns false for nil", func(t *testing.T) {
		gt.False(t, model.IsExecutorError(nil))
	})
}

func TestGetExitCode(t *testing.T) {
	t.Run("returns 0 for nil error", func(t *testing.T) {
		gt.Equal(t, model.GetExitCode(nil), 0)
	})

	t.Run("returns exit code from ExecutorError", func(t *testing.T) {
		err := model.NewExecutorError(errors.New("fail"), 42)
		gt.Equal(t, model.GetExitCode(err), 42)
	})

	t.Run("returns exit code from wrapped ExecutorError", func(t *testing.T) {
		execErr := model.NewExecutorError(errors.New("fail"), 42)
		wrapped := fmt.Errorf("outer: %w", execErr)
		gt.Equal(t, model.GetExitCode(wrapped), 42)
	})

	t.Run("returns 1 for non-ExecutorError", func(t *testing.T) {
		err := errors.New("generic error")
		gt.Equal(t, model.GetExitCode(err), 1)
	})
}
