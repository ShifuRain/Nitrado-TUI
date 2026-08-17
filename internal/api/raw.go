package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Raw issues an authenticated request and returns the response body
// unparsed (pretty-printed if it's JSON). It exists for debugging/schema
// discovery against the real API — see `nitui debug api`.
func (c *Client) Raw(ctx context.Context, method, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Nitrado API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, body, "", "  ") == nil {
		body = pretty.Bytes()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return body, nil
}
