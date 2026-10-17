package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/update"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func detailModel(t *testing.T) model {
	t.Helper()
	m := newTestModel(t, auth.NewMemoryStore())
	m.client = api.New("t")
	m.screen = screenServerDetail
	m.selectedSvc = api.Service{ID: 7, Details: api.ServiceDetails{Name: "My Server"}}
	return m
}

func TestRestartKey_OpensConfirmWithMessageThenRestarts(t *testing.T) {
	m := detailModel(t)

	next, _ := m.handleServerDetailKey(key("r"))
	m = next.(model)
	if m.confirm == nil || !m.confirm.withInput {
		t.Fatal("r should open a confirmation with a message input")
	}

	// Typing goes to the message input; enter confirms.
	next, _ = m.Update(key("h"))
	m = next.(model)
	next, cmd := m.Update(key("enter"))
	m = next.(model)
	if m.confirm != nil || !m.switchBusy || cmd == nil {
		t.Fatalf("confirm=%v busy=%v cmd=%v, want confirmed + request issued", m.confirm, m.switchBusy, cmd)
	}
}

func TestConfirm_EscCancels(t *testing.T) {
	m := detailModel(t)
	next, _ := m.handleServerDetailKey(key("x"))
	m = next.(model)
	next, cmd := m.Update(key("esc"))
	m = next.(model)
	if m.confirm != nil || m.switchBusy || cmd != nil {
		t.Error("esc should cancel without doing anything")
	}
}

func TestLifecycleDone_StartsWaitingAndFinishesOnlyAfterRestartObserved(t *testing.T) {
	m := detailModel(t)
	next, cmd := m.Update(lifecycleDoneMsg{verb: "restart"})
	m = next.(model)
	if m.waiting == nil || cmd == nil {
		t.Fatal("expected a wait to start with an immediate poll")
	}
	gen := m.gen

	// Still "started" right after the request must NOT finish the wait.
	next, _ = m.Update(statusPolledMsg{gen: gen, detail: &api.GameServer{Status: "started"}})
	m = next.(model)
	if m.waiting == nil {
		t.Fatal("wait must not complete before the server leaves 'started'")
	}
	next, _ = m.Update(statusPolledMsg{gen: gen, detail: &api.GameServer{Status: "restarting"}})
	m = next.(model)
	if m.waiting == nil || m.detail.Status != "restarting" {
		t.Fatal("still waiting while restarting; detail should show live status")
	}
	next, cmd = m.Update(statusPolledMsg{gen: gen, detail: &api.GameServer{Status: "started"}})
	m = next.(model)
	if m.waiting != nil || m.status != "Server is running again." {
		t.Fatalf("waiting=%v status=%q, want completion", m.waiting, m.status)
	}
	if cmd == nil {
		t.Error("the passive refresh chain should continue")
	}
}

func TestLifecycleDone_StopCompletesOnStopped(t *testing.T) {
	m := detailModel(t)
	next, _ := m.Update(lifecycleDoneMsg{verb: "stop"})
	m = next.(model)
	next, _ = m.Update(statusPolledMsg{gen: m.gen, detail: &api.GameServer{Status: "stopping"}})
	m = next.(model)
	if m.waiting == nil {
		t.Fatal("stopping is not stopped")
	}
	next, _ = m.Update(statusPolledMsg{gen: m.gen, detail: &api.GameServer{Status: "stopped"}})
	m = next.(model)
	if m.waiting != nil || m.status != "Server stopped." {
		t.Fatalf("waiting=%v status=%q", m.waiting, m.status)
	}
}

func TestWait_TimesOut(t *testing.T) {
	m := detailModel(t)
	next, _ := m.Update(lifecycleDoneMsg{verb: "stop"})
	m = next.(model)
	m.waiting.watch = api.NewStatusWatch(api.WaitOptions{
		Want: func(s string) bool { return s == "stopped" }, Timeout: time.Nanosecond,
	}, time.Now().Add(-time.Hour))

	next, _ = m.Update(statusPolledMsg{gen: m.gen, detail: &api.GameServer{Status: "stopping"}})
	m = next.(model)
	if m.waiting != nil || m.err == nil {
		t.Fatalf("waiting=%v err=%v, want a give-up error", m.waiting, m.err)
	}
}

