package executor

import "github.com/m-mizutani/zenv/pkg/model"

type ExecuteFunc func(cmd string, args []string, envVars []*model.EnvVar) (int, error)
