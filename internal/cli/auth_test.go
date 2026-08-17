package cli

import (
	"strings"
	"testing"

	"nitui/internal/auth"
)

const okServicesBody = `{"status":"success","data":{"services":[{"id":17732920,"status":"active","details":{"game":"Enshrouded"}}]}}`

func TestAuthLogin_Success(t *testing.T) {
	srv := jsonServer(t, 200, okServicesBody)
	store := withTestEnv(t, srv)

	out, err := runCLI(t, "auth", "login", "--token", "good-token")
	if err != nil {
		t.Fatalf("auth login error = %v, output: %s", err, out)
	}
	if !strings.Contains(out, "Logged in") {
		t.Errorf("output = %q, want it to mention logging in", out)
	}
	got, err := store.Get()
	if err != nil || got != "good-token" {
		t.Errorf("store.Get() = (%q, %v), want (good-token, nil)", got, err)
	}
}

func TestAuthLogin_InvalidTokenNotSaved(t *testing.T) {
	srv := jsonServer(t, 401, `{"status":"error","message":"invalid token"}`)
	store := withTestEnv(t, srv)

	out, err := runCLI(t, "auth", "login", "--token", "bad-token")
	if err == nil {
		t.Fatalf("auth login with bad token: expected an error, output: %s", out)
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("error = %q, want the friendly 'run auth login again' message", err)
	}
	if _, getErr := store.Get(); getErr != auth.ErrNotLoggedIn {
		t.Error("store should not have saved the token after a failed validation call")
	}
}

func TestAuthStatus_NotLoggedIn(t *testing.T) {
	withTestEnv(t, nil)

	out, err := runCLI(t, "auth", "status")
	if err != nil {
		t.Fatalf("auth status error = %v", err)
	}
	if !strings.Contains(out, "Not logged in") {
		t.Errorf("output = %q, want 'Not logged in'", out)
	}
}

func TestAuthStatus_LoggedIn(t *testing.T) {
	srv := jsonServer(t, 200, okServicesBody)
	store := withTestEnv(t, srv)
	_ = store.Save("good-token")

	out, err := runCLI(t, "auth", "status")
	if err != nil {
		t.Fatalf("auth status error = %v", err)
	}
	if !strings.Contains(out, "1 server(s)") {
		t.Errorf("output = %q, want it to report 1 server", out)
	}
}

func TestAuthLogoff_RemovesToken(t *testing.T) {
	store := withTestEnv(t, nil)
	_ = store.Save("good-token")

	out, err := runCLI(t, "auth", "logoff")
	if err != nil {
		t.Fatalf("auth logoff error = %v", err)
	}
	if !strings.Contains(out, "Logged off") {
		t.Errorf("output = %q, want 'Logged off'", out)
	}
	if _, getErr := store.Get(); getErr != auth.ErrNotLoggedIn {
		t.Error("token should be gone after logoff")
	}
}

func TestAuthLogoff_AlreadyLoggedOut(t *testing.T) {
	withTestEnv(t, nil)

	out, err := runCLI(t, "auth", "logoff")
	if err != nil {
		t.Fatalf("auth logoff error = %v", err)
	}
	if !strings.Contains(out, "Already logged out") {
		t.Errorf("output = %q, want 'Already logged out'", out)
	}
}
