package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// SendCommand sends a console command into the running game server.
// POST /services/{id}/gameservers/command {"command": cmd}.
func (c *Client) SendCommand(ctx context.Context, serviceID int, cmd string) error {
	return c.do(ctx, "POST", "/services/"+strconv.Itoa(serviceID)+"/gameservers/command", map[string]string{"command": cmd}, nil)
}

// AdminLogs returns the admin log as display lines. The payload shape is
// unverified, so entries are rendered from common keys (date + message),
// falling back to the entry's JSON.
// GET /services/{id}/gameservers/admin_logs.
func (c *Client) AdminLogs(ctx context.Context, serviceID int) ([]string, error) {
	var env dataEnvelope
	if err := c.do(ctx, "GET", "/services/"+strconv.Itoa(serviceID)+"/gameservers/admin_logs", nil, &env); err != nil {
		return nil, err
	}
	var lines []string
	for _, item := range firstArray(env.Data, "logs", "admin_logs", "entries") {
		var s string
		if json.Unmarshal(item, &s) == nil {
			lines = append(lines, s)
			continue
		}
		var obj map[string]any
		if json.Unmarshal(item, &obj) != nil {
			lines = append(lines, string(item))
			continue
		}
		msg := field(obj, "message", "text", "log", "action")
		if msg == "" {
			lines = append(lines, string(item))
			continue
		}
		lines = append(lines, strings.TrimSpace(field(obj, "date", "time", "created_at", "timestamp")+" "+msg))
	}
	return lines, nil
}

// GetStats returns the raw stats payload for the last hours hours. The
// shape is unverified, so it's left as JSON for the caller to summarize.
// GET /services/{id}/gameservers/stats?hours=.
func (c *Client) GetStats(ctx context.Context, serviceID, hours int) (json.RawMessage, error) {
	var env struct {
		Data struct {
			Stats json.RawMessage `json:"stats"`
		} `json:"data"`
	}
	q := url.Values{"hours": {strconv.Itoa(hours)}}
	if err := c.do(ctx, "GET", "/services/"+strconv.Itoa(serviceID)+"/gameservers/stats?"+q.Encode(), nil, &env); err != nil {
		return nil, err
	}
	return env.Data.Stats, nil
}
