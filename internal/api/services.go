package api

import (
	"context"
	"strconv"
)

// ---------------------------------------------------------------------
// Endpoint paths and payload shapes below are confirmed either against
// the official NitrAPI-PHP SDK source / community danstis/go-nitrado Go
// SDK, or against a live account's real responses (ListServices,
// GetGameServer, ListGames — see internal/api/types.go doc comments).
// Two things remain unverified: the exact error text/status returned
// when the per-server installed-games limit is hit, and whether
// "install" vs "start" is the right call for switching to an
// already-installed-but-inactive game vs installing a brand new one.
// ---------------------------------------------------------------------

type listServicesResponse struct {
	Data struct {
		Services []Service `json:"services"`
	} `json:"data"`
}

// ListServices returns every service (server) on the authenticated
// account. GET /services.
func (c *Client) ListServices(ctx context.Context) ([]Service, error) {
	var resp listServicesResponse
	if err := c.do(ctx, "GET", "/services", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Services, nil
}

type getGameServerResponse struct {
	Data struct {
		GameServer GameServer `json:"gameserver"`
	} `json:"data"`
}

// GetGameServer returns gameserver-specific details for a service.
// GET /services/{id}/gameservers.
func (c *Client) GetGameServer(ctx context.Context, serviceID int) (*GameServer, error) {
	var resp getGameServerResponse
	if err := c.do(ctx, "GET", "/services/"+strconv.Itoa(serviceID)+"/gameservers", nil, &resp); err != nil {
		return nil, err
	}
	gs := resp.Data.GameServer
	gs.ServiceID = serviceID
	return &gs, nil
}

// Restart restarts the gameserver. POST /services/{id}/gameservers/restart.
func (c *Client) Restart(ctx context.Context, serviceID int) error {
	return c.do(ctx, "POST", "/services/"+strconv.Itoa(serviceID)+"/gameservers/restart", nil, nil)
}

// Stop stops the gameserver. POST /services/{id}/gameservers/stop.
func (c *Client) Stop(ctx context.Context, serviceID int) error {
	return c.do(ctx, "POST", "/services/"+strconv.Itoa(serviceID)+"/gameservers/stop", nil, nil)
}

type listGamesResponse struct {
	Data struct {
		Games []Game `json:"games"`
	} `json:"data"`
}

// ListGames returns the games available to install/switch to on a server.
// GET /services/{id}/gameservers/games.
func (c *Client) ListGames(ctx context.Context, serviceID int) ([]Game, error) {
	var resp listGamesResponse
	if err := c.do(ctx, "GET", "/services/"+strconv.Itoa(serviceID)+"/gameservers/games", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Games, nil
}

type installGameRequest struct {
	Game    string `json:"game"`
	Modpack string `json:"modpack,omitempty"`
}

// SwitchGame installs/switches the active game on a gameserver. This is
// the operation most likely to hit the "N games installed" limit
// (surfaced via Error.FriendlyMessage()).
// POST /services/{id}/gameservers/games/install, body {"game": gameID}.
func (c *Client) SwitchGame(ctx context.Context, serviceID int, gameID string) error {
	body := installGameRequest{Game: gameID}
	return c.do(ctx, "POST", "/services/"+strconv.Itoa(serviceID)+"/gameservers/games/install", body, nil)
}

type gameRequest struct {
	Game string `json:"game"`
}

// StartGame (re)starts a specific already-installed game on the server.
// POST /services/{id}/gameservers/games/start, body {"game": gameID}.
func (c *Client) StartGame(ctx context.Context, serviceID int, gameID string) error {
	body := gameRequest{Game: gameID}
	return c.do(ctx, "POST", "/services/"+strconv.Itoa(serviceID)+"/gameservers/games/start", body, nil)
}

// UninstallGame removes a game from the server.
// DELETE /services/{id}/gameservers/games/uninstall, body {"game": gameID}.
func (c *Client) UninstallGame(ctx context.Context, serviceID int, gameID string) error {
	body := gameRequest{Game: gameID}
	return c.do(ctx, "DELETE", "/services/"+strconv.Itoa(serviceID)+"/gameservers/games/uninstall", body, nil)
}
