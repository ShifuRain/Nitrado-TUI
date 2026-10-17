package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"nitui/internal/api"
	"nitui/internal/state"
)

type reqLog struct {
	mu   sync.Mutex
	reqs []string // "METHOD /path?query body"
}

func (l *reqLog) add(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reqs = append(l.reqs, s)
}

func (l *reqLog) has(sub string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.reqs {
		if strings.Contains(r, sub) {
			return true
		}
	}
	return false
}

// routeServer answers by "METHOD /path" prefix match against routes (longest
// match wins) and logs every request. Unmatched requests get a 404.
func routeServer(t *testing.T, routes map[string]string) (*httptest.Server, *reqLog) {
	t.Helper()
	log := &reqLog{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.add(r.Method + " " + r.URL.RequestURI() + " " + string(body))
		key := r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if resp, ok := routes[key]; ok {
			_, _ = w.Write([]byte(resp))
			return
		}
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"status":"error","message":"no route"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, log
}

func setup(t *testing.T, routes map[string]string) *reqLog {
	t.Helper()
	srv, log := routeServer(t, routes)
	store := withTestEnv(t, srv)
	_ = store.Save("token")
	if err := state.SetSelectedServer("17"); err != nil {
		t.Fatal(err)
	}
	return log
}

const ok = `{"status":"success","data":{}}`

func gsBody(status string) string {
	return `{"data":{"gameserver":{"status":"` + status + `","game":"valheim","game_human":"Valheim","ip":"1.2.3.4","port":2456,"slots":10,` +
		`"settings":{"config":{"name":"My World","slots":"10"},"general":{"motd":"hi"}},"game_specific":{"update_status":"up_to_date"}}}}`
}

func TestServerStatus_ShowsUpdateStatusAndJSON(t *testing.T) {
	setup(t, map[string]string{"GET /services/17/gameservers": gsBody("started")})

	out, err := runCLI(t, "server", "status")
	if err != nil || !strings.Contains(out, "started") || !strings.Contains(out, "update: up_to_date") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	out, err = runCLI(t, "server", "status", "--json")
	if err != nil || !strings.Contains(out, `"Status": "started"`) && !strings.Contains(out, `"status": "started"`) {
		t.Fatalf("json out=%q err=%v", out, err)
	}
}

func TestServerFlag_OverridesSelection(t *testing.T) {
	log := setup(t, map[string]string{"GET /services/99/gameservers": gsBody("started")})

	if _, err := runCLI(t, "server", "status", "--server", "99"); err != nil {
		t.Fatal(err)
	}
	if !log.has("GET /services/99/gameservers") {
		t.Errorf("expected request for server 99, got %v", log.reqs)
	}
	if _, err := runCLI(t, "server", "status", "--server", "-4"); err == nil {
		t.Error("negative --server should be rejected")
	}
}

func TestServerRestart_WaitPollsUntilStartedAgain(t *testing.T) {
	prev := pollInterval
	pollInterval = time.Millisecond
	t.Cleanup(func() { pollInterval = prev })

	var mu sync.Mutex
	polls := 0
	seq := []string{"started", "restarting", "restarting", "started"}
	mux := http.NewServeMux()
	mux.HandleFunc("/services/17/gameservers/restart", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(ok))
	})
	mux.HandleFunc("/services/17/gameservers", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		s := seq[min(polls, len(seq)-1)]
		polls++
		mu.Unlock()
		_, _ = w.Write([]byte(gsBody(s)))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	store := withTestEnv(t, srv)
	_ = store.Save("token")
	_ = state.SetSelectedServer("17")

	out, err := runCLI(t, "server", "restart", "--wait", "--message", "brb")
	if err != nil {
		t.Fatalf("err = %v, out = %q", err, out)
	}
	if !strings.Contains(out, "restarting") || !strings.Contains(out, "restarted: server 17") {
		t.Errorf("out = %q, want progress and completion", out)
	}
	if polls < 4 {
		t.Errorf("polls = %d, want >= 4", polls)
	}
}

func TestServerStop_WaitTimesOut(t *testing.T) {
	prev := pollInterval
	pollInterval = time.Millisecond
	t.Cleanup(func() { pollInterval = prev })
	setup(t, map[string]string{
		"POST /services/17/gameservers/stop": ok,
		"GET /services/17/gameservers":       gsBody("stopping"),
	})

	_, err := runCLI(t, "server", "stop", "--wait", "--timeout", "30ms")
	if err == nil || !strings.Contains(err.Error(), "stopping") {
		t.Fatalf("err = %v, want a timeout mentioning the last status", err)
	}
}

