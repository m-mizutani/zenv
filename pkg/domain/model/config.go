package model

import (
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

type Config struct {
	KeychainNamespacePrefix types.NamespacePrefix
	DotEnvFiles             []types.FilePath
	OverrideEnvFile         types.FilePath
	IgnoreErrors            map[types.IgnoreError]struct{}
}
