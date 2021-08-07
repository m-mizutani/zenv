package infra

import (
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

type Mock struct {
	ExecMock     func(vars []*model.EnvVar, args []string) error
	ReadFileMock func(filename string) ([]byte, error)
	PromptMock   func(msg string) string
	StdoutMock   func(format string, v ...interface{})

	PutKeyChainValuesMock func(envVars []*model.EnvVar, namespace string) error
	GetKeyChainValuesMock func(namespace string) ([]*model.EnvVar, error)

	keychainDB map[string]map[string]*model.EnvVar
}

func NewMock() *Mock {
	mock := &Mock{
		keychainDB: make(map[string]map[string]*model.EnvVar),
	}
	mock.ReadFileMock = mock.readFile
	mock.PutKeyChainValuesMock = mock.putKeyChainValues
	mock.GetKeyChainValuesMock = mock.getKeyChainValues
	return mock
}

func (x *Mock) Exec(vars []*model.EnvVar, args []string) error {
	return x.ExecMock(vars, args)
}

func (x *Mock) ReadFile(filename string) ([]byte, error) {
	return x.ReadFileMock(filename)
}

func (x *Mock) readFile(filename string) ([]byte, error) {
	return nil, nil
}

func (x *Mock) Stdout(format string, v ...interface{}) {
	x.StdoutMock(format, v)
}

func (x *Mock) Prompt(msg string) string {
	return x.PromptMock(msg)
}

func (x *Mock) PutKeyChainValues(envVars []*model.EnvVar, namespace string) error {

	return x.PutKeyChainValuesMock(envVars, namespace)
}

func (x *Mock) putKeyChainValues(envVars []*model.EnvVar, namespace string) error {
	db, ok := x.keychainDB[namespace]
	if !ok {
		db = make(map[string]*model.EnvVar)
		x.keychainDB[namespace] = db
	}

	for _, v := range envVars {
		db[v.Key] = v
	}
	return nil
}

func (x *Mock) GetKeyChainValues(namespace string) ([]*model.EnvVar, error) {
	return x.GetKeyChainValuesMock(namespace)
}

func (x *Mock) getKeyChainValues(namespace string) ([]*model.EnvVar, error) {
	db, ok := x.keychainDB[namespace]
	if !ok {
		return nil, goerr.Wrap(model.ErrKeychainNotFound).With("namespace", namespace)
	}

	var vars []*model.EnvVar
	for _, v := range db {
		vars = append(vars, v)
	}
	return vars, nil
}
