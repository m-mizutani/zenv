package model

import (
	"github.com/m-mizutani/goerr/v2"
)

type TOMLConfig map[string]TOMLValue

type TOMLValue struct {
	Value   *string  `toml:"value,omitempty"`
	File    *string  `toml:"file,omitempty"`
	Command *string  `toml:"command,omitempty"`
	Args    []string `toml:"args,omitempty"`
}

func (v TOMLValue) Validate() error {
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

	if count == 0 {
		return goerr.New("no value specified")
	}
	if count > 1 {
		return goerr.New("multiple value types specified")
	}
	return nil
}
