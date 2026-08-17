package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSelect_ServerOnAccount(t *testing.T) {
	srv := jsonServer(t, 200, okServicesBody) // contains server 17732920
	store := withTestEnv(t, srv)
	_ = store.Save("token")

	out, err := runCLI(t, "select", "17732920")
	if err != nil {
		t.Fatalf("select error = %v", err)
	}
	if !strings.Contains(out, "Selected server 17732920") {
		t.Errorf("output = %q", out)
	}
}

func TestSelect_ServerNotOnAccount(t *testing.T) {
	srv := jsonServer(t, 200, okServicesBody) // does not contain 999
	store := withTestEnv(t, srv)
	_ = store.Save("token")

	_, err := runCLI(t, "select", "999")
	if err == nil {
		t.Fatal("select on a server id not on the account: expected an error")
	}
	if !strings.Contains(err.Error(), "servers list") {
		t.Errorf("error = %q, want it to suggest `nitui servers list`", err)
	}
}

func TestSelect_NonNumericID(t *testing.T) {
	store := withTestEnv(t, nil)
	_ = store.Save("token")

	_, err := runCLI(t, "select", "not-a-number")
	if err == nil {
		t.Fatal("select with a non-numeric id: expected an error")
	}
}

func TestSwitch_NoServerSelected(t *testing.T) {
	store := withTestEnv(t, nil)
	_ = store.Save("token")

	_, err := runCLI(t, "switch", "valheim")
	if err == nil {
		t.Fatal("switch with no server selected: expected an error")
	}
	if !strings.Contains(err.Error(), "nitui select") {
		t.Errorf("error = %q, want it to suggest `nitui select`", err)
	}
}

func TestSwitch_CallsInstallWithSelectedServerAndGame(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/services") && !strings.Contains(r.URL.Path, "gameservers") {
			// ListServices call made by `select`.
			w.Write([]byte(okServicesBody)) //nolint:errcheck
			return
		}
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Write([]byte(`{"status":"success","data":{}}`)) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)

	store := withTestEnv(t, srv)
	_ = store.Save("token")

	if _, err := runCLI(t, "select", "17732920"); err != nil {
		t.Fatalf("select error = %v", err)
	}
	out, err := runCLI(t, "switch", "valheim")
	if err != nil {
		t.Fatalf("switch error = %v, output: %s", err, out)
	}

	if gotPath != "/services/17732920/gameservers/games/install" {
		t.Errorf("path = %q, want /services/17732920/gameservers/games/install", gotPath)
	}
	if !strings.Contains(gotBody, `"game":"valheim"`) {
		t.Errorf("body = %q, want it to contain the game slug", gotBody)
	}
}

func TestSwitch_LimitReachedShowsGuidance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/services") && !strings.Contains(r.URL.Path, "gameservers") {
			w.Write([]byte(okServicesBody)) //nolint:errcheck
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status":"error","message":"This server has reached the limit of 5 installed games."}`)) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)

	store := withTestEnv(t, srv)
	_ = store.Save("token")
	_, _ = runCLI(t, "select", "17732920")

	_, err := runCLI(t, "switch", "newgame")
	if err == nil {
		t.Fatal("expected an error when the games limit is hit")
	}
	if !strings.Contains(err.Error(), "Uninstall a game") {
		t.Errorf("error = %q, want actionable limit guidance", err)
	}
}
