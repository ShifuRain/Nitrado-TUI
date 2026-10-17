package tui

import (
	"fmt"
	pathpkg "path"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
)

// maxPreviewBytes caps what the TUI will load into the viewer/editor.
const maxPreviewBytes = 64 * 1024

func (m model) updateFiles(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case filesLoadedMsg:
		m.filesLoad = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.filesDir = msg.dir
		var items []list.Item
		if msg.dir != "/" && msg.dir != "" {
			items = append(items, rowItem{title: "..", desc: "parent directory", data: api.FileEntry{Type: "dir", Path: parentDir(msg.dir)}})
		}
		for _, e := range msg.entries {
			title, desc := e.Name, fmt.Sprintf("%d bytes", e.Size)
			if e.IsDir() {
				title, desc = e.Name+"/", "directory"
			}
			items = append(items, rowItem{title: title, desc: desc, data: e})
		}
		m.files.Title = "Files — " + msg.dir
		return m, m.files.SetItems(items)

	case fileReadMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if len(msg.data) > maxPreviewBytes || !utf8.Valid(msg.data) {
			m.status = "That file is binary or larger than 64 KB — use `nitui files get` instead."
			return m, nil
		}
		m.err = nil
		m.filePath = msg.path
		m.fileContent = string(msg.data)
		m.fileView.SetContent(m.fileContent)
		m.fileView.GotoTop()
		m.screen = screenFilePreview
		return m, nil

	case fileWrittenMsg:
		m.saveBusy = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.fileContent = m.fileEditor.Value()
		m.fileView.SetContent(m.fileContent)
		m.screen = screenFilePreview
		m.status = "Saved " + msg.path + "."
		return m, nil

	case fileDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.status = "Deleted " + msg.path + "."
		m.filesLoad = true
		return m, loadFiles(m.client, m.selectedSvc.ID, m.filesDir)
	}
	return m, nil
}

func parentDir(dir string) string {
	p := pathpkg.Dir(dir)
	if p == "." || p == "" {
		return "/"
	}
	return p
}

func (m model) handleFilesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	id := m.selectedSvc.ID

	switch m.screen {
	case screenFilePreview:
		switch msg.String() {
		case "esc", "q":
			m.screen = screenFiles
			return m, nil
		case "e":
			m.fileEditor.SetValue(m.fileContent)
			m.fileEditor.Focus()
			m.screen = screenFileEdit
			m.status = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.fileView, cmd = m.fileView.Update(msg)
		return m, cmd

	case screenFileEdit:
		switch msg.String() {
		case "esc":
			m.screen = screenFilePreview
			m.err = nil
			return m, nil
		case "ctrl+s":
			if m.saveBusy {
				return m, nil
			}
			m.saveBusy = true
			dir, name := pathpkg.Split(m.filePath)
			if dir == "" {
				dir = "/"
			}
			return m, writeRemoteFile(m.client, id, dir, name, []byte(m.fileEditor.Value()))
		}
		var cmd tea.Cmd
		m.fileEditor, cmd = m.fileEditor.Update(msg)
		return m, cmd
	}

	// screenFiles
	if m.files.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.files, cmd = m.files.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "esc":
		chain := (&m).enterDetail()
		return m, chain
	case "backspace", "h":
		if m.filesDir != "/" {
			m.filesLoad = true
			return m, loadFiles(m.client, id, parentDir(m.filesDir))
		}
		return m, nil
	case "enter":
		item, ok := m.files.SelectedItem().(rowItem)
		if !ok {
			return m, nil
		}
		e := item.data.(api.FileEntry)
		if e.IsDir() {
			m.filesLoad = true
			return m, loadFiles(m.client, id, e.Path)
		}
		m.status = ""
		return m, readRemoteFile(m.client, id, e.Path)
	case "d":
		item, ok := m.files.SelectedItem().(rowItem)
		if !ok {
			return m, nil
		}
		e := item.data.(api.FileEntry)
		if item.title == ".." {
			return m, nil
		}
		m.confirm = newConfirm(fmt.Sprintf("Delete %s? This cannot be undone.", e.Path), func(m model, _ string) (model, tea.Cmd) {
			return m, deleteRemoteFile(m.client, id, e.Path)
		})
		return m, nil
	}

	var cmd tea.Cmd
	m.files, cmd = m.files.Update(msg)
	return m, cmd
}

func (m model) viewFiles() string {
	switch m.screen {
	case screenFilePreview:
		return m.styles.Title.Render(m.filePath) + "\n" + m.fileView.View() + "\n" +
			m.styles.Help.Render("↑/↓ scroll  •  e: edit  •  esc: back")
	case screenFileEdit:
		busy := ""
		if m.saveBusy {
			busy = "  " + m.styles.Warning.Render("Saving...")
		}
		return m.styles.Title.Render("Editing "+m.filePath) + "\n" + m.fileEditor.View() + "\n" +
			m.styles.Help.Render("ctrl+s: save  •  esc: cancel") + busy
	}
	if m.filesLoad {
		return m.styles.Border.Render(m.styles.Normal.Render("Loading files..."))
	}
	return m.files.View() + "\n" +
		m.styles.Help.Render("enter: open  •  backspace: up  •  d: delete  •  esc: back")
}
