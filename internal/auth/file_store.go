package auth

import (
	"errors"
	"os"
	"path/filepath"
)

// FileStore stores the token in plaintext in a file under the given
// directory. This is NOT for production use — it exists only so the token
// can survive across separate `nitui` process invocations in environments
// with no OS keychain (namely this project's own devcontainer, which has
// no D-Bus Secret Service). Selected via NITUI_TOKEN_STORE=file; see
// cli.Execute. Real installs always default to KeychainStore.
type FileStore struct {
	path string
}

func NewFileStore(dir string) FileStore {
	return FileStore{path: filepath.Join(dir, "dev-token")}
}

func (f FileStore) Save(token string) error {
	return os.WriteFile(f.path, []byte(token), 0o600)
}

func (f FileStore) Get() (string, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNotLoggedIn
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f FileStore) Delete() error {
	err := os.Remove(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotLoggedIn
	}
	return err
}
