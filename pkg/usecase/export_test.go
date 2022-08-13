package usecase

import (
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
)

func NewWithMock(options ...Option) (*Usecase, *infra.Mock) {
	mock := infra.NewMock()
	uc := New(append(options, WithClient(mock))...)
	uc.config.KeychainNamespacePrefix = "zenv."
	uc.client = mock
	return uc, mock
}

func ParseArgs(uc *Usecase, args types.Arguments) (types.Arguments, []*model.EnvVar, error) {
	return uc.parseArgs(args)
}
