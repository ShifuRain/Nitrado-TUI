// Package tui is the interactive Bubble Tea front-end for nitui. It
// covers the same ground as the CLI subcommands (login, list/select
// servers, view live status, switch/start/uninstall games, stop/restart,
// settings, backups, files, console) through a keyboard-driven UI, styled
// from the user's config.Theme.
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/config"
	"nitui/internal/update"
)

type screen int

const (
	screenLogin screen = iota
	screenServerList
	screenServerDetail
	screenGamePicker
	screenSettingsCats
	screenSettingsKeys
	screenSettingEdit
	screenBackups
	screenFiles
	screenFilePreview
	screenFileEdit
	screenConsole
)

type model struct {
	styles styles
	store  auth.Store
	client *api.Client

	version      string
	checkUpdates bool
	updateNotice string // one-line "new version" banner, empty if none

	screen screen
	width  int
	height int
	err    error
	status string // transient success/info message shown at the bottom

	confirm *confirmState // non-nil while a confirmation modal is open

	loginInput textinput.Model

	servers     list.Model
	selectedSvc api.Service
	detail      *api.GameServer
	detailLoad  bool

	// Live status polling. gen identifies the active tick chain so stale
	// ticks (from a screen we've left) are ignored.
	gen     int
	waiting *waitState

	games      list.Model
	gamesLoad  bool
	switchBusy bool

	// Settings
	settingsLoad    bool
	settings        api.Settings
	settingDefaults api.Settings
	settingsCats    list.Model
	settingsKeys    list.Model
	settingCat      string
	settingKey      string
	settingInput    textinput.Model
	saveBusy        bool

	// Backups
	backups     list.Model
	backupsLoad bool

	// Files
	filesDir    string
	files       list.Model
	filesLoad   bool
	filePath    string
	fileView    viewport.Model
	fileEditor  textarea.Model
	fileContent string

	// Console
	consoleInput textinput.Model
	consoleLog   []string
	consoleBusy  bool
}

// Run starts the interactive TUI. cfg controls styling; store is where the
// API token is read from (and written to, if the user logs in from here);
// version is nitui's own version, used for the update notice.
func Run(cfg config.Config, store auth.Store, version string) error {
	m := newModel(cfg, store)
	m.version = version
	m.checkUpdates = !update.Disabled(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel(cfg config.Config, store auth.Store) model {
	st := newStyles(cfg.Theme)

	ti := textinput.New()
	ti.Placeholder = "paste your Nitrado long-life token"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()

	si := textinput.New()
	si.Prompt = "> "
	si.CharLimit = 0

	ci := textinput.New()
	ci.Prompt = "> "
	ci.Placeholder = "console command (enter to send, ctrl+l loads the admin log)"

	ed := textarea.New()
	ed.ShowLineNumbers = false
	ed.CharLimit = 0

	m := model{
		styles:       st,
		store:        store,
		screen:       screenLogin,
		loginInput:   ti,
		servers:      newList(st, "Servers"),
		games:        newList(st, "Available games"),
		settingsCats: newList(st, "Settings"),
		settingsKeys: newList(st, "Settings"),
		settingInput: si,
		backups:      newList(st, "Backups"),
		files:        newList(st, "Files"),
		fileView:     viewport.New(0, 0),
		fileEditor:   ed,
		consoleInput: ci,
	}

	if token, err := store.Get(); err == nil {
		m.client = newClient(token)
		m.screen = screenServerList
	}

	return m
}

func newList(st styles, title string) list.Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.Styles.Title = st.Title
	l.SetShowHelp(true)
	return l
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.screen == screenServerList {
		cmds = append(cmds, loadServers(m.client))
	} else {
		cmds = append(cmds, textinput.Blink)
	}
	if m.checkUpdates {
		cmds = append(cmds, checkForUpdate(m.version))
	}
	return tea.Batch(cmds...)
}

// apiCtx bounds every network call the TUI makes so a dropped connection
// can't hang the interface indefinitely.
func apiCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
