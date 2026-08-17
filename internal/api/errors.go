package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Error represents a non-2xx response from the Nitrado API.
type Error struct {
	StatusCode int
	// Message is Nitrado's own error text, parsed from {"status":"error","message":"..."}.
	Message string
	// Raw is the unparsed response body, kept for debugging/-v output.
	Raw string
	// RateLimitReset is when the current rate-limit window resets, parsed
	// from the X-RateLimit-Reset header on a 429 response. Zero if absent.
	RateLimitReset time.Time
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("nitrado api: %s (HTTP %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("nitrado api: HTTP %d", e.StatusCode)
}

// envelope is Nitrado's confirmed error shape: {"status":"error","message":"..."}.
type envelope struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ParseError builds an *Error from a non-2xx HTTP status, response body,
// and response headers (used to read rate-limit info on a 429).
func ParseError(statusCode int, body []byte, headers http.Header) *Error {
	e := &Error{StatusCode: statusCode, Raw: string(body)}

	var env envelope
	if json.Unmarshal(body, &env) == nil && env.Message != "" {
		e.Message = env.Message
	}

	if statusCode == http.StatusTooManyRequests {
		if resetStr := headers.Get("X-RateLimit-Reset"); resetStr != "" {
			if unix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
				e.RateLimitReset = time.Unix(unix, 0)
			}
		}
	}
	return e
}

// FriendlyMessage turns an API error into user-facing guidance. It adds
// context Nitrado's own message doesn't include (what command to run next),
// while still surfacing Nitrado's original wording so nothing is hidden.
func (e *Error) FriendlyMessage() string {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "Authentication failed — your token may be invalid or expired. Run `nitui auth login` to sign in again."
	case http.StatusNotFound:
		return "Not found — double check the server/game id. Run `nitui servers list` to see what's available on your account."
	case http.StatusTooManyRequests:
		if !e.RateLimitReset.IsZero() {
			return fmt.Sprintf("Nitrado's API rate limit is exceeded. Try again after %s.", e.RateLimitReset.Local().Format(time.Kitchen))
		}
		return "Nitrado's API rate limit is exceeded. Wait a moment and try again."
	case 428:
		return "Another change is already in progress for this server. Wait for it to finish and try again."
	case http.StatusServiceUnavailable:
		return "Nitrado's API is in maintenance right now. Try again shortly."
	}

	msg := e.Message
	if msg == "" {
		msg = strings.TrimSpace(e.Raw)
	}
	if msg == "" {
		return fmt.Sprintf("Nitrado API request failed with HTTP %d.", e.StatusCode)
	}

	// Nitrado doesn't document the exact wording for the per-server
	// installed-games limit, so this is a heuristic rather than an exact
	// match — it still surfaces Nitrado's real message either way.
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "limit") {
		return fmt.Sprintf("%s\nUninstall a game or free up capacity before installing/switching to a new one.", msg)
	}
	return msg
}
