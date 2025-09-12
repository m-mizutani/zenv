package model

import (
	"github.com/m-mizutani/goerr/v2"
)

type TOMLConfig map[string]TOMLValue

type TOMLValue struct {
	Value    *string  `toml:"value,omitempty"`
	File     *string  `toml:"file,omitempty"`
	Command  *string  `toml:"command,omitempty"`
	Args     []string `toml:"args,omitempty"`
	Alias    *string  `toml:"alias,omitempty"`
	Template *string  `toml:"template,omitempty"`
	Refs     []string `toml:"refs,omitempty"`
}

func (v TOMLValue) Validate() error {
	// Refs should only be used with template (check this first to give more specific error)
	if v.Template == nil && len(v.Refs) > 0 {
		return goerr.New("refs can only be used with template")
	}

	count := 0
	if v.Value != nil {
		count++
	}
	if v.File != nil {
		count++
	}
	if v.Command != nil {
		count++
	}
	if v.Alias != nil {
		count++
	}
	if v.Template != nil {
		count++
	}

	if count == 0 {
		return goerr.New("no value specified")
	}
	if count > 1 {
		return goerr.New("multiple value types specified (only one of value, file, command, alias, or template can be specified)")
	}

	// Refs can only be used with templates
	if len(v.Refs) > 0 && v.Template == nil {
		return goerr.New("refs can only be used with template")
	}

	return nil
}
