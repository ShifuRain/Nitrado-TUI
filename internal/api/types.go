package api

import (
	"encoding/json"
	"fmt"
)

// Service is a Nitrado "service" — the billing/ownership unit that a
// gameserver (among other product types) lives under.
// Confirmed against a live GET /services/{id} response.
type Service struct {
	ID         int            `json:"id"`
	LocationID int            `json:"location_id"`
	Status     string         `json:"status"`
	Type       string         `json:"type"`
	TypeHuman  string         `json:"type_human"`
	Details    ServiceDetails `json:"details"`
}

type ServiceDetails struct {
	Address string `json:"address"` // "host:port"
	Name    string `json:"name"`
	Game    string `json:"game"` // human-readable name, e.g. "Enshrouded"
	Slots   int    `json:"slots"`
}

// GameServer holds the gameserver-specific details of a Service: which
// game is installed, whether it's running, and connection info.
// Confirmed against a live GET /services/{id}/gameservers response.
type GameServer struct {
	ServiceID int    `json:"-"`
	Status    string `json:"status"` // e.g. "started", "stopped", "restarting"
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	QueryPort int    `json:"query_port"`
	RconPort  int    `json:"rcon_port"`
	// Game is the short slug identifying the installed/active game (e.g.
	// "enshrouded") — this is what's passed to SwitchGame/StartGame, not
	// Game.CatalogID from ListGames.
	Game      string `json:"game"`
	GameHuman string `json:"game_human"`
	Slots     int    `json:"slots"`
	Location  string `json:"location"` // country code, e.g. "DE"
	Username  string `json:"username"` // FTP/file-server user
	Label     string `json:"label"`

	// Settings is the per-game configuration, category -> key -> value.
	// Decoding is tolerant: an unexpected shape yields nil, never an error.
	Settings Settings `json:"settings"`
	// GameSpecific carries the last game-update status, when Nitrado reports one.
	GameSpecific GameSpecific `json:"game_specific"`
}

// UpdateStatus is the last game-update status Nitrado reported, or "".
func (g GameServer) UpdateStatus() string { return g.GameSpecific.UpdateStatus }

// GameSpecific holds the game-specific block of a gameserver response.
type GameSpecific struct {
	UpdateStatus string `json:"update_status"`
}

// UnmarshalJSON tolerates a missing or differently-shaped block.
func (g *GameSpecific) UnmarshalJSON(b []byte) error {
	var raw struct {
		UpdateStatus any `json:"update_status"`
	}
	if json.Unmarshal(b, &raw) != nil {
		return nil
	}
	if s, ok := raw.UpdateStatus.(string); ok {
		g.UpdateStatus = s
	}
	return nil
}

// Address is the connect address as "host:port".
func (g GameServer) Address() string {
	return fmt.Sprintf("%s:%d", g.IP, g.Port)
}

// Game is an installable/switchable game offering for a gameserver.
// Confirmed against a live GET /services/{id}/gameservers/games response.
type Game struct {
	CatalogID int `json:"id"` // Nitrado's internal catalog id — NOT used for install/switch
	// Slug is the identifier to pass to SwitchGame/StartGame (matches
	// GameServer.Game once installed and active).
	Slug               string `json:"portlist_short"`
	Name               string `json:"name"`
	Installed          bool   `json:"installed"`
	Active             bool   `json:"active"`
	MinimumSlots       int    `json:"minimum_slots"`
	MaximumRecommended *int   `json:"maximum_recommended_slots"`
	EnoughSlots        bool   `json:"enough_slots"`
	TooManySlots       bool   `json:"too_many_slots"`
}
