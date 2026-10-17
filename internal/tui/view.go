package tui

import (
	"fmt"
	"strings"
	"time"

	"nitui/internal/api"
)

func (m model) View() string {
	var body string
	switch {
	case m.confirm != nil:
		body = m.viewConfirm()
	case m.screen == screenLogin:
		body = m.viewLogin()
	case m.screen == screenServerList:
		body = m.servers.View() + "\n" + m.styles.Help.Render("enter: open server  •  /: filter  •  q: quit")
	case m.screen == screenServerDetail:
		body = m.viewServerDetail()
	case m.screen == screenGamePicker:
		body = m.viewGamePicker()
	case m.screen == screenSettingsCats, m.screen == screenSettingsKeys, m.screen == screenSettingEdit:
		body = m.viewSettings()
	case m.screen == screenBackups:
		body = m.viewBackups()
	case m.screen == screenFiles, m.screen == screenFilePreview, m.screen == screenFileEdit:
		body = m.viewFiles()
	case m.screen == screenConsole:
		body = m.viewConsole()
	}

	var footer string
	if m.err != nil {
		footer = m.styles.Error.Render("Error: " + errMessage(m.err))
	} else if m.status != "" {
		footer = m.styles.Success.Render(m.status)
	}

	if m.updateNotice != "" {
		if footer != "" {
			footer += "\n"
		}
		footer += m.styles.Help.Render(m.updateNotice)
	}

	if footer == "" {
		return body
	}
	return body + "\n\n" + footer
}

func errMessage(err error) string {
	if e, ok := err.(*api.Error); ok {
		return e.FriendlyMessage()
	}
	return err.Error()
}

func (m model) viewLogin() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render("nitui — log in"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Normal.Render("Generate a long-life token at nitrado.net: My Account -> Developer Portal -> Long-life tokens."))
	b.WriteString("\n\n")
	b.WriteString(m.loginInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("enter: log in  •  esc: quit"))
	return m.styles.Border.Render(b.String())
}

func (m model) viewServerDetail() string {
	var b strings.Builder
	b.WriteString(m.styles.Title.Render(m.serverName()))
	b.WriteString("\n\n")

	switch {
	case m.detail != nil:
		d := m.detail
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Status:  %s", statusStyled(m, d.Status))))
		b.WriteString("\n")
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Game:    %s", d.GameHuman)))
		b.WriteString("\n")
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Address: %s", d.Address())))
		b.WriteString("\n")
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Slots:   %d", d.Slots)))
		if d.Location != "" {
			b.WriteString("\n")
			b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Region:  %s", d.Location)))
		}
		if u := d.UpdateStatus(); u != "" {
			b.WriteString("\n")
			b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Update:  %s", u)))
		}
	case m.detailLoad:
		b.WriteString(m.styles.Normal.Render("Loading..."))
	}

	if m.waiting != nil {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Warning.Render(m.waiting.describe(time.Now())))
	} else if m.switchBusy {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Warning.Render("Working..."))
	}

	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("r: restart  •  x: stop  •  u: update  •  s: games  •  e: settings  •  b: backups  •  f: files  •  c: console  •  esc: back  •  q: quit"))
	return m.styles.Border.Render(b.String())
}

func statusStyled(m model, status string) string {
	switch status {
	case "started", "running":
		return m.styles.Success.Render(status)
	case "stopped":
		return m.styles.Error.Render(status)
	default:
		return m.styles.Warning.Render(status)
	}
}

func (m model) viewGamePicker() string {
	if m.gamesLoad {
		return m.styles.Border.Render(m.styles.Normal.Render("Loading available games..."))
	}
	help := m.styles.Help.Render("enter: switch to / start this game  •  u: uninstall  •  esc: back")
	return m.games.View() + "\n" + help
}
