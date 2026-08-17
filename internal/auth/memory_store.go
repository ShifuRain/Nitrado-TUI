package auth

// MemoryStore is an in-memory Store implementation. It exists because the
// OS keychain isn't available in headless/CI/container environments (e.g.
// this project's own devcontainer has no Secret Service daemon), so tests
// and local dev builds can swap it in instead of KeychainStore.
type MemoryStore struct {
	token string
	set   bool
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }

func (m *MemoryStore) Save(token string) error {
	m.token, m.set = token, true
	return nil
}

func (m *MemoryStore) Get() (string, error) {
	if !m.set {
		return "", ErrNotLoggedIn
	}
	return m.token, nil
}

func (m *MemoryStore) Delete() error {
	if !m.set {
		return ErrNotLoggedIn
	}
	m.token, m.set = "", false
	return nil
}
