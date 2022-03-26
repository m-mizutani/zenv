package model

import (
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

type EnvVar struct {
	Key    types.EnvKey
	Value  types.EnvValue
	Secret bool
}
