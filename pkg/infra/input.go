package infra

import (
	"github.com/Songmu/prompter"
)

func (x *client) Prompt(msg string) string {
	return prompter.Password(msg)
}
