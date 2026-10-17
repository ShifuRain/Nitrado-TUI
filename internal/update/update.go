// Package update checks GitHub for a newer nitui release. The check is
// advisory only: it is cached, time-boxed, and every failure is swallowed by
// callers so it can never slow down or break a command.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"nitui/internal/config"
)

const (
	// LatestURL is GitHub's "latest release" endpoint for nitui.
	LatestURL = "https://api.github.com/repos/ShifuRain/Nitrado-TUI/releases/latest"
	cacheFile = "update_check.yaml"
	// CheckInterval is how often the network is consulted at most.
	CheckInterval = 24 * time.Hour
)

// Release describes a newer version.
type Release struct {
	Version string // e.g. "1.2.0" (no leading v)
	URL     string // human-facing release page
}

// Options customizes Check; the zero value uses production defaults.
type Options struct {
	URL      string
	Client   *http.Client
	Now      func() time.Time
	Interval time.Duration
}

type cache struct {
	CheckedAt time.Time `yaml:"checked_at"`
	Version   string    `yaml:"version"`
	URL       string    `yaml:"url"`
}

// Disabled reports whether update checks are switched off, via the
// NITUI_NO_UPDATE_CHECK env var or `update_check: false` in the config.
func Disabled(cfg config.Config) bool {
	if os.Getenv("NITUI_NO_UPDATE_CHECK") != "" {
		return true
	}
	return cfg.UpdateCheck != nil && !*cfg.UpdateCheck
}

// Check returns the newest release if it is newer than current, or nil.
// Development builds ("dev" or non-semver versions) are never checked. The
// network is hit at most once per Interval; in between, the cached answer is
// used.
func Check(ctx context.Context, current string, opts Options) (*Release, error) {
	if _, ok := parseVersion(current); !ok {
		return nil, nil
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Interval == 0 {
		opts.Interval = CheckInterval
	}
	if opts.URL == "" {
		opts.URL = LatestURL
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 3 * time.Second}
	}

	c := loadCache()
	if c.Version == "" || opts.Now().Sub(c.CheckedAt) >= opts.Interval {
		fresh, err := fetchLatest(ctx, opts)
		if err != nil {
			return nil, err
		}
		c = cache{CheckedAt: opts.Now(), Version: fresh.Version, URL: fresh.URL}
		saveCache(c)
	}
	if IsNewer(current, c.Version) {
		return &Release{Version: strings.TrimPrefix(c.Version, "v"), URL: c.URL}, nil
	}
	return nil, nil
}

func fetchLatest(ctx context.Context, opts Options) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", opts.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "nitui-update-check")
	resp, err := opts.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update check: HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if _, ok := parseVersion(body.TagName); !ok {
		return nil, fmt.Errorf("update check: unrecognized tag %q", body.TagName)
	}
	return &Release{Version: strings.TrimPrefix(body.TagName, "v"), URL: body.HTMLURL}, nil
}

func loadCache() cache {
	var c cache
	path, err := config.FilePath(cacheFile)
	if err != nil {
		return c
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &c)
	}
	return c
}

func saveCache(c cache) {
	path, err := config.FilePath(cacheFile)
	if err != nil {
		return
	}
	if data, err := yaml.Marshal(c); err == nil {
		_ = os.WriteFile(path, data, 0o600)
	}
}

// IsNewer reports whether latest is a strictly higher version than current.
func IsNewer(current, latest string) bool {
	c, ok1 := parseVersion(current)
	l, ok2 := parseVersion(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// parseVersion reads "v1.2.3" / "1.2.3" (a "-suffix" is ignored).
func parseVersion(s string) ([3]int, bool) {
	var v [3]int
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

// Notice is the one-line message shown to users about a newer release.
func Notice(r *Release) string {
	return fmt.Sprintf("A new nitui version is available: %s — update with `winget upgrade ShifuRain.nitui`, your AUR helper (nitui-bin), or download it from %s", r.Version, r.URL)
}