func TestSettings_ListGetSetReset(t *testing.T) {
	log := setup(t, map[string]string{
		"GET /services/17/gameservers":                   gsBody("started"),
		"GET /services/17/gameservers/settings/defaults": `{"data":{"settings":{"config":{"name":"Default","slots":"10"},"general":{"motd":"hi"}}}}`,
		"POST /services/17/gameservers/settings":         ok,
		"DELETE /services/17/gameservers/settings":       ok,
	})

	out, err := runCLI(t, "settings", "list")
	if err != nil || !strings.Contains(out, "My World *") || strings.Contains(out, "hi *") {
		t.Fatalf("list out=%q err=%v (only 'name' differs from default)", out, err)
	}
	if _, err := runCLI(t, "settings", "list", "nope"); err == nil {
		t.Error("unknown category should error")
	}
	out, err = runCLI(t, "settings", "get", "config", "name")
	if err != nil || strings.TrimSpace(out) != "My World" {
		t.Fatalf("get out=%q err=%v", out, err)
	}
	if _, err := runCLI(t, "settings", "get", "config", "missing"); err == nil {
		t.Error("unknown key should error")
	}
	if _, err := runCLI(t, "settings", "set", "config", "name", "New Name"); err != nil {
		t.Fatal(err)
	}
	if !log.has(`"key":"name"`) || !log.has(`"value":"New Name"`) {
		t.Errorf("set request not seen: %v", log.reqs)
	}
	if _, err := runCLI(t, "settings", "reset"); err == nil {
		t.Error("reset without --yes and without a terminal must refuse")
	}
	if _, err := runCLI(t, "settings", "reset", "--yes"); err != nil {
		t.Fatal(err)
	}
	if !log.has("DELETE /services/17/gameservers/settings") {
		t.Error("reset should call DELETE")
	}
}

func TestConfirm_InteractivePromptAcceptsYes(t *testing.T) {
	prev := stdinIsTerminal
	stdinIsTerminal = func() bool { return true }
	t.Cleanup(func() { stdinIsTerminal = prev })
	log := setup(t, map[string]string{"DELETE /services/17/gameservers/settings": ok})

	if _, err := runCLISTDIN(t, "n\n", "settings", "reset"); err == nil {
		t.Error("answering n must cancel")
	}
	if log.has("DELETE") {
		t.Error("nothing should have been deleted")
	}
	if _, err := runCLISTDIN(t, "y\n", "settings", "reset"); err != nil {
		t.Fatalf("answering y: %v", err)
	}
	if !log.has("DELETE") {
		t.Error("expected delete after y")
	}
}

func TestRestart_NoTerminalDoesNotBlockScripts(t *testing.T) {
	log := setup(t, map[string]string{"POST /services/17/gameservers/restart": ok})
	if _, err := runCLI(t, "server", "restart"); err != nil {
		t.Fatalf("restart without a terminal must still work for scripts: %v", err)
	}
	if !log.has("POST /services/17/gameservers/restart") {
		t.Error("restart not sent")
	}
}

