package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
	"nitui/internal/auth"
)

// Messages produced by the async commands below and handled in update.go.

type loginResultMsg struct {
	client *api.Client
	err    error
}

type serversLoadedMsg struct {
	services []api.Service
	err      error
}

type detailLoadedMsg struct {
	detail *api.GameServer
	err    error
}

type gamesLoadedMsg struct {
	games []api.Game
	err   error
}

type actionDoneMsg struct {
	verb string // "Switched", "Stopped", "Restarted"
	err  error
}

// apiBaseURL overrides the Nitrado API base URL when set. Tests point this
// at an httptest.Server; production code leaves it empty to use the real API.
var apiBaseURL string

func newClient(token string) *api.Client {
	if apiBaseURL != "" {
		return api.New(token, api.WithBaseURL(apiBaseURL))
	}
	return api.New(token)
}

func doLogin(store auth.Store, token string) tea.Cmd {
	return func() tea.Msg {
		client := newClient(token)
		ctx, cancel := apiCtx()
		defer cancel()
		if _, err := client.ListServices(ctx); err != nil {
			return loginResultMsg{err: err}
		}
		if err := store.Save(token); err != nil {
			return loginResultMsg{err: err}
		}
		return loginResultMsg{client: client}
	}
}

func loadServers(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		services, err := client.ListServices(ctx)
		return serversLoadedMsg{services: services, err: err}
	}
}

func loadDetail(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		detail, err := client.GetGameServer(ctx, serviceID)
		return detailLoadedMsg{detail: detail, err: err}
	}
}

func loadGames(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		games, err := client.ListGames(ctx, serviceID)
		return gamesLoadedMsg{games: games, err: err}
	}
}

func doSwitchGame(client *api.Client, serviceID int, gameID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.SwitchGame(ctx, serviceID, gameID)
		return actionDoneMsg{verb: "Switched", err: err}
	}
}

func doStop(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.Stop(ctx, serviceID)
		return actionDoneMsg{verb: "Stopped", err: err}
	}
}

func doRestart(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.Restart(ctx, serviceID)
		return actionDoneMsg{verb: "Restarted", err: err}
	}
}
