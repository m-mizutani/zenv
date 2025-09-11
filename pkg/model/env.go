package model

type EnvVar struct {
	Name   string
	Value  string
	Source EnvSource
}

type EnvSource int

const (
	SourceSystem EnvSource = iota
	SourceDotEnv
	SourceTOML
	SourceInline
)
