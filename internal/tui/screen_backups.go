package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateBackups(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case backupsLoadedMsg:
		m.backupsLoad = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		items := make([]list.Item, len(msg.backups))
		for i, b := range msg.backups {
			desc := strings.TrimSpace(b.Created + "  " + b.Size)
			items[i] = rowItem{title: b.Name, desc: desc, data: b.Name}
		}
		return m, m.backups.SetItems(items)

	case backupActionMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = msg.verb + "."
		m.backupsLoad = true
		return m, loadBackups(m.client, m.selectedSvc.ID)
	}
	return m, nil
}

func (m model) handleBackupsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.backups.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.backups, cmd = m.backups.Update(msg)
		return m, cmd
	}
	id := m.selectedSvc.ID
	selected := func() (string, bool) {
		item, ok := m.backups.SelectedItem().(rowItem)
		if !ok {
			return "", false
		}
		return item.data.(string), true
	}

	switch msg.String() {
	case "esc":
		chain := (&m).enterDetail()
		return m, chain
	case "c":
		m.status = "Requesting a backup..."
		return m, doBackupAction("Backup requested (it may take a few minutes to appear)", func(ctx context.Context) error {
			return m.client.CreateBackup(ctx, id, "game")
		})
	case "R":
		m.backupsLoad = true
		return m, loadBackups(m.client, id)
	case "enter":
		name, ok := selected()
		if !ok {
			return m, nil
		}
		m.confirm = newConfirm(fmt.Sprintf("Restore %q? This overwrites the server's current files.", name), func(m model, _ string) (model, tea.Cmd) {
			return m, doBackupAction("Restoring "+name, func(ctx context.Context) error {
				return m.client.RestoreBackup(ctx, id, name)
			})
		})
		return m, nil
	case "d":
		name, ok := selected()
		if !ok {
			return m, nil
		}
		m.confirm = newConfirm(fmt.Sprintf("Delete backup %q?", name), func(m model, _ string) (model, tea.Cmd) {
			return m, doBackupAction("Deleted "+name, func(ctx context.Context) error {
				return m.client.DeleteBackup(ctx, id, name)
			})
		})
		return m, nil
	}

	var cmd tea.Cmd
	m.backups, cmd = m.backups.Update(msg)
	return m, cmd
}

func (m model) viewBackups() string {
	if m.backupsLoad {
		return m.styles.Border.Render(m.styles.Normal.Render("Loading backups..."))
	}
	return m.backups.View() + "\n" +
		m.styles.Help.Render("c: create backup  •  enter: restore  •  d: delete  •  R: refresh  •  esc: back")
}
