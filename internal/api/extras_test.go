package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFiles_ListAndTokenTransfers(t *testing.T) {
	var uploaded, uploadToken, downloadToken string
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/services/9/gameservers/file_server/list", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("dir") != "/game" {
			t.Errorf("dir = %q", r.URL.Query().Get("dir"))
		}
		_, _ = w.Write([]byte(`{"data":{"entries":[{"type":"dir","name":"saves","path":"/game/saves"},{"type":"file","name":"a.cfg","path":"/game/a.cfg","size":12}]}}`))
	})
	mux.HandleFunc("/services/9/gameservers/file_server/download", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"token":{"url":"` + srv.URL + `/blob","token":"dl-tok"}}}`))
	})
	mux.HandleFunc("/services/9/gameservers/file_server/upload", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"token":{"url":"` + srv.URL + `/blob","token":"up-tok"}}}`))
	})
	mux.HandleFunc("/blob", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			downloadToken = r.Header.Get("token")
			_, _ = w.Write([]byte("hello"))
			return
		}
		uploadToken = r.Header.Get("token")
		b, _ := io.ReadAll(r.Body)
		uploaded = string(b)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	c := New("bearer", WithBaseURL(srv.URL))
	ctx := context.Background()

	entries, err := c.ListFiles(ctx, 9, "/game")
	if err != nil || len(entries) != 2 || !entries[0].IsDir() || entries[1].IsDir() {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	data, err := c.ReadFile(ctx, 9, "/game/a.cfg")
	if err != nil || string(data) != "hello" || downloadToken != "dl-tok" {
		t.Errorf("read: %q tok=%q err=%v", data, downloadToken, err)
	}
	if err := c.WriteFile(ctx, 9, "/game", "a.cfg", []byte("new")); err != nil || uploaded != "new" || uploadToken != "up-tok" {
		t.Errorf("write: %q tok=%q err=%v", uploaded, uploadToken, err)
	}
}

func TestFiles_SimpleOps(t *testing.T) {
	rs := newRecordingServer(t, `{"status":"success","data":{}}`)
	c := New("t", WithBaseURL(rs.URL))
	ctx := context.Background()

	if err := c.MakeDir(ctx, 9, "/game", "new"); err != nil || rs.Path != "/services/9/gameservers/file_server/mkdir" || rs.Body["name"] != "new" {
		t.Errorf("mkdir: %s %v err=%v", rs.Path, rs.Body, err)
	}
	if err := c.DeleteFile(ctx, 9, "/game/a.cfg"); err != nil || rs.Method != "DELETE" || rs.Body["path"] != "/game/a.cfg" {
		t.Errorf("delete: %s %v err=%v", rs.Method, rs.Body, err)
	}
	if err := c.MoveFile(ctx, 9, "/a", "/b", "c"); err != nil || rs.Body["target_filename"] != "c" {
		t.Errorf("move: %v err=%v", rs.Body, err)
	}
}

func TestBackups(t *testing.T) {
	rs := newRecordingServer(t, `{"data":{"backups":[{"name":"b1","size":100,"created_at":"2026-01-01"},"b2"]}}`)
	c := New("t", WithBaseURL(rs.URL))
	ctx := context.Background()

	list, err := c.ListBackups(ctx, 3)
	if err != nil || len(list) != 2 || list[0].Name != "b1" || list[0].Size != "100" || list[1].Name != "b2" {
		t.Fatalf("list = %+v err=%v", list, err)
	}
	if err := c.CreateBackup(ctx, 3, ""); err != nil || rs.Body["type"] != "game" {
		t.Errorf("create: %v err=%v", rs.Body, err)
	}
	if err := c.RestoreBackup(ctx, 3, "b1"); err != nil || rs.Path != "/services/3/backups/extract" || rs.Body["name"] != "b1" {
		t.Errorf("restore: %s %v err=%v", rs.Path, rs.Body, err)
	}
	if err := c.DeleteBackup(ctx, 3, "b1"); err != nil || rs.Method != "DELETE" {
		t.Errorf("delete: %s err=%v", rs.Method, err)
	}
}

func TestConsole(t *testing.T) {
	rs := newRecordingServer(t, `{"data":{"logs":[{"date":"d1","message":"banned x"},{"foo":1},"plain"]}}`)
	c := New("t", WithBaseURL(rs.URL))
	ctx := context.Background()

	lines, err := c.AdminLogs(ctx, 4)
	if err != nil || len(lines) != 3 || lines[0] != "d1 banned x" || !strings.Contains(lines[1], "foo") || lines[2] != "plain" {
		t.Fatalf("lines = %q err=%v", lines, err)
	}
	if err := c.SendCommand(ctx, 4, "say hi"); err != nil || rs.Path != "/services/4/gameservers/command" || rs.Body["command"] != "say hi" {
		t.Errorf("command: %s %v err=%v", rs.Path, rs.Body, err)
	}
}

func TestStats(t *testing.T) {
	rs := newRecordingServer(t, `{"data":{"stats":{"cpu":[1,2]}}}`)
	c := New("t", WithBaseURL(rs.URL))
	raw, err := c.GetStats(context.Background(), 4, 6)
	if err != nil || !strings.Contains(string(raw), "cpu") {
		t.Fatalf("raw=%s err=%v", raw, err)
	}
}
