package auth

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "nitui"
	keyringUser    = "api-token"
)

// KeychainStore stores the API token in the OS-native credential store:
// Windows Credential Manager, macOS Keychain, or the Secret Service on
// Linux (gnome-keyring, KWallet, etc. via D-Bus).
//
// On a Linux system with no Secret Service available (e.g. a bare headless
// server, or a container with no keyring daemon running) Save/Get/Delete
// will return an error explaining that rather than silently failing.
type KeychainStore struct{}

func NewKeychainStore() KeychainStore { return KeychainStore{} }

func (KeychainStore) Save(token string) error {
	if err := keyring.Set(keyringService, keyringUser, token); err != nil {
		return fmt.Errorf("saving token to the OS keychain: %w (no Secret Service/Keychain/Credential Manager available?)", err)
	}
	return nil
}

func (KeychainStore) Get() (string, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotLoggedIn
	}
	if err != nil {
		return "", fmt.Errorf("reading token from the OS keychain: %w", err)
	}
	return token, nil
}

func (KeychainStore) Delete() error {
	err := keyring.Delete(keyringService, keyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotLoggedIn
	}
	if err != nil {
		return fmt.Errorf("deleting token from the OS keychain: %w", err)
	}
	return nil
}