func TestStalePollsAndTicksAreIgnored(t *testing.T) {
	m := detailModel(t)
	m.gen = 5
	next, cmd := m.Update(statusPolledMsg{gen: 4, detail: &api.GameServer{Status: "stopped"}})
	if cmd != nil || next.(model).detail != nil {
		t.Error("a poll from an old chain must be dropped")
	}
	_, cmd = m.Update(tickMsg{gen: 4})
	if cmd != nil {
		t.Error("a tick from an old chain must be dropped")
	}
	_, cmd = m.Update(tickMsg{gen: 5})
	if cmd == nil {
		t.Error("a current tick should trigger a poll")
	}

	m.screen = screenSettingsCats
	_, cmd = m.Update(tickMsg{gen: 5})
	if cmd != nil {
		t.Error("ticks off the detail screen must stop the chain")
	}
}

func TestLeavingDetailInvalidatesChain(t *testing.T) {
	m := detailModel(t)
	m.gen = 3
	next, _ := m.handleServerDetailKey(key("esc"))
	if next.(model).gen == 3 {
		t.Error("leaving detail should bump gen so in-flight ticks die")
	}
}

func TestPassiveRefreshUpdatesDetailAndReschedules(t *testing.T) {
	m := detailModel(t)
	m.gen = 2
	next, cmd := m.Update(statusPolledMsg{gen: 2, detail: &api.GameServer{Status: "started", GameHuman: "Valheim"}})
	nm := next.(model)
	if nm.detail == nil || nm.detail.GameHuman != "Valheim" || cmd == nil {
		t.Fatal("poll should update detail and schedule the next tick")
	}
}

func TestLifecycleDone_ErrorSurfaces(t *testing.T) {
	m := detailModel(t)
	m.switchBusy = true
	next, _ := m.Update(lifecycleDoneMsg{verb: "restart", err: errors.New("busy")})
	nm := next.(model)
	if nm.err == nil || nm.switchBusy || nm.waiting != nil {
		t.Errorf("err=%v busy=%v waiting=%v", nm.err, nm.switchBusy, nm.waiting)
	}
}

