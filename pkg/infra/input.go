package infra

import (
	"github.com/Songmu/prompter"
)

func (x *Infrastructure) Prompt(msg string) string {
	return prompter.Password(msg)
}
