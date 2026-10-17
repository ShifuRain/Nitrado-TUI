package tui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/state"
	"nitui/internal/update"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		contentHeight := msg.Height - 4
		if contentHeight < 0 {
			contentHeight = 0
		}
		for _, l := range []*list.Model{&m.servers, &m.games, &m.settingsCats, &m.settingsKeys, &m.backups, &m.files} {
			l.SetSize(msg.Width, contentHeight)
		}
		m.fileView.Width, m.fileView.Height = msg.Width, contentHeight
		m.fileEditor.SetWidth(max(msg.Width-4, 10))
		m.fileEditor.SetHeight(max(contentHeight-2, 3))
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.confirm != nil {
			return m.handleConfirmKey(msg)
		}
		return m.handleKey(msg)

	case updateAvailableMsg:
		m.updateNotice = update.Notice(msg.release)
		return m, nil

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

	case tickMsg:
		if msg.gen != m.gen || m.screen != screenServerDetail {
			return m, nil
		}
		return m, pollStatus(m.client, m.selectedSvc.ID, msg.gen)

	case statusPolledMsg:
		return m.handleStatusPolled(msg)

	case lifecycleDoneMsg:
		m.switchBusy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.screen = screenServerDetail
		m.status = ""
		cmd := (&m).startWaiting(msg.verb, time.Now())
		return m, cmd

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

	case gamesChangedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = msg.verb + "."
		m.gamesLoad = true
		return m, loadGames(m.client, m.selectedSvc.ID)

	case actionDoneMsg:
		m.switchBusy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = msg.verb + "."
		m.detailLoad = true
		load := loadDetail(m.client, m.selectedSvc.ID)
		if m.screen == screenGamePicker {
			chain := (&m).enterDetail()
			return m, tea.Batch(load, chain)
		}
		return m, load

	case settingsLoadedMsg, settingSavedMsg, settingsResetMsg:
		return m.updateSettings(msg)

	case backupsLoadedMsg, backupActionMsg:
		return m.updateBackups(msg)

	case filesLoadedMsg, fileReadMsg, fileWrittenMsg, fileDeletedMsg:
		return m.updateFiles(msg)

	case commandSentMsg, logsLoadedMsg:
		return m.updateConsole(msg)
	}

	return m, nil
}

// handleStatusPolled applies a status poll: refresh the detail, finish or
// time out a pending restart/stop wait, and schedule the next tick.
func (m model) handleStatusPolled(msg statusPolledMsg) (tea.Model, tea.Cmd) {
	if msg.gen != m.gen || m.screen != screenServerDetail {
		return m, nil
	}
	interval := idleRefresh
	if msg.err == nil {
		m.detail = msg.detail
		m.detailLoad = false
	}
	if w := m.waiting; w != nil {
		interval = activeRefresh
		now := time.Now()
		switch {
		case msg.err == nil && w.watch.Observe(msg.detail.Status, now):
			if w.verb == "stop" {
				m.status = "Server stopped."
			} else {
				m.status = "Server is running again."
			}
			m.waiting = nil
			interval = idleRefresh
		case w.watch.TimedOut(now):
			last := "unknown"
			if m.detail != nil {
				last = m.detail.Status
			}
			m.err = fmt.Errorf("gave up waiting for the server (last status: %s)", last)
			m.waiting = nil
			interval = idleRefresh
		}
	}
	return m, tickCmd(m.gen, interval)
}

func tickCmd(gen int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{gen: gen} })
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
	case screenSettingsCats, screenSettingsKeys, screenSettingEdit:
		return m.handleSettingsKey(msg)
	case screenBackups:
		return m.handleBackupsKey(msg)
	case screenFiles, screenFilePreview, screenFileEdit:
		return m.handleFilesKey(msg)
	case screenConsole:
		return m.handleConsoleKey(msg)
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
		m.detail = nil
		m.detailLoad = true
		m.waiting = nil
		m.status = ""
		m.err = nil
		_ = state.SetSelectedServer(strconv.Itoa(item.Service.ID))
		chain := (&m).enterDetail()
		return m, tea.Batch(loadDetail(m.client, item.Service.ID), chain)
	}

	var cmd tea.Cmd
	m.servers, cmd = m.servers.Update(msg)
	return m, cmd
}

func (m model) handleServerDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	name := m.serverName()
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		m.gen++ // stop polling
		m.screen = screenServerList
		m.status = ""
		m.err = nil
		return m, loadServers(m.client)
	case "s":
		m.screen = screenGamePicker
		m.gamesLoad = true
		m.status = ""
		return m, loadGames(m.client, m.selectedSvc.ID)
	case "e":
		m.screen = screenSettingsCats
		m.settingsLoad = true
		m.status = ""
		return m, loadSettings(m.client, m.selectedSvc.ID)
	case "b":
		m.screen = screenBackups
		m.backupsLoad = true
		m.status = ""
		return m, loadBackups(m.client, m.selectedSvc.ID)
	case "f":
		m.screen = screenFiles
		m.filesDir = "/"
		m.filesLoad = true
		m.status = ""
		return m, loadFiles(m.client, m.selectedSvc.ID, "/")
	case "c":
		m.screen = screenConsole
		m.status = ""
		m.consoleInput.Focus()
		return m, nil
	case "r":
		m.confirm = newConfirmWithMessage(fmt.Sprintf("Restart %s?", name), m.restartAction)
		return m, nil
	case "u":
		m.confirm = newConfirmWithMessage(
			fmt.Sprintf("Restart %s so Nitrado applies a pending game update? (unverified)", name), m.restartAction)
		return m, nil
	case "x":
		m.confirm = newConfirmWithMessage(fmt.Sprintf("Stop %s?", name), func(m model, text string) (model, tea.Cmd) {
			m.switchBusy = true
			m.status, m.err = "", nil
			return m, doStop(m.client, m.selectedSvc.ID, text)
		})
		return m, nil
	}
	return m, nil
}

func (m model) restartAction(mm model, text string) (model, tea.Cmd) {
	mm.switchBusy = true
	mm.status, mm.err = "", nil
	return mm, doRestart(mm.client, mm.selectedSvc.ID, text)
}

// serverName is the friendliest label we have for the selected server.
func (m model) serverName() string {
	if n := m.selectedSvc.Details.Name; n != "" {
		return n
	}
	return fmt.Sprintf("server %d", m.selectedSvc.ID)
}

func (m model) handleGamePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.games.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.games, cmd = m.games.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		chain := (&m).enterDetail()
		return m, chain
	case "enter":
		item, ok := m.games.SelectedItem().(gameItem)
		if !ok {
			return m, nil
		}
		switch {
		case item.Active:
			m.status = item.Name + " is already the active game."
			return m, nil
		case item.Installed:
			m.switchBusy = true
			return m, doStartGame(m.client, m.selectedSvc.ID, item.Game.Slug)
		}
		m.switchBusy = true
		return m, doSwitchGame(m.client, m.selectedSvc.ID, item.Game.Slug)
	case "u":
		item, ok := m.games.SelectedItem().(gameItem)
		if !ok || !item.Installed || item.Active {
			m.status = "Only an installed game that isn't active can be uninstalled."
			return m, nil
		}
		m.confirm = newConfirm(fmt.Sprintf("Uninstall %s? Its files on the server may be lost.", item.Name), func(m model, _ string) (model, tea.Cmd) {
			return m, doUninstallGame(m.client, m.selectedSvc.ID, item.Game.Slug)
		})
		return m, nil
	}

	var cmd tea.Cmd
	m.games, cmd = m.games.Update(msg)
	return m, cmd
}
