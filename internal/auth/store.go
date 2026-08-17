// Package auth handles storage and retrieval of the user's Nitrado
// long-life API token.
package auth

import "errors"

// ErrNotLoggedIn is returned by Store.Get when no token has been saved yet.
var ErrNotLoggedIn = errors.New("not logged in: run `nitui auth login` first")

// Store persists the Nitrado API token. The production implementation
// (KeychainStore) uses the OS-native credential store; tests and other
// callers can substitute an in-memory implementation.
type Store interface {
	Save(token string) error
	Get() (string, error)
	Delete() error
}
