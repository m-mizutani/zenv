package usecase

import (
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
	"github.com/m-mizutani/zenv/pkg/infra"
)

func NewWithMock() (interfaces.Usecase, *infra.Mock) {
	mock := infra.NewMock()
	uc := newUsecase()
	uc.infra = mock
	return uc, mock
}
