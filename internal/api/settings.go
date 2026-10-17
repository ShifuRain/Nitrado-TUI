package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// Settings is a game's configuration as category -> key -> value.
type Settings map[string]map[string]any

// UnmarshalJSON tolerates unexpected shapes (it keeps whatever categories do
// look like objects and drops the rest) so a surprise in the settings payload
// can never break loading the rest of a gameserver.
func (s *Settings) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil {
		return nil
	}
	out := make(Settings, len(raw))
	for cat, v := range raw {
		var kv map[string]any
		if json.Unmarshal(v, &kv) == nil {
			out[cat] = kv
		}
	}
	*s = out
	return nil
}

// Categories returns the category names in sorted order.
func (s Settings) Categories() []string { return sortedKeys(s) }

// Keys returns the keys of one category in sorted order.
func (s Settings) Keys(category string) []string { return sortedKeys(s[category]) }

// Get returns the value as display text and whether it exists.
func (s Settings) Get(category, key string) (string, bool) {
	v, ok := s[category][key]
	if !ok {
		return "", false
	}
	return FormatValue(v), true
}

// FormatValue renders a settings value for display.
func FormatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(t)
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type settingsDefaultsResponse struct {
	Data struct {
		Settings Settings `json:"settings"`
	} `json:"data"`
}

// GetSettingsDefaults returns the default value of every setting.
// GET /services/{id}/gameservers/settings/defaults.
func (c *Client) GetSettingsDefaults(ctx context.Context, serviceID int) (Settings, error) {
	var resp settingsDefaultsResponse
	if err := c.do(ctx, "GET", "/services/"+strconv.Itoa(serviceID)+"/gameservers/settings/defaults", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Settings, nil
}

type setSettingRequest struct {
	Category string `json:"category"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

// SetSetting changes one setting. Values are sent as strings, the way
// Nitrado's own panel does (unverified for every game).
// POST /services/{id}/gameservers/settings.
func (c *Client) SetSetting(ctx context.Context, serviceID int, category, key, value string) error {
	body := setSettingRequest{Category: category, Key: key, Value: value}
	return c.do(ctx, "POST", "/services/"+strconv.Itoa(serviceID)+"/gameservers/settings", body, nil)
}

// ResetSettings restores every setting to its default.
// DELETE /services/{id}/gameservers/settings.
func (c *Client) ResetSettings(ctx context.Context, serviceID int) error {
	return c.do(ctx, "DELETE", "/services/"+strconv.Itoa(serviceID)+"/gameservers/settings", nil, nil)
}
