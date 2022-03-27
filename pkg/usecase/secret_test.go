package usecase_test

import (
	"os"
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, mock.PutKeyChainValues(vars, ns))

	fd, err := os.CreateTemp("", "")
	fpath := types.FilePath(fd.Name())
	require.NoError(t, err)
	require.NoError(t, fd.Close())
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
		require.NoError(t, uc.ExportSecret(&model.ExportSecretInput{
			Output: types.FilePath(fpath),
		}))
		assert.Equal(t, 1, calledPrompt)
		assert.Equal(t, 1, calledStdout)
	}

	// remove temporary
	require.NoError(t, mock.DeleteKeyChainValue(ns, vars[0].Key))
	_, err = mock.GetKeyChainValues(ns)
	require.NoError(t, err)

	{
		// import from file
		require.NoError(t, uc.ImportSecret(&model.ImportSecretInput{
			Input: fpath,
		}))
		assert.Equal(t, 2, calledPrompt)
		assert.Equal(t, 2, calledStdout)
	}

	resp, err := mock.GetKeyChainValues(ns)
	require.NoError(t, err)
	require.Len(t, resp, 1)
	assert.Equal(t, resp[0], vars[0])

	{
		// fail to import invalid passphrase
		mock.PromptMock = func(msg string) string {
			return "invalid_phrase"
		}
		require.Error(t, uc.ImportSecret(&model.ImportSecretInput{
			Input: fpath,
		}))
	}
}
