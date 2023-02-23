package usecase_test

import (
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
	"github.com/m-mizutani/zenv/pkg/usecase"
)

func TestExportImportSecret(t *testing.T) {
	mock := infra.NewMock()
	uc := usecase.New(usecase.WithClient(mock))
	ns := types.Namespace("zenv.test")
	vars := []*model.EnvVar{
		{
			Key:    types.EnvKey("test"),
			Value:  types.EnvValue("blue"),
			Secret: true,
		},
	}
	gt.NoError(t, mock.PutKeyChainValues(vars, ns)).Must()

	fd, err := os.CreateTemp("", "")
	fpath := types.FilePath(fd.Name())
	gt.NoError(t, err).Must()
	gt.NoError(t, fd.Close()).Must()
	defer os.Remove(string(fpath))

	var calledStdout, calledPrompt int
	mock.StdoutMock = func(format string, v ...interface{}) {
		calledStdout++
	}
	mock.PromptMock = func(msg string) string {
		calledPrompt++
		return "hoge"
	}

	{
		// export to file
		gt.NoError(t, uc.ExportSecret(&model.ExportSecretInput{
			Output: types.FilePath(fpath),
		})).Must()
		gt.V(t, calledPrompt).Equal(1)
		gt.V(t, calledStdout).Equal(1)
	}

	// remove temporary
	gt.NoError(t, mock.DeleteKeyChainValue(ns, vars[0].Key)).Must()
	gt.R1(mock.GetKeyChainValues(ns)).NoError(t)

	{
		// import from file
		gt.NoError(t, uc.ImportSecret(&model.ImportSecretInput{
			Input: fpath,
		})).Must()
		gt.V(t, calledPrompt).Equal(2)
		gt.V(t, calledStdout).Equal(2)
	}

	resp := gt.R1(mock.GetKeyChainValues(ns)).NoError(t)
	gt.V(t, resp).Equal(vars)

	{
		// fail to import invalid passphrase
		mock.PromptMock = func(msg string) string {
			return "invalid_phrase"
		}
		gt.Error(t, uc.ImportSecret(&model.ImportSecretInput{
			Input: fpath,
		}))
	}
}
