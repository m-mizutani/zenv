package usecase

import (
	"github.com/m-mizutani/zenv/pkg/infra"
)

func NewWithMock(options ...Option) (*Usecase, *infra.Mock) {
	mock := infra.NewMock()
	uc := New(append(options, WithClient(mock))...)
	uc.config.KeychainNamespacePrefix = "zenv."
	uc.client = mock
	return uc, mock
}
