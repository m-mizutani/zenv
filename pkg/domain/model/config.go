package model

import (
	"regexp"

	"github.com/m-mizutani/zenv/pkg/domain/types"
)

type Config struct {
	KeychainNamespacePrefix types.NamespacePrefix
	DotEnvFiles             []types.FilePath
	OverrideEnvFile         types.FilePath
}

var envVarNameRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")
