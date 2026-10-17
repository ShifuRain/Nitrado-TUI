package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"nitui/internal/config"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		cur, latest string
		want        bool
	}{
		{"1.0.2", "v1.0.3", true},
		{"v1.0.2", "1.1.0", true},
		{"1.0.2", "2.0.0", true},
		{"1.0.2", "1.0.2", false},
		{"1.2.0", "1.1.9", false},
		{"1.10.0", "1.9.0", false},
		{"dev", "1.0.0", false},
		{"1.0.0", "garbage", false},
		{"1.0.0-rc1", "1.0.1", true},
	}
	for _, tt := range tests {
		if got := IsNewer(tt.cur, tt.latest); got != tt.want {
			t.Errorf("IsNewer(%q,%q) = %v, want %v", tt.cur, tt.latest, got, tt.want)
		}
	}
}

func releaseServer(t *testing.T, tag string, hits *int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"https://example.test/rel"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheck_NewerFoundAndCached(t *testing.T) {
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
	var hits int32
	srv := releaseServer(t, "v1.2.0", &hits)
	now := time.Now()
	opts := Options{URL: srv.URL, Now: func() time.Time { return now }}

	r, err := Check(context.Background(), "1.0.2", opts)
	if err != nil || r == nil || r.Version != "1.2.0" || r.URL != "https://example.test/rel" {
		t.Fatalf("r=%v err=%v", r, err)
	}

	// Within the interval the network is not hit again.
	now = now.Add(time.Hour)
	if r, _ := Check(context.Background(), "1.0.2", opts); r == nil {
		t.Error("cached newer version should still be reported")
	}
	if hits != 1 {
		t.Errorf("hits = %d, want 1 (cached)", hits)
	}

	// After the interval it refetches.
	now = now.Add(25 * time.Hour)
	_, _ = Check(context.Background(), "1.0.2", opts)
	if hits != 2 {
		t.Errorf("hits = %d, want 2 after the interval", hits)
	}
}

func TestCheck_UpToDateAndDev(t *testing.T) {
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
	var hits int32
	srv := releaseServer(t, "v1.0.2", &hits)

	if r, err := Check(context.Background(), "1.0.2", Options{URL: srv.URL}); r != nil || err != nil {
		t.Errorf("up to date: r=%v err=%v", r, err)
	}
	if r, err := Check(context.Background(), "dev", Options{URL: srv.URL}); r != nil || err != nil {
		t.Errorf("dev: r=%v err=%v", r, err)
	}
	if hits != 1 {
		t.Errorf("hits = %d, dev builds must not hit the network", hits)
	}
}

func TestCheck_ServerErrorIsReturnedNotPanicked(t *testing.T) {
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)
	if _, err := Check(context.Background(), "1.0.0", Options{URL: srv.URL}); err == nil {
		t.Error("expected an error on HTTP 500")
	}
}

func TestDisabled(t *testing.T) {
	off := false
	if !Disabled(config.Config{UpdateCheck: &off}) {
		t.Error("update_check:false should disable")
	}
	if Disabled(config.Config{}) {
		t.Error("default should be enabled")
	}
	t.Setenv("NITUI_NO_UPDATE_CHECK", "1")
	if !Disabled(config.Config{}) {
		t.Error("env var should disable")
	}
}
