package config

import (
	"errors"
	"os"
	"testing"
)

func withTempConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
}

func TestLoad_MissingFileYieldsDefaults(t *testing.T) {
	withTempConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Theme != DefaultTheme() {
		t.Errorf("Load() with no file = %+v, want defaults %+v", cfg.Theme, DefaultTheme())
	}
}

func TestLoad_UserOverridesMergeWithDefaults(t *testing.T) {
	withTempConfigDir(t)

	path, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	yaml := "theme:\n  colors:\n    accent: \"#ABCDEF\"\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Theme.Colors.Accent != "#ABCDEF" {
		t.Errorf("Accent = %q, want user override %q", cfg.Theme.Colors.Accent, "#ABCDEF")
	}
	if cfg.Theme.Colors.Primary != DefaultTheme().Colors.Primary {
		t.Errorf("Primary = %q, want default fallback", cfg.Theme.Colors.Primary)
	}
}

func TestWriteDefault_DoesNotOverwriteExisting(t *testing.T) {
	withTempConfigDir(t)

	path1, err := WriteDefault()
	if err != nil {
		t.Fatalf("first WriteDefault() error = %v", err)
	}
	if err := os.WriteFile(path1, []byte("custom content"), 0o644); err != nil {
		t.Fatalf("overwriting for test: %v", err)
	}

	_, err = WriteDefault()
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("second WriteDefault() error = %v, want os.ErrExist", err)
	}

	data, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(data) != "custom content" {
		t.Error("WriteDefault() overwrote an existing config file")
	}
}
