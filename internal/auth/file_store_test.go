package auth

import (
	"errors"
	"testing"
)

func TestFileStore_RoundTrip(t *testing.T) {
	s := NewFileStore(t.TempDir())

	if _, err := s.Get(); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("Get() before Save() = %v, want ErrNotLoggedIn", err)
	}

	if err := s.Save("secret-token"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "secret-token" {
		t.Errorf("Get() = %q, want %q", got, "secret-token")
	}

	if err := s.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.Get(); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("Get() after Delete() = %v, want ErrNotLoggedIn", err)
	}
	if err := s.Delete(); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("second Delete() = %v, want ErrNotLoggedIn", err)
	}
}

func TestFileStore_OverwritesOnSecondSave(t *testing.T) {
	s := NewFileStore(t.TempDir())
	_ = s.Save("first")
	_ = s.Save("second")

	got, err := s.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "second" {
		t.Errorf("Get() = %q, want %q (overwritten)", got, "second")
	}
}
