package tui

import (
	"fmt"
	"strings"

	"nitui/internal/api"
)

func (m model) View() string {
	var body string
	switch m.screen {
	case screenLogin:
		body = m.viewLogin()
	case screenServerList:
		body = m.servers.View()
	case screenServerDetail:
		body = m.viewServerDetail()
	case screenGamePicker:
		body = m.viewGamePicker()
	}

	var footer string
	if m.err != nil {
		footer = m.styles.Error.Render("Error: " + errMessage(m.err))
	} else if m.status != "" {
		footer = m.styles.Success.Render(m.status)
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
	b.WriteString(m.styles.Title.Render(fmt.Sprintf("Server %d", m.selectedSvc.ID)))
	b.WriteString("\n\n")

	switch {
	case m.detailLoad:
		b.WriteString(m.styles.Normal.Render("Loading..."))
	case m.detail != nil:
		d := m.detail
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Status:  %s", statusStyled(m, d.Status))))
		b.WriteString("\n")
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Game:    %s", d.GameHuman)))
		b.WriteString("\n")
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Address: %s", d.Address())))
		b.WriteString("\n")
		b.WriteString(m.styles.Normal.Render(fmt.Sprintf("Slots:   %d", d.Slots)))
	}

	if m.switchBusy {
		b.WriteString("\n\n")
		b.WriteString(m.styles.Warning.Render("Working..."))
	}

	b.WriteString("\n\n")
	b.WriteString(m.styles.Help.Render("s: switch game  •  r: restart  •  x: stop  •  esc: back  •  q: quit"))
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
	help := m.styles.Help.Render("enter: switch to this game  •  esc: back")
	return m.games.View() + "\n" + help
}
