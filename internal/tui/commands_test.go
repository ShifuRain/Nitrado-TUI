package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPollStatus_CarriesGenAndDetail(t *testing.T) {
	client := testClient(t, 200, `{"data":{"gameserver":{"status":"restarting"}}}`)
	res := pollStatus(client, 3, 9)().(statusPolledMsg)
	if res.err != nil || res.gen != 9 || res.detail.Status != "restarting" || res.detail.ServiceID != 3 {
		t.Errorf("res = %+v", res)
	}
}

func TestDoStopAndRestart_ReturnLifecycleMsgWithMessage(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 200)
		n, _ := r.Body.Read(b)
		body = string(b[:n])
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	t.Cleanup(srv.Close)
	client := api.New("t", api.WithBaseURL(srv.URL))

	res := doRestart(client, 1, "brb")().(lifecycleDoneMsg)
	if res.err != nil || res.verb != "restart" || !strings.Contains(body, "brb") {
		t.Errorf("res=%+v body=%q", res, body)
	}
	res = doStop(client, 1, "")().(lifecycleDoneMsg)
	if res.err != nil || res.verb != "stop" {
		t.Errorf("res=%+v", res)
	}
}

func TestCheckForUpdate_ReturnsMsgOnlyWhenNewer(t *testing.T) {
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","html_url":"https://example.test"}`))
	}))
	t.Cleanup(srv.Close)
	prev := updateURL
	updateURL = srv.URL
	t.Cleanup(func() { updateURL = prev })

	if msg, ok := checkForUpdate("1.0.0")().(updateAvailableMsg); !ok || msg.release.Version != "2.0.0" {
		t.Errorf("want updateAvailableMsg for 2.0.0, got %v", msg)
	}
	if msg := checkForUpdate("2.0.0")(); msg != nil {
		t.Errorf("up to date should yield nil, got %v", msg)
	}
	if msg := checkForUpdate("dev")(); msg != nil {
		t.Errorf("dev build should yield nil, got %v", msg)
	}
}

func TestSettingsAndBackupCommands(t *testing.T) {
	client := testClient(t, 200, `{"data":{"gameserver":{"status":"started","settings":{"a":{"b":"c"}}},"settings":{"a":{"b":"d"}},"backups":[{"name":"x"}]}}`)
	sl := loadSettings(client, 1)().(settingsLoadedMsg)
	if sl.err != nil || sl.detail.Settings["a"]["b"] != "c" {
		t.Errorf("settings = %+v", sl)
	}
	if bl := loadBackups(client, 1)().(backupsLoadedMsg); bl.err != nil || len(bl.backups) != 1 {
		t.Errorf("backups = %+v", bl)
	}
	if saved := saveSetting(client, 1, "a", "b", "v")().(settingSavedMsg); saved.err != nil || saved.value != "v" {
		t.Errorf("saved = %+v", saved)
	}
}
