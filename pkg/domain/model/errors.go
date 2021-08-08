package model

import "github.com/m-mizutani/goerr"

var (
	ErrInvalidArgumentFormat    = goerr.New("invalid argument format")
	ErrNotEnoughArgument        = goerr.New("not enough arguments")
	ErrKeychainNotFound         = goerr.New("keychain item not found")
	ErrKeychainQueryFailed      = goerr.New("failed to query keychain item")
	ErrKeychainInvalidNamespace = goerr.New("invalid keychain namespace")
	ErrKeychainNotSupported     = goerr.New("keychain is not supported on the OS")
)
