package cmd_test

import (
	"testing"

	"github.com/m-mizutani/zenv/pkg/controller/cmd"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretWriteAndRead(t *testing.T) {
	mock := infra.NewMock()

	uc := usecase.New(usecase.WithClient(mock))
	cli := cmd.New(cmd.WithUsecase(uc))

	mock.PromptMock = func(msg string) string {
		return "my_key"
	}
	require.NoError(t, cli.Run([]string{"zenv", "secret", "write", "@aws", "AWS_ACCESS_KEY_ID"}))

	mock.PromptMock = func(msg string) string {
		return "my_secret"
	}
	require.NoError(t, cli.Run([]string{"zenv", "secret", "write", "@aws", "AWS_SECRET_ACCESS_KEY"}))

	var calledExec int
	mock.ExecMock = func(vars []*model.EnvVar, args types.Arguments) error {
		calledExec++
		v1 := &model.EnvVar{
			Key:    types.EnvKey("AWS_ACCESS_KEY_ID"),
			Value:  types.EnvValue("my_key"),
			Secret: true,
		}
		v2 := &model.EnvVar{
			Key:    types.EnvKey("AWS_SECRET_ACCESS_KEY"),
			Value:  types.EnvValue("my_secret"),
			Secret: true,
		}

		assert.Contains(t, vars, v1)
		assert.Contains(t, vars, v2)
		require.Len(t, args, 3)
		assert.Equal(t, types.Argument("aws"), args[0])
		assert.Equal(t, types.Argument("s3"), args[1])
		assert.Equal(t, types.Argument("ls"), args[2])
		return nil
	}
	require.NoError(t, cli.Run([]string{"zenv", "@aws", "aws", "s3", "ls"}))
	assert.Equal(t, 1, calledExec)
}
