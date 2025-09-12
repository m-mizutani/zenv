package executor

import (
	"context"

	"github.com/m-mizutani/zenv/v2/pkg/model"
)

type ExecuteFunc func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error
