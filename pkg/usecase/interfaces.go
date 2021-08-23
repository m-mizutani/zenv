package usecase

import "github.com/m-mizutani/zenv/pkg/domain/model"

type Interface interface {
	Exec(input *model.ExecInput) error
	List(input *model.ListInput) error
	Write(input *model.WriteSecretInput) error
	Generate(input *model.GenerateSecretInput) error
	SetConfig(config *model.Config)
}
