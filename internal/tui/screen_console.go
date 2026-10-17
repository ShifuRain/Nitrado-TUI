package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// maxConsoleLines bounds the scrollback kept in memory.
const maxConsoleLines = 500

func (m model) updateConsole(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commandSentMsg:
		m.consoleBusy = false
		if msg.err != nil {
			m.consoleLog = appendLog(m.consoleLog, "! "+errMessage(msg.err))
		} else {
			m.consoleLog = appendLog(m.consoleLog, "  sent.")
		}
	case logsLoadedMsg:
		m.consoleBusy = false
		if msg.err != nil {
			m.consoleLog = appendLog(m.consoleLog, "! "+errMessage(msg.err))
		} else if len(msg.lines) == 0 {
			m.consoleLog = appendLog(m.consoleLog, "  (admin log is empty)")
		} else {
			m.consoleLog = appendLog(m.consoleLog, msg.lines...)
		}
	}
	return m, nil
}

func appendLog(log []string, lines ...string) []string {
	log = append(log, lines...)
	if len(log) > maxConsoleLines {
		log = log[len(log)-maxConsoleLines:]
	}
	return log
}

func (m model) handleConsoleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.consoleInput.Blur()
		chain := (&m).enterDetail()
		return m, chain
	case tea.KeyEnter:
		line := strings.TrimSpace(m.consoleInput.Value())
		if line == "" || m.consoleBusy {
			return m, nil
		}
		m.consoleInput.SetValue("")
		m.consoleBusy = true
		m.consoleLog = appendLog(m.consoleLog, "> "+line)
		return m, sendConsoleCommand(m.client, m.selectedSvc.ID, line)
	case tea.KeyCtrlL:
		if m.consoleBusy {
			return m, nil
		}
		m.consoleBusy = true
		m.consoleLog = appendLog(m.consoleLog, "-- admin log --")
		return m, loadAdminLogs(m.client, m.selectedSvc.ID)
	}
	var cmd tea.Cmd
	m.consoleInput, cmd = m.consoleInput.Update(msg)
	return m, cmd
}

func (m model) viewConsole() string {
	rows := m.height - 7
	if rows < 3 {
		rows = 3
	}
	log := m.consoleLog
	if len(log) > rows {
		log = log[len(log)-rows:]
	}
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("Console — " + m.serverName()))
	b.WriteString("\n\n")
	if len(log) == 0 {
		b.WriteString(m.styles.Help.Render("Commands you send appear here. The game must support remote commands."))
	} else {
		b.WriteString(m.styles.Normal.Render(strings.Join(log, "\n")))
	}
	b.WriteString("\n\n")
	b.WriteString(m.consoleInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("enter: send  •  ctrl+l: load admin log  •  esc: back"))
	return m.styles.Border.Render(b.String())
}
