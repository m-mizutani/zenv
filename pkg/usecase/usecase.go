package usecase

import (
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/infra"
)

type Usecase struct {
	infra  interfaces.Infra
	config *model.Config
}

func newUsecase() *Usecase {
	return &Usecase{
		infra:  infra.New(),
		config: &model.Config{},
	}
}

func New() interfaces.Usecase {
	return newUsecase()
}

func (x *Usecase) SetConfig(config *model.Config) {
	x.config = config
}
