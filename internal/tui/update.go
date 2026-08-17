package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/state"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		contentHeight := msg.Height - 4
		if contentHeight < 0 {
			contentHeight = 0
		}
		m.servers.SetSize(msg.Width, contentHeight)
		m.games.SetSize(msg.Width, contentHeight)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)

	case loginResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.client = msg.client
		m.err = nil
		m.screen = screenServerList
		return m, loadServers(m.client)

	case serversLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, len(msg.services))
		for i, s := range msg.services {
			items[i] = serviceItem{s}
		}
		m.err = nil
		return m, m.servers.SetItems(items)

	case detailLoadedMsg:
		m.detailLoad = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.detail = msg.detail
		return m, nil

	case gamesLoadedMsg:
		m.gamesLoad = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		items := make([]list.Item, len(msg.games))
		for i, g := range msg.games {
			items[i] = gameItem{g}
		}
		return m, m.games.SetItems(items)

	case actionDoneMsg:
		m.switchBusy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = msg.verb + "."
		if m.screen == screenGamePicker {
			m.screen = screenServerDetail
		}
		m.detailLoad = true
		return m, loadDetail(m.client, m.selectedSvc.ID)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenLogin:
		return m.handleLoginKey(msg)
	case screenServerList:
		return m.handleServerListKey(msg)
	case screenServerDetail:
		return m.handleServerDetailKey(msg)
	case screenGamePicker:
		return m.handleGamePickerKey(msg)
	}
	return m, nil
}

func (m model) handleLoginKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		token := m.loginInput.Value()
		if token == "" {
			return m, nil
		}
		m.err = nil
		return m, doLogin(m.store, token)
	case tea.KeyEsc:
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.loginInput, cmd = m.loginInput.Update(msg)
	return m, cmd
}

func (m model) handleServerListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.servers.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.servers, cmd = m.servers.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "enter":
		item, ok := m.servers.SelectedItem().(serviceItem)
		if !ok {
			return m, nil
		}
		m.selectedSvc = item.Service
		m.screen = screenServerDetail
		m.detail = nil
		m.detailLoad = true
		m.status = ""
		_ = state.SetSelectedServer(strconv.Itoa(item.Service.ID))
		return m, loadDetail(m.client, item.Service.ID)
	}

	var cmd tea.Cmd
	m.servers, cmd = m.servers.Update(msg)
	return m, cmd
}

func (m model) handleServerDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenServerList
		m.status = ""
		return m, nil
	case "s":
		m.screen = screenGamePicker
		m.gamesLoad = true
		m.status = ""
		return m, loadGames(m.client, m.selectedSvc.ID)
	case "r":
		m.switchBusy = true
		m.status = ""
		return m, doRestart(m.client, m.selectedSvc.ID)
	case "x":
		m.switchBusy = true
		m.status = ""
		return m, doStop(m.client, m.selectedSvc.ID)
	}
	return m, nil
}

func (m model) handleGamePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.games.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.games, cmd = m.games.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.screen = screenServerDetail
		return m, nil
	case "enter":
		item, ok := m.games.SelectedItem().(gameItem)
		if !ok {
			return m, nil
		}
		m.switchBusy = true
		return m, doSwitchGame(m.client, m.selectedSvc.ID, item.Game.Slug)
	}

	var cmd tea.Cmd
	m.games, cmd = m.games.Update(msg)
	return m, cmd
}
