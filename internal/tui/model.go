// Package tui is the interactive Bubble Tea front-end for nitui. It
// covers the same ground as the CLI subcommands (login, list/select
// servers, view status, switch game, stop/restart) through a keyboard-
// driven UI, styled from the user's config.Theme.
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/config"
)

type screen int

const (
	screenLogin screen = iota
	screenServerList
	screenServerDetail
	screenGamePicker
)

type model struct {
	styles styles
	store  auth.Store
	client *api.Client

	screen screen
	width  int
	height int
	err    error
	status string // transient success/info message shown at the bottom

	loginInput textinput.Model

	servers     list.Model
	selectedSvc api.Service
	detail      *api.GameServer
	detailLoad  bool

	games      list.Model
	gamesLoad  bool
	switchBusy bool
}

// Run starts the interactive TUI. cfg controls styling; store is where the
// API token is read from (and written to, if the user logs in from here).
func Run(cfg config.Config, store auth.Store) error {
	m := newModel(cfg, store)
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

	m := model{
		styles:     st,
		store:      store,
		screen:     screenLogin,
		loginInput: ti,
		servers:    newList(st, "Servers"),
		games:      newList(st, "Available games"),
	}

	if token, err := store.Get(); err == nil {
		m.client = api.New(token)
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
	if m.screen == screenServerList {
		return loadServers(m.client)
	}
	return textinput.Blink
}

// apiCtx bounds every network call the TUI makes so a dropped connection
// can't hang the interface indefinitely.
func apiCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
