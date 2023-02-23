package cmd_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/pkg/controller/cmd"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
	"github.com/m-mizutani/zenv/pkg/usecase"
)

func TestSecretWriteAndRead(t *testing.T) {
	mock := infra.NewMock()

	uc := usecase.New(usecase.WithClient(mock))
	cli := cmd.New(cmd.WithUsecase(uc))

	mock.PromptMock = func(msg string) string {
		return "my_key"
	}
	gt.NoError(t, cli.Run([]string{"zenv", "secret", "write", "@aws", "AWS_ACCESS_KEY_ID"})).Must()

	mock.PromptMock = func(msg string) string {
		return "my_secret"
	}
	gt.NoError(t, cli.Run([]string{"zenv", "secret", "write", "@aws", "AWS_SECRET_ACCESS_KEY"})).Must()

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

		gt.Array(t, vars).Have(v1).Have(v2)
		gt.Array(t, args).Length(3).
			Contain([]types.Argument{"aws", "s3", "ls"})
		return nil
	}
	gt.NoError(t, cli.Run([]string{"zenv", "@aws", "aws", "s3", "ls"})).Must()
	gt.Number(t, calledExec).Equal(1)
}
