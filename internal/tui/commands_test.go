package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nitui/internal/api"
	"nitui/internal/auth"
)

func testClient(t *testing.T, status int, body string) *api.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return api.New("t", api.WithBaseURL(srv.URL))
}

func withTestBaseURL(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := apiBaseURL
	apiBaseURL = srv.URL
	t.Cleanup(func() { apiBaseURL = prev })
}

func TestDoLogin_SuccessSavesTokenAndReturnsClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"services":[]}}`))
	}))
	t.Cleanup(srv.Close)
	withTestBaseURL(t, srv)
	store := auth.NewMemoryStore()

	msg := doLogin(store, "my-token")()
	res, ok := msg.(loginResultMsg)
	if !ok {
		t.Fatalf("msg = %T, want loginResultMsg", msg)
	}
	if res.err != nil {
		t.Fatalf("loginResultMsg.err = %v", res.err)
	}
	if res.client == nil {
		t.Fatal("loginResultMsg.client is nil, want a client")
	}
	got, err := store.Get()
	if err != nil || got != "my-token" {
		t.Errorf("store.Get() = (%q, %v), want (my-token, nil)", got, err)
	}
}

func TestDoLogin_FailureDoesNotSaveToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"error","message":"invalid token"}`))
	}))
	t.Cleanup(srv.Close)
	withTestBaseURL(t, srv)
	store := auth.NewMemoryStore()

	msg := doLogin(store, "bad-token")()
	res := msg.(loginResultMsg)
	if res.err == nil {
		t.Fatal("expected an error for an invalid token")
	}
	if _, err := store.Get(); err != auth.ErrNotLoggedIn {
		t.Error("token should not be saved when validation fails")
	}
}

func TestLoadServers_DecodesServiceList(t *testing.T) {
	client := testClient(t, 200, `{"status":"success","data":{"services":[{"id":1,"status":"active"}]}}`)

	msg := loadServers(client)()
	res, ok := msg.(serversLoadedMsg)
	if !ok {
		t.Fatalf("msg = %T, want serversLoadedMsg", msg)
	}
	if res.err != nil || len(res.services) != 1 {
		t.Errorf("serversLoadedMsg = %+v", res)
	}
}

func TestLoadDetail_SetsServiceID(t *testing.T) {
	client := testClient(t, 200, `{"status":"success","data":{"gameserver":{"status":"started","game":"enshrouded"}}}`)

	msg := loadDetail(client, 42)()
	res, ok := msg.(detailLoadedMsg)
	if !ok {
		t.Fatalf("msg = %T, want detailLoadedMsg", msg)
	}
	if res.err != nil {
		t.Fatalf("detailLoadedMsg.err = %v", res.err)
	}
	if res.detail.ServiceID != 42 {
		t.Errorf("ServiceID = %d, want 42", res.detail.ServiceID)
	}
}

func TestDoSwitchGame_ReturnsActionDoneMsg(t *testing.T) {
	client := testClient(t, 200, `{"status":"success","data":{}}`)

	msg := doSwitchGame(client, 1, "valheim")()
	res, ok := msg.(actionDoneMsg)
	if !ok {
		t.Fatalf("msg = %T, want actionDoneMsg", msg)
	}
	if res.err != nil || res.verb != "Switched" {
		t.Errorf("actionDoneMsg = %+v", res)
	}
}

func TestDoSwitchGame_LimitErrorSurfaced(t *testing.T) {
	client := testClient(t, 400, `{"status":"error","message":"This server has reached the limit of 5 installed games."}`)

	msg := doSwitchGame(client, 1, "valheim")()
	res := msg.(actionDoneMsg)
	if res.err == nil {
		t.Fatal("expected an error when the games limit is hit")
	}
}
