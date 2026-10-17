package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// FileEntry is one file or directory in the gameserver's file server.
type FileEntry struct {
	Type       string `json:"type"` // "file" or "dir"
	Path       string `json:"path"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt int64  `json:"modified_at"`
}

// IsDir reports whether the entry is a directory.
func (f FileEntry) IsDir() bool { return f.Type == "dir" || f.Type == "directory" }

func (c *Client) filesPath(serviceID int, op string) string {
	return "/services/" + strconv.Itoa(serviceID) + "/gameservers/file_server/" + op
}

type listFilesResponse struct {
	Data struct {
		Entries []FileEntry `json:"entries"`
	} `json:"data"`
}

// ListFiles lists a directory. GET …/file_server/list?dir=.
func (c *Client) ListFiles(ctx context.Context, serviceID int, dir string) ([]FileEntry, error) {
	var resp listFilesResponse
	q := url.Values{"dir": {dir}}
	if err := c.do(ctx, "GET", c.filesPath(serviceID, "list")+"?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Entries, nil
}

type fileTokenResponse struct {
	Data struct {
		Token struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		} `json:"token"`
	} `json:"data"`
}

// transfer performs the second leg of an upload/download: a request to the
// URL Nitrado handed out, authenticated by the one-time token header (not
// the API bearer token).
func (c *Client) transfer(ctx context.Context, method, url, token string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("token", token)
	if body != nil {
		req.Header.Set("Content-Type", "application/binary")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transferring file: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading transfer response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ParseError(resp.StatusCode, data, resp.Header)
	}
	return data, nil
}

// ReadFile downloads a file's contents. GET …/file_server/download?file= to
// get a one-time URL, then GET that URL.
func (c *Client) ReadFile(ctx context.Context, serviceID int, path string) ([]byte, error) {
	var resp fileTokenResponse
	q := url.Values{"file": {path}}
	if err := c.do(ctx, "GET", c.filesPath(serviceID, "download")+"?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	t := resp.Data.Token
	if t.URL == "" || t.Token == "" {
		return nil, fmt.Errorf("nitrado returned no download token for %s", path)
	}
	return c.transfer(ctx, "GET", t.URL, t.Token, nil)
}

// WriteFile uploads content to dir/name, overwriting any existing file.
// POST …/file_server/upload to get a one-time URL, then POST the bytes to it.
func (c *Client) WriteFile(ctx context.Context, serviceID int, dir, name string, content []byte) error {
	var resp fileTokenResponse
	body := map[string]string{"path": dir, "file": name}
	if err := c.do(ctx, "POST", c.filesPath(serviceID, "upload"), body, &resp); err != nil {
		return err
	}
	t := resp.Data.Token
	if t.URL == "" || t.Token == "" {
		return fmt.Errorf("nitrado returned no upload token for %s/%s", dir, name)
	}
	_, err := c.transfer(ctx, "POST", t.URL, t.Token, bytes.NewReader(content))
	return err
}

// DeleteFile deletes a file or directory (recursively).
// DELETE …/file_server/delete, with the path in both body and query because
// the exact transport is unverified.
func (c *Client) DeleteFile(ctx context.Context, serviceID int, path string) error {
	q := url.Values{"path": {path}}
	return c.do(ctx, "DELETE", c.filesPath(serviceID, "delete")+"?"+q.Encode(), map[string]string{"path": path}, nil)
}

// MakeDir creates directory name inside dir. POST …/file_server/mkdir.
func (c *Client) MakeDir(ctx context.Context, serviceID int, dir, name string) error {
	return c.do(ctx, "POST", c.filesPath(serviceID, "mkdir"), map[string]string{"path": dir, "name": name}, nil)
}

// MoveFile moves/renames a file. POST …/file_server/move.
func (c *Client) MoveFile(ctx context.Context, serviceID int, source, targetDir, targetName string) error {
	body := map[string]string{"source_path": source, "target_path": targetDir, "target_filename": targetName}
	return c.do(ctx, "POST", c.filesPath(serviceID, "move"), body, nil)
}
