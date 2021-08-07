package infra

import (
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
)

type Infrastructure struct{}

func New() interfaces.Infra {
	return &Infrastructure{}
}
