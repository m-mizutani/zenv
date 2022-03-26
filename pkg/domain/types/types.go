package types

import (
	"regexp"
	"strings"

	"github.com/m-mizutani/goerr"
)

type (
	EnvKey    string
	EnvValue  string
	EnvSecret string

	// Namespace is keychain namespace
	Namespace       string
	NamespaceSuffix string
	NamespacePrefix string

	Argument  string
	Arguments []Argument

	FilePath string
)

func (x EnvKey) String() string          { return string(x) }
func (x Namespace) String() string       { return string(x) }
func (x NamespacePrefix) String() string { return string(x) }
func (x Argument) String() string        { return string(x) }

func (x Arguments) Strings() []string {
	resp := make([]string, len(x))
	for i := range x {
		resp[i] = x[i].String()
	}
	return resp
}

func (x Namespace) HasPrefix(p NamespacePrefix) bool {
	return strings.HasPrefix(x.String(), p.String())
}

func isKeychainNamespace(s string) bool {
	return strings.HasPrefix(s, keychainNamespaceHead) &&
		len(s) > len(keychainNamespaceHead)
}
func (x NamespaceSuffix) Validate() error {
	if !isKeychainNamespace(string(x)) {
		return goerr.Wrap(ErrInvalidArgument, "malformed namespace").With("namespace", x)
	}
	return nil
}

func (x NamespaceSuffix) ToNamespace(prefix NamespacePrefix) Namespace {
	return Namespace(string(prefix) + strings.TrimPrefix(string(x), keychainNamespaceHead))
}

const (
	envVarSeparator       = "="
	envVarFileLoader      = "&"
	keychainNamespaceHead = "@"
)

func (x Argument) HasEnvVarSeparator() bool {
	return strings.Index(string(x), envVarSeparator) > 0
}

func (x Argument) ToEnvVar() (EnvKey, EnvValue) {
	v := strings.SplitN(string(x), envVarSeparator, 2)
	return EnvKey(v[0]), EnvValue(v[1])
}

func (x Argument) IsKeychainNamespace() bool {
	return isKeychainNamespace(string(x))
}

func (x Argument) ReplaceAll(key EnvKey, value EnvValue) Argument {
	return Argument(strings.Replace(string(x), string(key), string(value), -1))
}

var envVarNameRegex = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

func (x EnvKey) Validate() error {
	if !envVarNameRegex.MatchString(string(x)) {
		return goerr.Wrap(ErrInvalidArgument, "malformed environment variable name").With("name", x)
	}
	return nil
}

func (x EnvValue) IsFilePath() bool {
	return strings.HasPrefix(string(x), envVarFileLoader) &&
		len(x) > len(envVarFileLoader)
}

func (x EnvValue) ToFilePath() FilePath {
	return FilePath(strings.TrimPrefix(string(x), envVarFileLoader))
}

func (x EnvValue) ToHiddenValue() EnvValue {
	return EnvValue(strings.Repeat("*", len(x)) + " (hidden)")
}
