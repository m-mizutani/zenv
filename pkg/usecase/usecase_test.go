package usecase_test

import (
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExec(t *testing.T) {
	t.Run("exec with env vars", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args []string) error {
			require.Len(t, args, 2)
			assert.Equal(t, "this", args[0])
			assert.Equal(t, "test", args[1])

			require.Len(t, vars, 3)
			assert.Equal(t, "COLOR", vars[0].Key)
			assert.Equal(t, "blue", vars[0].Value)
			assert.Equal(t, "NUMBER", vars[1].Key)
			assert.Equal(t, "five", vars[1].Value)
			assert.Equal(t, "TIME", vars[2].Key)
			assert.Equal(t, "insane", vars[2].Value)

			return nil
		}

		uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "COLOR", Value: "blue"},
				{Key: "NUMBER", Value: "five"},
				{Key: "TIME", Value: "insane"},
			},
			Args: []string{"this", "test"},
		})
	})

	t.Run("exec with env vars in args", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args []string) error {
			require.Len(t, args, 2)
			assert.Equal(t, "this", args[0])
			assert.Equal(t, "test", args[1])

			require.Len(t, vars, 2)
			assert.Equal(t, "COLOR", vars[0].Key)
			assert.Equal(t, "blue", vars[0].Value)
			assert.Equal(t, "NUMBER", vars[1].Key)
			assert.Equal(t, "five", vars[1].Value)

			return nil
		}

		uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "COLOR", Value: "blue"},
			},
			Args: []string{"NUMBER=five", "this", "test"},
		})
	})

	t.Run("with keychain", func(t *testing.T) {
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

		require.NoError(t, uc.Write(&model.WriteInput{
			Namespace: "@tower",
			EnvVar: model.EnvVar{
				Key:   "COLOR",
				Value: "blue",
			},
		}))

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    []string{"@tower", "this", "test"},
		}))
	})
}
