package usecase_test

import (
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	t.Run("load keychain variables", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args []string) error {
			require.Len(t, args, 2)
			assert.Equal(t, "this", args[0])
			assert.Equal(t, "test", args[1])

			require.Len(t, vars, 1)
			assert.Equal(t, "COLOR", vars[0].Key)
			assert.Equal(t, "blue", vars[0].Value)

			return nil
		}

		mock.PromptMock = func(msg string) string { return "blue" }
		require.NoError(t, uc.Write(&model.WriteInput{
			Namespace: "@tower",
			Args:      []string{"COLOR"},
		}))

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    []string{"@tower", "this", "test"},
		}))
	})

	t.Run("keychain namespace not found", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		require.Error(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    []string{"@tower", "this", "test"},
		}))
	})
}
