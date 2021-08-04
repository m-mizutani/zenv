package model

import "strings"

type Config struct {
	KeychainNamespacePrefix string
	ConfigFilePath          string
}

const keychainNamespaceHead = "@"

func IsKeychainNamespace(name string) bool {
	return strings.HasPrefix(name, keychainNamespaceHead)
}

func KeychainNamespace(prefix, name string) string {
	return prefix + strings.TrimPrefix(name, keychainNamespaceHead)
}
