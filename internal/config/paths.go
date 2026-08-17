package config

import (
	"os"
	"path/filepath"
)

// Dir returns the directory nitui stores its config and state in.
// NITUI_CONFIG_DIR overrides it outright when set (useful for portable
// installs, and for tests that don't want to touch a real user profile).
// Otherwise it follows each OS's conventional location (%AppData% on
// Windows, ~/Library/Application Support on macOS, $XDG_CONFIG_HOME or
// ~/.config on Linux). Created if it doesn't exist yet.
func Dir() (string, error) {
	if override := os.Getenv("NITUI_CONFIG_DIR"); override != "" {
		if err := os.MkdirAll(override, 0o700); err != nil {
			return "", err
		}
		return override, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "nitui")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// FilePath returns the absolute path to a named file inside the nitui
// config directory (e.g. "config.yaml", "state.yaml").
func FilePath(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
