package usecase

import (
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/infra"
)

type Usecase struct {
	client infra.Client
	config *model.Config
}

func New(options ...Option) *Usecase {
	core := &Usecase{
		client: infra.New(),
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

type Option func(usecase *Usecase)

func WithClient(client infra.Client) Option {
	return func(usecase *Usecase) {
		usecase.client = client
	}
}

func WithConfig(cfg *model.Config) Option {
	return func(usecase *Usecase) {
		usecase.config = cfg
	}
}
