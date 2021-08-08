package usecase_test

import (
	"fmt"
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicExec(t *testing.T) {
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

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "COLOR", Value: "blue"},
				{Key: "NUMBER", Value: "five"},
				{Key: "TIME", Value: "insane"},
			},
			Args: []string{"this", "test"},
		}))
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

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "COLOR", Value: "blue"},
			},
			Args: []string{"NUMBER=five", "this", "test"},
		}))
	})
}

func TestDotEnv(t *testing.T) {
	t.Run("exec with dotenv file", func(t *testing.T) {
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
		mock.ReadFileMock = func(filename string) ([]byte, error) {
			assert.Equal(t, ".mydotenv", filename)
			return []byte(`# ignore comment line
COLOR=blue

NUMBER=five
`), nil
		}
		uc.SetConfig(&model.Config{
			DotEnvFile: ".mydotenv",
		})
		require.NoError(t, uc.Exec(&model.ExecInput{
			Args: []string{"this", "test"},
		}))
	})

	t.Run("error when invalid line in dotenv file", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ReadFileMock = func(filename string) ([]byte, error) {
			assert.Equal(t, ".env", filename)
			return []byte(`COLOR=blue
NoEqualMark
NUMBER=five
`), nil
		}
		uc.SetConfig(&model.Config{DotEnvFile: ".env"})
		require.ErrorIs(t, uc.Exec(&model.ExecInput{
			Args: []string{"this", "test"},
		}), model.ErrInvalidArgumentFormat)
	})

	t.Run("something bad in reading dotenv", func(t *testing.T) {
		err := fmt.Errorf("something bad")
		uc, mock := usecase.NewWithMock()
		mock.ReadFileMock = func(filename string) ([]byte, error) {
			return nil, err
		}
		uc.SetConfig(&model.Config{DotEnvFile: ".env"})

		require.ErrorIs(t, uc.Exec(&model.ExecInput{
			Args: []string{"this", "test"},
		}), err)

	})
}

func TestReplacement(t *testing.T) {
	t.Run("exec with replacement values", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args []string) error {
			require.Len(t, args, 4)
			assert.Equal(t, "test", args[0])
			assert.Equal(t, "five", args[1])
			assert.Equal(t, "onetime", args[2])
			assert.Equal(t, "%BLUE", args[3])

			return nil
		}

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "VAR", Value: "one"},
				{Key: "VARIABLE", Value: "five"},
			},
			Args: []string{"test", "%VARIABLE", "%VARtime", "%BLUE"},
		}))
	})
}
