package infra

import (
	"io"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

type Mock struct {
	ExecMock     func(vars []*model.EnvVar, args types.Arguments) error
	CommandMock  func(argv types.Arguments) (io.Reader, error)
	ReadFileMock func(filename types.FilePath) ([]byte, error)
	PromptMock   func(msg string) string
	StdoutMock   func(format string, v ...interface{})

	PutKeyChainValuesMock      func([]*model.EnvVar, types.Namespace) error
	GetKeyChainValuesMock      func(types.Namespace) ([]*model.EnvVar, error)
	DeleteKeyChainValuesMock   func(types.Namespace, types.EnvKey) error
	ListKeyChainNamespacesMock func(types.NamespacePrefix) ([]types.Namespace, error)
	DeleteKeyChainValueMock    func(types.Namespace, types.EnvKey) error

	keychainDB map[types.Namespace]map[types.EnvKey]*model.EnvVar
}

func NewMock() *Mock {
	mock := &Mock{
		keychainDB: make(map[types.Namespace]map[types.EnvKey]*model.EnvVar),
	}
	mock.ReadFileMock = mock.readFile
	mock.PutKeyChainValuesMock = mock.putKeyChainValues
	mock.GetKeyChainValuesMock = mock.getKeyChainValues
	mock.DeleteKeyChainValuesMock = mock.deleteKeyChainValue
	mock.ListKeyChainNamespacesMock = mock.listKeyChainNamespaces
	return mock
}

func (x *Mock) Exec(vars []*model.EnvVar, args types.Arguments) error {
	return x.ExecMock(vars, args)
}

func (x *Mock) Command(args types.Arguments) (io.Reader, error) {
	return x.CommandMock(args)
}

func (x *Mock) ReadFile(filename types.FilePath) ([]byte, error) {
	return x.ReadFileMock(filename)
}

func (x *Mock) readFile(filename types.FilePath) ([]byte, error) {
	return nil, nil
}

func (x *Mock) Stdout(format string, v ...interface{}) {
	x.StdoutMock(format, v)
}

func (x *Mock) Prompt(msg string) string {
	return x.PromptMock(msg)
}

func (x *Mock) PutKeyChainValues(envVars []*model.EnvVar, namespace types.Namespace) error {

	return x.PutKeyChainValuesMock(envVars, namespace)
}

func (x *Mock) GetKeyChainValues(namespace types.Namespace) ([]*model.EnvVar, error) {
	return x.GetKeyChainValuesMock(namespace)
}

func (x *Mock) ListKeyChainNamespaces(prefix types.NamespacePrefix) ([]types.Namespace, error) {
	return x.ListKeyChainNamespacesMock(prefix)
}

func (x *Mock) DeleteKeyChainValue(ns types.Namespace, key types.EnvKey) error {
	return x.DeleteKeyChainValuesMock(ns, key)
}

func (x *Mock) putKeyChainValues(envVars []*model.EnvVar, namespace types.Namespace) error {
	db, ok := x.keychainDB[namespace]
	if !ok {
		db = make(map[types.EnvKey]*model.EnvVar)
		x.keychainDB[namespace] = db
	}

	for _, v := range envVars {
		db[v.Key] = v
	}
	return nil
}

func (x *Mock) getKeyChainValues(ns types.Namespace) ([]*model.EnvVar, error) {
	db, ok := x.keychainDB[ns]
	if !ok {
		return nil, goerr.Wrap(types.ErrKeychainNotFound).With("namespace", ns)
	}

	var vars []*model.EnvVar
	for _, v := range db {
		vars = append(vars, v)
	}
	return vars, nil
}

func (x *Mock) listKeyChainNamespaces(prefix types.NamespacePrefix) ([]types.Namespace, error) {
	var keys []types.Namespace
	for k := range x.keychainDB {
		if k.HasPrefix(prefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (x *Mock) deleteKeyChainValue(namespace types.Namespace, key types.EnvKey) error {
	db, ok := x.keychainDB[namespace]
	if !ok {
		return goerr.Wrap(types.ErrKeychainNotFound).With("namespace", namespace)
	}

	delete(db, key)
	return nil
}
