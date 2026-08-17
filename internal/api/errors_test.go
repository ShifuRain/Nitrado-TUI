package api

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFriendlyMessage_KnownStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
		wantSubstr string
	}{
		{"unauthorized", http.StatusUnauthorized, `{"status":"error","message":"invalid token"}`, "auth login"},
		{"forbidden", http.StatusForbidden, `{"status":"error","message":"forbidden"}`, "auth login"},
		{"not found", http.StatusNotFound, `{"status":"error","message":"not found"}`, "servers list"},
		{"maintenance", http.StatusServiceUnavailable, ``, "maintenance"},
		{"concurrency conflict", 428, ``, "already in progress"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := ParseError(tc.statusCode, []byte(tc.body), http.Header{})
			got := e.FriendlyMessage()
			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("FriendlyMessage() = %q, want substring %q", got, tc.wantSubstr)
			}
		})
	}
}

func TestFriendlyMessage_LimitReached(t *testing.T) {
	body := `{"status":"error","message":"This server has reached the limit of 5 installed games."}`
	e := ParseError(http.StatusBadRequest, []byte(body), http.Header{})

	got := e.FriendlyMessage()
	if !strings.Contains(got, "limit of 5 installed games") {
		t.Errorf("FriendlyMessage() = %q, want it to include Nitrado's original message", got)
	}
	if !strings.Contains(got, "Uninstall a game") {
		t.Errorf("FriendlyMessage() = %q, want actionable guidance appended", got)
	}
}

func TestFriendlyMessage_RateLimitWithReset(t *testing.T) {
	reset := time.Now().Add(10 * time.Minute)
	headers := http.Header{}
	headers.Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))

	e := ParseError(http.StatusTooManyRequests, nil, headers)
	if e.RateLimitReset.IsZero() {
		t.Fatal("expected RateLimitReset to be parsed from X-RateLimit-Reset header")
	}

	got := e.FriendlyMessage()
	if !strings.Contains(got, "Try again after") {
		t.Errorf("FriendlyMessage() = %q, want it to mention when the rate limit resets", got)
	}
}

func TestFriendlyMessage_UnparseableBodyFallsBackToRaw(t *testing.T) {
	e := ParseError(http.StatusInternalServerError, []byte("upstream timeout"), http.Header{})
	got := e.FriendlyMessage()
	if !strings.Contains(got, "upstream timeout") {
		t.Errorf("FriendlyMessage() = %q, want raw body surfaced when it isn't JSON", got)
	}
}
