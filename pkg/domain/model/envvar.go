package model

import (
	"time"

	"github.com/m-mizutani/zenv/pkg/domain/types"
)

type EnvVar struct {
	Key    types.EnvKey
	Value  types.EnvValue
	Secret bool
}

type NamespaceVars struct {
	Namespace types.NamespaceSuffix
	Vars      []*EnvVar
}

type Backup struct {
	CreatedAt time.Time
	Encrypted []byte
	IV        []byte
}
