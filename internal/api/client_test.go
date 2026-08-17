package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDo_SetsAuthHeaderAndEncodesBody(t *testing.T) {
	var gotAuth, gotMethod, gotPath string
	var gotBody map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
	}))
	defer srv.Close()

	c := New("my-token", WithBaseURL(srv.URL))
	err := c.do(context.Background(), "POST", "/services/1/gameservers/games/install", map[string]string{"game": "valheim"}, nil)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}

	if gotAuth != "Bearer my-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer my-token")
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/services/1/gameservers/games/install" {
		t.Errorf("path = %q, want /services/1/gameservers/games/install", gotPath)
	}
	if gotBody["game"] != "valheim" {
		t.Errorf("body[game] = %q, want %q", gotBody["game"], "valheim")
	}
}

func TestDo_DecodesSuccessResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success","data":{"services":[{"id":42,"status":"active"}]}}`))
	}))
	defer srv.Close()

	c := New("t", WithBaseURL(srv.URL))
	var resp listServicesResponse
	if err := c.do(context.Background(), "GET", "/services", nil, &resp); err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if len(resp.Data.Services) != 1 || resp.Data.Services[0].ID != 42 {
		t.Errorf("decoded services = %+v, want one service with id 42", resp.Data.Services)
	}
}

func TestDo_NonSuccessStatusReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"error","message":"invalid token"}`))
	}))
	defer srv.Close()

	c := New("bad-token", WithBaseURL(srv.URL))
	err := c.do(context.Background(), "GET", "/services", nil, nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 401, got nil")
	}

	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid token" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "invalid token")
	}
}
