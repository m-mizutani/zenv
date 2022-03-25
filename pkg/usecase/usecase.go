package usecase

import (
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/infra"
)

type Usecase struct {
	infra  infra.Interface
	config *model.Config
}

func New(options ...Option) *Usecase {
	core := &Usecase{
		infra:  infra.New(),
		config: &model.Config{},
	}

	for _, opt := range options {
		opt(core)
	}
	return core
}

func (x *Usecase) Clone(options ...Option) *Usecase {
	core := *x
	for _, opt := range options {
		opt(&core)
	}
	return &core
}

type Option func(core *Usecase)

func WithInfra(clients infra.Interface) Option {
	return func(core *Usecase) {
		core.infra = clients
	}
}

func WithConfig(cfg *model.Config) Option {
	return func(core *Usecase) {
		core.config = cfg
	}
}
