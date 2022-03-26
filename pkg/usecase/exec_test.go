package usecase_test

import (
	"fmt"
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicExec(t *testing.T) {
	t.Run("exec with env vars", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			require.Len(t, args, 2)
			assert.Equal(t, types.Argument("this"), args[0])
			assert.Equal(t, types.Argument("test"), args[1])

			require.Len(t, vars, 3)
			assert.Equal(t, types.EnvKey("COLOR"), vars[0].Key)
			assert.Equal(t, types.EnvValue("blue"), vars[0].Value)
			assert.Equal(t, types.EnvKey("NUMBER"), vars[1].Key)
			assert.Equal(t, types.EnvValue("five"), vars[1].Value)
			assert.Equal(t, types.EnvKey("TIME"), vars[2].Key)
			assert.Equal(t, types.EnvValue("insane"), vars[2].Value)

			return nil
		}

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "COLOR", Value: "blue"},
				{Key: "NUMBER", Value: "five"},
				{Key: "TIME", Value: "insane"},
			},
			Args: types.Arguments{"this", "test"},
		}))
	})

	t.Run("exec with env vars in args", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			require.Len(t, args, 2)
			assert.Equal(t, types.Argument("this"), args[0])
			assert.Equal(t, types.Argument("test"), args[1])

			require.Len(t, vars, 2)
			assert.Equal(t, types.EnvKey("COLOR"), vars[0].Key)
			assert.Equal(t, types.EnvValue("blue"), vars[0].Value)
			assert.Equal(t, types.EnvKey("NUMBER"), vars[1].Key)
			assert.Equal(t, types.EnvValue("five"), vars[1].Value)

			return nil
		}

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "COLOR", Value: "blue"},
			},
			Args: types.Arguments{"NUMBER=five", "this", "test"},
		}))
	})
}

func TestDotEnv(t *testing.T) {
	t.Run("exec with dotenv file", func(t *testing.T) {
		uc, mock := usecase.NewWithMock(usecase.WithConfig(&model.Config{
			DotEnvFile: ".mydotenv",
		}))
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			require.Len(t, args, 2)
			assert.Equal(t, types.Argument("this"), args[0])
			assert.Equal(t, types.Argument("test"), args[1])

			require.Len(t, vars, 2)
			assert.Equal(t, types.EnvKey("COLOR"), vars[0].Key)
			assert.Equal(t, types.EnvValue("blue"), vars[0].Value)
			assert.Equal(t, types.EnvKey("NUMBER"), vars[1].Key)
			assert.Equal(t, types.EnvValue("five"), vars[1].Value)

			return nil
		}
		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			assert.Equal(t, types.FilePath(".mydotenv"), filename)
			return []byte(`# ignore comment line
COLOR=blue

NUMBER=five
`), nil
		}

		require.NoError(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"this", "test"},
		}))
	})

	t.Run("error when invalid line in dotenv file", func(t *testing.T) {
		uc, mock := usecase.NewWithMock(usecase.WithConfig(
			&model.Config{DotEnvFile: ".env"},
		))

		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			assert.Equal(t, types.FilePath(".env"), filename)
			return []byte(`COLOR=blue
NoEqualMark
NUMBER=five
`), nil
		}

		require.ErrorIs(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"this", "test"},
		}), types.ErrInvalidArgumentFormat)
	})

	t.Run("something bad in reading dotenv", func(t *testing.T) {
		err := fmt.Errorf("something bad")
		uc, mock := usecase.NewWithMock(usecase.WithConfig(
			&model.Config{DotEnvFile: ".env"},
		))
		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			return nil, err
		}

		require.ErrorIs(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"this", "test"},
		}), err)

	})
}

func TestReplacement(t *testing.T) {
	t.Run("exec with replacement values", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			require.Len(t, args, 4)
			assert.Equal(t, types.Argument("test"), args[0])
			assert.Equal(t, types.Argument("five"), args[1])
			assert.Equal(t, types.Argument("onetime"), args[2])
			assert.Equal(t, types.Argument("%BLUE"), args[3])

			return nil
		}

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "VAR", Value: "one"},
				{Key: "VARIABLE", Value: "five"},
			},
			Args: types.Arguments{"test", "%VARIABLE", "%VARtime", "%BLUE"},
		}))
	})
}
