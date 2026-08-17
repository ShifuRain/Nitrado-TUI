package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// recordingServer captures the last request's method, path, and decoded
// JSON body, and replies with a canned success envelope.
type recordingServer struct {
	*httptest.Server
	Method, Path string
	Body         map[string]any
}

func newRecordingServer(t *testing.T, respBody string) *recordingServer {
	t.Helper()
	rs := &recordingServer{}
	rs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.Method = r.Method
		rs.Path = r.URL.Path
		if r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&rs.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(rs.Server.Close)
	return rs
}

func TestServiceActions_EndpointsAndPayloads(t *testing.T) {
	const ok = `{"status":"success","data":{}}`

	tests := []struct {
		name       string
		call       func(c *Client) error
		wantMethod string
		wantPath   string
		wantBody   map[string]any
	}{
		{
			name:       "Restart",
			call:       func(c *Client) error { return c.Restart(context.Background(), 17) },
			wantMethod: "POST",
			wantPath:   "/services/17/gameservers/restart",
		},
		{
			name:       "Stop",
			call:       func(c *Client) error { return c.Stop(context.Background(), 17) },
			wantMethod: "POST",
			wantPath:   "/services/17/gameservers/stop",
		},
		{
			name:       "SwitchGame",
			call:       func(c *Client) error { return c.SwitchGame(context.Background(), 17, "valheim") },
			wantMethod: "POST",
			wantPath:   "/services/17/gameservers/games/install",
			wantBody:   map[string]any{"game": "valheim"},
		},
		{
			name:       "StartGame",
			call:       func(c *Client) error { return c.StartGame(context.Background(), 17, "valheim") },
			wantMethod: "POST",
			wantPath:   "/services/17/gameservers/games/start",
			wantBody:   map[string]any{"game": "valheim"},
		},
		{
			name:       "UninstallGame",
			call:       func(c *Client) error { return c.UninstallGame(context.Background(), 17, "valheim") },
			wantMethod: "DELETE",
			wantPath:   "/services/17/gameservers/games/uninstall",
			wantBody:   map[string]any{"game": "valheim"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rs := newRecordingServer(t, ok)
			c := New("t", WithBaseURL(rs.URL))
			if err := tc.call(c); err != nil {
				t.Fatalf("%s() error = %v", tc.name, err)
			}
			if rs.Method != tc.wantMethod {
				t.Errorf("method = %q, want %q", rs.Method, tc.wantMethod)
			}
			if rs.Path != tc.wantPath {
				t.Errorf("path = %q, want %q", rs.Path, tc.wantPath)
			}
			if tc.wantBody != nil {
				for k, v := range tc.wantBody {
					if rs.Body[k] != v {
						t.Errorf("body[%q] = %v, want %v", k, rs.Body[k], v)
					}
				}
			}
		})
	}
}

func TestListServices(t *testing.T) {
	rs := newRecordingServer(t, `{"status":"success","data":{"services":[
		{"id":1,"status":"active","details":{"game":"Enshrouded","address":"1.2.3.4:1000","slots":10}}
	]}}`)
	c := New("t", WithBaseURL(rs.URL))

	services, err := c.ListServices(context.Background())
	if err != nil {
		t.Fatalf("ListServices() error = %v", err)
	}
	if rs.Method != "GET" || rs.Path != "/services" {
		t.Errorf("request = %s %s, want GET /services", rs.Method, rs.Path)
	}
	if len(services) != 1 || services[0].Details.Game != "Enshrouded" {
		t.Errorf("services = %+v, want one service for Enshrouded", services)
	}
}

func TestGetGameServer_SetsServiceID(t *testing.T) {
	rs := newRecordingServer(t, `{"status":"success","data":{"gameserver":{
		"status":"started","ip":"1.2.3.4","port":1000,"game":"enshrouded","game_human":"Enshrouded","slots":10
	}}}`)
	c := New("t", WithBaseURL(rs.URL))

	gs, err := c.GetGameServer(context.Background(), 17732920)
	if err != nil {
		t.Fatalf("GetGameServer() error = %v", err)
	}
	if rs.Path != "/services/17732920/gameservers" {
		t.Errorf("path = %q, want /services/17732920/gameservers", rs.Path)
	}
	if gs.ServiceID != 17732920 {
		t.Errorf("ServiceID = %d, want 17732920 (not part of the JSON, set by the client)", gs.ServiceID)
	}
	if gs.Address() != "1.2.3.4:1000" {
		t.Errorf("Address() = %q, want 1.2.3.4:1000", gs.Address())
	}
}

func TestListGames(t *testing.T) {
	rs := newRecordingServer(t, `{"status":"success","data":{"games":[
		{"id":906,"portlist_short":"valheim","name":"Valheim","installed":false,"active":false,"enough_slots":true,"too_many_slots":false},
		{"id":231,"portlist_short":"enshrouded","name":"Enshrouded","installed":true,"active":true,"enough_slots":true,"too_many_slots":false}
	]}}`)
	c := New("t", WithBaseURL(rs.URL))

	games, err := c.ListGames(context.Background(), 17)
	if err != nil {
		t.Fatalf("ListGames() error = %v", err)
	}
	if rs.Path != "/services/17/gameservers/games" {
		t.Errorf("path = %q, want /services/17/gameservers/games", rs.Path)
	}
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
	if games[1].Slug != "enshrouded" || !games[1].Active {
		t.Errorf("games[1] = %+v, want active enshrouded", games[1])
	}
}
