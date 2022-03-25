package infra

type Infrastructure struct{}

func New() Interface {
	return &Infrastructure{}
}