func TestBackups_Commands(t *testing.T) {
	log := setup(t, map[string]string{
		"GET /services/17/backups":          `{"data":{"backups":[{"name":"b1","size":5,"created_at":"yesterday"}]}}`,
		"POST /services/17/backups":         ok,
		"POST /services/17/backups/extract": ok,
		"DELETE /services/17/backups":       ok,
	})

	out, err := runCLI(t, "backups", "list")
	if err != nil || !strings.Contains(out, "b1") || !strings.Contains(out, "yesterday") {
		t.Fatalf("list out=%q err=%v", out, err)
	}
	if _, err := runCLI(t, "backups", "create"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, "backups", "restore", "b1"); err == nil {
		t.Error("restore needs --yes")
	}
	if _, err := runCLI(t, "backups", "restore", "b1", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCLI(t, "backups", "delete", "b1", "--yes"); err != nil {
		t.Fatal(err)
	}
	if !log.has(`"name":"b1"`) || !log.has("DELETE /services/17/backups?prefix=b1") {
		t.Errorf("requests = %v", log.reqs)
	}
}

func TestFiles_LsAndMkdirAndRm(t *testing.T) {
	log := setup(t, map[string]string{
		"GET /services/17/gameservers/file_server/list":      `{"data":{"entries":[{"type":"dir","name":"saves","path":"/g/saves"},{"type":"file","name":"a.cfg","path":"/g/a.cfg","size":7}]}}`,
		"POST /services/17/gameservers/file_server/mkdir":    ok,
		"DELETE /services/17/gameservers/file_server/delete": ok,
	})

	out, err := runCLI(t, "files", "ls", "/g")
	if err != nil || !strings.Contains(out, "saves/") || !strings.Contains(out, "a.cfg") {
		t.Fatalf("ls out=%q err=%v", out, err)
	}
	if _, err := runCLI(t, "files", "mkdir", "/g/new"); err != nil {
		t.Fatal(err)
	}
	if !log.has(`"path":"/g/"`) || !log.has(`"name":"new"`) {
		t.Errorf("mkdir requests = %v", log.reqs)
	}
	if _, err := runCLI(t, "files", "rm", "/g/a.cfg"); err == nil {
		t.Error("rm needs --yes")
	}
	if _, err := runCLI(t, "files", "rm", "/g/a.cfg", "--yes"); err != nil {
		t.Fatal(err)
	}
}

func TestCommandAndLogs(t *testing.T) {
	log := setup(t, map[string]string{
		"POST /services/17/gameservers/command":   ok,
		"GET /services/17/gameservers/admin_logs": `{"data":{"logs":[{"date":"d","message":"kicked bob"}]}}`,
	})

	out, err := runCLI(t, "command", "say", "hello", "world")
	if err != nil || !strings.Contains(out, "say hello world") || !log.has(`"command":"say hello world"`) {
		t.Fatalf("out=%q err=%v reqs=%v", out, err, log.reqs)
	}
	out, err = runCLI(t, "logs")
	if err != nil || !strings.Contains(out, "d kicked bob") {
		t.Fatalf("logs out=%q err=%v", out, err)
	}
}

func TestGames_StartUninstallAndSwitchPicksStartForInstalled(t *testing.T) {
	log := setup(t, map[string]string{
		"GET /services/17/gameservers/games":              gamesListBody,
		"POST /services/17/gameservers/games/start":       ok,
		"POST /services/17/gameservers/games/install":     ok,
		"DELETE /services/17/gameservers/games/uninstall": ok,
	})

	// valheim is not installed -> install; enshrouded installed+active -> no-op.
	if _, err := runCLI(t, "switch", "valheim"); err != nil {
		t.Fatal(err)
	}
	if !log.has("POST /services/17/gameservers/games/install") {
		t.Errorf("uninstalled game should be installed: %v", log.reqs)
	}
	out, err := runCLI(t, "switch", "enshrouded")
	if err != nil || !strings.Contains(out, "already the active") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if _, err := runCLI(t, "games", "start", "enshrouded"); err != nil || !log.has("games/start") {
		t.Fatalf("start err=%v", err)
	}
	if _, err := runCLI(t, "games", "uninstall", "valheim"); err == nil {
		t.Error("uninstall needs --yes")
	}
	if _, err := runCLI(t, "games", "uninstall", "valheim", "--yes"); err != nil || !log.has("DELETE /services/17/gameservers/games/uninstall") {
		t.Fatalf("uninstall err=%v", err)
	}
}

func TestSwitch_InstalledInactiveUsesStart(t *testing.T) {
	log := setup(t, map[string]string{
		"GET /services/17/gameservers/games":        `{"data":{"games":[{"portlist_short":"valheim","name":"Valheim","installed":true,"active":false,"enough_slots":true}]}}`,
		"POST /services/17/gameservers/games/start": ok,
	})
	if _, err := runCLI(t, "switch", "valheim"); err != nil {
		t.Fatal(err)
	}
	if !log.has("games/start") || log.has("games/install") {
		t.Errorf("requests = %v", log.reqs)
	}
}

func TestServersList_JSON(t *testing.T) {
	setup(t, map[string]string{"GET /services": `{"data":{"services":[{"id":17,"status":"active","details":{"game":"Valheim"}}]}}`})
	out, err := runCLI(t, "servers", "list", "--json")
	if err != nil || !strings.Contains(out, `"id": 17`) {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestWatchStatus_PrintsOnlyChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seq := []string{"started", "started", "restarting", "restarting", "started"}
	n := 0
	fetch := func(context.Context) (*api.GameServer, error) {
		s := seq[min(n, len(seq)-1)]
		n++
		if n > len(seq) {
			cancel()
		}
		return &api.GameServer{Status: s}, nil
	}
	var out bytes.Buffer
	if err := watchStatus(ctx, &out, 1, fetch, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 change lines (started, restarting, started), got %d: %q", len(lines), out.String())
	}
	if !strings.Contains(lines[1], "restarting") {
		t.Errorf("lines = %q", lines)
	}
}

func TestFormatStatusLine_IncludesUpdateStatus(t *testing.T) {
	if !strings.Contains(formatStatusLine(1, mustGS(t)), "update: up_to_date") {
		t.Error("status line should include update status")
	}
}
