package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"nitui/internal/api"
	"nitui/internal/auth"
	"nitui/internal/config"
)

func newTestModel(t *testing.T, store auth.Store) model {
	t.Helper()
	t.Setenv("NITUI_CONFIG_DIR", t.TempDir())
	return newModel(config.Config{Theme: config.DefaultTheme()}, store)
}

func TestNewModel_ScreenDependsOnStoredToken(t *testing.T) {
	loggedOut := newTestModel(t, auth.NewMemoryStore())
	if loggedOut.screen != screenLogin {
		t.Errorf("screen with no token = %v, want screenLogin", loggedOut.screen)
	}

	store := auth.NewMemoryStore()
	_ = store.Save("a-token")
	loggedIn := newTestModel(t, store)
	if loggedIn.screen != screenServerList {
		t.Errorf("screen with a saved token = %v, want screenServerList", loggedIn.screen)
	}
}

func TestUpdate_LoginResult_Error(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	next, _ := m.Update(loginResultMsg{err: errors.New("boom")})
	nm := next.(model)

	if nm.err == nil {
		t.Error("err should be set after a failed login")
	}
	if nm.screen != screenLogin {
		t.Errorf("screen = %v, want to stay on screenLogin after a failed login", nm.screen)
	}
}

func TestUpdate_LoginResult_Success(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	next, cmd := m.Update(loginResultMsg{client: api.New("t")})
	nm := next.(model)

	if nm.screen != screenServerList {
		t.Errorf("screen = %v, want screenServerList after a successful login", nm.screen)
	}
	if nm.err != nil {
		t.Errorf("err = %v, want nil after a successful login", nm.err)
	}
	if cmd == nil {
		t.Error("expected a command to load servers after login")
	}
}

func TestUpdate_ServersLoaded_PopulatesList(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	next, _ := m.Update(serversLoadedMsg{services: []api.Service{{ID: 1}, {ID: 2}}})
	nm := next.(model)

	if len(nm.servers.Items()) != 2 {
		t.Errorf("len(servers.Items()) = %d, want 2", len(nm.servers.Items()))
	}
}

func TestUpdate_ActionDone_FromDetailScreenStaysOnDetail(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	m.screen = screenServerDetail
	m.selectedSvc = api.Service{ID: 7}
	m.switchBusy = true

	next, cmd := m.Update(actionDoneMsg{verb: "Stopped"})
	nm := next.(model)

	if nm.switchBusy {
		t.Error("switchBusy should be cleared after the action completes")
	}
	if nm.screen != screenServerDetail {
		t.Errorf("screen = %v, want to stay on screenServerDetail", nm.screen)
	}
	if nm.status != "Stopped." {
		t.Errorf("status = %q, want %q", nm.status, "Stopped.")
	}
	if !nm.detailLoad || cmd == nil {
		t.Error("expected detail to be reloaded after the action completes")
	}
}

func TestUpdate_ActionDone_FromGamePickerReturnsToDetail(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	m.screen = screenGamePicker
	m.selectedSvc = api.Service{ID: 7}

	next, _ := m.Update(actionDoneMsg{verb: "Switched"})
	nm := next.(model)

	if nm.screen != screenServerDetail {
		t.Errorf("screen = %v, want screenServerDetail after switching games", nm.screen)
	}
}

func TestUpdate_ActionDone_ErrorKeepsScreenAndSetsErr(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	m.screen = screenGamePicker

	next, _ := m.Update(actionDoneMsg{err: errors.New("limit reached")})
	nm := next.(model)

	if nm.err == nil {
		t.Error("err should be set when the action fails")
	}
	if nm.screen != screenGamePicker {
		t.Errorf("screen = %v, want to stay on screenGamePicker after a failed action", nm.screen)
	}
}

func TestHandleServerDetailKey_SwitchOpensGamePicker(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	m.screen = screenServerDetail
	m.selectedSvc = api.Service{ID: 7}

	next, cmd := m.handleServerDetailKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	nm := next.(model)

	if nm.screen != screenGamePicker {
		t.Errorf("screen = %v, want screenGamePicker", nm.screen)
	}
	if !nm.gamesLoad || cmd == nil {
		t.Error("expected games to start loading")
	}
}

func TestHandleGamePickerKey_EscReturnsToDetail(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())
	m.screen = screenGamePicker

	next, _ := m.handleGamePickerKey(tea.KeyMsg{Type: tea.KeyEsc})
	nm := next.(model)

	if nm.screen != screenServerDetail {
		t.Errorf("screen = %v, want screenServerDetail", nm.screen)
	}
}

func TestHandleLoginKey_EmptyInputDoesNothing(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())

	_, cmd := m.handleLoginKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command when submitting an empty token")
	}
}

func TestHandleLoginKey_EscQuits(t *testing.T) {
	m := newTestModel(t, auth.NewMemoryStore())

	_, cmd := m.handleLoginKey(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a quit command on esc")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected the command to produce a tea.QuitMsg")
	}
}
