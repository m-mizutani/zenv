package usecase_test

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/usecase"
)

func TestBasicExec(t *testing.T) {
	t.Run("exec with env vars", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			gt.Array(t, args).Equal([]types.Argument{"this", "test"})

			gt.Array(t, vars).Equal([]*model.EnvVar{
				{
					Key:   "COLOR",
					Value: "blue",
				},
				{
					Key:   "NUMBER",
					Value: "five",
				},
				{
					Key:   "TIME",
					Value: "insane",
				},
			})

			return nil
		}

		gt.NoError(t, uc.Exec(&model.ExecInput{
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
			gt.Array(t, args).
				Equal([]types.Argument{"this", "test"})

			gt.Array(t, vars).Equal([]*model.EnvVar{
				{
					Key:   "COLOR",
					Value: "blue",
				},
				{
					Key:   "NUMBER",
					Value: "five",
				},
			})

			return nil
		}

		gt.NoError(t, uc.Exec(&model.ExecInput{
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
			DotEnvFiles: []types.FilePath{".mydotenv"},
		}))
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			gt.Array(t, args).
				Equal([]types.Argument{"this", "test"})

			sort.Slice(vars, func(i, j int) bool {
				return vars[i].Key < vars[j].Key
			})
			gt.Array(t, vars).Equal([]*model.EnvVar{
				{
					Key:   "COLOR",
					Value: "blue",
				},
				{
					Key:   "NUMBER",
					Value: "five",
				},
			})

			return nil
		}
		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			gt.Value(t, filename).Equal(".mydotenv")
			return []byte(`# ignore comment line
COLOR=blue

NUMBER=five
`), nil
		}

		gt.NoError(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"this", "test"},
		}))
	})

	t.Run("error when invalid line in dotenv file", func(t *testing.T) {
		uc, mock := usecase.NewWithMock(usecase.WithConfig(
			&model.Config{DotEnvFiles: []types.FilePath{".env"}},
		))

		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			gt.V(t, filename).Equal(".env")
			return []byte(`COLOR=blue
NoEqualMark
NUMBER=five
`), nil
		}

		gt.Error(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"this", "test"},
		})).Is(types.ErrInvalidArgumentFormat)
	})

	t.Run("something bad in reading dotenv", func(t *testing.T) {
		err := fmt.Errorf("something bad")
		uc, mock := usecase.NewWithMock(usecase.WithConfig(
			&model.Config{DotEnvFiles: []types.FilePath{".env"}},
		))
		mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
			return nil, err
		}

		gt.Error(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"this", "test"},
		})).Is(err)

	})
}

func TestReplacement(t *testing.T) {
	t.Run("exec with replacement values", func(t *testing.T) {
		uc, mock := usecase.NewWithMock()
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			gt.Array(t, args).Equal([]types.Argument{
				"test", "five", "onetime", "%BLUE",
			})

			return nil
		}

		gt.NoError(t, uc.Exec(&model.ExecInput{
			EnvVars: []*model.EnvVar{
				{Key: "VAR", Value: "one"},
				{Key: "VARIABLE", Value: "five"},
			},
			Args: types.Arguments{"test", "%VARIABLE", "%VARtime", "%BLUE"},
		}))
	})
}

func TestIgnoreError(t *testing.T) {
	t.Run("ignore env file open error", func(t *testing.T) {
		uc, mock := usecase.NewWithMock(usecase.WithConfig(&model.Config{
			DotEnvFiles: []types.FilePath{".not_found_env"},
			IgnoreErrors: map[types.IgnoreError]struct{}{
				types.IgnoreEnvFileOpen: {},
			},
		}))
		mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
			gt.Array(t, args).Equal([]types.Argument{"test"})
			return nil
		}

		gt.NoError(t, uc.Exec(&model.ExecInput{
			Args: types.Arguments{"test"},
		}))
	})
}

func TestOverride(t *testing.T) {
	type testCase struct {
		inputFiles []types.FilePath
		envVars    []*model.EnvVar
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			var called int
			uc, mock := usecase.NewWithMock(usecase.WithConfig(&model.Config{
				DotEnvFiles: tc.inputFiles,
			}))
			mock.ReadFileMock = func(filename types.FilePath) ([]byte, error) {
				return os.ReadFile(string(filename))
			}

			mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
				called++

				sort.Slice(vars, func(i, j int) bool {
					return vars[i].Key < vars[j].Key
				})

				gt.Array(t, vars).Equal(tc.envVars)
				return nil
			}

			gt.NoError(t, uc.Exec(&model.ExecInput{
				Args: types.Arguments{"this", "test"},
			}))
			gt.Equal(t, called, 1)
		}
	}

	t.Run("no override env vars", runTest(testCase{
		inputFiles: []types.FilePath{"testdata/basic.env"},
		envVars: []*model.EnvVar{
			{Key: "COLOR", Value: "blue"},
		},
	}))

	t.Run("override env vars by one file", runTest(testCase{
		inputFiles: []types.FilePath{
			"testdata/basic.env",
			"testdata/override1.env",
		},
		envVars: []*model.EnvVar{
			{Key: "COLOR", Value: "orange"},
		},
	}))

	t.Run("override env vars by two files", runTest(testCase{
		inputFiles: []types.FilePath{
			"testdata/basic.env",
			"testdata/override1.env",
			"testdata/override2.env",
		},
		envVars: []*model.EnvVar{
			{Key: "COLOR", Value: "red"},
		},
	}))

	t.Run("override env vars prioritize by order", runTest(testCase{
		inputFiles: []types.FilePath{
			"testdata/basic.env",
			"testdata/override2.env",
			"testdata/override1.env",
		},
		envVars: []*model.EnvVar{
			{Key: "COLOR", Value: "orange"},
		},
	}))

	t.Run("override env vars prioritize by order (2)", runTest(testCase{
		inputFiles: []types.FilePath{
			"testdata/override2.env",
			"testdata/override1.env",
			"testdata/basic.env",
		},
		envVars: []*model.EnvVar{
			{Key: "COLOR", Value: "blue"},
		},
	}))
}
