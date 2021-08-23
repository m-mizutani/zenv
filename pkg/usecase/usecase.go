package usecase

import (
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/infra"
)

type usecase struct {
	infra  interfaces.Infra
	config *model.Config
}

func New() Interface {
	return &usecase{
		infra:  infra.New(),
		config: &model.Config{},
	}
}

func (x *usecase) SetConfig(config *model.Config) {
	x.config = config
}
