package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
)

// updateSettings handles the async results for the settings screens.
func (m model) updateSettings(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsLoadedMsg:
		m.settingsLoad = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.detail = msg.detail
		m.settings = msg.detail.Settings
		if m.settings == nil {
			m.settings = api.Settings{}
		}
		m.settingDefaults = msg.defaults
		return m, m.rebuildSettings()

	case settingSavedMsg:
		m.saveBusy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		if m.settings[msg.category] == nil {
			m.settings[msg.category] = map[string]any{}
		}
		m.settings[msg.category][msg.key] = msg.value
		m.screen = screenSettingsKeys
		m.status = fmt.Sprintf("Saved %s/%s.", msg.category, msg.key)
		cmd := m.rebuildSettings()
		m.confirm = newConfirm("Restart the server now so the change takes effect?", m.restartAction)
		return m, cmd

	case settingsResetMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = "All settings reset to defaults."
		m.settingsLoad = true
		m.screen = screenSettingsCats
		return m, loadSettings(m.client, m.selectedSvc.ID)
	}
	return m, nil
}

// rebuildSettings refreshes both settings lists from m.settings.
func (m *model) rebuildSettings() tea.Cmd {
	cats := m.settings.Categories()
	catItems := make([]list.Item, len(cats))
	for i, c := range cats {
		catItems[i] = rowItem{
			title: c,
			desc:  fmt.Sprintf("%d settings%s", len(m.settings[c]), m.changedSuffix(c)),
			data:  c,
		}
	}
	cmd1 := m.settingsCats.SetItems(catItems)

	var keyItems []list.Item
	for _, k := range m.settings.Keys(m.settingCat) {
		val, _ := m.settings.Get(m.settingCat, k)
		desc := "value: " + val
		if def, ok := m.settingDefaults.Get(m.settingCat, k); ok {
			if def != val {
				desc += "   (default: " + def + ") *"
			}
		}
		keyItems = append(keyItems, rowItem{title: k, desc: desc, data: k})
	}
	cmd2 := m.settingsKeys.SetItems(keyItems)
	m.settingsKeys.Title = "Settings — " + m.settingCat
	return tea.Batch(cmd1, cmd2)
}

// changedSuffix notes how many settings in a category differ from default.
func (m model) changedSuffix(cat string) string {
	n := 0
	for k := range m.settings[cat] {
		val, _ := m.settings.Get(cat, k)
		if def, ok := m.settingDefaults.Get(cat, k); ok && def != val {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf(", %d changed", n)
}

func (m model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenSettingsCats:
		if m.settingsCats.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.settingsCats, cmd = m.settingsCats.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "esc":
			chain := (&m).enterDetail()
			return m, chain
		case "enter":
			item, ok := m.settingsCats.SelectedItem().(rowItem)
			if !ok {
				return m, nil
			}
			m.settingCat = item.data.(string)
			m.screen = screenSettingsKeys
			cmd := m.rebuildSettings()
			return m, cmd
		case "D":
			m.confirm = newConfirm("Reset ALL settings of this server to their defaults?", func(m model, _ string) (model, tea.Cmd) {
				return m, resetSettings(m.client, m.selectedSvc.ID)
			})
			return m, nil
		}
		var cmd tea.Cmd
		m.settingsCats, cmd = m.settingsCats.Update(msg)
		return m, cmd

	case screenSettingsKeys:
		if m.settingsKeys.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.settingsKeys, cmd = m.settingsKeys.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "esc":
			m.screen = screenSettingsCats
			return m, nil
		case "enter":
			item, ok := m.settingsKeys.SelectedItem().(rowItem)
			if !ok {
				return m, nil
			}
			m.settingKey = item.data.(string)
			val, _ := m.settings.Get(m.settingCat, m.settingKey)
			m.settingInput.SetValue(val)
			m.settingInput.CursorEnd()
			m.settingInput.Focus()
			m.screen = screenSettingEdit
			m.status, m.err = "", nil
			return m, nil
		}
		var cmd tea.Cmd
		m.settingsKeys, cmd = m.settingsKeys.Update(msg)
		return m, cmd

	case screenSettingEdit:
		switch msg.Type {
		case tea.KeyEsc:
			m.screen = screenSettingsKeys
			m.err = nil
			return m, nil
		case tea.KeyEnter:
			if m.saveBusy {
				return m, nil
			}
			m.saveBusy = true
			m.err = nil
			return m, saveSetting(m.client, m.selectedSvc.ID, m.settingCat, m.settingKey, m.settingInput.Value())
		}
		var cmd tea.Cmd
		m.settingInput, cmd = m.settingInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) viewSettings() string {
	switch m.screen {
	case screenSettingsCats:
		if m.settingsLoad {
			return m.styles.Border.Render(m.styles.Normal.Render("Loading settings..."))
		}
		if len(m.settings) == 0 {
			return m.styles.Border.Render(m.styles.Normal.Render("This server reports no editable settings.") + "\n\n" + m.styles.Help.Render("esc: back"))
		}
		return m.settingsCats.View() + "\n" + m.styles.Help.Render("enter: open category  •  D: reset all to defaults  •  esc: back")
	case screenSettingsKeys:
		return m.settingsKeys.View() + "\n" + m.styles.Help.Render("enter: edit  •  * = changed from default  •  esc: back")
	default: // screenSettingEdit
		var b strings.Builder
		b.WriteString(m.styles.Title.Render(m.settingCat + " / " + m.settingKey))
		b.WriteString("\n\n")
		if def, ok := m.settingDefaults.Get(m.settingCat, m.settingKey); ok {
			b.WriteString(m.styles.Help.Render("default: " + def))
			b.WriteString("\n\n")
		}
		b.WriteString(m.settingInput.View())
		if m.saveBusy {
			b.WriteString("\n\n" + m.styles.Warning.Render("Saving..."))
		}
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render("enter: save  •  esc: cancel"))
		return m.styles.Border.Render(b.String())
	}
}
