package loader

import (
	"context"

	"github.com/m-mizutani/zenv/v2/pkg/model"
)

type LoadFunc func(ctx context.Context) ([]*model.EnvVar, error)
