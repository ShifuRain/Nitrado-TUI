package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/update"
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
	verb string // "Switched", "Started", "Uninstalled"
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

// lifecycleDoneMsg reports a stop/restart request that Nitrado accepted (or
// rejected). Unlike actionDoneMsg, success starts the live "wait until it
// actually happened" polling.
type lifecycleDoneMsg struct {
	verb string // "restart" or "stop"
	err  error
}

func doStop(client *api.Client, serviceID int, message string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.StopWithMessage(ctx, serviceID, message)
		return lifecycleDoneMsg{verb: "stop", err: err}
	}
}

func doRestart(client *api.Client, serviceID int, message string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.RestartWithMessage(ctx, serviceID, message)
		return lifecycleDoneMsg{verb: "restart", err: err}
	}
}

// tickMsg fires when it's time for the next status poll of chain gen.
type tickMsg struct{ gen int }

// statusPolledMsg carries one poll result for chain gen.
type statusPolledMsg struct {
	gen    int
	detail *api.GameServer
	err    error
}

func pollStatus(client *api.Client, serviceID, gen int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		gs, err := client.GetGameServer(ctx, serviceID)
		return statusPolledMsg{gen: gen, detail: gs, err: err}
	}
}

// updateAvailableMsg is sent when a newer nitui release exists.
type updateAvailableMsg struct{ release *update.Release }

// checkForUpdate runs the cached, time-boxed new-version lookup. Failures
// are silent: an update notice must never get in the way.
func checkForUpdate(version string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		r, err := update.Check(ctx, version, update.Options{URL: updateURL})
		if err != nil || r == nil {
			return nil
		}
		return updateAvailableMsg{release: r}
	}
}

// updateURL overrides the release-lookup URL; tests point it at a server.
var updateURL string

func doStartGame(client *api.Client, serviceID int, gameID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.StartGame(ctx, serviceID, gameID)
		return actionDoneMsg{verb: "Started", err: err}
	}
}

func doUninstallGame(client *api.Client, serviceID int, gameID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.UninstallGame(ctx, serviceID, gameID)
		return gamesChangedMsg{verb: "Uninstalled", err: err}
	}
}

// gamesChangedMsg reports a games-list change (uninstall) that keeps the
// user on the picker and refreshes it.
type gamesChangedMsg struct {
	verb string
	err  error
}

// --- settings ---

type settingsLoadedMsg struct {
	detail   *api.GameServer
	defaults api.Settings
	err      error
}

func loadSettings(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		gs, err := client.GetGameServer(ctx, serviceID)
		if err != nil {
			return settingsLoadedMsg{err: err}
		}
		defaults, _ := client.GetSettingsDefaults(ctx, serviceID) // best-effort
		return settingsLoadedMsg{detail: gs, defaults: defaults}
	}
}

type settingSavedMsg struct {
	category, key, value string
	err                  error
}

func saveSetting(client *api.Client, serviceID int, category, key, value string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.SetSetting(ctx, serviceID, category, key, value)
		return settingSavedMsg{category: category, key: key, value: value, err: err}
	}
}

type settingsResetMsg struct{ err error }

func resetSettings(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		return settingsResetMsg{err: client.ResetSettings(ctx, serviceID)}
	}
}

// --- backups ---

type backupsLoadedMsg struct {
	backups []api.Backup
	err     error
}

func loadBackups(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		b, err := client.ListBackups(ctx, serviceID)
		return backupsLoadedMsg{backups: b, err: err}
	}
}

type backupActionMsg struct {
	verb string
	err  error
}

func doBackupAction(verb string, fn func(ctx context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		return backupActionMsg{verb: verb, err: fn(ctx)}
	}
}

// --- files ---

type filesLoadedMsg struct {
	dir     string
	entries []api.FileEntry
	err     error
}

func loadFiles(client *api.Client, serviceID int, dir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		e, err := client.ListFiles(ctx, serviceID, dir)
		return filesLoadedMsg{dir: dir, entries: e, err: err}
	}
}

type fileReadMsg struct {
	path string
	data []byte
	err  error
}

func readRemoteFile(client *api.Client, serviceID int, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		d, err := client.ReadFile(ctx, serviceID, path)
		return fileReadMsg{path: path, data: d, err: err}
	}
}

type fileWrittenMsg struct {
	path string
	err  error
}

func writeRemoteFile(client *api.Client, serviceID int, dir, name string, content []byte) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		err := client.WriteFile(ctx, serviceID, dir, name, content)
		return fileWrittenMsg{path: dir + "/" + name, err: err}
	}
}

type fileDeletedMsg struct {
	path string
	err  error
}

func deleteRemoteFile(client *api.Client, serviceID int, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		return fileDeletedMsg{path: path, err: client.DeleteFile(ctx, serviceID, path)}
	}
}

// --- console ---

type commandSentMsg struct {
	command string
	err     error
}

func sendConsoleCommand(client *api.Client, serviceID int, command string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		return commandSentMsg{command: command, err: client.SendCommand(ctx, serviceID, command)}
	}
}

type logsLoadedMsg struct {
	lines []string
	err   error
}

func loadAdminLogs(client *api.Client, serviceID int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apiCtx()
		defer cancel()
		l, err := client.AdminLogs(ctx, serviceID)
		return logsLoadedMsg{lines: l, err: err}
	}
}
