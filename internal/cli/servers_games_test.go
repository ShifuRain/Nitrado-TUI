package cli

import (
	"strings"
	"testing"

	"nitui/internal/state"
)

func TestServersList_ShowsAccountServers(t *testing.T) {
	srv := jsonServer(t, 200, `{"status":"success","data":{"services":[
		{"id":17732920,"status":"active","details":{"game":"Enshrouded","address":"1.2.3.4:1000","slots":10}}
	]}}`)
	store := withTestEnv(t, srv)
	_ = store.Save("token")

	out, err := runCLI(t, "servers", "list")
	if err != nil {
		t.Fatalf("servers list error = %v", err)
	}
	for _, want := range []string{"17732920", "active", "Enshrouded", "1.2.3.4:1000"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestServersList_NotLoggedIn(t *testing.T) {
	withTestEnv(t, nil)

	_, err := runCLI(t, "servers", "list")
	if err == nil {
		t.Fatal("servers list while logged out: expected an error")
	}
	if !strings.Contains(err.Error(), "nitui auth login") {
		t.Errorf("error = %q, want it to suggest logging in", err)
	}
}

const gamesListBody = `{"status":"success","data":{"games":[
	{"id":906,"portlist_short":"valheim","name":"Valheim","installed":false,"active":false,"enough_slots":true,"too_many_slots":false},
	{"id":231,"portlist_short":"enshrouded","name":"Enshrouded","installed":true,"active":true,"enough_slots":true,"too_many_slots":false}
]}}`

func TestGamesList_ShowsAllGames(t *testing.T) {
	srv := jsonServer(t, 200, gamesListBody)
	store := withTestEnv(t, srv)
	_ = store.Save("token")
	if err := state.SetSelectedServer("17732920"); err != nil {
		t.Fatalf("SetSelectedServer() error = %v", err)
	}

	out, err := runCLI(t, "games", "list")
	if err != nil {
		t.Fatalf("games list error = %v", err)
	}
	if !strings.Contains(out, "valheim") || !strings.Contains(out, "enshrouded") {
		t.Errorf("output = %q, want both games listed", out)
	}
}

func TestGamesInstalled_FiltersToInstalledOnly(t *testing.T) {
	srv := jsonServer(t, 200, gamesListBody)
	store := withTestEnv(t, srv)
	_ = store.Save("token")
	if err := state.SetSelectedServer("17732920"); err != nil {
		t.Fatalf("SetSelectedServer() error = %v", err)
	}

	out, err := runCLI(t, "games", "installed")
	if err != nil {
		t.Fatalf("games installed error = %v", err)
	}
	if strings.Contains(out, "valheim") {
		t.Errorf("output = %q, should not list valheim (not installed)", out)
	}
	if !strings.Contains(out, "enshrouded") {
		t.Errorf("output = %q, should list enshrouded (installed)", out)
	}
}