func TestUpdateAvailable_ShowsBannerInView(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	next, _ := m.Update(updateAvailableMsg{release: &update.Release{Version: "9.9.9", URL: "https://example.test"}})
	v := next.(model).View()
	if !contains(v, "9.9.9") {
		t.Errorf("view should mention the new version:\n%s", v)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSettingsFlow(t *testing.T) {
	m := detailModel(t)
	next, cmd := m.handleServerDetailKey(key("e"))
	m = next.(model)
	if m.screen != screenSettingsCats || !m.settingsLoad || cmd == nil {
		t.Fatal("e should open settings and start loading")
	}

	gs := &api.GameServer{Settings: api.Settings{"config": {"name": "My World", "slots": "10"}}}
	defaults := api.Settings{"config": {"name": "Default", "slots": "10"}}
	next, _ = m.Update(settingsLoadedMsg{detail: gs, defaults: defaults})
	m = next.(model)
	if len(m.settingsCats.Items()) != 1 {
		t.Fatalf("categories = %d", len(m.settingsCats.Items()))
	}

	// enter category
	m.settingsCats.Select(0)
	next, _ = m.handleSettingsKey(key("enter"))
	m = next.(model)
	if m.screen != screenSettingsKeys || len(m.settingsKeys.Items()) != 2 {
		t.Fatalf("screen=%v keys=%d", m.screen, len(m.settingsKeys.Items()))
	}
	desc := m.settingsKeys.Items()[0].(rowItem).desc // "name" sorts before "slots"
	if !contains(desc, "default: Default") {
		t.Errorf("changed value should show its default, got %q", desc)
	}

	// edit the first key
	m.settingsKeys.Select(0)
	next, _ = m.handleSettingsKey(key("enter"))
	m = next.(model)
	if m.screen != screenSettingEdit || m.settingInput.Value() != "My World" {
		t.Fatalf("screen=%v input=%q", m.screen, m.settingInput.Value())
	}
	m.settingInput.SetValue("Renamed")
	next, cmd = m.handleSettingsKey(key("enter"))
	m = next.(model)
	if !m.saveBusy || cmd == nil {
		t.Fatal("enter should save")
	}

	// save result: back to keys, value updated, restart offered
	next, _ = m.Update(settingSavedMsg{category: "config", key: "name", value: "Renamed"})
	m = next.(model)
	if m.screen != screenSettingsKeys || m.confirm == nil {
		t.Fatalf("screen=%v confirm=%v, want keys screen + restart offer", m.screen, m.confirm)
	}
	if v, _ := m.settings.Get("config", "name"); v != "Renamed" {
		t.Errorf("local value = %q", v)
	}
}

func TestSettingSave_ErrorStaysOnEdit(t *testing.T) {
	m := detailModel(t)
	m.screen = screenSettingEdit
	m.saveBusy = true
	next, _ := m.Update(settingSavedMsg{err: errors.New("nope")})
	nm := next.(model)
	if nm.screen != screenSettingEdit || nm.err == nil || nm.saveBusy {
		t.Errorf("screen=%v err=%v busy=%v", nm.screen, nm.err, nm.saveBusy)
	}
}

func TestGamePicker_EnterStartsInstalledInstallsOthersBlocksActive(t *testing.T) {
	m := detailModel(t)
	m.screen = screenGamePicker
	items := []list.Item{
		gameItem{api.Game{Slug: "a", Name: "A", Installed: true, Active: true}},
		gameItem{api.Game{Slug: "b", Name: "B", Installed: true}},
		gameItem{api.Game{Slug: "c", Name: "C"}},
	}
	m.games.SetItems(items)

	m.games.Select(0)
	next, cmd := m.handleGamePickerKey(key("enter"))
	if cmd != nil || next.(model).switchBusy {
		t.Error("the active game should not trigger any request")
	}
	m.games.Select(1)
	next, cmd = m.handleGamePickerKey(key("enter"))
	if cmd == nil || !next.(model).switchBusy {
		t.Error("an installed game should be started")
	}
	m.games.Select(2)
	next, cmd = m.handleGamePickerKey(key("enter"))
	if cmd == nil || !next.(model).switchBusy {
		t.Error("a new game should be installed")
	}
}

func TestGamePicker_UninstallNeedsConfirmAndInstalledInactive(t *testing.T) {
	m := detailModel(t)
	m.screen = screenGamePicker
	m.games.SetItems([]list.Item{
		gameItem{api.Game{Slug: "a", Name: "A", Installed: true, Active: true}},
		gameItem{api.Game{Slug: "b", Name: "B", Installed: true}},
	})

	m.games.Select(0)
	next, _ := m.handleGamePickerKey(key("u"))
	if next.(model).confirm != nil {
		t.Error("the active game cannot be uninstalled")
	}
	m.games.Select(1)
	next, _ = m.handleGamePickerKey(key("u"))
	if next.(model).confirm == nil {
		t.Error("uninstall must ask for confirmation first")
	}
}

func TestBackupsKeys_RestoreAndDeleteAskFirst(t *testing.T) {
	m := detailModel(t)
	m.screen = screenBackups
	m.backups.SetItems([]list.Item{rowItem{title: "b1", data: "b1"}})
	m.backups.Select(0)

	for _, k := range []string{"enter", "d"} {
		next, cmd := m.handleBackupsKey(key(k))
		if next.(model).confirm == nil || cmd != nil {
			t.Errorf("%q must open a confirmation and send nothing yet", k)
		}
	}
}

func TestFiles_DirNavigationAndBinarySkipped(t *testing.T) {
	m := detailModel(t)
	m.screen = screenFiles
	next, _ := m.Update(filesLoadedMsg{dir: "/g/saves", entries: []api.FileEntry{
		{Type: "dir", Name: "sub", Path: "/g/saves/sub"},
		{Type: "file", Name: "a.cfg", Path: "/g/saves/a.cfg", Size: 3},
	}})
	m = next.(model)
	if n := len(m.files.Items()); n != 3 { // .., sub/, a.cfg
		t.Fatalf("items = %d, want 3 (with parent entry)", n)
	}

	m.files.Select(1) // sub/
	next, cmd := m.handleFilesKey(key("enter"))
	if cmd == nil || !next.(model).filesLoad {
		t.Error("entering a directory should load it")
	}

	next, _ = m.Update(fileReadMsg{path: "/g/x.bin", data: []byte{0xff, 0xfe, 0x00}})
	if next.(model).screen == screenFilePreview {
		t.Error("binary files must not open in the previewer")
	}
	next, _ = m.Update(fileReadMsg{path: "/g/a.cfg", data: []byte("hello")})
	nm := next.(model)
	if nm.screen != screenFilePreview || nm.fileContent != "hello" {
		t.Errorf("screen=%v content=%q", nm.screen, nm.fileContent)
	}
	next, _ = nm.handleFilesKey(key("e"))
	if next.(model).screen != screenFileEdit {
		t.Error("e should open the editor")
	}
}

func TestParentDir(t *testing.T) {
	cases := map[string]string{"/a/b": "/a", "/a": "/", "/": "/", "a": "/"}
	for in, want := range cases {
		if got := parentDir(in); got != want {
			t.Errorf("parentDir(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConsole_SendAndLog(t *testing.T) {
	m := detailModel(t)
	m.screen = screenConsole
	m.consoleInput.SetValue("say hi")
	next, cmd := m.handleConsoleKey(key("enter"))
	m = next.(model)
	if cmd == nil || !m.consoleBusy || m.consoleLog[len(m.consoleLog)-1] != "> say hi" {
		t.Fatalf("cmd=%v busy=%v log=%v", cmd, m.consoleBusy, m.consoleLog)
	}
	next, _ = m.Update(commandSentMsg{command: "say hi"})
	m = next.(model)
	if m.consoleBusy || m.consoleLog[len(m.consoleLog)-1] != "  sent." {
		t.Errorf("busy=%v log=%v", m.consoleBusy, m.consoleLog)
	}
	next, _ = m.Update(commandSentMsg{err: errors.New("not running")})
	m = next.(model)
	if last := m.consoleLog[len(m.consoleLog)-1]; last != "! not running" {
		t.Errorf("last = %q", last)
	}

	// blank input is ignored
	m.consoleInput.SetValue("  ")
	if _, cmd := m.handleConsoleKey(key("enter")); cmd != nil {
		t.Error("blank command should not be sent")
	}
}

func TestConsoleLog_IsBounded(t *testing.T) {
	var log []string
	for i := 0; i < maxConsoleLines+50; i++ {
		log = appendLog(log, "x")
	}
	if len(log) != maxConsoleLines {
		t.Errorf("len = %d, want %d", len(log), maxConsoleLines)
	}
}

func TestViews_RenderWithoutPanicking(t *testing.T) {
	m := detailModel(t)
	m.width, m.height = 100, 30
	m.detail = &api.GameServer{Status: "started", GameHuman: "Valheim", IP: "1.2.3.4", Port: 1, Location: "DE",
		GameSpecific: api.GameSpecific{UpdateStatus: "ok"}}
	for _, s := range []screen{screenLogin, screenServerList, screenServerDetail, screenGamePicker, screenSettingsCats,
		screenSettingsKeys, screenSettingEdit, screenBackups, screenFiles, screenFilePreview, screenFileEdit, screenConsole} {
		m.screen = s
		if m.View() == "" {
			t.Errorf("screen %v rendered nothing", s)
		}
	}
	m.confirm = newConfirmWithMessage("sure?", nil)
	if !contains(m.View(), "sure?") {
		t.Error("confirm modal should render its prompt")
	}
}
