package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"nitui/internal/auth"
)

// withTestEnv points authStore at a fresh in-memory store and apiBaseURL at
// srv for the duration of the test, restoring the previous values after.
// It also redirects config/state to a temp dir so tests never touch the
// real user profile.
func withTestEnv(t *testing.T, srv *httptest.Server) *auth.MemoryStore {
	t.Helper()
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())

	prevStore, prevURL := authStore, apiBaseURL
	store := auth.NewMemoryStore()
	authStore = store
	if srv != nil {
		apiBaseURL = srv.URL
	}
	t.Cleanup(func() {
		authStore, apiBaseURL = prevStore, prevURL
	})
	return store
}

// runCLI executes a fresh root command with args and returns combined
// stdout output and any error.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// jsonServer replies to every request with body, regardless of path.
func jsonServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}
