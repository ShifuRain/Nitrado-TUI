// Package state persists small bits of local, non-secret CLI state —
// currently just which server `nitui select` last pointed at — so that
// follow-up commands like `nitui switch` don't need the server id repeated.
package state

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"

	"nitui/internal/config"
)

const fileName = "state.yaml"

// ErrNoServerSelected is returned when a command needs a selected server
// but none has been chosen yet via `nitui select`.
var ErrNoServerSelected = errors.New("no server selected: run `nitui select <server_id>` first")

type State struct {
	SelectedServerID string `yaml:"selected_server_id"`
}

func load() (State, error) {
	var s State
	path, err := config.FilePath(fileName)
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := yaml.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

func save(s State) error {
	path, err := config.FilePath(fileName)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SelectedServer returns the currently selected server id, or
// ErrNoServerSelected if none has been set.
func SelectedServer() (string, error) {
	s, err := load()
	if err != nil {
		return "", err
	}
	if s.SelectedServerID == "" {
		return "", ErrNoServerSelected
	}
	return s.SelectedServerID, nil
}

// SetSelectedServer records id as the active server for follow-up commands.
func SetSelectedServer(id string) error {
	s, err := load()
	if err != nil {
		return err
	}
	s.SelectedServerID = id
	return save(s)
}
