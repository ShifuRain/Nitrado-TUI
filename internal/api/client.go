// Package api is a client for the Nitrado REST API
// (https://doc.nitrado.net/). Endpoint paths and response shapes are
// documented in comments next to each call; several are marked TODO
// pending confirmation against Nitrado's docs/official SDKs and should not
// be trusted blindly.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.nitrado.net"

// Client talks to the Nitrado API using a long-life personal access token.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Option customizes a Client built by New.
type Option func(*Client)

// WithBaseURL points the client at a different API base URL. Used by tests
// to run against an httptest.Server instead of the real Nitrado API.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// New creates a Client authenticated with the given long-life token.
func New(token string, opts ...Option) *Client {
	c := &Client{
		baseURL: defaultBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do issues an HTTP request against the Nitrado API, decoding a JSON
// response body into out (if non-nil) and translating non-2xx responses
// into an *Error via ParseError.
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling Nitrado API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading Nitrado API response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ParseError(resp.StatusCode, respBody, resp.Header)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decoding Nitrado API response: %w", err)
		}
	}
	return nil
}
