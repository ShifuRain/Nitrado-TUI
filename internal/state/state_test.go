package state

import (
	"errors"
	"testing"
)

func withTempConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
}

func TestSelectedServer_NoneSelectedYet(t *testing.T) {
	withTempConfigDir(t)

	if _, err := SelectedServer(); !errors.Is(err, ErrNoServerSelected) {
		t.Errorf("SelectedServer() = %v, want ErrNoServerSelected", err)
	}
}

func TestSetSelectedServer_PersistsAcrossLoads(t *testing.T) {
	withTempConfigDir(t)

	if err := SetSelectedServer("17732920"); err != nil {
		t.Fatalf("SetSelectedServer() error = %v", err)
	}

	got, err := SelectedServer()
	if err != nil {
		t.Fatalf("SelectedServer() error = %v", err)
	}
	if got != "17732920" {
		t.Errorf("SelectedServer() = %q, want %q", got, "17732920")
	}
}

func TestSetSelectedServer_Overwrites(t *testing.T) {
	withTempConfigDir(t)

	_ = SetSelectedServer("1")
	_ = SetSelectedServer("2")

	got, err := SelectedServer()
	if err != nil {
		t.Fatalf("SelectedServer() error = %v", err)
	}
	if got != "2" {
		t.Errorf("SelectedServer() = %q, want %q (overwritten)", got, "2")
	}
}
