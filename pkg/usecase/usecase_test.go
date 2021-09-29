package usecase_test

import (
	"os"
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
		require.NoError(t, uc.Write(&model.WriteSecretInput{
			Namespace: "@tower",
			Key:       "COLOR",
		}))

		require.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    []string{"@tower", "this", "test"},
		}))
	})

	t.Run("keychain namespace not found", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		require.ErrorIs(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    []string{"@tower", "this", "test"},
		}), model.ErrKeychainNotFound)
	})
}

func TestGenerate(t *testing.T) {
	t.Run("generate random secure variable", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.PutKeyChainValuesMock = func(envVars []*model.EnvVar, namespace string) error {
			require.Len(t, envVars, 1)
			assert.Equal(t, "zenv.bridge", namespace)
			assert.Equal(t, "SECRET", envVars[0].Key)
			assert.Len(t, envVars[0].Value, 24)
			return nil
		}
		require.NoError(t, uc.Generate(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Key:       "SECRET",
			Length:    24,
		}))
	})

	t.Run("fail if length <= 0", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		require.ErrorIs(t, uc.Generate(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Key:       "SECRET",
			Length:    0,
		}), model.ErrInvalidArgument)
	})

	t.Run("fail if length > 2^16", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		require.ErrorIs(t, uc.Generate(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Key:       "SECRET",
			Length:    65536,
		}), model.ErrInvalidArgument)
	})

	t.Run("fail if key is empty", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		require.ErrorIs(t, uc.Generate(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Length:    24,
		}), model.ErrEnvVarInvalidName)
	})

	t.Run("fail if namespaec is empty", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		require.ErrorIs(t, uc.Generate(&model.GenerateSecretInput{
			Key:    "blue",
			Length: 24,
		}), model.ErrKeychainInvalidNamespace)
	})
}

func TestFileLoader(t *testing.T) {
	t.Run("replace value with a file", func(t *testing.T) {
		var calledExec int
		uc, mock := usecase.NewWithMock()
		mock.ReadFileMock = func(filename string) ([]byte, error) {
			assert.Equal(t, "myfile.txt", filename)
			return []byte("yummy"), nil
		}
		mock.ExecMock = func(vars []*model.EnvVar, args []string) error {
			calledExec++
			require.Len(t, vars, 1)
			require.Len(t, args, 1)
			assert.Equal(t, &model.EnvVar{
				Key:    "FILE_VAL",
				Value:  "yummy",
				Secret: false,
			}, vars[0])
			assert.Equal(t, "gogo", args[0])
			return nil
		}
		require.NoError(t, uc.Exec(&model.ExecInput{
			Args: []string{
				"FILE_VAL=@myfile.txt",
				"gogo",
			},
		}))
		assert.Equal(t, 1, calledExec)
	})

	t.Run("fail if not existing file specified", func(t *testing.T) {
		var calledExec int
		uc, mock := usecase.NewWithMock()
		mock.ReadFileMock = func(filename string) ([]byte, error) {
			return nil, os.ErrNotExist
		}
		mock.ExecMock = func(vars []*model.EnvVar, args []string) error {
			calledExec++
			return nil
		}
		require.ErrorIs(t, uc.Exec(&model.ExecInput{
			Args: []string{
				"FILE_VAL=@myfile.txt",
				"gogo",
			},
		}), os.ErrNotExist)
		assert.Equal(t, 0, calledExec)
	})
}
