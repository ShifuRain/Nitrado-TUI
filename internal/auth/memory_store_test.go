package auth

import (
	"errors"
	"testing"
)

func TestMemoryStore_RoundTrip(t *testing.T) {
	s := NewMemoryStore()

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
