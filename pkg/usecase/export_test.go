package usecase

import (
	"github.com/m-mizutani/zenv/pkg/infra"
)

func NewWithMock() (Interface, *infra.Mock) {
	mock := infra.NewMock()
	uc := New().(*usecase)
	uc.config.KeychainNamespacePrefix = "zenv."
	uc.infra = mock
	return uc, mock
}
