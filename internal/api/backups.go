package api

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// Backup is one backup archive. The list payload's exact shape is
// unverified, so fields are filled from whichever common keys are present
// and Raw keeps the original object.
type Backup struct {
	Name    string
	Size    string
	Created string
	Raw     map[string]any
}

func (c *Client) backupsPath(serviceID int) string {
	return "/services/" + strconv.Itoa(serviceID) + "/backups"
}

// ListBackups lists backup archives. GET /services/{id}/backups.
func (c *Client) ListBackups(ctx context.Context, serviceID int) ([]Backup, error) {
	var env dataEnvelope
	if err := c.do(ctx, "GET", c.backupsPath(serviceID), nil, &env); err != nil {
		return nil, err
	}
	var out []Backup
	for _, item := range firstArray(env.Data, "backups", "archives") {
		var obj map[string]any
		if json.Unmarshal(item, &obj) != nil {
			var name string
			if json.Unmarshal(item, &name) == nil && name != "" {
				out = append(out, Backup{Name: name, Raw: map[string]any{"name": name}})
			}
			continue
		}
		out = append(out, Backup{
			Name:    field(obj, "name", "archive", "id", "prefix"),
			Size:    field(obj, "size", "bytes"),
			Created: field(obj, "created_at", "created", "time", "timestamp", "date"),
			Raw:     obj,
		})
	}
	return out, nil
}

// CreateBackup creates a backup. kind is "game" (default) or whatever
// Nitrado accepts for the server. POST /services/{id}/backups {"type":kind}.
func (c *Client) CreateBackup(ctx context.Context, serviceID int, kind string) error {
	if kind == "" {
		kind = "game"
	}
	return c.do(ctx, "POST", c.backupsPath(serviceID), map[string]string{"type": kind}, nil)
}

// RestoreBackup restores the named archive. POST …/backups/extract.
func (c *Client) RestoreBackup(ctx context.Context, serviceID int, name string) error {
	body := map[string]any{"name": name, "paths": []string{}}
	return c.do(ctx, "POST", c.backupsPath(serviceID)+"/extract", body, nil)
}

// DeleteBackup deletes the named archive. DELETE …/backups?prefix=name.
func (c *Client) DeleteBackup(ctx context.Context, serviceID int, name string) error {
	q := url.Values{"prefix": {name}}
	return c.do(ctx, "DELETE", c.backupsPath(serviceID)+"?"+q.Encode(), nil, nil)
}
