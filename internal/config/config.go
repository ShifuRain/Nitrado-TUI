package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const fileName = "config.yaml"

// Config is the top-level shape of ~/.config/nitui/config.yaml (or the
// platform equivalent).
type Config struct {
	Theme Theme `yaml:"theme"`
}

// Load reads the user's config file, if present, and merges it over the
// defaults. A missing file is not an error — it simply yields defaults.
func Load() (Config, error) {
	cfg := Config{Theme: DefaultTheme()}

	path, err := FilePath(fileName)
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	var user Config
	if err := yaml.Unmarshal(data, &user); err != nil {
		return cfg, err
	}

	cfg.Theme = user.Theme.merge(DefaultTheme())
	return cfg, nil
}

// Path returns the on-disk location of the config file, for messages like
// "edit ~/.config/nitui/config.yaml to customize colors".
func Path() (string, error) {
	return FilePath(fileName)
}

// WriteDefault writes a fully-commented example config file to disk,
// without overwriting an existing one. Used by `nitui config init`.
func WriteDefault() (string, error) {
	path, err := FilePath(fileName)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, os.ErrExist
	}
	if err := os.WriteFile(path, []byte(exampleConfig), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

const exampleConfig = `# nitui config file
# Every field below is optional; omit anything you want left at its default.
# Colors accept a hex string ("#7D56F4") or an ANSI 256 color index ("212").

theme:
  # rounded | normal | thick | double | hidden
  border_style: rounded

  colors:
    primary: "#7D56F4"    # headers, active borders
    secondary: "#5A5A7A"  # secondary text, inactive borders
    accent: "#F25D94"     # selected list item, highlights
    text: "#E4E4E7"       # normal body text
    muted: "#71717A"      # dim/help text
    success: "#4ADE80"    # server running, success messages
    warning: "#FBBF24"    # approaching a limit, degraded state
    error: "#F87171"      # failures, limit reached
    background: ""        # leave empty to use the terminal's own background
`
