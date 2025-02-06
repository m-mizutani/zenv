package types

import "github.com/m-mizutani/goerr"

var (
	ErrInvalidArgumentFormat = goerr.New("invalid argument format")
	ErrInvalidArgument       = goerr.New("invalid argument")
	ErrNotEnoughArgument     = goerr.New("not enough arguments")
	ErrKeychainNotFound      = goerr.New("keychain item not found")
	ErrKeychainQueryFailed   = goerr.New("failed to query keychain item")
	ErrKeychainNotSupported  = goerr.New("keychain is not supported on the OS")
	ErrEnvVarInvalidName     = goerr.New("invalid environment variable name")
	ErrGenerateRandom        = goerr.New("crypto/rand failed")
	ErrInnerCommandFailed    = goerr.New("inner command failed")
)

type IgnoreError string

const (
	IgnoreEnvFileOpen IgnoreError = "env_file_open"
)

var ignoreErrorMap = map[IgnoreError]struct{}{
	IgnoreEnvFileOpen: {},
}

func IsIgnoreErrorCode(s string) bool {
	_, ok := ignoreErrorMap[IgnoreError(s)]
	return ok
}
