package model

import "github.com/m-mizutani/goerr"

var (
	ErrNotEnoughArgument        = goerr.New("not enough arguments")
	ErrKeychainNotFound         = goerr.New("keychain item not found")
	ErrKeychainQueryFailed      = goerr.New("failed to query keychain item")
	ErrKeychainInvalidNamespace = goerr.New("invalid keychain namespace")
)
