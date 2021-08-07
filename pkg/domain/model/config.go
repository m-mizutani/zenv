package model

import (
	"strings"

	"github.com/m-mizutani/goerr"
)

type Config struct {
	KeychainNamespacePrefix string
	DotEnvFile              string
}

const keychainNamespaceHead = "@"

func trimNamespace(name string) string {
	return strings.TrimPrefix(name, keychainNamespaceHead)
}

func ValidateKeychainNamespace(name string) error {
	if !strings.HasPrefix(name, keychainNamespaceHead) {
		return goerr.Wrap(ErrKeychainInvalidNamespace).With("reason", "@ is required as prefix")
	}
	namespace := trimNamespace(name)
	if len(namespace) == 0 {
		return goerr.Wrap(ErrKeychainInvalidNamespace).With("reason", "no name after @")
	}

	return nil
}

func KeychainNamespace(prefix, name string) string {
	return prefix + trimNamespace(name)
}
