package usecase_test

import (
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/usecase"
)

func TestWrite(t *testing.T) {
	t.Run("load keychain variables", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			gt.Array(t, args).Equal([]types.Argument{"this", "test"})
			gt.Array(t, vars).Equal([]*model.EnvVar{
				{
					Key:    "COLOR",
					Value:  "blue",
					Secret: true,
				},
			})

			return nil
		}

		mock.PromptMock = func(msg string) string { return "blue" }
		gt.NoError(t, uc.WriteSecret(&model.WriteSecretInput{
			Namespace: "@tower",
			Key:       "COLOR",
		}))

		gt.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    types.Arguments{"@tower", "this", "test"},
		}))
	})

	t.Run("keychain namespace not found", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		gt.Error(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{},
			Args:    types.Arguments{"@tower", "this", "test"},
		})).Is(types.ErrKeychainNotFound)
	})
}

func TestGenerate(t *testing.T) {
	t.Run("generate random secure variable", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.PutKeyChainValuesMock = func(envVars []*model.EnvVar, namespace types.Namespace) error {
			gt.V(t, namespace).Equal("zenv.bridge")
			gt.A(t, envVars).Length(1).
				Elem(0, func(t testing.TB, v *model.EnvVar) {
					gt.Value(t, v.Key).Equal("SECRET")
					gt.N(t, len(v.Value)).Equal(24)
				})
			return nil
		}
		gt.NoError(t, uc.GenerateSecret(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Key:       "SECRET",
			Length:    24,
		}))
	})

	t.Run("fail if length <= 0", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		gt.Error(t, uc.GenerateSecret(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Key:       "SECRET",
			Length:    0,
		})).Is(types.ErrInvalidArgument)
	})

	t.Run("fail if length > 2^16", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		gt.Error(t, uc.GenerateSecret(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Key:       "SECRET",
			Length:    65536,
		})).Is(types.ErrInvalidArgument)
	})

	t.Run("fail if key is empty", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		gt.Error(t, uc.GenerateSecret(&model.GenerateSecretInput{
			Namespace: "@bridge",
			Length:    24,
		})).Is(types.ErrInvalidArgument)
	})

	t.Run("fail if namespaec is empty", func(t *testing.T) {
		uc, _ := usecase.NewWithMock()
		gt.Error(t, uc.GenerateSecret(&model.GenerateSecretInput{
			Key:    "blue",
			Length: 24,
		})).Is(types.ErrInvalidArgument)
	})
}

func TestFileLoader(t *testing.T) {
	t.Run("replace value with a file", func(t *testing.T) {
		var calledExec int
		uc, mock := usecase.NewWithMock()
		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			gt.Value(t, filename).Equal("myfile.txt")
			return []byte("yummy"), nil
		}
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			calledExec++
			gt.Array(t, vars).Equal([]*model.EnvVar{
				{
					Key:    "FILE_VAL",
					Value:  "yummy",
					Secret: false,
				},
			})
			gt.Array(t, args).Length(1)
			gt.Value(t, args[0]).Equal("gogo")
			return nil
		}
		gt.NoError(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{
				"FILE_VAL=&myfile.txt",
				"gogo",
			},
		}))
		gt.N(t, calledExec).Equal(1)
	})

	t.Run("fail if not existing file specified", func(t *testing.T) {
		var calledExec int
		uc, mock := usecase.NewWithMock()
		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			return nil, os.ErrNotExist
		}
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			calledExec++
			return nil
		}
		gt.Error(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{
				"FILE_VAL=&myfile.txt",
				"gogo",
			},
		})).Is(os.ErrNotExist)
		gt.N(t, calledExec).Equal(0)
	})
}

func TestAssign(t *testing.T) {
	uc, mock := usecase.NewWithMock(usecase.WithConfig(&model.Config{
		DotEnvFiles: []types.FilePath{".env"},
	}))
	mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
		return []byte("BLUE=%ORANGE"), nil
	}

	args, vars := gt.R2(usecase.ParseArgs(uc, types.Arguments{
		"ORANGE=red",
		"hello",
	})).NoError(t)

	gt.Array(t, vars).Length(2).EqualAt(0, &model.EnvVar{
		Key:   "BLUE",
		Value: "red",
	})
	gt.Array(t, args).Equal(types.Arguments{"hello"})
}
